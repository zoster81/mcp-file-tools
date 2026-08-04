//go:build linux || darwin

package backupstore

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

type storeLock struct {
	file *os.File
}

func acquireStoreLock(path string) (*storeLock, error) {
	fd, err := unix.Open(path, unix.O_RDWR|unix.O_CREAT|unix.O_EXCL|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0o600)
	created := err == nil
	if errors.Is(err, unix.EEXIST) {
		fd, err = unix.Open(path, unix.O_RDWR|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
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
	if err := unix.Flock(fd, unix.LOCK_EX|unix.LOCK_NB); err != nil {
		return cleanup(err)
	}
	return &storeLock{file: file}, nil
}

func isLockConflict(err error) bool {
	return errors.Is(err, unix.EWOULDBLOCK) || errors.Is(err, unix.EAGAIN)
}

func (lock *storeLock) close() error {
	if lock == nil || lock.file == nil {
		return nil
	}
	fd := int(lock.file.Fd())
	unlockErr := unix.Flock(fd, unix.LOCK_UN)
	closeErr := lock.file.Close()
	lock.file = nil
	return errors.Join(unlockErr, closeErr)
}
