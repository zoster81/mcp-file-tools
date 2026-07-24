//go:build darwin

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
	return unix.RenamexNp(source, destination, unix.RENAME_EXCL)
}

func movePathNoReplace(source, destination string) error {
	return unix.RenamexNp(source, destination, unix.RENAME_EXCL)
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
