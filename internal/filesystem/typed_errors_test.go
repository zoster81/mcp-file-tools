package filesystem

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/zoster81/scripthold/internal/operation"
)

func TestCopyFileReturnsTypedNotFound(t *testing.T) {
	dir := t.TempDir()
	err := CopyFile(filepath.Join(dir, "missing.txt"), filepath.Join(dir, "destination.txt"))
	if got := operation.KindOf(err); got != operation.KindNotFound {
		t.Fatalf("error kind = %v, want %v; error=%v", got, operation.KindNotFound, err)
	}
}

func TestCopyFileReturnsTypedConflict(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source.txt")
	destination := filepath.Join(dir, "destination.txt")
	if err := os.WriteFile(source, []byte("source"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(destination, []byte("destination"), 0644); err != nil {
		t.Fatal(err)
	}

	err := CopyFile(source, destination)
	if !errors.Is(err, ErrDestinationExists) {
		t.Fatalf("error = %v, want ErrDestinationExists", err)
	}
	if got := operation.KindOf(err); got != operation.KindConflict {
		t.Fatalf("error kind = %v, want %v", got, operation.KindConflict)
	}
}

func TestSnapshotVerifyReturnsTypedConflict(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "target.txt")
	original := []byte("original")
	if err := os.WriteFile(path, original, 0644); err != nil {
		t.Fatal(err)
	}
	snapshot, err := CaptureSnapshotWithData(path, original)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("changed"), 0644); err != nil {
		t.Fatal(err)
	}

	err = snapshot.Verify(path)
	if !errors.Is(err, ErrConcurrentModification) {
		t.Fatalf("error = %v, want ErrConcurrentModification", err)
	}
	if got := operation.KindOf(err); got != operation.KindConflict {
		t.Fatalf("error kind = %v, want %v", got, operation.KindConflict)
	}
}
