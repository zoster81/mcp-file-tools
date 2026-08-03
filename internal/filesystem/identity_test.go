package filesystem

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileIdentityDetectsSameContentPathReplacement(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "target.txt")
	if err := os.WriteFile(path, []byte("same"), 0644); err != nil {
		t.Fatal(err)
	}
	identity, err := OpenFileIdentity(path)
	if err != nil {
		t.Fatal(err)
	}
	defer identity.Close()

	matches, err := identity.Matches(path)
	if err != nil || !matches {
		t.Fatalf("initial identity match=%v err=%v", matches, err)
	}
	replacement := filepath.Join(root, "replacement.txt")
	if err := os.WriteFile(replacement, []byte("same"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(replacement, path); err != nil {
		t.Fatal(err)
	}

	matches, err = identity.Matches(path)
	if err != nil {
		t.Fatal(err)
	}
	if matches {
		t.Fatal("path replacement retained the original file identity")
	}
}
