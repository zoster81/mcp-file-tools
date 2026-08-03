package handler

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const maxPatchHunks = 1000

var unifiedHunkHeader = regexp.MustCompile(`^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@(?: .*)?$`)

type patchHunk struct {
	oldStart int
	oldCount int
	newStart int
	newCount int
	lines    []string
}

func applyUnifiedPatch(content, patchText, targetPath string, maxBytes int64) (string, error) {
	hunks, err := parseUnifiedPatch(patchText, targetPath, maxBytes)
	if err != nil {
		return "", err
	}
	return applyPatchHunks(content, hunks)
}

func parseUnifiedPatch(patchText, targetPath string, maxBytes int64) ([]patchHunk, error) {
	if strings.TrimSpace(patchText) == "" {
		return nil, fmt.Errorf("patch is empty")
	}
	if int64(len(patchText)) > maxBytes {
		return nil, fmt.Errorf("patch exceeds the %d-byte file edit limit", maxBytes)
	}
	lines := strings.Split(ConvertLineEndings(patchText, LineEndingLF), "\n")
	if len(lines) < 3 || !strings.HasPrefix(lines[0], "--- ") || !strings.HasPrefix(lines[1], "+++ ") {
		return nil, fmt.Errorf("patch must start with one ---/+++ file header pair")
	}
	if err := validatePatchHeaderPath(strings.TrimSpace(strings.TrimPrefix(lines[0], "--- ")), targetPath); err != nil {
		return nil, err
	}
	if err := validatePatchHeaderPath(strings.TrimSpace(strings.TrimPrefix(lines[1], "+++ ")), targetPath); err != nil {
		return nil, err
	}

	hunks := make([]patchHunk, 0)
	for index := 2; index < len(lines); {
		if lines[index] == "" {
			index++
			continue
		}
		if strings.HasPrefix(lines[index], "--- ") || strings.HasPrefix(lines[index], "+++ ") {
			return nil, fmt.Errorf("multi-file patches are not supported")
		}
		match := unifiedHunkHeader.FindStringSubmatch(lines[index])
		if match == nil {
			return nil, fmt.Errorf("invalid hunk header at patch line %d", index+1)
		}
		hunk := patchHunk{
			oldStart: mustPatchInt(match[1]),
			oldCount: patchCount(match[2]),
			newStart: mustPatchInt(match[3]),
			newCount: patchCount(match[4]),
		}
		index++
		oldSeen, newSeen := 0, 0
		for index < len(lines) && unifiedHunkHeader.FindStringSubmatch(lines[index]) == nil {
			line := lines[index]
			if strings.HasPrefix(line, "--- ") || strings.HasPrefix(line, "+++ ") {
				return nil, fmt.Errorf("multi-file patches are not supported")
			}
			if line == `\ No newline at end of file` {
				return nil, fmt.Errorf("patches with no-newline markers are not supported")
			}
			if line == "" && index == len(lines)-1 {
				break
			}
			if line == "" {
				return nil, fmt.Errorf("patch body line %d lacks a prefix", index+1)
			}
			switch line[0] {
			case ' ':
				oldSeen++
				newSeen++
			case '-':
				oldSeen++
			case '+':
				newSeen++
			default:
				return nil, fmt.Errorf("invalid patch line prefix at line %d", index+1)
			}
			hunk.lines = append(hunk.lines, line)
			index++
		}
		if oldSeen != hunk.oldCount || newSeen != hunk.newCount {
			return nil, fmt.Errorf("hunk count mismatch: header declares -%d +%d, body has -%d +%d", hunk.oldCount, hunk.newCount, oldSeen, newSeen)
		}
		hunks = append(hunks, hunk)
		if len(hunks) > maxPatchHunks {
			return nil, fmt.Errorf("patch contains more than %d hunks", maxPatchHunks)
		}
	}
	if len(hunks) == 0 {
		return nil, fmt.Errorf("patch contains no hunks")
	}
	return hunks, nil
}

func validatePatchHeaderPath(header, targetPath string) error {
	pathField := strings.Fields(header)
	if len(pathField) == 0 || pathField[0] == "/dev/null" {
		return fmt.Errorf("patch creation/deletion is not supported")
	}
	patchPath := strings.TrimPrefix(filepath.ToSlash(pathField[0]), "a/")
	patchPath = strings.TrimPrefix(patchPath, "b/")
	if filepath.Base(filepath.FromSlash(patchPath)) != filepath.Base(targetPath) {
		return fmt.Errorf("patch header %q does not match target file %q", pathField[0], filepath.Base(targetPath))
	}
	return nil
}

func mustPatchInt(value string) int {
	parsed, _ := strconv.Atoi(value)
	return parsed
}

func patchCount(value string) int {
	if value == "" {
		return 1
	}
	return mustPatchInt(value)
}

func applyPatchHunks(content string, hunks []patchHunk) (string, error) {
	source := strings.Split(content, "\n")
	output := make([]string, 0, len(source))
	sourceIndex := 0
	lastOldEnd := 0
	for _, hunk := range hunks {
		if hunk.oldStart < 0 || hunk.newStart < 0 || hunk.oldStart == 0 && hunk.oldCount != 0 || hunk.newStart == 0 && hunk.newCount != 0 {
			return "", fmt.Errorf("zero hunk line numbers are valid only for empty ranges")
		}
		hunkIndex := hunk.oldStart - 1
		if hunk.oldCount == 0 {
			hunkIndex = hunk.oldStart
		}
		if hunkIndex < sourceIndex || hunk.oldStart < lastOldEnd || hunkIndex > len(source) {
			return "", fmt.Errorf("patch hunks must be ordered, non-overlapping, and inside the target file")
		}
		output = append(output, source[sourceIndex:hunkIndex]...)
		expectedNewStart := len(output) + 1
		if hunk.newCount == 0 {
			expectedNewStart = len(output)
		}
		if hunk.newStart != expectedNewStart {
			return "", fmt.Errorf("hunk new start is %d, want %d", hunk.newStart, expectedNewStart)
		}
		sourceIndex = hunkIndex
		for _, patchLine := range hunk.lines {
			text := patchLine[1:]
			switch patchLine[0] {
			case ' ':
				if sourceIndex >= len(source) || source[sourceIndex] != text {
					return "", fmt.Errorf("patch context mismatch at source line %d", sourceIndex+1)
				}
				output = append(output, text)
				sourceIndex++
			case '-':
				if sourceIndex >= len(source) || source[sourceIndex] != text {
					return "", fmt.Errorf("patch deletion mismatch at source line %d", sourceIndex+1)
				}
				sourceIndex++
			case '+':
				output = append(output, text)
			}
		}
		lastOldEnd = hunk.oldStart + hunk.oldCount
	}
	output = append(output, source[sourceIndex:]...)
	return strings.Join(output, "\n"), nil
}
