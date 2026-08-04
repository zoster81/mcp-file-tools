package backupstore

import (
	"errors"
	"io"
	"os"
	"sort"
)

// readDirectoryBounded reads at most limit+1 entries, allowing callers to
// detect saturation without materializing an attacker-controlled directory.
func readDirectoryBounded(path string, limit int) ([]os.DirEntry, bool, error) {
	if limit < 0 {
		return nil, false, errors.New("directory entry limit must not be negative")
	}
	directory, err := os.Open(path)
	if err != nil {
		return nil, false, err
	}
	defer directory.Close()
	entries, readErr := directory.ReadDir(limit + 1)
	if readErr != nil && !errors.Is(readErr, io.EOF) {
		return nil, false, readErr
	}
	overflow := len(entries) > limit
	if overflow {
		entries = entries[:limit]
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})
	return entries, overflow, nil
}
