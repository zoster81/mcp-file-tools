//go:build windows

package backupstore

import (
	"errors"
	"os"

	"golang.org/x/sys/windows"
)

func validateSingleLink(path string, expected os.FileInfo) error {
	if expected == nil || !expected.Mode().IsRegular() {
		return errors.New("filesystem entry is not a regular file")
	}
	pathPtr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	handle, err := windows.CreateFile(
		pathPtr,
		windows.FILE_READ_ATTRIBUTES,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_ATTRIBUTE_NORMAL|windows.FILE_FLAG_OPEN_REPARSE_POINT,
		0,
	)
	if err != nil {
		return err
	}
	file := os.NewFile(uintptr(handle), path)
	if file == nil {
		_ = windows.CloseHandle(handle)
		return os.ErrInvalid
	}
	defer file.Close()
	var handleInfo windows.ByHandleFileInformation
	if err := windows.GetFileInformationByHandle(handle, &handleInfo); err != nil {
		return err
	}
	if handleInfo.FileAttributes&(windows.FILE_ATTRIBUTE_REPARSE_POINT|windows.FILE_ATTRIBUTE_DIRECTORY) != 0 {
		return errors.New("filesystem entry is linked or not a regular file")
	}
	if handleInfo.NumberOfLinks != 1 {
		return errors.New("filesystem entry has multiple hard links")
	}
	actual, err := file.Stat()
	if err != nil {
		return err
	}
	if !os.SameFile(expected, actual) {
		return errors.New("filesystem entry identity changed during validation")
	}
	return nil
}
