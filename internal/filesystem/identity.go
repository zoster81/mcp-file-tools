package filesystem

import (
	"fmt"
	"os"

	"github.com/zoster81/mcp-file-tools/internal/operation"
)

// FileIdentity retains an open reference to one regular file so callers can
// detect path replacement without preventing a later atomic replacement.
type FileIdentity struct {
	file *os.File
}

// OpenFileIdentity opens path with platform-appropriate sharing and retains a
// stable filesystem identity until Close is called.
func OpenFileIdentity(path string) (identity *FileIdentity, err error) {
	defer func() {
		err = operation.WrapFilesystem("open_file_identity", path, err)
	}()
	file, err := openIdentityFile(path)
	if err != nil {
		return nil, err
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, err
	}
	if !info.Mode().IsRegular() {
		_ = file.Close()
		return nil, fmt.Errorf("path is not a regular file: %s", path)
	}
	return &FileIdentity{file: file}, nil
}

// Matches reports whether path still names the same open filesystem object.
func (identity *FileIdentity) Matches(path string) (matches bool, err error) {
	defer func() {
		err = operation.WrapFilesystem("match_file_identity", path, err)
	}()
	if identity == nil || identity.file == nil {
		return false, fmt.Errorf("file identity is unavailable")
	}
	openInfo, err := identity.file.Stat()
	if err != nil {
		return false, err
	}
	currentInfo, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	return os.SameFile(openInfo, currentInfo), nil
}

// Close releases the retained filesystem reference.
func (identity *FileIdentity) Close() error {
	if identity == nil || identity.file == nil {
		return nil
	}
	err := identity.file.Close()
	identity.file = nil
	return err
}
