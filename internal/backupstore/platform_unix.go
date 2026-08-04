//go:build linux || darwin

package backupstore

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

func isLinkOrReparse(info os.FileInfo) bool {
	return info == nil || info.Mode()&os.ModeSymlink != 0
}

func restrictPathPermissions(path string, directory bool) error {
	mode := os.FileMode(0o600)
	if directory {
		mode = 0o700
	}
	if err := os.Chmod(path, mode); err != nil {
		return err
	}
	return validatePathPermissions(path, directory)
}

func validatePathPermissions(path string, directory bool) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	return validateUnixOwnerOnlyInfo(info, directory)
}

func validateUnixOwnerOnlyInfo(info os.FileInfo, directory bool) error {
	if info == nil {
		return errors.New("filesystem metadata is unavailable")
	}
	expected := os.FileMode(0o600)
	if directory {
		expected = 0o700
	}
	if info.Mode().Perm() != expected {
		return fmt.Errorf("permissions are %04o, want %04o", info.Mode().Perm(), expected)
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat == nil {
		return errors.New("filesystem owner metadata is unavailable")
	}
	if stat.Uid != uint32(os.Geteuid()) {
		return errors.New("filesystem owner does not match the process identity")
	}
	return nil
}

func syncDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}
