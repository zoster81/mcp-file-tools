package backupstore

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/zoster81/mcp-file-tools/internal/filesystem"
)

func FuzzDecodeManifest(f *testing.F) {
	descriptor := fuzzDescriptor()
	objectDigest := sha256.Sum256([]byte("manifest fuzz seed"))
	digestText := hex.EncodeToString(objectDigest[:])
	fingerprint, err := filesystem.FingerprintRegularFileContentDigest(int64(len("manifest fuzz seed")), digestText)
	if err != nil {
		f.Fatal(err)
	}
	manifest, err := finalizeManifestChecksum(Manifest{
		FormatVersion:      ManifestVersion,
		StoreFormatVersion: FormatVersion,
		StoreID:            descriptor.StoreID,
		BackupID:           strings.Repeat("b", 64),
		CreatedAt:          "2026-08-04T17:00:00Z",
		TargetPath:         fuzzAbsolutePath(),
		SourceOperation:    SourceOperationEdit,
		ObjectAlgorithm:    ObjectAlgorithm,
		ObjectDigest:       digestText,
		ObjectBytes:        int64(len("manifest fuzz seed")),
		ContentFingerprint: fingerprint,
		OriginalMode:       0o600,
		OriginalModTime:    "2026-08-04T16:59:00Z",
	})
	if err != nil {
		f.Fatal(err)
	}
	valid, err := encodeManifest(manifest)
	if err != nil {
		f.Fatal(err)
	}
	f.Add(valid)
	f.Add([]byte{})
	f.Add([]byte(`{"formatVersion":"backup-manifest-v1"}`))

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > maxManifestBytes+1 {
			t.Skip()
		}
		_, _ = decodeManifest(io.LimitReader(bytes.NewReader(data), maxManifestBytes+1), descriptor)
	})
}

func FuzzDecodeIndex(f *testing.F) {
	descriptor := fuzzDescriptor()
	index := buildIndex(descriptor, nil, map[string]scannedObject{})
	valid, err := encodeIndex(index)
	if err != nil {
		f.Fatal(err)
	}
	f.Add(valid)
	f.Add([]byte{})
	f.Add([]byte(`{"formatVersion":"backup-index-v1"}`))

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 1024*1024 {
			t.Skip()
		}
		_, _ = decodeIndex(io.LimitReader(bytes.NewReader(data), maxIndexBytes+1), descriptor)
	})
}

func fuzzAbsolutePath() string {
	if runtime.GOOS == "windows" {
		return `C:\backup\target.txt`
	}
	return "/backup/target.txt"
}

func fuzzDescriptor() Descriptor {
	return Descriptor{
		FormatVersion:   FormatVersion,
		StoreID:         strings.Repeat("a", 64),
		CreatedAt:       time.Date(2026, 8, 4, 17, 0, 0, 0, time.UTC).Format(time.RFC3339Nano),
		ObjectAlgorithm: ObjectAlgorithm,
		ManifestVersion: ManifestVersion,
		IndexVersion:    IndexVersion,
	}
}
