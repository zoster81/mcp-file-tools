package handler

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zoster81/mcp-file-tools/internal/encoding"
	"github.com/zoster81/mcp-file-tools/internal/filesystem"
	"github.com/zoster81/mcp-file-tools/internal/security"
)

const defaultMaxFiles = 1000

// HandleTree returns a compact indented tree view optimized for AI consumption.
// Uses ~70-80% fewer tokens than JSON format.
func (h *Handler) HandleTree(ctx context.Context, req *mcp.CallToolRequest, input TreeInput) (*mcp.CallToolResult, TreeOutput, error) {
	v := h.ValidatePath(input.Path)
	if !v.Ok() {
		return v.Result, TreeOutput{}, nil
	}
	stat, err := os.Stat(v.Path)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to access path: %v", err)), TreeOutput{}, nil
	}
	if !stat.IsDir() {
		return errorResult(ErrPathMustBeDirectory.Error()), TreeOutput{}, nil
	}
	maxFiles := input.MaxFiles
	if maxFiles == 0 {
		maxFiles = defaultMaxFiles
	}
	state := &treeState{
		maxFiles:     maxFiles,
		dirsOnly:     input.DirsOnly,
		exclude:      input.Exclude,
		showEncoding: input.ShowEncoding,
	}
	allowedDirs := h.ResolvedAllowedDirs()
	var sb strings.Builder
	_ = filesystem.Walk(ctx, v.Path, filesystem.WalkOptions{
		ResolvedAllowedDirs: allowedDirs,
		MaxDepth:            input.MaxDepth,
	}, func(entry filesystem.Entry) (filesystem.WalkAction, error) {
		if state.totalCount() >= state.maxFiles {
			state.truncated = true
			return filesystem.WalkStop, nil
		}
		if shouldExcludeTree(entry.Name, state.exclude) {
			if entry.DirEntry.IsDir() {
				return filesystem.WalkSkipDir, nil
			}
			return filesystem.WalkContinue, nil
		}

		indent := strings.Repeat("  ", entry.Depth-1)
		if entry.DirEntry.IsDir() {
			state.dirCount++
			sb.WriteString(indent)
			sb.WriteString(entry.Name)
			sb.WriteString("/\n")
			return filesystem.WalkContinue, nil
		}
		if state.dirsOnly {
			return filesystem.WalkContinue, nil
		}

		state.fileCount++
		sb.WriteString(indent)
		sb.WriteString(entry.Name)
		if state.showEncoding {
			if safePath, safe := security.ResolvePathSafe(entry.Path, allowedDirs); safe {
				if enc := detectFileEncoding(safePath); enc != "" {
					sb.WriteString("  [")
					sb.WriteString(enc)
					sb.WriteString("]")
				}
			}
		}
		sb.WriteString("\n")
		return filesystem.WalkContinue, nil
	})
	if ctx.Err() != nil {
		state.truncated = true
	}
	return &mcp.CallToolResult{}, TreeOutput{
		Tree:      sb.String(),
		FileCount: state.fileCount,
		DirCount:  state.dirCount,
		Truncated: state.truncated,
	}, nil
}

type treeState struct {
	maxFiles     int
	dirsOnly     bool
	exclude      []string
	showEncoding bool
	fileCount    int
	dirCount     int
	truncated    bool
}

func (s *treeState) totalCount() int {
	return s.fileCount + s.dirCount
}

// detectFileEncoding returns the detected encoding name for a file, or "" on error.
// Uses sample mode for speed since this is called per-file in tree traversal.
func detectFileEncoding(path string) string {
	result, err := encoding.DetectFromFile(path, "sample")
	if err != nil || result.Confidence < encoding.MinConfidenceThreshold {
		return ""
	}
	return result.Charset
}

func shouldExcludeTree(name string, patterns []string) bool {
	for _, pattern := range patterns {
		if name == pattern {
			return true
		}
		if matched, _ := filepath.Match(pattern, name); matched {
			return true
		}
	}
	return false
}
