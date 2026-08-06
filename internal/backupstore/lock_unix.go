//go:build linux || darwin

package backupstore

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

type storeLock struct {
	file     *os.File
	identity os.FileInfo
}

func acquireStoreLock(path string) (*storeLock, error) {
	return openStoreLock(path, true)
}

func acquireExistingStoreLock(path string) (*storeLock, error) {
	return openStoreLock(path, false)
}

func openStoreLock(path string, create bool) (*storeLock, error) {
	flags := unix.O_RDWR | unix.O_CLOEXEC | unix.O_NOFOLLOW
	created := false
	var fd int
	var err error
	if create {
		fd, err = unix.Open(path, flags|unix.O_CREAT|unix.O_EXCL, 0o600)
		created = err == nil
		if errors.Is(err, unix.EEXIST) {
			fd, err = unix.Open(path, flags, 0)
		}
	} else {
		fd, err = unix.Open(path, flags, 0)
	}
	if err != nil {
		return nil, err
	}
	file := os.NewFile(uintptr(fd), path)
	if file == nil {
		_ = unix.Close(fd)
		return nil, os.ErrInvalid
	}
	cleanup := func(err error) (*storeLock, error) {
		_ = file.Close()
		return nil, err
	}
	info, err := file.Stat()
	if err != nil {
		return cleanup(err)
	}
	if !info.Mode().IsRegular() {
		return cleanup(errors.New("backup store lock is not a regular file"))
	}
	if created {
		if err := file.Chmod(0o600); err != nil {
			return cleanup(err)
		}
		info, err = file.Stat()
		if err != nil {
			return cleanup(err)
		}
	}
	if err := validateUnixOwnerOnlyInfo(info, false); err != nil {
		return cleanup(err)
	}
	if err := validateSingleLink(path, info); err != nil {
		return cleanup(err)
	}
	if err := unix.Flock(fd, unix.LOCK_EX|unix.LOCK_NB); err != nil {
		return cleanup(err)
	}
	return &storeLock{file: file, identity: info}, nil
}

func isLockConflict(err error) bool {
	return errors.Is(err, unix.EWOULDBLOCK) || errors.Is(err, unix.EAGAIN)
}

func (lock *storeLock) validateExpected(path string, expected os.FileInfo) error {
	if lock == nil || lock.file == nil || expected == nil {
		return errors.New("backup store lock acquisition identity is unavailable")
	}
	handleInfo, err := lock.file.Stat()
	if err != nil {
		return err
	}
	if !os.SameFile(expected, handleInfo) || expected.Mode() != handleInfo.Mode() || expected.Size() != handleInfo.Size() || !expected.ModTime().Equal(handleInfo.ModTime()) {
		return errors.New("backup store lock changed during acquisition")
	}
	return lock.validate(path)
}

func (lock *storeLock) validate(path string) error {
	if lock == nil || lock.file == nil || lock.identity == nil {
		return errors.New("backup store lock identity is unavailable")
	}
	handleInfo, err := lock.file.Stat()
	if err != nil {
		return err
	}
	if !handleInfo.Mode().IsRegular() || !os.SameFile(lock.identity, handleInfo) {
		return errors.New("backup store lock handle identity changed")
	}
	if err := validateUnixOwnerOnlyInfo(handleInfo, false); err != nil {
		return err
	}
	pathInfo, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if isLinkOrReparse(pathInfo) || !pathInfo.Mode().IsRegular() || !os.SameFile(handleInfo, pathInfo) {
		return errors.New("backup store lock path identity changed")
	}
	if err := validateUnixOwnerOnlyInfo(pathInfo, false); err != nil {
		return err
	}
	return validateSingleLink(path, pathInfo)
}

func (lock *storeLock) close() error {
	if lock == nil || lock.file == nil {
		return nil
	}
	fd := int(lock.file.Fd())
	unlockErr := unix.Flock(fd, unix.LOCK_UN)
	closeErr := lock.file.Close()
	lock.file = nil
	lock.identity = nil
	return errors.Join(unlockErr, closeErr)
}
