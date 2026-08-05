package filesystem

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestCaptureSnapshotWithDigestDetectsSameMetadataContentChange(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "script.bin")
	if err := os.WriteFile(path, []byte("first1"), 0o700); err != nil {
		t.Fatal(err)
	}

	snapshot, err := CaptureSnapshotWithDigest(path)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("second"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, info.ModTime(), info.ModTime()); err != nil {
		t.Fatal(err)
	}

	if err := snapshot.Verify(path); !errors.Is(err, ErrConcurrentModification) {
		t.Fatalf("Verify() error = %v, want ErrConcurrentModification", err)
	}
}

func TestCaptureSnapshotWithDigestRejectsDirectory(t *testing.T) {
	_, err := CaptureSnapshotWithDigest(t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "not a regular file") {
		t.Fatalf("CaptureSnapshotWithDigest() error = %v, want regular-file rejection", err)
	}
}

func TestStageReplacementExactModePreservesZeroPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows mode bits do not represent Unix permission mode 0000")
	}
	dir := t.TempDir()
	target := filepath.Join(dir, "restored.bin")
	expected, err := CaptureSnapshot(target)
	if err != nil {
		t.Fatal(err)
	}
	modTime := time.Unix(1_700_000_000, 0).UTC()
	staged, err := StageReplacementExactMode(target, strings.NewReader("restored"), 0, &modTime)
	if err != nil {
		t.Fatal(err)
	}
	changed, err := staged.Commit(ReplaceOptions{Expected: &expected})
	if err != nil || !changed {
		t.Fatalf("Commit() changed=%v error=%v", changed, err)
	}
	info, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0 {
		t.Fatalf("restored mode=%#o, want 0000", info.Mode().Perm())
	}
	if !info.ModTime().Equal(modTime) {
		t.Fatalf("restored modtime=%v, want %v", info.ModTime(), modTime)
	}
}

func TestStageReplacementStreamsAndCommits(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "target.txt")
	original := []byte("original")
	if err := os.WriteFile(path, original, 0o600); err != nil {
		t.Fatal(err)
	}
	expected, err := CaptureSnapshotWithData(path, original)
	if err != nil {
		t.Fatal(err)
	}

	staged, err := StageReplacement(path, &singleByteMutationReader{data: []byte("replacement")}, 0o600, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer staged.Cleanup()
	changed, err := staged.Commit(ReplaceOptions{Expected: &expected})
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("changed = false, want true")
	}
	actual, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(actual) != "replacement" {
		t.Fatalf("content = %q, want replacement", actual)
	}
	assertNoMutationTemps(t, dir)
}

func TestStageReplacementIdenticalBytesSkipCommitAndBackup(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "target.txt")
	backupPath := path + ".bak"
	original := []byte("unchanged")
	fixedTime := time.Unix(1_700_000_000, 0)
	if err := os.WriteFile(path, original, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, fixedTime, fixedTime); err != nil {
		t.Fatal(err)
	}
	expected, err := CaptureSnapshotWithData(path, original)
	if err != nil {
		t.Fatal(err)
	}

	staged, err := StageReplacement(path, bytes.NewReader(original), 0o600, nil)
	if err != nil {
		t.Fatal(err)
	}
	changed, err := staged.Commit(ReplaceOptions{Expected: &expected, BackupPath: backupPath, SkipIdentical: true})
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Fatal("changed = true, want false")
	}
	if _, err := os.Stat(backupPath); !os.IsNotExist(err) {
		t.Fatalf("backup should not exist, stat error = %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !info.ModTime().Equal(fixedTime) {
		t.Fatalf("mtime = %v, want %v", info.ModTime(), fixedTime)
	}
	assertNoMutationTemps(t, dir)
}

func TestStageReplacementReaderFailureCleansTemp(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "target.txt")
	if err := os.WriteFile(path, []byte("original"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := StageReplacement(path, &failingMutationReader{}, 0o600, nil)
	if err == nil || !strings.Contains(err.Error(), "injected read failure") {
		t.Fatalf("error = %v, want injected read failure", err)
	}
	if actual, readErr := os.ReadFile(path); readErr != nil || string(actual) != "original" {
		t.Fatalf("target = %q, err=%v; want original", actual, readErr)
	}
	assertNoMutationTemps(t, dir)
}

func TestStageReplacementDiskFullCleansTempAndPreservesTarget(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "target.txt")
	if err := os.WriteFile(path, []byte("original"), 0o600); err != nil {
		t.Fatal(err)
	}

	ops := defaultMutationOps
	ops.copyStream = func(destination io.Writer, _ io.Reader) (int64, error) {
		written, err := io.WriteString(destination, "partial")
		if err != nil {
			return int64(written), err
		}
		return int64(written), syscall.ENOSPC
	}
	if _, err := stageReplacement(path, strings.NewReader("replacement"), 0o600, nil, true, ops); !errors.Is(err, syscall.ENOSPC) {
		t.Fatalf("error = %v, want ENOSPC", err)
	}
	if actual, readErr := os.ReadFile(path); readErr != nil || string(actual) != "original" {
		t.Fatalf("target = %q, err=%v; want original", actual, readErr)
	}
	assertNoMutationTemps(t, dir)
}

type singleByteMutationReader struct {
	data []byte
}

func (reader *singleByteMutationReader) Read(buffer []byte) (int, error) {
	if len(reader.data) == 0 {
		return 0, io.EOF
	}
	buffer[0] = reader.data[0]
	reader.data = reader.data[1:]
	return 1, nil
}

type failingMutationReader struct {
	delivered bool
}

func (reader *failingMutationReader) Read(buffer []byte) (int, error) {
	if !reader.delivered {
		reader.delivered = true
		copy(buffer, "partial")
		return len("partial"), nil
	}
	return 0, errors.New("injected read failure")
}

func TestReplaceFile_CommitsAndPreservesMode(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "target.txt")
	original := []byte("original")
	if err := os.WriteFile(path, original, 0600); err != nil {
		t.Fatal(err)
	}
	expected, err := CaptureSnapshotWithData(path, original)
	if err != nil {
		t.Fatal(err)
	}

	if err := ReplaceFile(path, []byte("replacement"), ReplaceOptions{
		Mode:     0600,
		Expected: &expected,
	}); err != nil {
		t.Fatal(err)
	}

	actual, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(actual) != "replacement" {
		t.Fatalf("content = %q, want replacement", actual)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0600 {
		t.Fatalf("mode = %o, want 600", info.Mode().Perm())
	}
	assertNoMutationTemps(t, dir)
}

func TestReplaceFile_WithBackupPreservesOriginalMetadata(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "target.txt")
	backupPath := path + ".bak"
	original := []byte("original")
	fixedTime := time.Unix(1_700_000_000, 0)
	if err := os.WriteFile(path, original, 0640); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, fixedTime, fixedTime); err != nil {
		t.Fatal(err)
	}
	expected, err := CaptureSnapshotWithData(path, original)
	if err != nil {
		t.Fatal(err)
	}

	if err := ReplaceFile(path, []byte("replacement"), ReplaceOptions{
		Mode:       0640,
		Expected:   &expected,
		BackupPath: backupPath,
	}); err != nil {
		t.Fatal(err)
	}

	backup, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(backup, original) {
		t.Fatalf("backup = %q, want %q", backup, original)
	}
	info, err := os.Stat(backupPath)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0640 {
		t.Fatalf("backup mode = %o, want 640", info.Mode().Perm())
	}
	if !info.ModTime().Equal(fixedTime) {
		t.Fatalf("backup mtime = %v, want %v", info.ModTime(), fixedTime)
	}
	assertNoMutationTemps(t, dir)
}

func TestReplaceFile_TargetCommitFailureLeavesTargetAndRestoresPreviousBackup(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "target.txt")
	backupPath := path + ".bak"
	original := []byte("original")
	previousBackup := []byte("previous backup")
	if err := os.WriteFile(path, original, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(backupPath, previousBackup, 0600); err != nil {
		t.Fatal(err)
	}
	expected, err := CaptureSnapshotWithData(path, original)
	if err != nil {
		t.Fatal(err)
	}

	ops := defaultMutationOps
	baseReplace := ops.replacePath
	ops.replacePath = func(source, destination string) error {
		if filepath.Clean(destination) == filepath.Clean(path) {
			return errors.New("injected target commit failure")
		}
		return baseReplace(source, destination)
	}

	err = replaceFile(path, []byte("replacement"), ReplaceOptions{
		Mode:       0644,
		Expected:   &expected,
		BackupPath: backupPath,
	}, ops)
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte("injected target commit failure")) {
		t.Fatalf("error = %v, want injected target commit failure", err)
	}

	actual, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(actual, original) {
		t.Fatalf("target = %q, want %q", actual, original)
	}
	backup, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(backup, previousBackup) {
		t.Fatalf("backup = %q, want previous backup", backup)
	}
	assertNoMutationTemps(t, dir)
}

func TestReplaceFile_ConcurrentModificationIsRejected(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "target.txt")
	original := []byte("original")
	if err := os.WriteFile(path, original, 0644); err != nil {
		t.Fatal(err)
	}
	expected, err := CaptureSnapshotWithData(path, original)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("intruder"), 0644); err != nil {
		t.Fatal(err)
	}

	err = ReplaceFile(path, []byte("replacement"), ReplaceOptions{Mode: 0644, Expected: &expected})
	if !errors.Is(err, ErrConcurrentModification) {
		t.Fatalf("error = %v, want ErrConcurrentModification", err)
	}
	actual, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(actual) != "intruder" {
		t.Fatalf("content = %q, want intruder", actual)
	}
	assertNoMutationTemps(t, dir)
}

func TestReplaceFile_MissingTargetDoesNotOverwriteConcurrentCreation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "target.txt")
	expected, err := CaptureSnapshot(path)
	if err != nil {
		t.Fatal(err)
	}
	if expected.Exists {
		t.Fatal("expected missing target snapshot")
	}

	ops := defaultMutationOps
	baseInstall := ops.installNoReplace
	ops.installNoReplace = func(source, destination string) error {
		if err := os.WriteFile(destination, []byte("intruder"), 0644); err != nil {
			return err
		}
		return baseInstall(source, destination)
	}

	err = replaceFile(path, []byte("replacement"), ReplaceOptions{
		Mode:     0644,
		Expected: &expected,
	}, ops)
	if !errors.Is(err, ErrConcurrentModification) {
		t.Fatalf("error = %v, want ErrConcurrentModification", err)
	}
	actual, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(actual) != "intruder" {
		t.Fatalf("content = %q, want intruder", actual)
	}
	assertNoMutationTemps(t, dir)
}

func TestReplaceFile_TargetFailureRemovesNewBackupWhenNoPreviousBackupExists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "target.txt")
	backupPath := path + ".bak"
	original := []byte("original")
	if err := os.WriteFile(path, original, 0644); err != nil {
		t.Fatal(err)
	}
	expected, err := CaptureSnapshotWithData(path, original)
	if err != nil {
		t.Fatal(err)
	}

	ops := defaultMutationOps
	baseReplace := ops.replacePath
	ops.replacePath = func(source, destination string) error {
		if filepath.Clean(destination) == filepath.Clean(path) {
			return errors.New("injected target commit failure")
		}
		return baseReplace(source, destination)
	}

	err = replaceFile(path, []byte("replacement"), ReplaceOptions{
		Mode:       0644,
		Expected:   &expected,
		BackupPath: backupPath,
	}, ops)
	if err == nil {
		t.Fatal("expected target commit failure")
	}
	if _, statErr := os.Stat(backupPath); !os.IsNotExist(statErr) {
		t.Fatalf("new backup should be removed during rollback, stat error = %v", statErr)
	}
	actual, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !bytes.Equal(actual, original) {
		t.Fatalf("target = %q, want original", actual)
	}
	assertNoMutationTemps(t, dir)
}

func TestReplaceFile_RollbackFailurePreservesRecoveryCopy(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "target.txt")
	backupPath := path + ".bak"
	original := []byte("original")
	previousBackup := []byte("previous backup")
	if err := os.WriteFile(path, original, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(backupPath, previousBackup, 0644); err != nil {
		t.Fatal(err)
	}
	expected, err := CaptureSnapshotWithData(path, original)
	if err != nil {
		t.Fatal(err)
	}

	ops := defaultMutationOps
	baseReplace := ops.replacePath
	backupReplaceCount := 0
	ops.replacePath = func(source, destination string) error {
		switch filepath.Clean(destination) {
		case filepath.Clean(path):
			return errors.New("injected target commit failure")
		case filepath.Clean(backupPath):
			backupReplaceCount++
			if backupReplaceCount == 2 {
				return errors.New("injected rollback failure")
			}
		}
		return baseReplace(source, destination)
	}

	err = replaceFile(path, []byte("replacement"), ReplaceOptions{
		Mode:       0644,
		Expected:   &expected,
		BackupPath: backupPath,
	}, ops)
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte("recovery copy preserved at")) {
		t.Fatalf("error = %v, want preserved recovery copy path", err)
	}
	if data, readErr := os.ReadFile(path); readErr != nil || !bytes.Equal(data, original) {
		t.Fatalf("target = %q, err=%v; want original", data, readErr)
	}

	entries, readDirErr := os.ReadDir(dir)
	if readDirErr != nil {
		t.Fatal(readDirErr)
	}
	var recoveryPaths []string
	for _, entry := range entries {
		if bytes.HasPrefix([]byte(entry.Name()), []byte(".target.txt.bak.")) && filepath.Ext(entry.Name()) == ".tmp" {
			recoveryPaths = append(recoveryPaths, filepath.Join(dir, entry.Name()))
		}
	}
	if len(recoveryPaths) != 1 {
		t.Fatalf("recovery paths = %v, want exactly one", recoveryPaths)
	}
	recovery, readErr := os.ReadFile(recoveryPaths[0])
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !bytes.Equal(recovery, previousBackup) {
		t.Fatalf("recovery copy = %q, want previous backup", recovery)
	}
}

func TestReplaceFile_SuccessReplacesExistingBackupWithOriginal(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "target.txt")
	backupPath := path + ".bak"
	original := []byte("original")
	if err := os.WriteFile(path, original, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(backupPath, []byte("stale backup"), 0644); err != nil {
		t.Fatal(err)
	}
	expected, err := CaptureSnapshotWithData(path, original)
	if err != nil {
		t.Fatal(err)
	}

	if err := ReplaceFile(path, []byte("replacement"), ReplaceOptions{
		Mode:       0644,
		Expected:   &expected,
		BackupPath: backupPath,
	}); err != nil {
		t.Fatal(err)
	}
	backup, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(backup, original) {
		t.Fatalf("backup = %q, want original", backup)
	}
	assertNoMutationTemps(t, dir)
}

func TestReplaceFile_SyncFailureLeavesOriginalAndCleansTemp(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "target.txt")
	original := []byte("original")
	if err := os.WriteFile(path, original, 0644); err != nil {
		t.Fatal(err)
	}
	expected, err := CaptureSnapshotWithData(path, original)
	if err != nil {
		t.Fatal(err)
	}

	ops := defaultMutationOps
	ops.syncFile = func(*os.File) error { return errors.New("injected sync failure") }
	err = replaceFile(path, []byte("replacement"), ReplaceOptions{Mode: 0644, Expected: &expected}, ops)
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte("injected sync failure")) {
		t.Fatalf("error = %v, want injected sync failure", err)
	}
	actual, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(actual, original) {
		t.Fatalf("target = %q, want %q", actual, original)
	}
	assertNoMutationTemps(t, dir)
}

func TestCopyFile_NoReplacePreservesModeAndTime(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source.txt")
	destination := filepath.Join(dir, "destination.txt")
	fixedTime := time.Unix(1_700_000_100, 0)
	if err := os.WriteFile(source, []byte("content"), 0751); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(source, fixedTime, fixedTime); err != nil {
		t.Fatal(err)
	}

	if err := CopyFile(source, destination); err != nil {
		t.Fatal(err)
	}
	actual, err := os.ReadFile(destination)
	if err != nil {
		t.Fatal(err)
	}
	if string(actual) != "content" {
		t.Fatalf("content = %q, want content", actual)
	}
	info, err := os.Stat(destination)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0751 {
		t.Fatalf("mode = %o, want 751", info.Mode().Perm())
	}
	if !info.ModTime().Equal(fixedTime) {
		t.Fatalf("mtime = %v, want %v", info.ModTime(), fixedTime)
	}

	if err := CopyFile(source, destination); !errors.Is(err, ErrDestinationExists) {
		t.Fatalf("second copy error = %v, want ErrDestinationExists", err)
	}
	assertNoMutationTemps(t, dir)
}

func TestMoveNoReplace_DestinationExists(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source.txt")
	destination := filepath.Join(dir, "destination.txt")
	if err := os.WriteFile(source, []byte("source"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(destination, []byte("destination"), 0644); err != nil {
		t.Fatal(err)
	}

	err := MoveNoReplace(source, destination)
	if !errors.Is(err, ErrDestinationExists) {
		t.Fatalf("error = %v, want ErrDestinationExists", err)
	}
	if data, _ := os.ReadFile(source); string(data) != "source" {
		t.Fatalf("source = %q, want source", data)
	}
	if data, _ := os.ReadFile(destination); string(data) != "destination" {
		t.Fatalf("destination = %q, want destination", data)
	}
}

func TestRemoveFile_ConcurrentModificationIsRejected(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "target.txt")
	original := []byte("original")
	if err := os.WriteFile(path, original, 0644); err != nil {
		t.Fatal(err)
	}
	expected, err := CaptureSnapshotWithData(path, original)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("intruder"), 0644); err != nil {
		t.Fatal(err)
	}

	err = RemoveFile(path, &expected)
	if !errors.Is(err, ErrConcurrentModification) {
		t.Fatalf("error = %v, want ErrConcurrentModification", err)
	}
	if data, err := os.ReadFile(path); err != nil || string(data) != "intruder" {
		t.Fatalf("file = %q, err=%v; want intruder", data, err)
	}
}

func assertNoMutationTemps(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) == ".tmp" || bytes.Contains([]byte(entry.Name()), []byte(".rollback-")) {
			t.Fatalf("unexpected mutation artifact: %s", entry.Name())
		}
	}
}
