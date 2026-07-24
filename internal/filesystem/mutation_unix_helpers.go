//go:build !windows

package filesystem

import (
	"errors"
	"fmt"
	"os"
)

func installRegularFileNoReplace(source, destination string) error {
	if err := os.Link(source, destination); err != nil {
		return err
	}
	if err := os.Remove(source); err != nil {
		rollbackErr := os.Remove(destination)
		return errors.Join(err, rollbackErr)
	}
	return nil
}

func movePortableNoReplace(source, destination string) error {
	info, err := os.Lstat(source)
	if err != nil {
		return err
	}
	if info.Mode().IsRegular() {
		return installRegularFileNoReplace(source, destination)
	}
	return fmt.Errorf("atomic no-replace move is not supported for %s on this platform", source)
}
