package handler

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync/atomic"
	"unicode"
	"unicode/utf8"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zoster81/mcp-file-tools/internal/concurrency"
	"github.com/zoster81/mcp-file-tools/internal/filesystem"
)

const (
	defaultMaxMatches = 1000
	binaryCheckSize   = 8192 // 8KB to catch files with text header but binary payload
)

// HandleGrep searches for a pattern in files with encoding support.
func (h *Handler) HandleGrep(ctx context.Context, req *mcp.CallToolRequest, input GrepInput) (*mcp.CallToolResult, GrepOutput, error) {
	if input.Pattern == "" {
		return errorResult("pattern is required"), GrepOutput{}, nil
	}
	if len(input.Paths) == 0 {
		return errorResult("paths is required"), GrepOutput{}, nil
	}
	re, err := compilePattern(input.Pattern, input.CaseSensitive)
	if err != nil {
		return errorResult(fmt.Sprintf("invalid regex pattern: %v", err)), GrepOutput{}, nil
	}
	maxMatches := input.MaxMatches
	if maxMatches <= 0 {
		maxMatches = defaultMaxMatches
	}
	files := h.collectFiles(ctx, input.Paths, input.Include, input.Exclude)
	if len(files) == 0 {
		return &mcp.CallToolResult{}, GrepOutput{Matches: []GrepMatch{}, FilesSearched: 0}, nil
	}
	matches, filesMatched, truncated := h.searchFiles(ctx, files, re, input, maxMatches)
	return &mcp.CallToolResult{}, GrepOutput{
		Matches:       matches,
		TotalMatches:  len(matches),
		FilesSearched: len(files),
		FilesMatched:  filesMatched,
		Truncated:     truncated,
	}, nil
}

// compilePattern compiles the regex pattern with optional case sensitivity.
func compilePattern(pattern string, caseSensitive *bool) (*regexp.Regexp, error) {
	if caseSensitive != nil && !*caseSensitive {
		pattern = "(?i)" + pattern
	}
	return regexp.Compile(pattern)
}

// collectFiles gathers all files to search from the given paths.
func (h *Handler) collectFiles(ctx context.Context, paths []string, include, exclude string) []string {
	var files []string
	seen := make(map[string]bool)
	allowedDirs := h.ResolvedAllowedDirs()
	for _, path := range paths {
		// Check for cancellation between paths
		select {
		case <-ctx.Done():
			return files
		default:
		}
		v := h.ValidatePath(path)
		if !v.Ok() {
			continue
		}
		info, err := os.Stat(v.Path)
		if err != nil {
			continue
		}
		if info.IsDir() {
			err := filesystem.Walk(ctx, v.Path, filesystem.WalkOptions{
				ResolvedAllowedDirs: allowedDirs,
				OnError: func(path string, _ int, err error) error {
					slog.Debug("skipping path due to error", "path", path, "error", err)
					return nil
				},
			}, func(entry filesystem.Entry) (filesystem.WalkAction, error) {
				if entry.DirEntry.IsDir() {
					return filesystem.WalkContinue, nil
				}
				if shouldIncludeFile(entry.Path, include, exclude) && !seen[entry.ResolvedPath] {
					seen[entry.ResolvedPath] = true
					files = append(files, entry.ResolvedPath)
				}
				return filesystem.WalkContinue, nil
			})
			if err != nil && ctx.Err() != nil {
				return files
			}
		} else if shouldIncludeFile(v.Path, include, exclude) && !seen[v.Path] {
			seen[v.Path] = true
			files = append(files, v.Path)
		}
	}
	return files
}

// shouldIncludeFile checks if a file matches include/exclude patterns.
// Matches against both full path (with forward slashes) and basename.
func shouldIncludeFile(path string, include, exclude string) bool {
	base := filepath.Base(path)
	normalized := filepath.ToSlash(path)
	if exclude != "" {
		if matchedBase, _ := filepath.Match(exclude, base); matchedBase {
			return false
		}
		if matchedPath, _ := filepath.Match(exclude, normalized); matchedPath {
			return false
		}
	}
	if include != "" {
		if matchedBase, _ := filepath.Match(include, base); matchedBase {
			return true
		}
		if matchedPath, _ := filepath.Match(include, normalized); matchedPath {
			return true
		}
		return false
	}
	return true
}

type grepFileResult struct {
	matches   []GrepMatch
	truncated bool
	err       error
}

// searchFiles searches files concurrently while committing results in file order.
// Only a bounded window is in flight, so per-file results cannot grow without limit.
func (h *Handler) searchFiles(ctx context.Context, files []string, re *regexp.Regexp, input GrepInput, maxMatches int) ([]GrepMatch, int, bool) {
	remaining := maxMatches
	var remainingBudget atomic.Int64
	remainingBudget.Store(int64(maxMatches))
	allMatches := make([]GrepMatch, 0, min(maxMatches, 64))
	matchedFiles := make(map[string]struct{})
	truncated := false

	concurrency.ProcessOrdered(ctx, files, concurrency.Options{}, func(ctx context.Context, _ int, path string) grepFileResult {
		limit := int(remainingBudget.Load())
		if limit <= 0 {
			limit = 1
		}
		return h.searchSingleFile(ctx, path, re, input, limit)
	}, func(_ int, current grepFileResult) bool {
		if current.err == nil && len(current.matches) > 0 {
			take := min(remaining, len(current.matches))
			allMatches = append(allMatches, current.matches[:take]...)
			remaining -= take
			remainingBudget.Store(int64(remaining))
			for _, match := range current.matches[:take] {
				matchedFiles[match.Path] = struct{}{}
			}
			if current.truncated || take < len(current.matches) {
				truncated = true
			}
		}

		if remaining == 0 && truncated {
			return false
		}
		return ctx.Err() == nil
	})

	return allMatches, len(matchedFiles), truncated
}

// searchSingleFile decodes and searches one file, returning at most maxMatches results.
func (h *Handler) searchSingleFile(ctx context.Context, path string, re *regexp.Regexp, input GrepInput, maxMatches int) grepFileResult {
	result := grepFileResult{}
	if maxMatches <= 0 {
		return result
	}

	validated := h.ValidatePath(path)
	if !validated.Ok() {
		result.err = validated.Err
		return result
	}

	document, err := h.readTextDocument(ctx, validated.Path, input.Encoding)
	if err != nil {
		result.err = err
		return result
	}
	if document.Text == "" || isLikelyBinaryText(document.Text) {
		return result
	}

	lines := splitGrepLines(document.Text)
	result.matches = make([]GrepMatch, 0, min(maxMatches, 16))
	for lineNum, line := range lines {
		select {
		case <-ctx.Done():
			result.matches = nil
			result.err = ctx.Err()
			return result
		default:
		}

		loc := re.FindStringIndex(line)
		if loc == nil {
			continue
		}
		if len(result.matches) >= maxMatches {
			result.truncated = true
			return result
		}

		match := GrepMatch{
			Path:     validated.Path,
			Line:     lineNum + 1,
			Column:   loc[0] + 1,
			Text:     line,
			Encoding: document.Charset,
		}
		if input.ContextBefore > 0 {
			match.Before = getContextBefore(lines, lineNum, input.ContextBefore)
		}
		if input.ContextAfter > 0 {
			match.After = getContextAfter(lines, lineNum, input.ContextAfter)
		}
		result.matches = append(result.matches, match)
	}
	return result
}

func splitGrepLines(content string) []string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")
	return strings.Split(content, "\n")
}

// isLikelyBinaryText classifies decoded content instead of raw encoded bytes.
func isLikelyBinaryText(content string) bool {
	if !utf8.ValidString(content) {
		return true
	}

	controlCount := 0
	runeCount := 0
	for byteIndex, r := range content {
		if byteIndex >= binaryCheckSize {
			break
		}
		runeCount++
		if r == 0 {
			return true
		}
		if unicode.IsControl(r) && r != '\n' && r != '\r' && r != '\t' {
			controlCount++
		}
	}
	return runeCount > 0 && controlCount*10 >= runeCount
}

// getContextBefore returns N lines before the given line index.
func getContextBefore(lines []string, lineIdx, count int) []string {
	start := lineIdx - count
	if start < 0 {
		start = 0
	}
	if start >= lineIdx {
		return nil
	}
	return lines[start:lineIdx]
}

// getContextAfter returns N lines after the given line index.
func getContextAfter(lines []string, lineIdx, count int) []string {
	end := lineIdx + count + 1
	if end > len(lines) {
		end = len(lines)
	}
	if lineIdx+1 >= end {
		return nil
	}
	return lines[lineIdx+1 : end]
}
