package handler

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/zoster81/scripthold/internal/config"
)

func TestHandleFingerprintPaths(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "data.txt")
	if err := os.WriteFile(path, []byte("payload"), 0644); err != nil {
		t.Fatal(err)
	}
	h := NewHandler([]string{root})
	result, output, err := h.HandleFingerprintPaths(context.Background(), nil, FingerprintPathsInput{
		Paths:          []string{path},
		IncludeEntries: true,
	})
	if err != nil || result.IsError {
		t.Fatalf("result=%+v output=%+v err=%v", result, output, err)
	}
	if output.Algorithm != "sha256" || output.Mode != "content-v1" || len(output.Fingerprint) != 64 {
		t.Fatalf("unexpected fingerprint metadata: %+v", output)
	}
	if output.RootCount != 1 || output.FileCount != 1 || output.DirectoryCount != 0 || output.TotalBytes != 7 {
		t.Fatalf("unexpected counts: %+v", output)
	}
	if len(output.Entries) != 1 || output.Entries[0].Path != "." || output.Entries[0].SHA256 == "" {
		t.Fatalf("unexpected entries: %+v", output.Entries)
	}
}

func TestHandleFingerprintPathsValidatesLimitsAndPaths(t *testing.T) {
	root := t.TempDir()
	first := filepath.Join(root, "first.txt")
	second := filepath.Join(root, "second.txt")
	for _, path := range []string{first, second} {
		if err := os.WriteFile(path, []byte(path), 0644); err != nil {
			t.Fatal(err)
		}
	}
	h := NewHandler([]string{root}, WithConfig(&config.Config{
		DefaultEncoding: "utf-8",
		Limits: config.Limits{
			MaxFileBytes:               config.DefaultMaxFileBytes,
			MaxDecodedCharacters:       config.DefaultMaxDecodedCharacters,
			MaxLineBytes:               config.DefaultMaxLineBytes,
			MaxBatchFiles:              1,
			MaxMatches:                 config.DefaultMaxMatches,
			MaxOutputBytes:             config.DefaultMaxOutputBytes,
			MaxSessions:                config.DefaultMaxSessions,
			MaxFingerprintEntries:      10,
			MaxFingerprintEntryDetails: 2,
		},
	}))

	result, _, err := h.HandleFingerprintPaths(context.Background(), nil, FingerprintPathsInput{Paths: []string{first, second}})
	if err != nil || result.Meta[ErrorCodeMetaKey] != ErrCodeLimit {
		t.Fatalf("batch limit result=%+v err=%v", result, err)
	}

	result, _, err = h.HandleFingerprintPaths(context.Background(), nil, FingerprintPathsInput{Paths: []string{""}})
	if err != nil || result.Meta[ErrorCodeMetaKey] != ErrCodeInvalidInput {
		t.Fatalf("empty path result=%+v err=%v", result, err)
	}

	result, _, err = h.HandleFingerprintPaths(context.Background(), nil, FingerprintPathsInput{
		Paths:           []string{first},
		IncludeEntries:  true,
		MaxEntryDetails: 3,
	})
	if err != nil || result.Meta[ErrorCodeMetaKey] != ErrCodeLimit {
		t.Fatalf("detail limit result=%+v err=%v", result, err)
	}

	outputLimited := NewHandler([]string{root}, WithConfig(&config.Config{
		DefaultEncoding: "utf-8",
		Limits: config.Limits{
			MaxOutputBytes: 1,
		},
	}))
	result, _, err = outputLimited.HandleFingerprintPaths(context.Background(), nil, FingerprintPathsInput{
		Paths:          []string{first},
		IncludeEntries: true,
	})
	if err != nil || result.Meta[ErrorCodeMetaKey] != ErrCodeLimit {
		t.Fatalf("output limit result=%+v err=%v", result, err)
	}
}
