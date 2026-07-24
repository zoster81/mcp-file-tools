//go:build windows

package security

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestValidatePath_RejectsJunctionEscape(t *testing.T) {
	allowedDir := t.TempDir()
	outsideDir := t.TempDir()
	junction := filepath.Join(allowedDir, "escape")
	createJunctionForTest(t, outsideDir, junction)

	_, err := ValidatePath(junction, []string{allowedDir})
	if !errors.Is(err, ErrSymlinkDenied) {
		t.Fatalf("error = %v, want ErrSymlinkDenied", err)
	}

	for _, path := range []string{
		filepath.Join(junction, "new.txt"),
		filepath.Join(junction, "missing", "nested", "new.txt"),
	} {
		_, err := ValidatePath(path, []string{allowedDir})
		if !errors.Is(err, ErrParentDirDenied) {
			t.Fatalf("ValidatePath(%q) error = %v, want ErrParentDirDenied", path, err)
		}
	}

	resolved, safe := ResolvePathSafe(junction, ResolveAllowedDirs([]string{allowedDir}))
	if safe {
		t.Fatalf("junction escape resolved as safe: %s", resolved)
	}
}

func TestValidatePath_AllowsMissingPathThroughSafeJunction(t *testing.T) {
	allowedDir := t.TempDir()
	target := filepath.Join(allowedDir, "target")
	if err := os.Mkdir(target, 0755); err != nil {
		t.Fatal(err)
	}
	junction := filepath.Join(allowedDir, "safe-link")
	createJunctionForTest(t, target, junction)
	requested := filepath.Join(junction, "missing", "nested", "new.txt")

	validated, err := ValidatePath(requested, []string{allowedDir})
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Clean(validated) != filepath.Clean(requested) {
		t.Fatalf("validated path = %q, want %q", validated, requested)
	}
}

func TestResolveExistingPath_ResolvesJunctionTarget(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target")
	if err := os.Mkdir(target, 0755); err != nil {
		t.Fatal(err)
	}
	junction := filepath.Join(root, "junction")
	createJunctionForTest(t, target, junction)

	resolved, err := resolveExistingPath(junction)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Clean(resolved) != filepath.Clean(target) {
		t.Fatalf("resolved path = %q, want %q", resolved, target)
	}
}

func TestNormalizeWindowsFinalPath(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{input: `\\?\C:\work\file.txt`, want: `C:\work\file.txt`},
		{input: `\\?\UNC\server\share\file.txt`, want: `\\server\share\file.txt`},
		{input: `\??\C:\work\file.txt`, want: `C:\work\file.txt`},
		{input: `\??\UNC\server\share\file.txt`, want: `\\server\share\file.txt`},
	}
	for _, tt := range tests {
		if got := normalizeWindowsFinalPath(tt.input); got != tt.want {
			t.Errorf("normalizeWindowsFinalPath(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func createJunctionForTest(t *testing.T, target, link string) {
	t.Helper()
	output, err := exec.Command("cmd", "/c", "mklink", "/J", link, target).CombinedOutput()
	if err != nil {
		t.Skipf("directory junction creation is not supported in this environment: %v (%s)", err, output)
	}
}
