package handler

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestHandleConvertEncoding_ReplacesExistingBackupWithOriginal(t *testing.T) {
	dir := t.TempDir()
	h := NewHandler([]string{dir})
	path := filepath.Join(dir, "convert.txt")
	backupPath := path + ".bak"
	original := []byte("Привет")
	if err := os.WriteFile(path, original, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(backupPath, []byte("stale backup"), 0644); err != nil {
		t.Fatal(err)
	}

	result, output, err := h.HandleConvertEncoding(context.Background(), nil, ConvertEncodingInput{
		Path:   path,
		From:   "utf-8",
		To:     "cp1251",
		Backup: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("expected success, got %v", result.Content)
	}
	if !sameExistingTestFile(t, output.BackupPath, backupPath) {
		t.Fatalf("backup path = %q, want same file as %q", output.BackupPath, backupPath)
	}
	backup, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(backup, original) {
		t.Fatalf("backup = %q, want original", backup)
	}
}

func TestHandleCopyFile_CancelledLeavesDestinationMissing(t *testing.T) {
	dir := t.TempDir()
	h := NewHandler([]string{dir})
	source := filepath.Join(dir, "source.txt")
	destination := filepath.Join(dir, "destination.txt")
	if err := os.WriteFile(source, []byte("source"), 0644); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result, _, err := h.HandleCopyFile(ctx, nil, CopyFileInput{Source: source, Destination: destination})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatal("expected cancellation error")
	}
	if _, err := os.Stat(destination); !os.IsNotExist(err) {
		t.Fatalf("destination should not exist, stat error = %v", err)
	}
}

func TestHandleMoveFile_CancelledLeavesSourceUnchanged(t *testing.T) {
	dir := t.TempDir()
	h := NewHandler([]string{dir})
	source := filepath.Join(dir, "source.txt")
	destination := filepath.Join(dir, "destination.txt")
	if err := os.WriteFile(source, []byte("source"), 0644); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result, _, err := h.HandleMoveFile(ctx, nil, MoveFileInput{Source: source, Destination: destination})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatal("expected cancellation error")
	}
	if data, err := os.ReadFile(source); err != nil || string(data) != "source" {
		t.Fatalf("source = %q, err=%v; want source", data, err)
	}
	if _, err := os.Stat(destination); !os.IsNotExist(err) {
		t.Fatalf("destination should not exist, stat error = %v", err)
	}
}

func TestHandleDeleteFile_CancelledLeavesFileUnchanged(t *testing.T) {
	dir := t.TempDir()
	h := NewHandler([]string{dir})
	path := filepath.Join(dir, "target.txt")
	if err := os.WriteFile(path, []byte("target"), 0644); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result, _, err := h.HandleDeleteFile(ctx, nil, DeleteFileInput{Path: path})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatal("expected cancellation error")
	}
	if data, err := os.ReadFile(path); err != nil || string(data) != "target" {
		t.Fatalf("file = %q, err=%v; want target", data, err)
	}
}

func TestHandleManageBom_CancelledLeavesFileUnchanged(t *testing.T) {
	dir := t.TempDir()
	h := NewHandler([]string{dir})
	path := filepath.Join(dir, "target.txt")
	original := append([]byte{0xEF, 0xBB, 0xBF}, []byte("target")...)
	if err := os.WriteFile(path, original, 0644); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result, _, err := h.HandleManageBom(ctx, nil, ManageBomInput{Path: path, Action: "strip"})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatal("expected cancellation error")
	}
	actual, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(actual, original) {
		t.Fatal("cancelled BOM operation changed file")
	}
}
