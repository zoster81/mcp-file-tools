//go:build windows

package backupstore

import (
	"path/filepath"
	"runtime"
	"testing"

	"golang.org/x/sys/windows"
)

func TestOpenRejectsAndDoesNotRepairPermissiveWindowsACL(t *testing.T) {
	for _, tc := range []struct {
		name      string
		relative  string
		directory bool
	}{
		{name: "store root", directory: true},
		{name: "lock file", relative: "store.lock"},
		{name: "descriptor", relative: "store.json"},
		{name: "layout directory", relative: "manifests", directory: true},
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
			if err := setPermissiveWindowsACL(path, tc.directory); err != nil {
				t.Fatal(err)
			}
			if err := validatePathPermissions(path, tc.directory); err == nil {
				t.Fatal("permissive ACL unexpectedly passed owner-only validation")
			}

			reopened, err := Open(Options{Directory: root})
			if reopened != nil {
				_ = reopened.Close()
			}
			if err == nil {
				t.Fatal("Open() accepted a permissive ACL")
			}
			if err := validatePathPermissions(path, tc.directory); err == nil {
				t.Fatal("Open() silently repaired the permissive ACL")
			}
		})
	}
}

func setPermissiveWindowsACL(path string, directory bool) error {
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return err
	}
	world, err := windows.CreateWellKnownSid(windows.WinWorldSid)
	if err != nil {
		return err
	}
	inheritance := uint32(0)
	if directory {
		inheritance = windows.SUB_CONTAINERS_AND_OBJECTS_INHERIT
	}
	acl, err := windows.ACLFromEntries([]windows.EXPLICIT_ACCESS{
		{
			AccessPermissions: windows.GENERIC_ALL,
			AccessMode:        windows.SET_ACCESS,
			Inheritance:       inheritance,
			Trustee: windows.TRUSTEE{
				TrusteeForm:  windows.TRUSTEE_IS_SID,
				TrusteeType:  windows.TRUSTEE_IS_USER,
				TrusteeValue: windows.TrusteeValueFromSID(user.User.Sid),
			},
		},
		{
			AccessPermissions: windows.GENERIC_READ,
			AccessMode:        windows.GRANT_ACCESS,
			Inheritance:       inheritance,
			Trustee: windows.TRUSTEE{
				TrusteeForm:  windows.TRUSTEE_IS_SID,
				TrusteeType:  windows.TRUSTEE_IS_WELL_KNOWN_GROUP,
				TrusteeValue: windows.TrusteeValueFromSID(world),
			},
		},
	}, nil)
	if err != nil {
		return err
	}
	err = windows.SetNamedSecurityInfo(
		path,
		windows.SE_FILE_OBJECT,
		windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION,
		nil,
		nil,
		acl,
		nil,
	)
	runtime.KeepAlive(user)
	runtime.KeepAlive(world)
	runtime.KeepAlive(acl)
	return err
}

func TestOpenExistingForDiagnosisRejectsAndDoesNotRepairPermissiveWindowsACL(t *testing.T) {
	for _, tc := range []struct {
		name      string
		relative  string
		directory bool
	}{
		{name: "store root", directory: true},
		{name: "lock file", relative: "store.lock"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := filepath.Join(canonicalTempDir(t), "backup-store")
			store, err := Open(Options{Directory: root})
			if err != nil {
				t.Fatal(err)
			}
			if err := store.Close(); err != nil {
				t.Fatal(err)
			}
			path := root
			if tc.relative != "" {
				path = filepath.Join(root, tc.relative)
			}
			if err := setPermissiveWindowsACL(path, tc.directory); err != nil {
				t.Fatal(err)
			}

			diagnostic, err := OpenExistingForDiagnosis(DiagnosticOpenOptions{Directory: root})
			if diagnostic != nil {
				_ = diagnostic.Close()
			}
			if err == nil {
				t.Fatal("diagnostic opener accepted a permissive ACL")
			}
			if err := validatePathPermissions(path, tc.directory); err == nil {
				t.Fatal("diagnostic opener silently repaired the permissive ACL")
			}
		})
	}
}
