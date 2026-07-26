package handler

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
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
	"github.com/zoster81/mcp-file-tools/internal/operation"
	"github.com/zoster81/mcp-file-tools/internal/textstream"
)

const (
	defaultMaxMatches = 1000
	binaryCheckSize   = 8192 // 8KB to catch files with text header but binary payload
	grepMatchOverhead = 256
	grepLineOverhead  = 32
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
	matches, filesMatched, truncated, err := h.searchFiles(ctx, files, re, input, maxMatches)
	if err != nil {
		return errorResultFromError(err), GrepOutput{}, nil
	}
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

type grepPlan struct {
	path               string
	worstRetainedBytes int64
}

// searchFiles keeps deterministic file order while bounding both match count
// and retained output/context memory. Parallelism is used only when the
// aggregate decoded worst case fits the configured budget.
func (h *Handler) searchFiles(ctx context.Context, files []string, re *regexp.Regexp, input GrepInput, maxMatches int) ([]GrepMatch, int, bool, error) {
	budget := h.memoryBudget()
	plans, worstTotal := h.planGrep(files, input, maxMatches)
	maxWorkers := 0
	if worstTotal > budget {
		maxWorkers = 1
	}

	remaining := maxMatches
	var remainingMatches atomic.Int64
	remainingMatches.Store(int64(maxMatches))
	var remainingOutput atomic.Int64
	remainingOutput.Store(budget)
	allMatches := make([]GrepMatch, 0, min(maxMatches, 64))
	matchedFiles := make(map[string]struct{})
	truncated := false
	var terminalErr error

	concurrency.ProcessOrdered(ctx, plans, concurrency.Options{MaxWorkers: maxWorkers}, func(ctx context.Context, _ int, plan grepPlan) grepFileResult {
		limit := int(remainingMatches.Load())
		if limit <= 0 {
			limit = 1
		}
		fileBudget := plan.worstRetainedBytes
		if maxWorkers == 1 || fileBudget <= 0 {
			fileBudget = remainingOutput.Load()
		}
		if fileBudget <= 0 {
			return grepFileResult{err: grepBudgetError(plan.path, 0)}
		}
		return h.searchSingleFileWithBudget(ctx, plan.path, re, input, limit, fileBudget)
	}, func(_ int, current grepFileResult) bool {
		if current.err != nil {
			if operation.KindOf(current.err) == operation.KindLimit {
				terminalErr = current.err
				return false
			}
			return ctx.Err() == nil
		}
		if len(current.matches) > 0 {
			take := min(remaining, len(current.matches))
			selected := current.matches[:take]
			allMatches = append(allMatches, selected...)
			remaining -= take
			remainingMatches.Store(int64(remaining))
			remainingOutput.Add(-grepMatchesRetainedBytes(selected))
			for _, match := range selected {
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

	return allMatches, len(matchedFiles), truncated, terminalErr
}

func (h *Handler) planGrep(files []string, input GrepInput, maxMatches int) ([]grepPlan, int64) {
	plans := make([]grepPlan, len(files))
	var total int64
	contextFactor := saturatingAdd(2, int64(max(0, input.ContextBefore)))
	contextFactor = saturatingAdd(contextFactor, int64(max(0, input.ContextAfter)))
	for index, path := range files {
		plans[index].path = path
		info, err := os.Stat(path)
		if err != nil || !info.Mode().IsRegular() {
			plans[index].worstRetainedBytes = math.MaxInt64
			total = math.MaxInt64
			continue
		}
		decoded := worstDecodedBytes(info.Size())
		contentWorst := saturatingMultiply(decoded, contextFactor)
		metadataPerMatch := int64(grepMatchOverhead + len(path) + 64)
		metadataWorst := saturatingMultiply(int64(maxMatches), metadataPerMatch)
		plans[index].worstRetainedBytes = saturatingAdd(contentWorst, metadataWorst)
		total = saturatingAdd(total, plans[index].worstRetainedBytes)
	}
	return plans, total
}

func saturatingMultiply(first, second int64) int64 {
	if first <= 0 || second <= 0 {
		return 0
	}
	if first > math.MaxInt64/second {
		return math.MaxInt64
	}
	return first * second
}

func saturatingAdd(first, second int64) int64 {
	if second > 0 && first > math.MaxInt64-second {
		return math.MaxInt64
	}
	return first + second
}

var errStopGrepScan = errors.New("grep scan complete")

type pendingGrepMatch struct {
	match          GrepMatch
	remainingAfter int
}

// searchSingleFile retains the compatibility helper used by focused tests.
func (h *Handler) searchSingleFile(ctx context.Context, path string, re *regexp.Regexp, input GrepInput, maxMatches int) grepFileResult {
	return h.searchSingleFileWithBudget(ctx, path, re, input, maxMatches, h.memoryBudget())
}

func grepBudgetError(path string, budget int64) error {
	return operation.Wrap(
		operation.KindLimit,
		"grep_text_files",
		path,
		fmt.Errorf("retained grep state exceeds the %d-byte grep output budget", budget),
	)
}

func grepMatchesRetainedBytes(matches []GrepMatch) int64 {
	var total int64
	for _, match := range matches {
		total = saturatingAdd(total, int64(grepMatchOverhead+len(match.Path)+len(match.Encoding)+len(match.Text)))
		for _, line := range match.Before {
			total = saturatingAdd(total, int64(grepLineOverhead+len(line)))
		}
		for _, line := range match.After {
			total = saturatingAdd(total, int64(grepLineOverhead+len(line)))
		}
	}
	return total
}

// searchSingleFileWithBudget decodes and searches one file incrementally while
// bounding line, context, match, and binary-probe state.
func (h *Handler) searchSingleFileWithBudget(ctx context.Context, path string, re *regexp.Regexp, input GrepInput, maxMatches int, maxOutputBytes int64) grepFileResult {
	result := grepFileResult{}
	if maxMatches <= 0 {
		return result
	}

	validated := h.ValidatePath(path)
	if !validated.Ok() {
		result.err = validated.Err
		return result
	}

	stream, err := h.openDecodedTextStream(ctx, validated.Path, input.Encoding)
	if err != nil {
		result.err = err
		return result
	}
	defer stream.Close()

	prefix := make([]byte, binaryCheckSize)
	read, prefixErr := io.ReadFull(stream.Reader, prefix)
	if prefixErr != nil && !errors.Is(prefixErr, io.EOF) && !errors.Is(prefixErr, io.ErrUnexpectedEOF) {
		result.err = prefixErr
		return result
	}
	prefix = prefix[:read]
	if len(prefix) == 0 || isLikelyBinaryText(string(trimIncompleteUTF8Suffix(prefix))) {
		return result
	}
	reader := io.MultiReader(bytes.NewReader(prefix), stream.Reader)

	result.matches = make([]GrepMatch, 0, min(maxMatches, 16))
	beforeCapacity := min(max(0, input.ContextBefore), 64)
	before := make([]string, 0, beforeCapacity)
	pending := make([]pendingGrepMatch, 0, min(maxMatches, 16))
	logicalLine := 0
	var retainedBytes int64

	reserve := func(amount int64) error {
		if amount < 0 || amount > maxOutputBytes-retainedBytes {
			return grepBudgetError(validated.Path, maxOutputBytes)
		}
		retainedBytes += amount
		return nil
	}
	release := func(amount int64) {
		retainedBytes -= amount
		if retainedBytes < 0 {
			retainedBytes = 0
		}
	}

	processLine := func(line []byte) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		logicalLine++

		if len(pending) > 0 {
			remainingPending := pending[:0]
			for _, current := range pending {
				if current.remainingAfter > 0 {
					if err := reserve(int64(grepLineOverhead + len(line))); err != nil {
						return err
					}
					current.match.After = append(current.match.After, string(line))
					current.remainingAfter--
				}
				if current.remainingAfter == 0 {
					result.matches = append(result.matches, current.match)
				} else {
					remainingPending = append(remainingPending, current)
				}
			}
			pending = remainingPending
		}

		if loc := re.FindIndex(line); loc != nil {
			selected := len(result.matches) + len(pending)
			if selected < maxMatches {
				required := int64(grepMatchOverhead + len(validated.Path) + len(stream.Charset) + len(line))
				for _, contextLine := range before {
					required = saturatingAdd(required, int64(grepLineOverhead+len(contextLine)))
				}
				if err := reserve(required); err != nil {
					return err
				}
				match := GrepMatch{
					Path:     validated.Path,
					Line:     logicalLine,
					Column:   loc[0] + 1,
					Text:     string(line),
					Encoding: stream.Charset,
				}
				if len(before) > 0 {
					match.Before = make([]string, len(before))
					for index, contextLine := range before {
						match.Before[index] = strings.Clone(contextLine)
					}
				}
				if input.ContextAfter > 0 {
					match.After = make([]string, 0, min(input.ContextAfter, 64))
					pending = append(pending, pendingGrepMatch{match: match, remainingAfter: input.ContextAfter})
				} else {
					result.matches = append(result.matches, match)
				}
			} else {
				result.truncated = true
			}
		}

		if input.ContextBefore > 0 {
			if len(before) == input.ContextBefore {
				release(int64(grepLineOverhead + len(before[0])))
				copy(before, before[1:])
				before = before[:len(before)-1]
			}
			if err := reserve(int64(grepLineOverhead + len(line))); err != nil {
				return err
			}
			before = append(before, string(line))
		}
		if result.truncated && len(pending) == 0 {
			return errStopGrepScan
		}
		return nil
	}

	_, scanErr := textstream.ScanLines(ctx, reader, textstream.DefaultMaxLineBytes, func(line textstream.Line) error {
		for _, segment := range bytes.Split(line.Data, []byte{'\r'}) {
			if err := processLine(segment); err != nil {
				return err
			}
		}
		return nil
	})
	if scanErr != nil && !errors.Is(scanErr, errStopGrepScan) {
		result.matches = nil
		result.err = scanErr
		return result
	}

	for _, current := range pending {
		result.matches = append(result.matches, current.match)
	}
	if scanErr == nil {
		if _, err := stream.Finish(); err != nil {
			result.matches = nil
			result.err = err
		}
	}
	return result
}

// trimIncompleteUTF8Suffix removes only a trailing partial UTF-8 sequence from
// a bounded probe. Invalid bytes inside the probe remain visible to binary
// classification.
func trimIncompleteUTF8Suffix(data []byte) []byte {
	if utf8.Valid(data) {
		return data
	}
	for trim := 1; trim < utf8.UTFMax && trim < len(data); trim++ {
		candidate := data[:len(data)-trim]
		if utf8.Valid(candidate) {
			return candidate
		}
	}
	return data
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
