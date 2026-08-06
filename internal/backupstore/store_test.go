package backupstore

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/zoster81/scripthold/internal/operation"
)

func TestOpenInitializesAndReopensVersionedStore(t *testing.T) {
	root := filepath.Join(canonicalTempDir(t), "backup-store")
	store, err := Open(Options{Directory: root})
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	first := store.Descriptor()
	if first.FormatVersion != FormatVersion || first.ObjectAlgorithm != ObjectAlgorithm ||
		first.ManifestVersion != ManifestVersion || first.IndexVersion != IndexVersion {
		t.Fatalf("descriptor = %#v", first)
	}
	if len(first.StoreID) != 64 {
		t.Fatalf("store ID length = %d, want 64", len(first.StoreID))
	}
	if _, err := time.Parse(time.RFC3339Nano, first.CreatedAt); err != nil {
		t.Fatalf("createdAt = %q: %v", first.CreatedAt, err)
	}

	for _, relative := range []string{
		"store.json",
		"store.lock",
		filepath.Join("objects", "sha256"),
		"manifests",
		"index",
		"staging",
		"trash",
	} {
		if _, err := os.Lstat(filepath.Join(root, relative)); err != nil {
			t.Fatalf("expected layout entry %q: %v", relative, err)
		}
	}
	assertOwnerOnlyPermissions(t, root, true)
	assertOwnerOnlyPermissions(t, filepath.Join(root, "store.json"), false)

	if err := store.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("second Close() error = %v", err)
	}

	reopened, err := Open(Options{Directory: root})
	if err != nil {
		t.Fatalf("reopen error = %v", err)
	}
	defer reopened.Close()
	if got := reopened.Descriptor(); got != first {
		t.Fatalf("descriptor changed after reopen: got %#v want %#v", got, first)
	}
}

func TestOpenRejectsInvalidInternalLimitsBeforeFilesystemMutation(t *testing.T) {
	base := canonicalTempDir(t)
	for _, tc := range []struct {
		name   string
		limits Limits
		kind   operation.Kind
	}{
		{name: "negative", limits: Limits{MaxTotalBytes: -1}, kind: operation.KindInvalidInput},
		{name: "above hard maximum", limits: Limits{MaxTotalBytes: hardMaxTotalBytes + 1}, kind: operation.KindLimit},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := filepath.Join(base, strings.ReplaceAll(tc.name, " ", "-"))
			store, err := Open(Options{Directory: root, Limits: tc.limits})
			if store != nil {
				_ = store.Close()
			}
			if operation.KindOf(err) != tc.kind {
				t.Fatalf("Open() error = %v, want %s", err, tc.kind)
			}
			if _, statErr := os.Lstat(root); !os.IsNotExist(statErr) {
				t.Fatalf("invalid limits created a store directory: %v", statErr)
			}
		})
	}
}

func TestOpenRejectsRelativeAndOverlappingDirectories(t *testing.T) {
	publicRoot := canonicalTempDir(t)
	volumeRoot := filepath.Clean(filepath.VolumeName(publicRoot) + string(filepath.Separator))
	cases := []struct {
		name      string
		directory string
		roots     []string
	}{
		{name: "relative", directory: "relative-backup"},
		{name: "filesystem root", directory: volumeRoot},
		{name: "equal", directory: publicRoot, roots: []string{publicRoot}},
		{name: "inside public root", directory: filepath.Join(publicRoot, "backups"), roots: []string{publicRoot}},
		{name: "contains public root", directory: filepath.Dir(publicRoot), roots: []string{publicRoot}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store, err := Open(Options{Directory: tc.directory, PublicAllowedDirectories: tc.roots})
			if store != nil {
				_ = store.Close()
			}
			if err == nil {
				t.Fatal("Open() unexpectedly succeeded")
			}
			for _, privatePath := range append([]string{tc.directory}, tc.roots...) {
				if filepath.IsAbs(privatePath) && strings.Contains(err.Error(), privatePath) {
					t.Fatalf("error exposed configured path %q: %v", privatePath, err)
				}
			}
		})
	}
}

func TestOpenAllowsPrefixLookalikeSibling(t *testing.T) {
	base := canonicalTempDir(t)
	publicRoot := filepath.Join(base, "project")
	storeRoot := filepath.Join(base, "project-backups")
	if err := os.Mkdir(publicRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	store, err := Open(Options{Directory: storeRoot, PublicAllowedDirectories: []string{publicRoot}})
	if err != nil {
		t.Fatalf("Open() rejected a component-distinct sibling: %v", err)
	}
	defer store.Close()
}

func TestOpenRejectsAliasedStorePath(t *testing.T) {
	base := canonicalTempDir(t)
	realRoot := filepath.Join(base, "real")
	if err := os.Mkdir(realRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(base, "alias")
	if err := os.Symlink(realRoot, alias); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	store, err := Open(Options{Directory: alias})
	if store != nil {
		_ = store.Close()
	}
	if err == nil {
		t.Fatal("Open() accepted an aliased store path")
	}
}

func TestOpenRejectsSecondWriter(t *testing.T) {
	root := filepath.Join(canonicalTempDir(t), "backup-store")
	first, err := Open(Options{Directory: root})
	if err != nil {
		t.Fatalf("first Open() error = %v", err)
	}
	defer first.Close()

	second, err := Open(Options{Directory: root})
	if second != nil {
		_ = second.Close()
	}
	if err == nil {
		t.Fatal("second writer unexpectedly acquired the store")
	}
	if operation.KindOf(err) != operation.KindConflict {
		t.Fatalf("second writer error kind = %s, want conflict: %v", operation.KindOf(err), err)
	}
}

func TestOpenDoesNotRepairStoreMissingDescriptor(t *testing.T) {
	root := filepath.Join(canonicalTempDir(t), "backup-store")
	initialized, err := Open(Options{Directory: root})
	if err != nil {
		t.Fatalf("initialize store: %v", err)
	}
	if err := initialized.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(root, "store.json")); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(root, "objects")); err != nil {
		t.Fatal(err)
	}

	store, err := Open(Options{Directory: root})
	if store != nil {
		_ = store.Close()
	}
	if err == nil {
		t.Fatal("Open() repaired a store with a missing descriptor")
	}
	if _, statErr := os.Lstat(filepath.Join(root, "store.json")); !os.IsNotExist(statErr) {
		t.Fatalf("store.json was created during failed recovery: %v", statErr)
	}
	if _, statErr := os.Lstat(filepath.Join(root, "objects")); !os.IsNotExist(statErr) {
		t.Fatalf("objects directory was recreated during failed recovery: %v", statErr)
	}
}

func TestOpenRejectsUnexpectedRootEntryBeforeMutation(t *testing.T) {
	root := filepath.Join(canonicalTempDir(t), "backup-store")
	initialized, err := Open(Options{Directory: root})
	if err != nil {
		t.Fatalf("initialize store: %v", err)
	}
	if err := initialized.Close(); err != nil {
		t.Fatal(err)
	}
	descriptorBefore, err := os.ReadFile(filepath.Join(root, "store.json"))
	if err != nil {
		t.Fatal(err)
	}
	unexpected := filepath.Join(root, "unexpected.txt")
	if err := os.WriteFile(unexpected, []byte("do not repair"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := restrictPathPermissions(unexpected, false); err != nil {
		t.Fatal(err)
	}

	store, err := Open(Options{Directory: root})
	if store != nil {
		_ = store.Close()
	}
	if err == nil {
		t.Fatal("Open() accepted an unexpected root entry")
	}
	descriptorAfter, readErr := os.ReadFile(filepath.Join(root, "store.json"))
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(descriptorAfter) != string(descriptorBefore) {
		t.Fatal("descriptor changed before corruption rejection")
	}
}

func TestOpenRejectsMalformedOrUnsupportedDescriptors(t *testing.T) {
	cases := []struct {
		name    string
		content func(Descriptor) []byte
	}{
		{
			name: "unsupported format",
			content: func(descriptor Descriptor) []byte {
				descriptor.FormatVersion = "backup-store-v999"
				data, _ := json.Marshal(descriptor)
				return data
			},
		},
		{
			name: "trailing JSON",
			content: func(descriptor Descriptor) []byte {
				data, _ := json.Marshal(descriptor)
				return append(data, []byte("\n{}")...)
			},
		},
		{
			name: "malformed JSON",
			content: func(Descriptor) []byte {
				return []byte(`{"formatVersion":`)
			},
		},
		{
			name: "oversized descriptor",
			content: func(Descriptor) []byte {
				return make([]byte, maxDescriptorBytes+1)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := filepath.Join(canonicalTempDir(t), "backup-store")
			initialized, err := Open(Options{Directory: root})
			if err != nil {
				t.Fatalf("initialize store: %v", err)
			}
			descriptor := initialized.Descriptor()
			if err := initialized.Close(); err != nil {
				t.Fatal(err)
			}
			descriptorPath := filepath.Join(root, "store.json")
			if err := os.Remove(descriptorPath); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(descriptorPath, tc.content(descriptor), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := restrictPathPermissions(descriptorPath, false); err != nil {
				t.Fatal(err)
			}

			store, err := Open(Options{Directory: root})
			if store != nil {
				_ = store.Close()
			}
			if err == nil {
				t.Fatal("Open() accepted an invalid descriptor")
			}
			if strings.Contains(err.Error(), root) || strings.Contains(err.Error(), descriptorPath) {
				t.Fatalf("error exposed an internal path: %v", err)
			}
		})
	}
}

func TestOpenRejectsUnknownDescriptorFields(t *testing.T) {
	root := filepath.Join(canonicalTempDir(t), "backup-store")
	initialized, err := Open(Options{Directory: root})
	if err != nil {
		t.Fatalf("initialize store: %v", err)
	}
	if err := initialized.Close(); err != nil {
		t.Fatal(err)
	}
	descriptor := map[string]any{
		"formatVersion":   FormatVersion,
		"storeId":         strings.Repeat("a", 64),
		"createdAt":       time.Now().UTC().Format(time.RFC3339Nano),
		"objectAlgorithm": ObjectAlgorithm,
		"manifestVersion": ManifestVersion,
		"indexVersion":    IndexVersion,
		"unexpected":      true,
	}
	data, err := json.Marshal(descriptor)
	if err != nil {
		t.Fatal(err)
	}
	descriptorPath := filepath.Join(root, "store.json")
	if err := os.Remove(descriptorPath); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(descriptorPath, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := restrictPathPermissions(descriptorPath, false); err != nil {
		t.Fatal(err)
	}

	store, err := Open(Options{Directory: root})
	if store != nil {
		_ = store.Close()
	}
	if err == nil {
		t.Fatal("Open() accepted an unknown descriptor field")
	}
}

func canonicalTempDir(t *testing.T) string {
	t.Helper()
	directory := t.TempDir()
	resolved, err := filepath.EvalSymlinks(directory)
	if err != nil {
		t.Fatalf("resolve temporary directory: %v", err)
	}
	return filepath.Clean(resolved)
}

func assertOwnerOnlyPermissions(t *testing.T, path string, directory bool) {
	t.Helper()
	if err := validatePathPermissions(path, directory); err != nil {
		t.Fatalf("owner-only permissions for %s: %v", filepath.Base(path), err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0o077 != 0 {
		t.Fatalf("permissions for %s = %o, want no group/other bits", path, info.Mode().Perm())
	}
	if directory && !info.IsDir() {
		t.Fatalf("%s is not a directory", path)
	}
}
