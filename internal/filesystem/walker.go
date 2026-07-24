package filesystem

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/dimitar-grigorov/mcp-file-tools/internal/security"
)

// WalkAction controls traversal after an entry is visited.
type WalkAction uint8

const (
	WalkContinue WalkAction = iota
	WalkSkipDir
	WalkStop
)

// Entry describes one filesystem entry relative to the traversal root.
type Entry struct {
	Path         string
	ResolvedPath string
	RelativePath string
	Name         string
	Depth        int
	DirEntry     fs.DirEntry
}

// WalkOptions defines shared traversal policy. Exclude prunes matching
// directories and skips matching files. MaxDepth is unlimited when <= 0.
type WalkOptions struct {
	ResolvedAllowedDirs []string
	MaxDepth            int
	Exclude             func(Entry) bool
	OnError             func(path string, depth int, err error) error
}

// Visitor processes a safe entry and selects the next traversal action.
type Visitor func(Entry) (WalkAction, error)

var errWalkStopped = errors.New("filesystem walk stopped")

// Walk traverses root in deterministic lexical order without following
// directory symlinks or reparse points. Every entry is resolved and checked
// against the allowed directories before it is exposed to the visitor.
func Walk(ctx context.Context, root string, options WalkOptions, visitor Visitor) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if visitor == nil {
		return errors.New("filesystem visitor is required")
	}
	if len(options.ResolvedAllowedDirs) == 0 {
		return errors.New("resolved allowed directories are required")
	}

	resolvedRoot, safe := security.ResolvePathSafe(root, options.ResolvedAllowedDirs)
	if !safe {
		if _, err := os.Stat(root); err != nil {
			return handleWalkError(options, root, 0, err)
		}
		return handleWalkError(options, root, 0, fmt.Errorf("root path resolves outside allowed directories: %s", root))
	}

	err := walkDirectory(ctx, resolvedRoot, "", 0, options, visitor)
	if errors.Is(err, errWalkStopped) {
		return nil
	}
	return err
}

func walkDirectory(ctx context.Context, dirPath, relativeDir string, depth int, options WalkOptions, visitor Visitor) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	resolvedDir, safe := security.ResolvePathSafe(dirPath, options.ResolvedAllowedDirs)
	if !safe {
		return handleWalkError(options, dirPath, depth, fmt.Errorf("path is no longer safe: %s", dirPath))
	}

	entries, err := os.ReadDir(resolvedDir)
	if err != nil {
		return handleWalkError(options, resolvedDir, depth, err)
	}

	for _, dirEntry := range entries {
		if err := ctx.Err(); err != nil {
			return err
		}

		childPath := filepath.Join(resolvedDir, dirEntry.Name())
		resolvedChild, safe := security.ResolvePathSafe(childPath, options.ResolvedAllowedDirs)
		if !safe {
			continue
		}

		relativePath := dirEntry.Name()
		if relativeDir != "" {
			relativePath = filepath.Join(relativeDir, dirEntry.Name())
		}
		entry := Entry{
			Path:         childPath,
			ResolvedPath: resolvedChild,
			RelativePath: relativePath,
			Name:         dirEntry.Name(),
			Depth:        depth + 1,
			DirEntry:     dirEntry,
		}
		if options.Exclude != nil && options.Exclude(entry) {
			continue
		}

		action, err := visitor(entry)
		if err != nil {
			return err
		}
		if action == WalkStop {
			return errWalkStopped
		}
		if !dirEntry.IsDir() || action == WalkSkipDir {
			continue
		}
		if options.MaxDepth > 0 && entry.Depth >= options.MaxDepth {
			continue
		}
		if err := walkDirectory(ctx, resolvedChild, relativePath, entry.Depth, options, visitor); err != nil {
			return err
		}
	}
	return nil
}

func handleWalkError(options WalkOptions, path string, depth int, err error) error {
	if options.OnError == nil {
		if depth == 0 {
			return err
		}
		return nil
	}
	return options.OnError(path, depth, err)
}
