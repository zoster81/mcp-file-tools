package security

import "github.com/zoster81/scripthold/internal/operation"

// Sentinel errors for path validation and security operations.
// Use errors.Is() to check for specific error types.

var (
	// ErrNoAllowedDirs is returned when no allowed directories are configured.
	ErrNoAllowedDirs = operation.New(operation.KindAccessDenied, "no allowed directories configured - please provide directories via CLI arguments or MCP roots protocol")

	// ErrPathDenied is returned when a path is outside all allowed directories.
	ErrPathDenied = operation.New(operation.KindAccessDenied, "access denied - path outside allowed directories")

	// ErrSymlinkDenied is returned when a symlink target is outside allowed directories.
	ErrSymlinkDenied = operation.New(operation.KindSymlinkEscape, "access denied - symlink target outside allowed directories")

	// ErrParentDirDenied is returned when a parent directory is outside allowed directories.
	ErrParentDirDenied = operation.New(operation.KindAccessDenied, "access denied - parent directory outside allowed directories")

	// ErrParentNotExists is returned when the parent directory does not exist.
	ErrParentNotExists = operation.New(operation.KindInvalidPath, "parent directory does not exist")

	// ErrNotDirectory is returned when a path is expected to be a directory but is not.
	ErrNotDirectory = operation.New(operation.KindInvalidPath, "path is not a directory")
)
