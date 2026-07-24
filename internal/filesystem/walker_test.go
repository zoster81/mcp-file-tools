package filesystem

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	"github.com/dimitar-grigorov/mcp-file-tools/internal/security"
)

func TestWalk_DeterministicOrderAndMetadata(t *testing.T) {
	root := t.TempDir()
	mustMkdirAll(t, filepath.Join(root, "a-dir"))
	mustMkdirAll(t, filepath.Join(root, "c-dir"))
	mustWriteFile(t, filepath.Join(root, "a-dir", "z.txt"))
	mustWriteFile(t, filepath.Join(root, "a-dir", "a.txt"))
	mustWriteFile(t, filepath.Join(root, "b.txt"))
	mustWriteFile(t, filepath.Join(root, "c-dir", "nested.txt"))

	allowedDirs := security.ResolveAllowedDirs([]string{root})
	var paths []string
	var depths []int
	err := Walk(context.Background(), root, WalkOptions{ResolvedAllowedDirs: allowedDirs}, func(entry Entry) (WalkAction, error) {
		paths = append(paths, filepath.ToSlash(entry.RelativePath))
		depths = append(depths, entry.Depth)
		if entry.Path == "" || entry.ResolvedPath == "" || entry.Name == "" || entry.DirEntry == nil {
			t.Fatalf("incomplete entry metadata: %+v", entry)
		}
		if !security.IsPathWithinAllowedDirectories(entry.ResolvedPath, allowedDirs) {
			t.Fatalf("resolved path escaped allowed directories: %s", entry.ResolvedPath)
		}
		return WalkContinue, nil
	})
	if err != nil {
		t.Fatal(err)
	}

	wantPaths := []string{"a-dir", "a-dir/a.txt", "a-dir/z.txt", "b.txt", "c-dir", "c-dir/nested.txt"}
	wantDepths := []int{1, 2, 2, 1, 1, 2}
	if !reflect.DeepEqual(paths, wantPaths) {
		t.Fatalf("paths = %v, want %v", paths, wantPaths)
	}
	if !reflect.DeepEqual(depths, wantDepths) {
		t.Fatalf("depths = %v, want %v", depths, wantDepths)
	}
}

func TestWalk_ExcludePrunesAndMaxDepth(t *testing.T) {
	root := t.TempDir()
	mustMkdirAll(t, filepath.Join(root, "keep", "deep"))
	mustMkdirAll(t, filepath.Join(root, "skip"))
	mustWriteFile(t, filepath.Join(root, "keep", "deep", "hidden.txt"))
	mustWriteFile(t, filepath.Join(root, "skip", "secret.txt"))
	mustWriteFile(t, filepath.Join(root, "root.txt"))

	var got []string
	err := Walk(context.Background(), root, WalkOptions{
		ResolvedAllowedDirs: security.ResolveAllowedDirs([]string{root}),
		MaxDepth:            2,
		Exclude: func(entry Entry) bool {
			return entry.Name == "skip"
		},
	}, func(entry Entry) (WalkAction, error) {
		got = append(got, filepath.ToSlash(entry.RelativePath))
		return WalkContinue, nil
	})
	if err != nil {
		t.Fatal(err)
	}

	want := []string{"keep", "keep/deep", "root.txt"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("visited = %v, want %v", got, want)
	}
}

func TestWalk_StopAndCancellation(t *testing.T) {
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "a.txt"))
	mustWriteFile(t, filepath.Join(root, "b.txt"))

	var visited []string
	err := Walk(context.Background(), root, WalkOptions{
		ResolvedAllowedDirs: security.ResolveAllowedDirs([]string{root}),
	}, func(entry Entry) (WalkAction, error) {
		visited = append(visited, entry.Name)
		return WalkStop, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(visited, []string{"a.txt"}) {
		t.Fatalf("visited = %v, want only first lexical entry", visited)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	visited = nil
	err = Walk(ctx, root, WalkOptions{
		ResolvedAllowedDirs: security.ResolveAllowedDirs([]string{root}),
	}, func(entry Entry) (WalkAction, error) {
		visited = append(visited, entry.Name)
		return WalkContinue, nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
	if len(visited) != 0 {
		t.Fatalf("visited %v after cancellation", visited)
	}
}

func TestWalk_OnErrorCanSkipChangedSubtree(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "a-dir")
	mustMkdirAll(t, dir)
	mustWriteFile(t, filepath.Join(dir, "child.txt"))
	mustWriteFile(t, filepath.Join(root, "b.txt"))

	var visited []string
	var errorDepth int
	var errorPath string
	err := Walk(context.Background(), root, WalkOptions{
		ResolvedAllowedDirs: security.ResolveAllowedDirs([]string{root}),
		OnError: func(path string, depth int, err error) error {
			errorPath = path
			errorDepth = depth
			return nil
		},
	}, func(entry Entry) (WalkAction, error) {
		visited = append(visited, filepath.ToSlash(entry.RelativePath))
		if entry.RelativePath == "a-dir" {
			if err := os.RemoveAll(entry.Path); err != nil {
				t.Fatal(err)
			}
			mustWriteFile(t, entry.Path)
		}
		return WalkContinue, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(visited, []string{"a-dir", "b.txt"}) {
		t.Fatalf("visited = %v, want traversal to continue after subtree error", visited)
	}
	if errorDepth != 1 || filepath.Clean(errorPath) != filepath.Clean(dir) {
		t.Fatalf("error callback = (%q, %d), want (%q, 1)", errorPath, errorDepth, dir)
	}
}

func TestWalk_SkipsDirectoryLinkEscape(t *testing.T) {
	allowedDir := t.TempDir()
	outsideDir := t.TempDir()
	mustWriteFile(t, filepath.Join(outsideDir, "secret.txt"))
	createDirectoryLinkForTest(t, outsideDir, filepath.Join(allowedDir, "escape"))
	mustWriteFile(t, filepath.Join(allowedDir, "keep.txt"))

	var got []string
	err := Walk(context.Background(), allowedDir, WalkOptions{
		ResolvedAllowedDirs: security.ResolveAllowedDirs([]string{allowedDir}),
	}, func(entry Entry) (WalkAction, error) {
		got = append(got, filepath.ToSlash(entry.RelativePath))
		return WalkContinue, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, []string{"keep.txt"}) {
		t.Fatalf("visited = %v, unsafe directory link must be skipped", got)
	}
}

func TestWalk_OnErrorReceivesUnsafeDirectorySwap(t *testing.T) {
	root := t.TempDir()
	outsideDir := t.TempDir()
	dir := filepath.Join(root, "a-dir")
	mustMkdirAll(t, dir)
	mustWriteFile(t, filepath.Join(root, "b.txt"))
	mustWriteFile(t, filepath.Join(outsideDir, "secret.txt"))

	var visited []string
	var errorPath string
	var errorDepth int
	err := Walk(context.Background(), root, WalkOptions{
		ResolvedAllowedDirs: security.ResolveAllowedDirs([]string{root}),
		OnError: func(path string, depth int, err error) error {
			errorPath = path
			errorDepth = depth
			return nil
		},
	}, func(entry Entry) (WalkAction, error) {
		visited = append(visited, filepath.ToSlash(entry.RelativePath))
		if entry.RelativePath == "a-dir" {
			if err := os.RemoveAll(entry.Path); err != nil {
				t.Fatal(err)
			}
			createDirectoryLinkForTest(t, outsideDir, entry.Path)
		}
		return WalkContinue, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(visited, []string{"a-dir", "b.txt"}) {
		t.Fatalf("visited = %v, want traversal to continue after unsafe swap", visited)
	}
	if filepath.Clean(errorPath) != filepath.Clean(dir) || errorDepth != 1 {
		t.Fatalf("error callback = (%q, %d), want (%q, 1)", errorPath, errorDepth, dir)
	}
}

func TestWalk_RejectsUnsafeRoot(t *testing.T) {
	allowedDir := t.TempDir()
	outsideDir := t.TempDir()

	err := Walk(context.Background(), outsideDir, WalkOptions{
		ResolvedAllowedDirs: security.ResolveAllowedDirs([]string{allowedDir}),
	}, func(Entry) (WalkAction, error) {
		t.Fatal("visitor must not receive entries for an unsafe root")
		return WalkContinue, nil
	})
	if err == nil {
		t.Fatal("expected unsafe root error")
	}
}

func createDirectoryLinkForTest(t *testing.T, target, link string) {
	t.Helper()
	if err := os.Symlink(target, link); err == nil {
		return
	} else if runtime.GOOS != "windows" {
		t.Skipf("directory symlink creation is not supported in this environment: %v", err)
	}

	output, err := exec.Command("cmd", "/c", "mklink", "/J", link, target).CombinedOutput()
	if err != nil {
		t.Skipf("directory junction creation is not supported in this environment: %v (%s)", err, output)
	}
}

func mustMkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0755); err != nil {
		t.Fatal(err)
	}
}

func mustWriteFile(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}
}
