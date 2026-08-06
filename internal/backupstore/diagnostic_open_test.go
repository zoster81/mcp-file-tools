package backupstore

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/zoster81/scripthold/internal/operation"
)

type diagnosticTreeEntry struct {
	Mode        os.FileMode
	Size        int64
	ModTimeNano int64
	Digest      string
}

func TestOpenExistingForDiagnosisRequiresExistingRootAndLockWithoutCreating(t *testing.T) {
	base := canonicalTempDir(t)
	missingRoot := filepath.Join(base, "missing-store")
	opened, err := OpenExistingForDiagnosis(DiagnosticOpenOptions{Directory: missingRoot})
	if opened != nil {
		_ = opened.Close()
	}
	if operation.KindOf(err) != operation.KindInvalidPath {
		t.Fatalf("missing root error=%v, want INVALID_PATH", err)
	}
	if _, statErr := os.Lstat(missingRoot); !os.IsNotExist(statErr) {
		t.Fatalf("diagnostic opener created missing root: %v", statErr)
	}

	root := filepath.Join(base, "existing-store")
	if err := os.Mkdir(root, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := restrictPathPermissions(root, true); err != nil {
		t.Fatal(err)
	}
	before := snapshotDiagnosticTree(t, root)
	opened, err = OpenExistingForDiagnosis(DiagnosticOpenOptions{Directory: root})
	if opened != nil {
		_ = opened.Close()
	}
	if err == nil {
		t.Fatal("diagnostic opener accepted a store without an existing lock")
	}
	if _, statErr := os.Lstat(filepath.Join(root, "store.lock")); !os.IsNotExist(statErr) {
		t.Fatalf("diagnostic opener created missing lock: %v", statErr)
	}
	assertDiagnosticTreeUnchanged(t, root, before)
}

func TestOpenExistingForDiagnosisUsesExclusiveLockWithoutMutation(t *testing.T) {
	root := filepath.Join(canonicalTempDir(t), "backup-store")
	initialized, err := Open(Options{Directory: root})
	if err != nil {
		t.Fatal(err)
	}
	if err := initialized.Close(); err != nil {
		t.Fatal(err)
	}
	before := snapshotDiagnosticTree(t, root)

	diagnostic, err := OpenExistingForDiagnosis(DiagnosticOpenOptions{Directory: root})
	if err != nil {
		t.Fatalf("open existing store for diagnosis: %v", err)
	}
	if err := diagnostic.validateIdentity(); err != nil {
		t.Fatalf("validate diagnostic identity: %v", err)
	}
	concurrent, concurrentErr := Open(Options{Directory: root})
	if concurrent != nil {
		_ = concurrent.Close()
	}
	if operation.KindOf(concurrentErr) != operation.KindConflict {
		t.Fatalf("concurrent normal open error=%v, want CONFLICT", concurrentErr)
	}
	if err := diagnostic.Close(); err != nil {
		t.Fatalf("close diagnostic store: %v", err)
	}
	if err := diagnostic.Close(); err != nil {
		t.Fatalf("second diagnostic close: %v", err)
	}
	assertDiagnosticTreeUnchanged(t, root, before)

	reopened, err := Open(Options{Directory: root})
	if err != nil {
		t.Fatalf("normal reopen after diagnosis: %v", err)
	}
	if err := reopened.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestOpenExistingForDiagnosisPreservesIncompleteAndResidualState(t *testing.T) {
	root := filepath.Join(canonicalTempDir(t), "backup-store")
	initialized, err := Open(Options{Directory: root})
	if err != nil {
		t.Fatal(err)
	}
	if err := initialized.Close(); err != nil {
		t.Fatal(err)
	}

	for _, path := range []string{
		filepath.Join(root, "store.json"),
		filepath.Join(root, "index", "index-v1.json"),
	} {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			t.Fatal(err)
		}
	}
	if err := os.RemoveAll(filepath.Join(root, "objects")); err != nil {
		t.Fatal(err)
	}
	for _, relative := range []string{
		filepath.Join("staging", ".capture-object-residue.tmp"),
		filepath.Join("trash", "operator-unknown.tmp"),
	} {
		path := filepath.Join(root, relative)
		if err := os.WriteFile(path, []byte(relative), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := restrictPathPermissions(path, false); err != nil {
			t.Fatal(err)
		}
	}
	before := snapshotDiagnosticTree(t, root)

	diagnostic, err := OpenExistingForDiagnosis(DiagnosticOpenOptions{Directory: root})
	if err != nil {
		t.Fatalf("diagnostic opener rejected incomplete store before scanning: %v", err)
	}
	if err := diagnostic.validateIdentity(); err != nil {
		t.Fatalf("validate diagnostic identity: %v", err)
	}
	if err := diagnostic.Close(); err != nil {
		t.Fatal(err)
	}
	assertDiagnosticTreeUnchanged(t, root, before)
}

func TestExistingStoreLockRejectsReplacementBetweenPrecheckAndAcquisition(t *testing.T) {
	root := filepath.Join(canonicalTempDir(t), "backup-store")
	store, err := Open(Options{Directory: root})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	lockPath := filepath.Join(root, "store.lock")
	expected, err := os.Lstat(lockPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(lockPath); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(lockPath, []byte("replacement"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := restrictPathPermissions(lockPath, false); err != nil {
		t.Fatal(err)
	}

	lock, err := acquireExistingStoreLock(lockPath)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.close()
	if err := lock.validateExpected(lockPath, expected); err == nil {
		t.Fatal("existing lock accepted a replacement between precheck and acquisition")
	}
}

func TestOpenExistingForDiagnosisRejectsInvalidLimitsBeforeFilesystemAccess(t *testing.T) {
	root := filepath.Join(canonicalTempDir(t), "must-not-exist")
	opened, err := OpenExistingForDiagnosis(DiagnosticOpenOptions{
		Directory: root,
		Limits:    Limits{MaxTotalBytes: -1},
	})
	if opened != nil {
		_ = opened.Close()
	}
	if operation.KindOf(err) != operation.KindInvalidInput {
		t.Fatalf("invalid limits error=%v, want INVALID_INPUT", err)
	}
	if _, statErr := os.Lstat(root); !os.IsNotExist(statErr) {
		t.Fatalf("invalid limits touched filesystem: %v", statErr)
	}
}

func TestOpenExistingForDiagnosisRejectsAliasedRootAndHardLinkedLock(t *testing.T) {
	base := canonicalTempDir(t)
	root := filepath.Join(base, "backup-store")
	initialized, err := Open(Options{Directory: root})
	if err != nil {
		t.Fatal(err)
	}
	if err := initialized.Close(); err != nil {
		t.Fatal(err)
	}

	alias := filepath.Join(base, "store-alias")
	if err := os.Symlink(root, alias); err == nil {
		opened, openErr := OpenExistingForDiagnosis(DiagnosticOpenOptions{Directory: alias})
		if opened != nil {
			_ = opened.Close()
		}
		if openErr == nil {
			t.Fatal("diagnostic opener accepted an aliased root")
		}
	} else {
		t.Logf("symlink unavailable: %v", err)
	}

	lockPath := filepath.Join(root, "store.lock")
	outsideAlias := filepath.Join(base, "lock-alias")
	if err := os.Link(lockPath, outsideAlias); err != nil {
		t.Skipf("hard links unavailable: %v", err)
	}
	before := snapshotDiagnosticTree(t, root)
	opened, err := OpenExistingForDiagnosis(DiagnosticOpenOptions{Directory: root})
	if opened != nil {
		_ = opened.Close()
	}
	if err == nil {
		t.Fatal("diagnostic opener accepted a hard-linked lock")
	}
	if strings.Contains(err.Error(), root) || strings.Contains(err.Error(), outsideAlias) {
		t.Fatalf("diagnostic error exposed a private path: %v", err)
	}
	assertDiagnosticTreeUnchanged(t, root, before)
}

func snapshotDiagnosticTree(t *testing.T, root string) map[string]diagnosticTreeEntry {
	t.Helper()
	entries := make(map[string]diagnosticTreeEntry)
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		entry := diagnosticTreeEntry{
			Mode:        info.Mode(),
			Size:        info.Size(),
			ModTimeNano: info.ModTime().UnixNano(),
		}
		if info.Mode().IsRegular() {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			digest := sha256.Sum256(data)
			entry.Digest = hex.EncodeToString(digest[:])
		}
		entries[relative] = entry
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return entries
}

func assertDiagnosticTreeUnchanged(t *testing.T, root string, before map[string]diagnosticTreeEntry) {
	t.Helper()
	after := snapshotDiagnosticTree(t, root)
	if !reflect.DeepEqual(after, before) {
		t.Fatalf("diagnostic opener changed store tree:\nbefore=%#v\nafter=%#v", before, after)
	}
}
