//go:build linux

package filesystem

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

func replacePath(source, destination string) error {
	return os.Rename(source, destination)
}

func installPathNoReplace(source, destination string) error {
	err := unix.Renameat2(unix.AT_FDCWD, source, unix.AT_FDCWD, destination, unix.RENAME_NOREPLACE)
	if errors.Is(err, unix.ENOSYS) || errors.Is(err, unix.EINVAL) || errors.Is(err, unix.EOPNOTSUPP) {
		return installRegularFileNoReplace(source, destination)
	}
	return err
}

func movePathNoReplace(source, destination string) error {
	err := unix.Renameat2(unix.AT_FDCWD, source, unix.AT_FDCWD, destination, unix.RENAME_NOREPLACE)
	if errors.Is(err, unix.ENOSYS) || errors.Is(err, unix.EINVAL) || errors.Is(err, unix.EOPNOTSUPP) {
		return movePortableNoReplace(source, destination)
	}
	return err
}

func syncDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}

func isDestinationExistsError(err error) bool {
	return errors.Is(err, unix.EEXIST)
}
