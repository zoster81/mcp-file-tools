//go:build !windows

package filesystem

import (
	"os"
	"path/filepath"
	"testing"
)

func sameTestPath(t *testing.T, first, second string) bool {
	t.Helper()
	firstInfo, firstErr := os.Stat(first)
	secondInfo, secondErr := os.Stat(second)
	if firstErr == nil && secondErr == nil && os.SameFile(firstInfo, secondInfo) {
		return true
	}
	return filepath.Clean(first) == filepath.Clean(second)
}

func TestSameTestPathResolvesDirectoryAliases(t *testing.T) {
	parent := t.TempDir()
	target := filepath.Join(parent, "target")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(parent, "alias")
	if err := os.Symlink(target, alias); err != nil {
		t.Skipf("directory symlink creation is unavailable: %v", err)
	}

	if !sameTestPath(t, alias, target) {
		t.Fatalf("paths %q and %q refer to the same directory", alias, target)
	}
}
