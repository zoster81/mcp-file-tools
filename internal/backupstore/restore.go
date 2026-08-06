package backupstore

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/zoster81/scripthold/internal/filesystem"
	"github.com/zoster81/scripthold/internal/operation"
)

// RestoreSourceOptions authorizes the original manifest target before any
// referenced object bytes are opened or hashed. The callback runs outside store
// locks.
type RestoreSourceOptions struct {
	AuthorizeTarget func(string) error
}

// RestoreSource retains verified identities for one immutable manifest and its
// referenced object. It exposes exact bytes only through bounded reads or the
// durable filesystem staging API, never through internal store paths.
type RestoreSource struct {
	store        *Store
	manifest     Manifest
	manifestFile *os.File
	manifestInfo os.FileInfo
	objectFile   *os.File
	objectInfo   os.FileInfo

	mu                sync.Mutex
	closed            bool
	referenceRetained bool
	closeErr          error
}

// OpenRestoreSource authorizes, opens, and fully verifies one immutable backup
// source for an original-target restore preview.
func (store *Store) OpenRestoreSource(ctx context.Context, backupID string, options RestoreSourceOptions) (*RestoreSource, error) {
	if store == nil {
		return nil, operation.New(operation.KindInvalidInput, "backup store is unavailable")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	manifest, err := store.readInspectManifest(ctx, backupID)
	if err != nil {
		return nil, err
	}
	if options.AuthorizeTarget != nil {
		if err := options.AuthorizeTarget(manifest.TargetPath); err != nil {
			return nil, err
		}
	}

	store.transactionMu.Lock()
	defer store.transactionMu.Unlock()
	if store.isClosed() {
		return nil, operation.New(operation.KindConflict, "backup store is closed")
	}
	if err := store.validateIdentityAndLayout(); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, operation.Wrap(operation.KindCancelled, "open_restore_source", "", err)
	}

	manifestPath := manifestPath(store.root, backupID)
	manifestLstat, err := os.Lstat(manifestPath)
	if err != nil {
		return nil, operation.New(operation.KindConflict, "backup manifest changed before restore preview")
	}
	currentManifest, err := readManifest(manifestPath, manifestLstat, store.descriptor)
	if err != nil || currentManifest != manifest {
		return nil, operation.New(operation.KindConflict, "backup manifest changed before restore preview")
	}
	manifestFile, manifestInfo, err := openImmutableRestoreFile(manifestPath, manifestLstat, manifest.ObjectBytes, false)
	if err != nil {
		return nil, operation.New(operation.KindFilesystem, "backup manifest identity is invalid")
	}

	objectPath := objectPath(store.root, manifest.ObjectDigest)
	objectLstat, err := os.Lstat(objectPath)
	if err != nil {
		_ = manifestFile.Close()
		return nil, operation.New(operation.KindFilesystem, "referenced backup object is unavailable")
	}
	if err := validateRestoreObjectMetadata(objectPath, objectLstat, manifest); err != nil {
		_ = manifestFile.Close()
		return nil, err
	}
	objectFile, objectInfo, err := openImmutableRestoreFile(objectPath, objectLstat, manifest.ObjectBytes, true)
	if err != nil {
		_ = manifestFile.Close()
		return nil, operation.New(operation.KindFilesystem, "referenced backup object identity is invalid")
	}

	source := &RestoreSource{
		store:        store,
		manifest:     manifest,
		manifestFile: manifestFile,
		manifestInfo: manifestInfo,
		objectFile:   objectFile,
		objectInfo:   objectInfo,
	}
	if err := source.verifyUnderTransaction(ctx, true); err != nil {
		_ = source.closeFiles()
		return nil, err
	}
	store.retainRestoreReference(manifest)
	source.referenceRetained = true
	return source, nil
}

// RestorePlanTTL returns the configured one-shot restore plan lifetime.
func (store *Store) RestorePlanTTL() time.Duration {
	if store == nil || store.limits.PlanTTLSeconds <= 0 {
		return 0
	}
	return time.Duration(store.limits.PlanTTLSeconds) * time.Second
}

// RestoreObjectLimit returns the configured maximum object size for bounded
// target fingerprinting and preview work.
func (store *Store) RestoreObjectLimit() int64 {
	if store == nil {
		return 0
	}
	return store.limits.MaxObjectBytes
}

// Manifest returns a detached immutable manifest value.
func (source *RestoreSource) Manifest() Manifest {
	if source == nil {
		return Manifest{}
	}
	source.mu.Lock()
	defer source.mu.Unlock()
	return source.manifest
}

// Verify revalidates the manifest and object identities and fully hashes the
// referenced object.
func (source *RestoreSource) Verify(ctx context.Context) error {
	if source == nil {
		return operation.New(operation.KindConflict, "restore source is unavailable")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	source.mu.Lock()
	defer source.mu.Unlock()
	if source.closed {
		return operation.New(operation.KindConflict, "restore source is closed")
	}
	source.store.transactionMu.Lock()
	defer source.store.transactionMu.Unlock()
	return source.verifyUnderTransaction(ctx, true)
}

// ReadAll returns exact verified object bytes under an explicit caller bound.
// Restore plans must not retain the returned bytes after preview construction.
func (source *RestoreSource) ReadAll(ctx context.Context, maxBytes int64) ([]byte, error) {
	if source == nil {
		return nil, operation.New(operation.KindConflict, "restore source is unavailable")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	source.mu.Lock()
	defer source.mu.Unlock()
	if source.closed {
		return nil, operation.New(operation.KindConflict, "restore source is closed")
	}
	if maxBytes <= 0 {
		return nil, operation.New(operation.KindInvalidInput, "restore read limit must be positive")
	}
	if source.manifest.ObjectBytes > maxBytes {
		return nil, operation.New(operation.KindLimit, "restore object exceeds the requested read limit")
	}
	if int64(int(source.manifest.ObjectBytes)) != source.manifest.ObjectBytes {
		return nil, operation.New(operation.KindLimit, "restore object size exceeds supported memory range")
	}
	if err := source.verifyStructure(ctx); err != nil {
		return nil, err
	}
	if _, err := source.objectFile.Seek(0, io.SeekStart); err != nil {
		return nil, operation.New(operation.KindFilesystem, "restore object could not be read")
	}
	data := make([]byte, int(source.manifest.ObjectBytes))
	hasher := sha256.New()
	for offset := 0; offset < len(data); {
		if err := ctx.Err(); err != nil {
			return nil, operation.Wrap(operation.KindCancelled, "read_restore_source", "", err)
		}
		end := min(offset+128*1024, len(data))
		read, readErr := io.ReadFull(source.objectFile, data[offset:end])
		if read > 0 {
			_, _ = hasher.Write(data[offset : offset+read])
			offset += read
		}
		if readErr != nil {
			return nil, operation.New(operation.KindFilesystem, "restore object size changed while reading")
		}
	}
	if hex.EncodeToString(hasher.Sum(nil)) != source.manifest.ObjectDigest {
		return nil, operation.New(operation.KindFilesystem, "restore object digest does not match its manifest")
	}
	if err := source.verifyStructure(ctx); err != nil {
		return nil, err
	}
	return data, nil
}

// Stage streams exact verified object bytes into a synced target-adjacent
// replacement and rejects any staged digest mismatch.
func (source *RestoreSource) Stage(ctx context.Context, target string, mode os.FileMode, modTime *time.Time) (*filesystem.StagedReplacement, error) {
	if source == nil {
		return nil, operation.New(operation.KindConflict, "restore source is unavailable")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	cleanTarget := filepath.Clean(target)
	if target == "" || strings.Contains(target, "\x00") || !filepath.IsAbs(cleanTarget) || cleanTarget != target {
		return nil, operation.New(operation.KindInvalidPath, "restore target path must be normalized and absolute")
	}
	source.mu.Lock()
	defer source.mu.Unlock()
	if source.closed {
		return nil, operation.New(operation.KindConflict, "restore source is closed")
	}
	if err := source.verifyStructure(ctx); err != nil {
		return nil, err
	}
	if _, err := source.objectFile.Seek(0, io.SeekStart); err != nil {
		return nil, operation.New(operation.KindFilesystem, "restore object could not be staged")
	}
	staged, err := filesystem.StageReplacementExactMode(target, io.LimitReader(source.objectFile, source.manifest.ObjectBytes), mode, modTime)
	if err != nil {
		return nil, err
	}
	digestBytes, _ := hex.DecodeString(source.manifest.ObjectDigest)
	var expectedDigest [sha256.Size]byte
	copy(expectedDigest[:], digestBytes)
	if !staged.MatchesContentDigest(source.manifest.ObjectBytes, expectedDigest) {
		cleanupErr := staged.Cleanup()
		return nil, errors.Join(operation.New(operation.KindConflict, "staged restore bytes do not match the backup object"), cleanupErr)
	}
	if err := source.verifyStructure(ctx); err != nil {
		return nil, errors.Join(err, staged.Cleanup())
	}
	return staged, nil
}

// Close releases retained manifest and object identities. It is idempotent.
func (source *RestoreSource) Close() error {
	if source == nil {
		return nil
	}
	source.mu.Lock()
	defer source.mu.Unlock()
	if source.closed {
		return source.closeErr
	}
	source.closed = true
	source.closeErr = source.closeFiles()
	if source.referenceRetained && source.store != nil {
		source.store.releaseRestoreReference(source.manifest)
		source.referenceRetained = false
	}
	return source.closeErr
}

func (source *RestoreSource) closeFiles() error {
	var closeErr error
	if source.objectFile != nil {
		closeErr = errors.Join(closeErr, source.objectFile.Close())
		source.objectFile = nil
	}
	if source.manifestFile != nil {
		closeErr = errors.Join(closeErr, source.manifestFile.Close())
		source.manifestFile = nil
	}
	return closeErr
}

func (source *RestoreSource) verifyStructure(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return operation.Wrap(operation.KindCancelled, "verify_restore_source", "", err)
	}
	source.store.transactionMu.Lock()
	defer source.store.transactionMu.Unlock()
	return source.verifyUnderTransaction(ctx, false)
}

func (source *RestoreSource) verifyUnderTransaction(ctx context.Context, hashObject bool) error {
	if source.store == nil || source.manifestFile == nil || source.objectFile == nil {
		return operation.New(operation.KindConflict, "restore source identity is unavailable")
	}
	if source.store.isClosed() {
		return operation.New(operation.KindConflict, "backup store is closed")
	}
	if err := source.store.validateIdentityAndLayout(); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return operation.Wrap(operation.KindCancelled, "verify_restore_source", "", err)
	}

	manifestPath := manifestPath(source.store.root, source.manifest.BackupID)
	manifestInfo, err := os.Lstat(manifestPath)
	if err != nil || !immutableFileIdentityMatches(source.manifestInfo, manifestInfo) {
		return operation.New(operation.KindConflict, "backup manifest changed after restore preview")
	}
	currentManifest, err := readManifest(manifestPath, manifestInfo, source.store.descriptor)
	if err != nil || currentManifest != source.manifest {
		return operation.New(operation.KindConflict, "backup manifest changed after restore preview")
	}
	openManifestInfo, err := source.manifestFile.Stat()
	if err != nil || !immutableFileIdentityMatches(source.manifestInfo, openManifestInfo) {
		return operation.New(operation.KindConflict, "backup manifest identity changed after restore preview")
	}

	objectPath := objectPath(source.store.root, source.manifest.ObjectDigest)
	objectInfo, err := os.Lstat(objectPath)
	if err != nil {
		return operation.New(operation.KindConflict, "backup object changed after restore preview")
	}
	if err := validateRestoreObjectMetadata(objectPath, objectInfo, source.manifest); err != nil {
		return err
	}
	if !immutableFileIdentityMatches(source.objectInfo, objectInfo) {
		return operation.New(operation.KindConflict, "backup object identity changed after restore preview")
	}
	openObjectInfo, err := source.objectFile.Stat()
	if err != nil || !immutableFileIdentityMatches(source.objectInfo, openObjectInfo) {
		return operation.New(operation.KindConflict, "backup object identity changed after restore preview")
	}
	if !hashObject {
		return nil
	}
	return hashOpenRestoreObject(ctx, source.objectFile, source.manifest)
}

func openImmutableRestoreFile(path string, lstatInfo os.FileInfo, expectedSize int64, requireExpectedSize bool) (*os.File, os.FileInfo, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, nil, err
	}
	if !info.Mode().IsRegular() || !os.SameFile(lstatInfo, info) || (requireExpectedSize && info.Size() != expectedSize) {
		_ = file.Close()
		return nil, nil, errors.New("immutable file identity is invalid")
	}
	return file, info, nil
}

func immutableFileIdentityMatches(expected, actual os.FileInfo) bool {
	return expected != nil && actual != nil && os.SameFile(expected, actual) &&
		expected.Mode() == actual.Mode() && expected.Size() == actual.Size() && expected.ModTime().Equal(actual.ModTime())
}

func validateRestoreObjectMetadata(path string, info os.FileInfo, manifest Manifest) error {
	if info == nil || isLinkOrReparse(info) || !info.Mode().IsRegular() || info.Size() != manifest.ObjectBytes {
		return operation.New(operation.KindFilesystem, "backup object metadata is invalid")
	}
	if err := validateSingleLink(path, info); err != nil {
		return operation.New(operation.KindFilesystem, "backup object hard-link state is invalid")
	}
	if err := validatePathPermissions(path, false); err != nil {
		return sanitizedFilesystemError("backup object permissions are not owner-only", err)
	}
	return nil
}

func hashOpenRestoreObject(ctx context.Context, file *os.File, manifest Manifest) error {
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return operation.New(operation.KindFilesystem, "backup object could not be read")
	}
	hasher := sha256.New()
	buffer := make([]byte, 128*1024)
	remaining := manifest.ObjectBytes
	for remaining > 0 {
		if err := ctx.Err(); err != nil {
			return operation.Wrap(operation.KindCancelled, "verify_restore_object", "", err)
		}
		readSize := int64(len(buffer))
		if remaining < readSize {
			readSize = remaining
		}
		read, readErr := io.ReadFull(file, buffer[:int(readSize)])
		if read > 0 {
			_, _ = hasher.Write(buffer[:read])
			remaining -= int64(read)
		}
		if readErr != nil {
			return operation.New(operation.KindFilesystem, "backup object size does not match its manifest")
		}
	}
	var extra [1]byte
	if read, readErr := file.Read(extra[:]); read != 0 || (readErr != nil && !errors.Is(readErr, io.EOF)) {
		return operation.New(operation.KindFilesystem, "backup object size does not match its manifest")
	}
	if hex.EncodeToString(hasher.Sum(nil)) != manifest.ObjectDigest {
		return operation.New(operation.KindFilesystem, "backup object digest does not match its manifest")
	}
	_, _ = file.Seek(0, io.SeekStart)
	return nil
}
