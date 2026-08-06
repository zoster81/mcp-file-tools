package handler

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zoster81/scripthold/internal/security"
)

// HandleListDirectory lists files in a directory with optional pattern filtering
func (h *Handler) HandleListDirectory(ctx context.Context, req *mcp.CallToolRequest, input ListDirectoryInput) (*mcp.CallToolResult, ListDirectoryOutput, error) {
	v := h.ValidatePath(input.Path)
	if !v.Ok() {
		return v.Result, ListDirectoryOutput{}, nil
	}

	pattern := "*"
	if input.Pattern != "" {
		pattern = input.Pattern
	}

	sortBy, err := normalizeSortBy(input.SortBy)
	if err != nil {
		return errorResult(err.Error()), ListDirectoryOutput{}, nil
	}
	entries, err := os.ReadDir(v.Path)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to read directory: %v", err)), ListDirectoryOutput{}, nil
	}

	type listedEntry struct {
		sortable sortablePath
		isDir    bool
	}
	selected := make([]listedEntry, 0, len(entries))
	for _, entry := range entries {
		matched, matchErr := filepath.Match(pattern, entry.Name())
		if matchErr != nil {
			return errorResult(fmt.Sprintf("invalid pattern: %v", matchErr)), ListDirectoryOutput{}, nil
		}
		if !matched {
			continue
		}
		entryPath := filepath.Join(v.Path, entry.Name())
		sortable := sortablePath{path: entryPath, name: entry.Name()}
		if sortBy == "mtime" || sortBy == "size" {
			resolved, safe := security.ResolvePathSafe(entryPath, h.ResolvedAllowedDirs())
			if !safe {
				continue
			}
			var statErr error
			sortable, statErr = statSortablePath(resolved)
			if statErr != nil {
				continue
			}
			sortable.name = entry.Name()
		}
		selected = append(selected, listedEntry{sortable: sortable, isDir: entry.IsDir()})
	}
	sort.SliceStable(selected, func(i, j int) bool {
		comparison := compareSortable(selected[i].sortable, selected[j].sortable, sortBy)
		if input.Reverse {
			return comparison > 0
		}
		return comparison < 0
	})

	files := make([]string, 0, len(selected))
	for _, entry := range selected {
		prefix := ""
		if entry.isDir {
			prefix = "[DIR] "
		}
		files = append(files, prefix+entry.sortable.name)
	}
	return &mcp.CallToolResult{}, ListDirectoryOutput{Files: files}, nil
}
