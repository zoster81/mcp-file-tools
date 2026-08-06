package backupstore

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/zoster81/scripthold/internal/filesystem"
	"github.com/zoster81/scripthold/internal/operation"
)

type scannedObject struct {
	Digest     string
	Bytes      int64
	References int
}

type persistedIndex struct {
	FormatVersion    string `json:"formatVersion"`
	StoreID          string `json:"storeId"`
	GeneratedAt      string `json:"generatedAt"`
	Generation       string `json:"generation"`
	TotalObjectBytes int64  `json:"totalObjectBytes"`
	ObjectCount      int    `json:"objectCount"`
	ManifestCount    int    `json:"manifestCount"`
	PinnedCount      int    `json:"pinnedCount"`
	TargetCount      int    `json:"targetCount"`
}

func indexPath(root string) string {
	return filepath.Join(root, "index", "index-v1.json")
}

func buildIndex(descriptor Descriptor, manifests []Manifest, objects map[string]scannedObject) Index {
	sortedManifests := append([]Manifest(nil), manifests...)
	sort.Slice(sortedManifests, func(i, j int) bool {
		if sortedManifests[i].CreatedAt != sortedManifests[j].CreatedAt {
			return sortedManifests[i].CreatedAt < sortedManifests[j].CreatedAt
		}
		return sortedManifests[i].BackupID < sortedManifests[j].BackupID
	})

	index := Index{
		FormatVersion: "backup-index-v1",
		StoreID:       descriptor.StoreID,
		GeneratedAt:   utcTimestamp(time.Now()),
		Manifests:     make([]ManifestSummary, 0, len(sortedManifests)),
		Objects:       make([]ObjectSummary, 0, len(objects)),
	}
	targets := make(map[string]TargetSummary)
	for _, manifest := range sortedManifests {
		index.Manifests = append(index.Manifests, ManifestSummary{
			BackupID:           manifest.BackupID,
			CreatedAt:          manifest.CreatedAt,
			TargetPath:         manifest.TargetPath,
			SourceOperation:    manifest.SourceOperation,
			ObjectDigest:       manifest.ObjectDigest,
			ObjectBytes:        manifest.ObjectBytes,
			ContentFingerprint: manifest.ContentFingerprint,
			Pinned:             manifest.Pinned,
			ManifestChecksum:   manifest.ManifestChecksum,
		})
		index.ManifestCount++
		if manifest.Pinned {
			index.PinnedCount++
		}
		target := targets[manifest.TargetPath]
		target.TargetPath = manifest.TargetPath
		target.ManifestCount++
		if manifest.Pinned {
			target.PinnedCount++
		}
		if target.LatestAt == "" || manifest.CreatedAt > target.LatestAt {
			target.LatestAt = manifest.CreatedAt
		}
		targets[manifest.TargetPath] = target
	}

	digests := make([]string, 0, len(objects))
	for digest := range objects {
		digests = append(digests, digest)
	}
	sort.Strings(digests)
	for _, digest := range digests {
		object := objects[digest]
		index.Objects = append(index.Objects, ObjectSummary{Digest: digest, Bytes: object.Bytes, References: object.References})
		index.ObjectCount++
		if !addNonNegativeInt64(&index.TotalObjectBytes, object.Bytes) {
			index.TotalObjectBytes = int64(^uint64(0) >> 1)
		}
	}

	targetPaths := make([]string, 0, len(targets))
	for targetPath := range targets {
		targetPaths = append(targetPaths, targetPath)
	}
	sort.Strings(targetPaths)
	index.Targets = make([]TargetSummary, 0, len(targetPaths))
	for _, targetPath := range targetPaths {
		index.Targets = append(index.Targets, targets[targetPath])
	}
	index.Generation = indexGeneration(index)
	return index
}

func indexGeneration(index Index) string {
	hasher := sha256.New()
	_, _ = hasher.Write([]byte("mcp-file-tools:backup-index-generation-v1\x00"))
	for _, manifest := range index.Manifests {
		writeIndexString(hasher, manifest.BackupID)
		writeIndexString(hasher, manifest.ManifestChecksum)
	}
	for _, object := range index.Objects {
		writeIndexString(hasher, object.Digest)
		var encoded [16]byte
		binary.BigEndian.PutUint64(encoded[:8], uint64(object.Bytes))
		binary.BigEndian.PutUint64(encoded[8:], uint64(object.References))
		_, _ = hasher.Write(encoded[:])
	}
	return hex.EncodeToString(hasher.Sum(nil))
}

func writeIndexString(target io.Writer, value string) {
	var encoded [8]byte
	binary.BigEndian.PutUint64(encoded[:], uint64(len(value)))
	_, _ = target.Write(encoded[:])
	_, _ = target.Write([]byte(value))
}

func persistedIndexFrom(index Index) persistedIndex {
	return persistedIndex{
		FormatVersion:    index.FormatVersion,
		StoreID:          index.StoreID,
		GeneratedAt:      index.GeneratedAt,
		Generation:       index.Generation,
		TotalObjectBytes: index.TotalObjectBytes,
		ObjectCount:      index.ObjectCount,
		ManifestCount:    index.ManifestCount,
		PinnedCount:      index.PinnedCount,
		TargetCount:      len(index.Targets),
	}
}

func encodeIndex(index Index) ([]byte, error) {
	data, err := json.MarshalIndent(persistedIndexFrom(index), "", "  ")
	if err != nil {
		return nil, operation.New(operation.KindFilesystem, "backup index could not be encoded")
	}
	data = append(data, '\n')
	if len(data) > maxIndexBytes {
		return nil, operation.New(operation.KindLimit, "backup index exceeds the maximum encoded size")
	}
	return data, nil
}

func persistIndex(root string, index Index) error {
	data, err := encodeIndex(index)
	if err != nil {
		return err
	}
	path := indexPath(root)
	expected, err := filesystem.CaptureSnapshot(path)
	if err != nil {
		return sanitizedFilesystemError("backup index cannot be inspected", err)
	}
	if err := filesystem.ReplaceFile(path, data, filesystem.ReplaceOptions{Mode: 0o600, Expected: &expected}); err != nil {
		return sanitizedFilesystemError("backup index could not be replaced", err)
	}
	if err := restrictPathPermissions(path, false); err != nil {
		return sanitizedFilesystemError("backup index permissions could not be restricted", err)
	}
	info, err := os.Lstat(path)
	if err != nil {
		return sanitizedFilesystemError("backup index cannot be inspected after replacement", err)
	}
	if err := validateSingleLink(path, info); err != nil {
		return operation.New(operation.KindFilesystem, "backup index hard-link state is invalid")
	}
	return nil
}

func loadIndex(root string, descriptor Descriptor) (persistedIndex, error) {
	path := indexPath(root)
	lstatInfo, err := os.Lstat(path)
	if err != nil {
		return persistedIndex{}, err
	}
	if isLinkOrReparse(lstatInfo) || !lstatInfo.Mode().IsRegular() || lstatInfo.Size() > maxIndexBytes {
		return persistedIndex{}, errors.New("backup index metadata is invalid")
	}
	if err := validateSingleLink(path, lstatInfo); err != nil {
		return persistedIndex{}, errors.New("backup index hard-link state is invalid")
	}
	if err := validatePathPermissions(path, false); err != nil {
		return persistedIndex{}, err
	}
	file, err := os.Open(path)
	if err != nil {
		return persistedIndex{}, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return persistedIndex{}, err
	}
	if !info.Mode().IsRegular() || !os.SameFile(lstatInfo, info) || info.Size() > maxIndexBytes {
		return persistedIndex{}, errors.New("backup index identity or size is invalid")
	}
	return decodeIndex(io.LimitReader(file, maxIndexBytes+1), descriptor)
}

func decodeIndex(reader io.Reader, descriptor Descriptor) (persistedIndex, error) {
	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()
	var index persistedIndex
	if err := decoder.Decode(&index); err != nil {
		return persistedIndex{}, err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return persistedIndex{}, errors.New("backup index contains trailing data")
	}
	if err := validatePersistedIndex(index, descriptor); err != nil {
		return persistedIndex{}, err
	}
	return index, nil
}

func validatePersistedIndex(index persistedIndex, descriptor Descriptor) error {
	if index.FormatVersion != IndexVersion || index.StoreID != descriptor.StoreID || !validHexIdentifier(index.Generation) {
		return errors.New("backup index format is invalid")
	}
	if index.TotalObjectBytes < 0 || index.ObjectCount < 0 || index.ManifestCount < 0 ||
		index.PinnedCount < 0 || index.PinnedCount > index.ManifestCount || index.TargetCount < 0 ||
		index.TargetCount > index.ManifestCount {
		return errors.New("backup index counts are invalid")
	}
	generatedAt, err := time.Parse(time.RFC3339Nano, index.GeneratedAt)
	if err != nil || generatedAt.Location() != time.UTC || !strings.HasSuffix(index.GeneratedAt, "Z") {
		return errors.New("backup index timestamp is invalid")
	}
	return nil
}

func indexesEquivalent(first persistedIndex, second Index) bool {
	expected := persistedIndexFrom(second)
	first.GeneratedAt = ""
	expected.GeneratedAt = ""
	return reflect.DeepEqual(first, expected)
}

func cloneIndex(index Index) Index {
	copyIndex := index
	copyIndex.Manifests = append([]ManifestSummary(nil), index.Manifests...)
	copyIndex.Objects = append([]ObjectSummary(nil), index.Objects...)
	copyIndex.Targets = append([]TargetSummary(nil), index.Targets...)
	return copyIndex
}
