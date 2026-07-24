package security

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func IsPathWithinAllowedDirectories(absolutePath string, allowedDirs []string) bool {
	if absolutePath == "" || len(allowedDirs) == 0 {
		return false
	}

	if strings.Contains(absolutePath, "\x00") {
		return false
	}

	normalized := filepath.Clean(absolutePath)
	if !filepath.IsAbs(normalized) {
		return false
	}

	normalized = normalizePath(normalized)

	for _, allowedDir := range allowedDirs {
		cleanAllowed := normalizePath(filepath.Clean(allowedDir))

		if normalized == cleanAllowed {
			return true
		}

		separator := string(filepath.Separator)
		allowedPrefix := cleanAllowed
		if !strings.HasSuffix(allowedPrefix, separator) {
			allowedPrefix += separator
		}
		if strings.HasPrefix(normalized, allowedPrefix) {
			return true
		}
	}

	return false
}

// ValidatePath resolves a path and ensures it's within allowed directories.
func ValidatePath(requestedPath string, allowedDirs []string) (string, error) {
	if len(allowedDirs) == 0 {
		return "", ErrNoAllowedDirs
	}

	expanded := ExpandHome(requestedPath)

	var absolute string
	if filepath.IsAbs(expanded) {
		absolute = filepath.Clean(expanded)
	} else {
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("failed to get current directory: %w", err)
		}
		absolute = filepath.Clean(filepath.Join(cwd, expanded))
	}

	normalized := normalizePath(absolute)

	if !IsPathWithinAllowedDirectories(normalized, allowedDirs) {
		return "", fmt.Errorf("%w: %s", ErrPathDenied, absolute)
	}

	resolvedAllowedDirs := make([]string, 0, len(allowedDirs))
	for _, dir := range allowedDirs {
		resolvedDir, _, err := resolvePathAllowMissing(dir)
		if err == nil {
			resolvedAllowedDirs = append(resolvedAllowedDirs, normalizePath(resolvedDir))
		}
	}

	resolvedPath, exists, err := resolvePathAllowMissing(absolute)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("%w: %s", ErrParentNotExists, filepath.Dir(absolute))
		}
		return "", fmt.Errorf("failed to resolve path: %w", err)
	}

	normalizedResolved := normalizePath(resolvedPath)
	if !IsPathWithinAllowedDirectories(normalizedResolved, resolvedAllowedDirs) {
		if exists {
			return "", fmt.Errorf("%w: %s", ErrSymlinkDenied, resolvedPath)
		}
		return "", fmt.Errorf("%w: %s", ErrParentDirDenied, filepath.Dir(resolvedPath))
	}
	if !exists {
		return absolute, nil
	}
	return resolvedPath, nil
}

// resolvePathAllowMissing resolves the nearest existing ancestor and projects
// any missing suffix onto that resolved path. Existing but unresolvable links
// fail closed instead of being treated as missing paths.
func resolvePathAllowMissing(path string) (resolved string, exists bool, err error) {
	current := filepath.Clean(path)
	missingParts := make([]string, 0, 4)

	for {
		resolvedCurrent, resolveErr := resolveExistingPath(current)
		if resolveErr == nil {
			resolvedCurrent = filepath.Clean(resolvedCurrent)
			for i := len(missingParts) - 1; i >= 0; i-- {
				resolvedCurrent = filepath.Join(resolvedCurrent, missingParts[i])
			}
			return filepath.Clean(resolvedCurrent), len(missingParts) == 0, nil
		}
		if !os.IsNotExist(resolveErr) {
			return "", false, resolveErr
		}

		if _, lstatErr := os.Lstat(current); lstatErr == nil {
			return "", false, fmt.Errorf("existing path cannot be resolved: %s: %w", current, resolveErr)
		} else if !os.IsNotExist(lstatErr) {
			return "", false, lstatErr
		}

		parent := filepath.Dir(current)
		if parent == current {
			return "", false, resolveErr
		}
		missingParts = append(missingParts, filepath.Base(current))
		current = parent
	}
}

func normalizePath(p string) string {
	p = strings.Trim(p, "\"' \t\n")
	p = filepath.Clean(p)
	if runtime.GOOS == "windows" && len(p) >= 2 && p[1] == ':' {
		p = strings.ToUpper(p[:1]) + p[1:]
	}

	return p
}

// ResolveAllowedDirs resolves allowed directories once. Missing directories
// are projected from their nearest existing ancestor; unresolvable existing
// links are omitted so callers fail closed.
func ResolveAllowedDirs(allowedDirs []string) []string {
	resolved := make([]string, 0, len(allowedDirs))
	for _, dir := range allowedDirs {
		resolvedDir, _, err := resolvePathAllowMissing(dir)
		if err != nil {
			continue
		}
		resolved = append(resolved, normalizePath(resolvedDir))
	}
	return resolved
}

// ResolvePathSafe resolves a path and returns the resolved path only when it remains
// within the pre-resolved allowed directories.
func ResolvePathSafe(path string, resolvedAllowedDirs []string) (string, bool) {
	if path == "" || len(resolvedAllowedDirs) == 0 {
		return "", false
	}

	resolved, err := resolveExistingPath(path)
	if err != nil {
		return "", false
	}
	resolved = filepath.Clean(resolved)
	if !IsPathWithinAllowedDirectories(resolved, resolvedAllowedDirs) {
		return "", false
	}
	return resolved, true
}

// IsPathSafeResolved checks if a path (after resolving symlinks) is within pre-resolved allowed dirs.
func IsPathSafeResolved(path string, resolvedAllowedDirs []string) bool {
	_, safe := ResolvePathSafe(path, resolvedAllowedDirs)
	return safe
}

func ExpandHome(path string) string {
	if strings.HasPrefix(path, "~/") || path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		if path == "~" {
			return home
		}
		return filepath.Join(home, path[2:])
	}
	return path
}

func NormalizeAllowedDirs(dirs []string) ([]string, error) {
	var normalized []string
	for _, dir := range dirs {
		expanded := ExpandHome(dir)

		absolute, err := filepath.Abs(expanded)
		if err != nil {
			return nil, fmt.Errorf("invalid directory %s: %w", dir, err)
		}

		resolved, exists, err := resolvePathAllowMissing(absolute)
		if err != nil {
			return nil, fmt.Errorf("cannot resolve directory %s: %w", dir, err)
		}
		if exists {
			info, err := os.Stat(resolved)
			if err != nil {
				return nil, fmt.Errorf("cannot stat directory %s: %w", resolved, err)
			}
			if !info.IsDir() {
				return nil, fmt.Errorf("%w: %s", ErrNotDirectory, resolved)
			}
		}

		normalized = append(normalized, normalizePath(resolved))
	}
	return normalized, nil
}
