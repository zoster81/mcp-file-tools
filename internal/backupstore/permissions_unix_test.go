//go:build linux || darwin

package backupstore

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpenRejectsAndDoesNotRepairPermissiveUnixMode(t *testing.T) {
	for _, tc := range []struct {
		name      string
		relative  string
		directory bool
		mode      os.FileMode
	}{
		{name: "store root", directory: true, mode: 0o755},
		{name: "lock file", relative: "store.lock", mode: 0o644},
		{name: "descriptor", relative: "store.json", mode: 0o644},
		{name: "layout directory", relative: "manifests", directory: true, mode: 0o755},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := filepath.Join(canonicalTempDir(t), "backup-store")
			store, err := Open(Options{Directory: root})
			if err != nil {
				t.Fatalf("initialize store: %v", err)
			}
			if err := store.Close(); err != nil {
				t.Fatal(err)
			}
			path := root
			if tc.relative != "" {
				path = filepath.Join(root, tc.relative)
			}
			if err := os.Chmod(path, tc.mode); err != nil {
				t.Fatal(err)
			}
			if err := validatePathPermissions(path, tc.directory); err == nil {
				t.Fatal("permissive mode unexpectedly passed owner-only validation")
			}

			reopened, err := Open(Options{Directory: root})
			if reopened != nil {
				_ = reopened.Close()
			}
			if err == nil {
				t.Fatal("Open() accepted a permissive mode")
			}
			info, statErr := os.Stat(path)
			if statErr != nil {
				t.Fatal(statErr)
			}
			if info.Mode().Perm() != tc.mode {
				t.Fatalf("Open() changed mode to %04o, want unchanged %04o", info.Mode().Perm(), tc.mode)
			}
		})
	}
}
