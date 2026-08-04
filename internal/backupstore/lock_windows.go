//go:build windows

package backupstore

import (
	"errors"

	"golang.org/x/sys/windows"
)

type storeLock struct {
	handle windows.Handle
}

func acquireStoreLock(path string) (*storeLock, error) {
	pathPtr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}
	const access = windows.GENERIC_READ | windows.GENERIC_WRITE | windows.READ_CONTROL | windows.WRITE_DAC
	const attributes = windows.FILE_ATTRIBUTE_NORMAL | windows.FILE_FLAG_OPEN_REPARSE_POINT

	handle, err := windows.CreateFile(
		pathPtr,
		access,
		0,
		nil,
		windows.CREATE_NEW,
		attributes,
		0,
	)
	created := err == nil
	if errors.Is(err, windows.ERROR_FILE_EXISTS) || errors.Is(err, windows.ERROR_ALREADY_EXISTS) {
		handle, err = windows.CreateFile(
			pathPtr,
			access,
			0,
			nil,
			windows.OPEN_EXISTING,
			attributes,
			0,
		)
	}
	if err != nil {
		return nil, err
	}

	var info windows.ByHandleFileInformation
	if err := windows.GetFileInformationByHandle(handle, &info); err != nil {
		_ = windows.CloseHandle(handle)
		return nil, err
	}
	if info.FileAttributes&(windows.FILE_ATTRIBUTE_REPARSE_POINT|windows.FILE_ATTRIBUTE_DIRECTORY) != 0 || info.NumberOfLinks != 1 {
		_ = windows.CloseHandle(handle)
		return nil, errors.New("backup store lock is not a single-link regular file")
	}
	if created {
		if err := restrictHandlePermissions(handle, false); err != nil {
			_ = windows.CloseHandle(handle)
			return nil, err
		}
	} else if err := validateHandlePermissions(handle, false); err != nil {
		_ = windows.CloseHandle(handle)
		return nil, err
	}
	return &storeLock{handle: handle}, nil
}

func isLockConflict(err error) bool {
	return errors.Is(err, windows.ERROR_SHARING_VIOLATION) || errors.Is(err, windows.ERROR_LOCK_VIOLATION)
}

func (lock *storeLock) close() error {
	if lock == nil || lock.handle == 0 || lock.handle == windows.InvalidHandle {
		return nil
	}
	err := windows.CloseHandle(lock.handle)
	lock.handle = 0
	return err
}
