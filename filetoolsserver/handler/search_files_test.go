package handler

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestHandleSearchFiles_SimplePattern(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	os.WriteFile(filepath.Join(tempDir, "file1.txt"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(tempDir, "file2.txt"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(tempDir, "file3.go"), []byte("test"), 0644)

	input := SearchFilesInput{Path: tempDir, Pattern: "*.txt"}

	result, output, err := h.HandleSearchFiles(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Errorf("expected success, got error")
	}
	if len(output.Files) != 2 {
		t.Errorf("expected 2 files, got %d: %v", len(output.Files), output.Files)
	}
}

func TestHandleSearchFiles_RecursivePattern(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	subDir := filepath.Join(tempDir, "subdir")
	os.Mkdir(subDir, 0755)
	deepDir := filepath.Join(subDir, "deep")
	os.Mkdir(deepDir, 0755)

	os.WriteFile(filepath.Join(tempDir, "root.txt"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(subDir, "sub.txt"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(deepDir, "deep.txt"), []byte("test"), 0644)

	input := SearchFilesInput{Path: tempDir, Pattern: "**/*.txt"}

	result, output, err := h.HandleSearchFiles(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Errorf("expected success, got error")
	}
	if len(output.Files) != 3 {
		t.Errorf("expected 3 files, got %d: %v", len(output.Files), output.Files)
	}
}

func TestHandleSearchFiles_WithExcludePatterns(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	subDir := filepath.Join(tempDir, "subdir")
	os.Mkdir(subDir, 0755)
	nodeModules := filepath.Join(tempDir, "node_modules")
	os.Mkdir(nodeModules, 0755)

	os.WriteFile(filepath.Join(tempDir, "root.txt"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(subDir, "sub.txt"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(nodeModules, "excluded.txt"), []byte("test"), 0644)

	input := SearchFilesInput{
		Path:            tempDir,
		Pattern:         "**/*.txt",
		ExcludePatterns: []string{"node_modules"},
	}

	result, output, err := h.HandleSearchFiles(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Errorf("expected success, got error")
	}
	if len(output.Files) != 2 {
		t.Errorf("expected 2 files (excluding node_modules), got %d: %v", len(output.Files), output.Files)
	}
}

func TestHandleSearchFiles_NoMatches(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	os.WriteFile(filepath.Join(tempDir, "file.go"), []byte("test"), 0644)

	input := SearchFilesInput{Path: tempDir, Pattern: "*.txt"}

	result, output, err := h.HandleSearchFiles(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Errorf("expected success, got error")
	}
	if len(output.Files) != 0 {
		t.Errorf("expected 0 files, got %d", len(output.Files))
	}
}

func TestHandleSearchFiles_ValidationErrors(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	tests := []struct {
		name  string
		input SearchFilesInput
	}{
		{"empty path", SearchFilesInput{Path: "", Pattern: "*.txt"}},
		{"empty pattern", SearchFilesInput{Path: tempDir, Pattern: ""}},
		{"outside allowed", SearchFilesInput{Path: "/random/path", Pattern: "*.txt"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, _, err := h.HandleSearchFiles(context.Background(), nil, tt.input)
			if err != nil {
				t.Fatal(err)
			}
			if !result.IsError {
				t.Errorf("expected error for %s", tt.name)
			}
		})
	}
}

func TestHandleSearchFiles_MaxResults(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	for i := 0; i < 20; i++ {
		os.WriteFile(filepath.Join(tempDir, fmt.Sprintf("file%03d.txt", i)), []byte("test"), 0644)
	}

	result, output, err := h.HandleSearchFiles(context.Background(), nil, SearchFilesInput{
		Path:       tempDir,
		Pattern:    "*.txt",
		MaxResults: 5,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Error("expected success")
	}
	if len(output.Files) != 5 {
		t.Errorf("expected 5 files (max), got %d", len(output.Files))
	}
	if !output.Truncated {
		t.Error("expected truncated to be true")
	}
}

func TestHandleSearchFiles_SkipsDirectoryLinkEscape(t *testing.T) {
	allowedDir := t.TempDir()
	outsideDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(outsideDir, "secret.txt"), []byte("secret"), 0644); err != nil {
		t.Fatal(err)
	}
	createDirectoryLinkForTest(t, outsideDir, filepath.Join(allowedDir, "escape"))

	h := NewHandler([]string{allowedDir})
	result, output, err := h.HandleSearchFiles(context.Background(), nil, SearchFilesInput{
		Path:    allowedDir,
		Pattern: "**",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("expected success, got %v", result.Content)
	}
	if len(output.Files) != 0 {
		t.Fatalf("unsafe directory link was returned: %v", output.Files)
	}
}

func TestHandleSearchFiles_DeterministicOrder(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "b-dir"), 0755); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{
		filepath.Join(root, "z.txt"),
		filepath.Join(root, "a.txt"),
		filepath.Join(root, "b-dir", "z.txt"),
		filepath.Join(root, "b-dir", "a.txt"),
	} {
		if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	h := NewHandler([]string{root})
	_, output, err := h.HandleSearchFiles(context.Background(), nil, SearchFilesInput{Path: root, Pattern: "**/*.txt"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		filepath.Join(root, "a.txt"),
		filepath.Join(root, "b-dir", "a.txt"),
		filepath.Join(root, "b-dir", "z.txt"),
		filepath.Join(root, "z.txt"),
	}
	if len(output.Files) != len(want) {
		t.Fatalf("files = %v, want %v", output.Files, want)
	}
	for i := range want {
		if !sameExistingTestFile(t, output.Files[i], want[i]) {
			t.Fatalf("files[%d] = %q, want same file as %q", i, output.Files[i], want[i])
		}
	}
}

func TestHandleSearchFiles_Cancelled(t *testing.T) {
	root := t.TempDir()
	h := NewHandler([]string{root})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result, _, err := h.HandleSearchFiles(ctx, nil, SearchFilesInput{Path: root, Pattern: "**"})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatal("expected cancellation error")
	}
	if text := extractTextFromResult(result.Content); text != "search cancelled" {
		t.Fatalf("error text = %q, want %q", text, "search cancelled")
	}
}
