//go:build linux || darwin

package backupstore

import (
	"errors"
	"os"
	"syscall"
)

func validateSingleLink(_ string, info os.FileInfo) error {
	if info == nil || !info.Mode().IsRegular() {
		return errors.New("filesystem entry is not a regular file")
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat == nil {
		return errors.New("filesystem link metadata is unavailable")
	}
	if stat.Nlink != 1 {
		return errors.New("filesystem entry has multiple hard links")
	}
	return nil
}
