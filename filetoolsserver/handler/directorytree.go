package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dimitar-grigorov/mcp-file-tools/internal/filesystem"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// HandleDirectoryTree returns a recursive tree view of files and directories as JSON.
func (h *Handler) HandleDirectoryTree(ctx context.Context, req *mcp.CallToolRequest, input DirectoryTreeInput) (*mcp.CallToolResult, DirectoryTreeOutput, error) {
	v := h.ValidatePath(input.Path)
	if !v.Ok() {
		return v.Result, DirectoryTreeOutput{}, nil
	}
	stat, err := os.Stat(v.Path)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to access path: %v", err)), DirectoryTreeOutput{}, nil
	}
	if !stat.IsDir() {
		return errorResult(ErrPathMustBeDirectory.Error()), DirectoryTreeOutput{}, nil
	}
	resolvedDirs := h.ResolvedAllowedDirs()
	tree, err := buildTree(ctx, v.Path, input.ExcludePatterns, resolvedDirs)
	if err != nil {
		if err == context.Canceled || err == context.DeadlineExceeded {
			return errorResult("operation cancelled"), DirectoryTreeOutput{}, nil
		}
		return errorResult(fmt.Sprintf("failed to build directory tree: %v", err)), DirectoryTreeOutput{}, nil
	}
	jsonBytes, err := json.MarshalIndent(tree, "", "  ")
	if err != nil {
		return errorResult(fmt.Sprintf("failed to marshal tree to JSON: %v", err)), DirectoryTreeOutput{}, nil
	}
	output := DirectoryTreeOutput{Tree: string(jsonBytes)}
	return &mcp.CallToolResult{}, output, nil
}

type treeEntryLocation struct {
	parent *[]TreeEntry
	index  int
	name   string
}

// buildTree builds a hierarchical result from the shared secure walker.
func buildTree(ctx context.Context, dirPath string, excludePatterns []string, allowedDirs []string) ([]TreeEntry, error) {
	result := make([]TreeEntry, 0)
	childrenByRelativePath := map[string]*[]TreeEntry{"": &result}
	directoryLocations := make(map[string]treeEntryLocation)

	err := filesystem.Walk(ctx, dirPath, filesystem.WalkOptions{
		ResolvedAllowedDirs: allowedDirs,
		Exclude: func(entry filesystem.Entry) bool {
			return shouldExclude(entry.Name, excludePatterns)
		},
		OnError: func(path string, depth int, err error) error {
			if depth == 0 {
				return err
			}
			location, ok := directoryLocations[filepath.Clean(path)]
			if !ok {
				return nil
			}
			entries := *location.parent
			if location.index >= 0 && location.index < len(entries) && entries[location.index].Name == location.name {
				*location.parent = append(entries[:location.index], entries[location.index+1:]...)
			}
			delete(directoryLocations, filepath.Clean(path))
			return nil
		},
	}, func(entry filesystem.Entry) (filesystem.WalkAction, error) {
		parentRelativePath := filepath.Dir(entry.RelativePath)
		if parentRelativePath == "." {
			parentRelativePath = ""
		}
		parent := childrenByRelativePath[parentRelativePath]
		if parent == nil {
			return filesystem.WalkSkipDir, nil
		}

		treeEntry := TreeEntry{Name: entry.Name, Type: "file"}
		if entry.DirEntry.IsDir() {
			children := make([]TreeEntry, 0)
			treeEntry.Type = "directory"
			treeEntry.Children = &children
		}
		*parent = append(*parent, treeEntry)

		if entry.DirEntry.IsDir() {
			childrenByRelativePath[entry.RelativePath] = treeEntry.Children
			directoryLocations[filepath.Clean(entry.ResolvedPath)] = treeEntryLocation{
				parent: parent,
				index:  len(*parent) - 1,
				name:   entry.Name,
			}
		}
		return filesystem.WalkContinue, nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// shouldExclude checks if a name matches any of the exclude patterns
func shouldExclude(name string, patterns []string) bool {
	for _, pattern := range patterns {
		// Try exact match first
		if name == pattern {
			return true
		}

		// Try glob pattern match
		matched, err := filepath.Match(pattern, name)
		if err == nil && matched {
			return true
		}

		// For patterns without wildcards, also try as substring/prefix
		// This mimics the JS behavior for patterns like "node_modules"
		if !containsGlobChars(pattern) {
			if strings.Contains(name, pattern) {
				return true
			}
		}
	}
	return false
}

// containsGlobChars checks if pattern contains glob metacharacters
func containsGlobChars(pattern string) bool {
	return strings.ContainsAny(pattern, "*?[")
}
