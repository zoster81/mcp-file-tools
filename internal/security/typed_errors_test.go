package security

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/zoster81/mcp-file-tools/internal/operation"
)

func TestValidatePathReturnsTypedErrors(t *testing.T) {
	allowed := filepath.Join(t.TempDir(), "allowed")
	if err := os.MkdirAll(allowed, 0755); err != nil {
		t.Fatal(err)
	}

	_, err := ValidatePath(filepath.Join(filepath.Dir(allowed), "outside", "file.txt"), []string{allowed})
	if !errors.Is(err, ErrPathDenied) {
		t.Fatalf("error = %v, want ErrPathDenied", err)
	}
	if got := operation.KindOf(err); got != operation.KindAccessDenied {
		t.Fatalf("error kind = %v, want %v", got, operation.KindAccessDenied)
	}

	_, err = ValidatePath(filepath.Join(allowed, "file.txt"), nil)
	if !errors.Is(err, ErrNoAllowedDirs) {
		t.Fatalf("error = %v, want ErrNoAllowedDirs", err)
	}
	if got := operation.KindOf(err); got != operation.KindAccessDenied {
		t.Fatalf("error kind = %v, want %v", got, operation.KindAccessDenied)
	}
}

func TestNormalizeAllowedDirsReturnsTypedInvalidPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "file.txt")
	if err := os.WriteFile(path, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := NormalizeAllowedDirs([]string{path})
	if !errors.Is(err, ErrNotDirectory) {
		t.Fatalf("error = %v, want ErrNotDirectory", err)
	}
	if got := operation.KindOf(err); got != operation.KindInvalidPath {
		t.Fatalf("error kind = %v, want %v", got, operation.KindInvalidPath)
	}
}
