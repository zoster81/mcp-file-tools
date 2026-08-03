package handler

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zoster81/mcp-file-tools/internal/filesystem"
)

const defaultMaxResults = 10000

// HandleSearchFiles recursively searches for files matching a glob pattern.
func (h *Handler) HandleSearchFiles(ctx context.Context, req *mcp.CallToolRequest, input SearchFilesInput) (*mcp.CallToolResult, SearchFilesOutput, error) {
	if input.Pattern == "" {
		return errorResult(ErrPatternRequired.Error()), SearchFilesOutput{}, nil
	}
	v := h.ValidatePath(input.Path)
	if !v.Ok() {
		return v.Result, SearchFilesOutput{}, nil
	}
	stat, err := os.Stat(v.Path)
	if err != nil {
		return errorResult("failed to access path: " + err.Error()), SearchFilesOutput{}, nil
	}
	if !stat.IsDir() {
		return errorResult(ErrPathMustBeDirectory.Error()), SearchFilesOutput{}, nil
	}
	maxResults := input.MaxResults
	if maxResults <= 0 {
		maxResults = defaultMaxResults
	}
	sortBy, sortErr := normalizeSortBy(input.SortBy)
	if sortErr != nil {
		return errorResult(sortErr.Error()), SearchFilesOutput{}, nil
	}
	if sortBy == "" && input.Reverse {
		sortBy = "name"
	}
	results, truncated, err := searchFilesWithOptions(ctx, v.Path, input.Pattern, input.ExcludePatterns, h.ResolvedAllowedDirs(), maxResults, sortBy, input.Reverse, shouldRespectGitignore(input.RespectGitignore))
	if err != nil {
		if err == context.Canceled || err == context.DeadlineExceeded {
			return errorResult("search cancelled"), SearchFilesOutput{}, nil
		}
		return errorResult("search failed: " + err.Error()), SearchFilesOutput{}, nil
	}
	return &mcp.CallToolResult{}, SearchFilesOutput{Files: results, Truncated: truncated}, nil
}

func shouldRespectGitignore(value *bool) bool {
	return value == nil || *value
}

// searchFilesWithOptions keeps result memory bounded by maxResults while still
// selecting the globally correct results for reverse, size, and mtime sorting.
func searchFilesWithOptions(ctx context.Context, rootPath, pattern string, excludePatterns, allowedDirs []string, maxResults int, sortBy string, reverse, respectGitignore bool) ([]string, bool, error) {
	if maxResults <= 0 {
		return []string{}, false, nil
	}
	if sortBy == "" && !reverse {
		results := make([]string, 0, min(maxResults, 64))
		truncated := false
		err := filesystem.Walk(ctx, rootPath, filesystem.WalkOptions{
			ResolvedAllowedDirs: allowedDirs,
			RespectGitignore:    respectGitignore,
			Exclude: func(entry filesystem.Entry) bool {
				return shouldExcludePath(filepath.ToSlash(entry.RelativePath), excludePatterns)
			},
			OnError: func(path string, _ int, err error) error {
				slog.Debug("skipping path due to error", "path", path, "error", err)
				return nil
			},
		}, func(entry filesystem.Entry) (filesystem.WalkAction, error) {
			if !matchGlobPattern(filepath.ToSlash(entry.RelativePath), pattern) {
				return filesystem.WalkContinue, nil
			}
			results = append(results, entry.Path)
			if len(results) >= maxResults {
				truncated = true
				return filesystem.WalkStop, nil
			}
			return filesystem.WalkContinue, nil
		})
		return results, truncated, err
	}

	collector := &boundedPathHeap{sortBy: sortBy, reverse: reverse}
	matchedCount := 0
	err := filesystem.Walk(ctx, rootPath, filesystem.WalkOptions{
		ResolvedAllowedDirs: allowedDirs,
		RespectGitignore:    respectGitignore,
		Exclude: func(entry filesystem.Entry) bool {
			return shouldExcludePath(filepath.ToSlash(entry.RelativePath), excludePatterns)
		},
		OnError: func(path string, _ int, err error) error {
			slog.Debug("skipping path due to error", "path", path, "error", err)
			return nil
		},
	}, func(entry filesystem.Entry) (filesystem.WalkAction, error) {
		if !matchGlobPattern(filepath.ToSlash(entry.RelativePath), pattern) {
			return filesystem.WalkContinue, nil
		}
		sortable, statErr := statSortablePath(entry.Path)
		if statErr != nil {
			return filesystem.WalkContinue, nil
		}
		matchedCount++
		collector.add(sortable, maxResults)
		return filesystem.WalkContinue, nil
	})
	if err != nil {
		return nil, false, err
	}
	selected := collector.sorted()
	results := make([]string, len(selected))
	for index, item := range selected {
		results[index] = item.path
	}
	return results, matchedCount > maxResults, nil
}

// matchGlobPattern matches a path against a glob pattern, supporting ** for recursive matching
func matchGlobPattern(path, pattern string) bool {
	// Normalize pattern to use forward slashes
	pattern = filepath.ToSlash(pattern)

	// Handle ** patterns (recursive glob)
	if strings.Contains(pattern, "**") {
		return matchDoubleStarPattern(path, pattern)
	}

	// Standard glob match using filepath.Match
	matched, err := filepath.Match(pattern, path)
	if err == nil && matched {
		return true
	}

	// Also try matching just the filename for patterns without path separators
	if !strings.Contains(pattern, "/") {
		filename := filepath.Base(path)
		matched, err = filepath.Match(pattern, filename)
		if err == nil && matched {
			return true
		}
	}

	return false
}

// matchDoubleStarPattern handles patterns containing **
func matchDoubleStarPattern(path, pattern string) bool {
	// Split pattern into parts around **
	parts := strings.Split(pattern, "**")

	if len(parts) == 2 {
		prefix := strings.TrimSuffix(parts[0], "/")
		suffix := strings.TrimPrefix(parts[1], "/")

		// Pattern like "**/*.ext" - match suffix against any subpath
		if prefix == "" {
			// Try matching the suffix against the path or any part of it
			if suffix != "" {
				// Match the suffix pattern against the filename or path ending
				return matchSuffix(path, suffix)
			}
			// Pattern is just "**" - matches everything
			return true
		}

		// Pattern like "dir/**" - match prefix then anything
		if suffix == "" {
			return strings.HasPrefix(path, prefix+"/") || path == prefix
		}

		// Pattern like "dir/**/file.ext"
		if strings.HasPrefix(path, prefix+"/") || prefix == "" {
			remaining := path
			if prefix != "" {
				remaining = strings.TrimPrefix(path, prefix+"/")
			}
			return matchSuffix(remaining, suffix)
		}
	}

	return false
}

// matchSuffix checks if the path ends with a pattern match
func matchSuffix(path, suffixPattern string) bool {
	// Try matching the entire path
	matched, err := filepath.Match(suffixPattern, path)
	if err == nil && matched {
		return true
	}

	// Try matching just the filename
	filename := filepath.Base(path)
	matched, err = filepath.Match(suffixPattern, filename)
	if err == nil && matched {
		return true
	}

	// Try matching the path with the suffix pattern at any depth
	parts := strings.Split(path, "/")
	for i := range parts {
		subpath := strings.Join(parts[i:], "/")
		matched, err = filepath.Match(suffixPattern, subpath)
		if err == nil && matched {
			return true
		}
	}

	return false
}

func containsGlobChars(pattern string) bool {
	return strings.ContainsAny(pattern, "*?[")
}

// shouldExcludePath checks if a path matches any of the exclude patterns.
func shouldExcludePath(path string, patterns []string) bool {
	for _, pattern := range patterns {
		pattern = filepath.ToSlash(pattern)

		// Try glob match
		if matchGlobPattern(path, pattern) {
			return true
		}

		// Also check if the path contains the pattern as a directory component
		if !containsGlobChars(pattern) {
			pathParts := strings.Split(path, "/")
			for _, part := range pathParts {
				if part == pattern {
					return true
				}
			}
		}
	}
	return false
}
