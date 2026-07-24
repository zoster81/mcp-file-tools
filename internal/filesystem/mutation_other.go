//go:build !windows && !linux && !darwin

package filesystem

import (
	"errors"
	"io/fs"
	"os"
)

func replacePath(source, destination string) error {
	return os.Rename(source, destination)
}

func installPathNoReplace(source, destination string) error {
	return installRegularFileNoReplace(source, destination)
}

func movePathNoReplace(source, destination string) error {
	return movePortableNoReplace(source, destination)
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
	return errors.Is(err, fs.ErrExist)
}
