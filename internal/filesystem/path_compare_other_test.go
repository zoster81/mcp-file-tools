//go:build !windows

package filesystem

import (
	"path/filepath"
	"testing"
)

func sameTestPath(t *testing.T, first, second string) bool {
	t.Helper()
	return filepath.Clean(first) == filepath.Clean(second)
}
