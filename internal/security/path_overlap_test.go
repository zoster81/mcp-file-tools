package security

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestPathsOverlapUsesRealComponentBoundaries(t *testing.T) {
	root := t.TempDir()
	child := filepath.Join(root, "child")
	sibling := root + "-sibling"

	if !PathsOverlap(root, root) {
		t.Fatal("equal paths did not overlap")
	}
	if !PathsOverlap(root, child) || !PathsOverlap(child, root) {
		t.Fatal("ancestor and descendant paths did not overlap symmetrically")
	}
	if PathsOverlap(root, sibling) {
		t.Fatalf("prefix lookalike %q overlapped %q", sibling, root)
	}
	if PathsOverlap(root, "relative") {
		t.Fatal("relative path was accepted for overlap comparison")
	}
}

func TestPathsOverlapUsesPlatformCaseSemantics(t *testing.T) {
	root := t.TempDir()
	variant := strings.ToUpper(root)
	got := PathsOverlap(root, variant)
	if runtime.GOOS == "windows" && !got {
		t.Fatal("Windows case variant did not overlap")
	}
	if runtime.GOOS != "windows" && variant != root && got {
		t.Fatal("case-distinct Unix path unexpectedly overlapped")
	}
}
