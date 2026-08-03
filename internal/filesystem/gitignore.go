package filesystem

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/zoster81/mcp-file-tools/internal/security"
)

const (
	maxGitignoreBytes           = 1 << 20
	maxGitignoreRules           = 10000
	maxGitignorePatternBytes    = 4096
	maxGitignorePatternSegments = 256
)

// ErrInvalidGitignore marks an ignore file that cannot be safely or boundedly
// consumed. Recursive public tools must fail closed instead of silently
// traversing with a different file set.
var ErrInvalidGitignore = errors.New("invalid .gitignore")

type ignoreRule struct {
	pattern       string
	negated       bool
	directoryOnly bool
	anchored      bool
	basenameOnly  bool
}

type ignoreScope struct {
	relativeDir string
	rules       []ignoreRule
}

func loadIgnoreScope(directory, relativeDir string, allowedDirectories []string) (ignoreScope, error) {
	scope := ignoreScope{relativeDir: filepath.ToSlash(relativeDir)}
	ignorePath := filepath.Join(directory, ".gitignore")
	info, err := os.Lstat(ignorePath)
	if errors.Is(err, os.ErrNotExist) {
		return scope, nil
	}
	if err != nil {
		return scope, fmt.Errorf("%w: inspect: %v", ErrInvalidGitignore, err)
	}
	if !info.Mode().IsRegular() {
		return scope, fmt.Errorf("%w: must be a regular file", ErrInvalidGitignore)
	}
	resolvedIgnore, safe := security.ResolvePathSafe(ignorePath, allowedDirectories)
	if !safe {
		return scope, fmt.Errorf("%w: resolves outside allowed directories", ErrInvalidGitignore)
	}
	file, err := os.Open(resolvedIgnore)
	if err != nil {
		return scope, fmt.Errorf("%w: open: %v", ErrInvalidGitignore, err)
	}
	defer file.Close()

	limited := io.LimitReader(file, maxGitignoreBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return scope, fmt.Errorf("%w: read: %v", ErrInvalidGitignore, err)
	}
	if len(data) > maxGitignoreBytes {
		return scope, fmt.Errorf("%w: exceeds the 1 MiB traversal limit", ErrInvalidGitignore)
	}
	rules, err := parseIgnoreRules(string(data))
	if err != nil {
		return scope, fmt.Errorf("%w: parse: %v", ErrInvalidGitignore, err)
	}
	scope.rules = rules
	return scope, nil
}

func parseIgnoreRules(content string) ([]ignoreRule, error) {
	scanner := bufio.NewScanner(strings.NewReader(content))
	scanner.Buffer(make([]byte, 4096), maxGitignoreBytes)
	rules := make([]ignoreRule, 0)
	for scanner.Scan() {
		line := strings.TrimSuffix(scanner.Text(), "\r")
		line = trimUnescapedTrailingSpaces(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		rule := ignoreRule{}
		if strings.HasPrefix(line, "!") {
			rule.negated = true
			line = strings.TrimPrefix(line, "!")
		}
		if strings.HasPrefix(line, `\#`) || strings.HasPrefix(line, `\!`) {
			line = line[1:]
		}
		if strings.HasSuffix(line, "/") {
			rule.directoryOnly = true
			line = strings.TrimSuffix(line, "/")
		}
		line = filepath.ToSlash(line)
		rule.anchored = strings.HasPrefix(line, "/")
		line = strings.TrimPrefix(line, "/")
		if line == "" {
			continue
		}
		if len(line) > maxGitignorePatternBytes {
			return nil, fmt.Errorf("pattern exceeds %d bytes", maxGitignorePatternBytes)
		}
		segments := strings.Split(line, "/")
		if len(segments) > maxGitignorePatternSegments {
			return nil, fmt.Errorf("pattern exceeds %d path segments", maxGitignorePatternSegments)
		}
		for _, segment := range segments {
			if segment == "**" {
				continue
			}
			if _, err := path.Match(segment, ""); err != nil {
				return nil, fmt.Errorf("invalid glob pattern %q: %v", line, err)
			}
		}
		rule.basenameOnly = !strings.Contains(line, "/")
		rule.pattern = line
		rules = append(rules, rule)
		if len(rules) > maxGitignoreRules {
			return nil, fmt.Errorf("rule count exceeds %d", maxGitignoreRules)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return rules, nil
}

func trimUnescapedTrailingSpaces(value string) string {
	for strings.HasSuffix(value, " ") {
		backslashes := 0
		for index := len(value) - 2; index >= 0 && value[index] == '\\'; index-- {
			backslashes++
		}
		if backslashes%2 == 1 {
			return value[:len(value)-2] + " "
		}
		value = strings.TrimSuffix(value, " ")
	}
	return value
}

func ignoredByScopes(scopes []ignoreScope, relativePath string, isDirectory bool) bool {
	relativePath = filepath.ToSlash(strings.TrimPrefix(relativePath, string(filepath.Separator)))
	ignored := false
	for _, scope := range scopes {
		candidate, applicable := pathWithinIgnoreScope(relativePath, scope.relativeDir)
		if !applicable {
			continue
		}
		for _, rule := range scope.rules {
			if rule.directoryOnly && !isDirectory {
				continue
			}
			if rule.matches(candidate) {
				ignored = !rule.negated
			}
		}
	}
	return ignored
}

func pathWithinIgnoreScope(relativePath, scopeDirectory string) (string, bool) {
	if scopeDirectory == "" {
		return relativePath, true
	}
	if relativePath == scopeDirectory {
		return "", true
	}
	prefix := scopeDirectory + "/"
	if !strings.HasPrefix(relativePath, prefix) {
		return "", false
	}
	return strings.TrimPrefix(relativePath, prefix), true
}

func (rule ignoreRule) matches(candidate string) bool {
	candidate = strings.TrimPrefix(filepath.ToSlash(candidate), "/")
	if candidate == "" {
		return false
	}
	if rule.basenameOnly && !rule.anchored {
		for _, segment := range strings.Split(candidate, "/") {
			if matched, err := path.Match(rule.pattern, segment); err == nil && matched {
				return true
			}
		}
		return false
	}
	return matchIgnoreSegments(strings.Split(rule.pattern, "/"), strings.Split(candidate, "/"))
}

func matchIgnoreSegments(patternSegments, pathSegments []string) bool {
	patternIndex, pathIndex := 0, 0
	globstarIndex, globstarPathIndex := -1, -1
	for pathIndex < len(pathSegments) {
		if patternIndex < len(patternSegments) && patternSegments[patternIndex] != "**" {
			matched, err := path.Match(patternSegments[patternIndex], pathSegments[pathIndex])
			if err == nil && matched {
				patternIndex++
				pathIndex++
				continue
			}
		}
		if patternIndex < len(patternSegments) && patternSegments[patternIndex] == "**" {
			globstarIndex = patternIndex
			globstarPathIndex = pathIndex
			patternIndex++
			continue
		}
		if globstarIndex >= 0 {
			globstarPathIndex++
			pathIndex = globstarPathIndex
			patternIndex = globstarIndex + 1
			continue
		}
		return false
	}
	for patternIndex < len(patternSegments) && patternSegments[patternIndex] == "**" {
		patternIndex++
	}
	return patternIndex == len(patternSegments)
}
