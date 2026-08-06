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

	"github.com/zoster81/scripthold/internal/operation"
)

const maxAuditIssues = 128

type scanOptions struct {
	mode       AuditMode
	maxObjects int
	maxBytes   int64
	checkIndex bool
}

type scanResult struct {
	manifests []Manifest
	objects   map[string]scannedObject
	report    AuditReport
}

// Audit performs a bounded read-only structural or complete integrity scan.
func (store *Store) Audit(ctx context.Context, options AuditOptions) (AuditReport, error) {
	if store == nil {
		return AuditReport{}, operation.New(operation.KindInvalidInput, "backup store is unavailable")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return AuditReport{}, operation.Wrap(operation.KindCancelled, "audit_backup_store", "", err)
	}
	mode := options.Mode
	if mode == "" {
		mode = AuditQuick
	}
	if mode != AuditQuick && mode != AuditFull {
		return AuditReport{}, operation.New(operation.KindInvalidInput, "backup audit mode is invalid")
	}
	maxObjects := options.MaxObjects
	if maxObjects < 0 {
		return AuditReport{}, operation.New(operation.KindInvalidInput, "backup audit object limit must not be negative")
	}
	if maxObjects == 0 {
		maxObjects = store.limits.MaxManifests
	}
	if maxObjects > store.limits.MaxManifests {
		return AuditReport{}, operation.New(operation.KindLimit, "backup audit object limit exceeds the configured maximum")
	}
	maxBytes := options.MaxBytes
	if maxBytes < 0 {
		return AuditReport{}, operation.New(operation.KindInvalidInput, "backup audit byte limit must not be negative")
	}
	if maxBytes == 0 {
		maxBytes = store.limits.MaxTotalBytes
	}
	if maxBytes > store.limits.MaxTotalBytes {
		return AuditReport{}, operation.New(operation.KindLimit, "backup audit byte limit exceeds the configured maximum")
	}

	store.transactionMu.Lock()
	defer store.transactionMu.Unlock()
	if store.isClosed() {
		return AuditReport{}, operation.New(operation.KindConflict, "backup store is closed")
	}
	if err := store.validateRootIdentity(); err != nil {
		return AuditReport{}, err
	}
	result, err := scanStore(ctx, store.root, store.descriptor, scanOptions{
		mode:       mode,
		maxObjects: maxObjects,
		maxBytes:   maxBytes,
		checkIndex: true,
	})
	if err != nil {
		return AuditReport{}, err
	}
	return result.report, nil
}

func scanStore(ctx context.Context, root string, descriptor Descriptor, options scanOptions) (scanResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return scanResult{}, operation.Wrap(operation.KindCancelled, "scan_backup_store", "", err)
	}
	if options.maxObjects <= 0 || options.maxBytes <= 0 {
		return scanResult{}, operation.New(operation.KindInvalidInput, "backup scan limits must be positive")
	}

	result := scanResult{
		objects: make(map[string]scannedObject),
		report: AuditReport{
			Mode: options.mode,
		},
	}
	addIssue := func(code, message string) {
		if len(result.report.Issues) >= maxAuditIssues {
			if len(result.report.Issues) == maxAuditIssues {
				result.report.Issues = append(result.report.Issues, AuditIssue{Code: AuditIssueLimit, Message: "additional audit issues were truncated"})
			}
			return
		}
		result.report.Issues = append(result.report.Issues, AuditIssue{Code: code, Message: message})
	}
	if rootErr := validateRootEntries(root); rootErr != nil {
		addIssue(AuditIssueStoreEntry, "backup store root layout is invalid")
	}

	manifestEntries, manifestOverflow, err := readDirectoryBounded(filepath.Join(root, "manifests"), options.maxObjects)
	if err != nil {
		return scanResult{}, sanitizedFilesystemError("backup manifests cannot be inspected", err)
	}
	if manifestOverflow {
		addIssue(AuditIssueLimit, "manifest count exceeds the audit object limit")
	}
	manifestIDs := make(map[string]struct{}, len(manifestEntries))
	references := make(map[string]int)
	expectedObjectBytes := make(map[string]int64)
	for _, entry := range manifestEntries {
		if err := ctx.Err(); err != nil {
			return scanResult{}, operation.Wrap(operation.KindCancelled, "scan_backup_manifests", "", err)
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".json") || !validHexIdentifier(strings.TrimSuffix(name, ".json")) {
			addIssue(AuditIssueManifest, "manifest filename is invalid")
			continue
		}
		path := filepath.Join(root, "manifests", name)
		info, statErr := os.Lstat(path)
		if statErr != nil {
			addIssue(AuditIssueManifest, "manifest metadata cannot be inspected")
			continue
		}
		manifest, readErr := readManifest(path, info, descriptor)
		if readErr != nil {
			message := "manifest is invalid"
			if strings.Contains(readErr.Error(), "checksum") {
				message = "manifest checksum is invalid"
			}
			addIssue(AuditIssueManifest, message)
			continue
		}
		fileID := strings.TrimSuffix(name, ".json")
		if manifest.BackupID != fileID {
			addIssue(AuditIssueManifest, "manifest filename does not match its backup identifier")
			continue
		}
		if _, duplicate := manifestIDs[manifest.BackupID]; duplicate {
			addIssue(AuditIssueManifest, "duplicate backup identifier")
			continue
		}
		manifestIDs[manifest.BackupID] = struct{}{}
		if expectedBytes, exists := expectedObjectBytes[manifest.ObjectDigest]; exists && expectedBytes != manifest.ObjectBytes {
			addIssue(AuditIssueManifest, "manifests disagree about object size")
			continue
		}
		expectedObjectBytes[manifest.ObjectDigest] = manifest.ObjectBytes
		result.manifests = append(result.manifests, manifest)
		references[manifest.ObjectDigest]++
	}
	result.report.ManifestCount = len(result.manifests)

	objectRoot := filepath.Join(root, "objects", "sha256")
	objectRootInfo, err := os.Lstat(objectRoot)
	if err != nil || isLinkOrReparse(objectRootInfo) || !objectRootInfo.IsDir() || validatePathPermissions(objectRoot, true) != nil {
		addIssue(AuditIssueStoreEntry, "object algorithm directory is invalid")
	}
	shards, shardOverflow, err := readDirectoryBounded(objectRoot, 256)
	if err != nil {
		return scanResult{}, sanitizedFilesystemError("backup objects cannot be inspected", err)
	}
	if shardOverflow {
		addIssue(AuditIssueStoreEntry, "object algorithm directory contains too many shards")
	}
	objectCount := 0
	var fullAuditBytes int64
	for _, shard := range shards {
		if err := ctx.Err(); err != nil {
			return scanResult{}, operation.Wrap(operation.KindCancelled, "scan_backup_objects", "", err)
		}
		shardName := shard.Name()
		shardPath := filepath.Join(objectRoot, shardName)
		info, statErr := os.Lstat(shardPath)
		if statErr != nil || len(shardName) != 2 || !isLowerHex(shardName) || isLinkOrReparse(info) || !info.IsDir() {
			addIssue(AuditIssueStoreEntry, "object shard is invalid")
			continue
		}
		if permissionErr := validatePathPermissions(shardPath, true); permissionErr != nil {
			addIssue(AuditIssueStoreEntry, "object shard permissions are invalid")
			continue
		}
		remainingObjects := options.maxObjects - objectCount
		if remainingObjects < 0 {
			remainingObjects = 0
		}
		entries, entryOverflow, readErr := readDirectoryBounded(shardPath, remainingObjects)
		if readErr != nil {
			addIssue(AuditIssueStoreEntry, "object shard cannot be inspected")
			continue
		}
		if entryOverflow {
			addIssue(AuditIssueLimit, "object count exceeds the audit limit")
		}
		for _, entry := range entries {
			if err := ctx.Err(); err != nil {
				return scanResult{}, operation.Wrap(operation.KindCancelled, "scan_backup_objects", "", err)
			}
			objectCount++
			digest := entry.Name()
			path := filepath.Join(shardPath, digest)
			objectInfo, statErr := os.Lstat(path)
			if statErr != nil || !validHexIdentifier(digest) || digest[:2] != shardName ||
				isLinkOrReparse(objectInfo) || !objectInfo.Mode().IsRegular() {
				addIssue(AuditIssueObjectMetadata, "object metadata is invalid")
				continue
			}
			if linkErr := validateSingleLink(path, objectInfo); linkErr != nil {
				addIssue(AuditIssueObjectMetadata, "object hard-link state is invalid")
				continue
			}
			if permissionErr := validatePathPermissions(path, false); permissionErr != nil {
				addIssue(AuditIssueObjectMetadata, "object permissions are invalid")
				continue
			}
			object := scannedObject{Digest: digest, Bytes: objectInfo.Size(), References: references[digest]}
			result.objects[digest] = object
			if object.References == 0 {
				result.report.OrphanObjectCount++
				if !addNonNegativeInt64(&result.report.OrphanObjectBytes, object.Bytes) {
					addIssue(AuditIssueLimit, "orphan object bytes exceed the supported range")
				}
			} else if !addNonNegativeInt64(&result.report.ReferencedBytes, object.Bytes) {
				addIssue(AuditIssueLimit, "referenced object bytes exceed the supported range")
			}
			if options.mode == AuditFull && object.References > 0 {
				if object.Bytes < 0 || object.Bytes > options.maxBytes-fullAuditBytes {
					addIssue(AuditIssueLimit, "referenced object bytes exceed the full-audit limit")
					continue
				}
				fullAuditBytes += object.Bytes
				actual, hashErr := hashRegularFile(ctx, path, object.Bytes)
				if hashErr != nil {
					if operation.KindOf(hashErr) == operation.KindCancelled {
						return scanResult{}, hashErr
					}
					addIssue(AuditIssueObjectMetadata, "object could not be hashed")
					continue
				}
				if actual != digest {
					addIssue(AuditIssueObjectDigest, "object digest does not match its identifier")
				}
			}
		}
	}
	result.report.ObjectCount = len(result.objects)
	for digest := range references {
		object, exists := result.objects[digest]
		if !exists {
			addIssue(AuditIssueObjectMissing, "referenced object is missing")
			continue
		}
		if expectedObjectBytes[digest] != object.Bytes {
			addIssue(AuditIssueObjectMetadata, "referenced object size does not match its manifest")
		}
	}

	var residualErr error
	result.report.StagingEntryCount, result.report.StagingEntryBytes, residualErr = scanResidualDirectory(ctx, filepath.Join(root, "staging"), options.maxObjects, addIssue)
	if residualErr != nil {
		return scanResult{}, residualErr
	}
	result.report.TrashEntryCount, result.report.TrashEntryBytes, residualErr = scanResidualDirectory(ctx, filepath.Join(root, "trash"), options.maxObjects, addIssue)
	if residualErr != nil {
		return scanResult{}, residualErr
	}

	indexEntries, indexOverflow, indexReadErr := readDirectoryBounded(filepath.Join(root, "index"), 1)
	if indexReadErr != nil {
		addIssue(AuditIssueStoreEntry, "index directory cannot be inspected")
	} else {
		if indexOverflow {
			addIssue(AuditIssueStoreEntry, "index directory contains unexpected entries")
		}
		for _, entry := range indexEntries {
			if entry.Name() != "index-v1.json" {
				addIssue(AuditIssueStoreEntry, "index directory contains an unexpected entry")
			}
		}
	}

	rebuilt := buildIndex(descriptor, result.manifests, result.objects)
	result.report.Generation = rebuilt.Generation
	if options.checkIndex {
		persisted, indexErr := loadIndex(root, descriptor)
		if indexErr != nil || !indexesEquivalent(persisted, rebuilt) {
			addIssue(AuditIssueIndex, "derived index is missing, corrupt, or stale")
			result.report.IndexConsistent = false
		} else {
			result.report.IndexConsistent = true
		}
	} else {
		result.report.IndexConsistent = true
	}
	result.report.Healthy = len(result.report.Issues) == 0
	return result, nil
}

func scanResidualDirectory(ctx context.Context, root string, maxEntries int, addIssue func(string, string)) (count int, bytes int64, err error) {
	if maxEntries <= 0 {
		return 0, 0, operation.New(operation.KindInvalidInput, "recovery entry limit must be positive")
	}
	rootInfo, err := os.Lstat(root)
	if err != nil || isLinkOrReparse(rootInfo) || !rootInfo.IsDir() {
		addIssue(AuditIssueStoreEntry, "recovery directory is invalid")
		return 0, 0, nil
	}
	if permissionErr := validatePathPermissions(root, true); permissionErr != nil {
		addIssue(AuditIssueStoreEntry, "recovery directory permissions are invalid")
	}

	pending := []string{root}
	visited := 0
	for len(pending) > 0 {
		if contextErr := ctx.Err(); contextErr != nil {
			return 0, 0, operation.Wrap(operation.KindCancelled, "scan_backup_recovery", "", contextErr)
		}
		directory := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		remaining := maxEntries - visited
		if remaining <= 0 {
			addIssue(AuditIssueLimit, "recovery entry count exceeds the audit limit")
			break
		}
		entries, overflow, readErr := readDirectoryBounded(directory, remaining)
		if readErr != nil {
			addIssue(AuditIssueStoreEntry, "recovery directory cannot be scanned")
			continue
		}
		if overflow {
			addIssue(AuditIssueLimit, "recovery entry count exceeds the audit limit")
		}
		for _, entry := range entries {
			visited++
			path := filepath.Join(directory, entry.Name())
			info, infoErr := os.Lstat(path)
			if infoErr != nil || isLinkOrReparse(info) {
				addIssue(AuditIssueStoreEntry, "recovery directory contains an invalid entry")
				continue
			}
			if info.IsDir() {
				if permissionErr := validatePathPermissions(path, true); permissionErr != nil {
					addIssue(AuditIssueStoreEntry, "recovery directory permissions are invalid")
					continue
				}
				pending = append(pending, path)
				continue
			}
			if !info.Mode().IsRegular() {
				addIssue(AuditIssueStoreEntry, "recovery directory contains a special file")
				continue
			}
			if linkErr := validateSingleLink(path, info); linkErr != nil {
				addIssue(AuditIssueStoreEntry, "recovery file hard-link state is invalid")
				continue
			}
			if permissionErr := validatePathPermissions(path, false); permissionErr != nil {
				addIssue(AuditIssueStoreEntry, "recovery file permissions are invalid")
				continue
			}
			count++
			if !addNonNegativeInt64(&bytes, info.Size()) {
				addIssue(AuditIssueLimit, "recovery byte count exceeds the supported range")
				continue
			}
		}
		if overflow {
			break
		}
	}
	return count, bytes, nil
}

func hashRegularFile(ctx context.Context, path string, expectedSize int64) (string, error) {
	lstatInfo, err := os.Lstat(path)
	if err != nil || isLinkOrReparse(lstatInfo) || !lstatInfo.Mode().IsRegular() || lstatInfo.Size() != expectedSize {
		return "", operation.New(operation.KindFilesystem, "backup object metadata changed during audit")
	}
	if err := validateSingleLink(path, lstatInfo); err != nil {
		return "", operation.New(operation.KindFilesystem, "backup object hard-link state changed during audit")
	}
	if err := validatePathPermissions(path, false); err != nil {
		return "", sanitizedFilesystemError("backup object permissions are not owner-only", err)
	}
	file, err := os.Open(path)
	if err != nil {
		return "", sanitizedFilesystemError("backup object cannot be opened", err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() != expectedSize || !os.SameFile(lstatInfo, info) {
		return "", operation.New(operation.KindFilesystem, "backup object identity changed during audit")
	}
	hasher := sha256.New()
	buffer := make([]byte, 128*1024)
	remaining := expectedSize
	for remaining > 0 {
		if err := ctx.Err(); err != nil {
			return "", operation.Wrap(operation.KindCancelled, "hash_backup_object", "", err)
		}
		readSize := int64(len(buffer))
		if remaining < readSize {
			readSize = remaining
		}
		read, readErr := io.ReadFull(file, buffer[:readSize])
		if read > 0 {
			_, _ = hasher.Write(buffer[:read])
			remaining -= int64(read)
		}
		if readErr != nil {
			return "", operation.New(operation.KindFilesystem, "backup object was truncated during audit")
		}
	}
	var extra [1]byte
	if read, readErr := file.Read(extra[:]); read != 0 || (readErr != nil && !errors.Is(readErr, io.EOF)) {
		return "", operation.New(operation.KindFilesystem, "backup object size changed during audit")
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func isLowerHex(value string) bool {
	if value == "" || value != strings.ToLower(value) {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func firstStructuralIssue(report AuditReport) error {
	for _, issue := range report.Issues {
		if issue.Code == AuditIssueIndex {
			continue
		}
		kind := operation.KindFilesystem
		if issue.Code == AuditIssueLimit {
			kind = operation.KindLimit
		}
		return operation.New(kind, issue.Message)
	}
	return nil
}
