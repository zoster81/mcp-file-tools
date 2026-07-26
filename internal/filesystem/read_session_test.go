package filesystem

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestReadSessionSamplesWithoutMovingSequentialCursor(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.bin")
	original := []byte("BOM-payload-content")
	if err := os.WriteFile(path, original, 0o640); err != nil {
		t.Fatal(err)
	}

	session, err := OpenReadSession(path)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	sample := make([]byte, 7)
	if _, err := session.ReadAt(sample, 4); err != nil {
		t.Fatal(err)
	}
	if string(sample) != "payload" {
		t.Fatalf("sample = %q, want payload", sample)
	}
	if err := session.Start(4); err != nil {
		t.Fatal(err)
	}
	payload, err := io.ReadAll(session)
	if err != nil {
		t.Fatal(err)
	}
	if string(payload) != "payload-content" {
		t.Fatalf("payload = %q", payload)
	}

	snapshot, err := session.Finish()
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Size != int64(len(original)) {
		t.Fatalf("snapshot = %+v", snapshot)
	}
	if runtime.GOOS != "windows" && snapshot.Mode.Perm() != 0o640 {
		t.Fatalf("snapshot mode = %o, want 640", snapshot.Mode.Perm())
	}
	if err := snapshot.Verify(path); err != nil {
		t.Fatalf("snapshot verification failed: %v", err)
	}
}

func TestReadSessionFinishRequiresCompleteConsumption(t *testing.T) {
	path := filepath.Join(t.TempDir(), "partial.txt")
	if err := os.WriteFile(path, []byte("complete"), 0o600); err != nil {
		t.Fatal(err)
	}

	session, err := OpenReadSession(path)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	if err := session.Start(0); err != nil {
		t.Fatal(err)
	}
	buffer := make([]byte, 2)
	if _, err := session.Read(buffer); err != nil {
		t.Fatal(err)
	}
	if _, err := session.Finish(); !errors.Is(err, ErrIncompleteRead) {
		t.Fatalf("Finish error = %v, want ErrIncompleteRead", err)
	}
}

func TestReadSessionSnapshotDetectsLaterSameMetadataChange(t *testing.T) {
	path := filepath.Join(t.TempDir(), "changed.txt")
	original := []byte("first1")
	if err := os.WriteFile(path, original, 0o600); err != nil {
		t.Fatal(err)
	}

	session, err := OpenReadSession(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := session.Start(0); err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(io.Discard, session); err != nil {
		t.Fatal(err)
	}
	snapshot, err := session.Finish()
	if err != nil {
		t.Fatal(err)
	}
	if err := session.Close(); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("second"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, info.ModTime(), info.ModTime()); err != nil {
		t.Fatal(err)
	}
	if err := snapshot.Verify(path); !errors.Is(err, ErrConcurrentModification) {
		t.Fatalf("Verify error = %v, want ErrConcurrentModification", err)
	}
}

func TestReadSessionRejectsDirectory(t *testing.T) {
	if _, err := OpenReadSession(t.TempDir()); err == nil {
		t.Fatal("expected regular-file rejection")
	}
}

func TestReadSessionStartIncludesSkippedPrefixInDigest(t *testing.T) {
	path := filepath.Join(t.TempDir(), "prefix.bin")
	data := []byte{0xEF, 0xBB, 0xBF, 'a', 'b', 'c'}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	session, err := OpenReadSession(path)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	if err := session.Start(3); err != nil {
		t.Fatal(err)
	}
	payload, err := io.ReadAll(session)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(payload, []byte("abc")) {
		t.Fatalf("payload = %q", payload)
	}
	snapshot, err := session.Finish()
	if err != nil {
		t.Fatal(err)
	}
	if err := snapshot.Verify(path); err != nil {
		t.Fatal(err)
	}
}
