//go:build linux || darwin

package backupstore

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiagnosticStoreDetectsUnixLockPathReplacement(t *testing.T) {
	base := canonicalTempDir(t)
	root := filepath.Join(base, "backup-store")
	store, err := Open(Options{Directory: root})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	diagnostic, err := OpenExistingForDiagnosis(DiagnosticOpenOptions{Directory: root})
	if err != nil {
		t.Fatal(err)
	}
	lockPath := filepath.Join(root, "store.lock")
	displaced := filepath.Join(base, "displaced-lock")
	if err := os.Rename(lockPath, displaced); err != nil {
		_ = diagnostic.Close()
		t.Skipf("lock replacement unavailable: %v", err)
	}
	if err := os.WriteFile(lockPath, nil, 0o600); err != nil {
		_ = diagnostic.Close()
		t.Fatal(err)
	}
	if err := restrictPathPermissions(lockPath, false); err != nil {
		_ = diagnostic.Close()
		t.Fatal(err)
	}

	validationErr := diagnostic.validateIdentity()
	if validationErr == nil {
		_ = diagnostic.Close()
		t.Fatal("diagnostic identity accepted a replaced lock path")
	}
	if strings.Contains(validationErr.Error(), root) || strings.Contains(validationErr.Error(), displaced) {
		_ = diagnostic.Close()
		t.Fatalf("diagnostic identity error exposed a private path: %v", validationErr)
	}
	if err := diagnostic.Close(); err != nil {
		t.Fatal(err)
	}
}
