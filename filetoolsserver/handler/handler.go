package handler

import (
	"os"
	"sync"
	"time"

	"github.com/zoster81/mcp-file-tools/internal/config"
	"github.com/zoster81/mcp-file-tools/internal/filesystem"
	"github.com/zoster81/mcp-file-tools/internal/security"
)

// Default permissions for new files and directories
const (
	DefaultFileMode os.FileMode = 0644
	DefaultDirMode  os.FileMode = 0755
)

// Handler handles all file tool operations
type Handler struct {
	config                   *config.Config
	executionPolicy          *ExecutionPolicy
	configuredRequestedDirs  []string // immutable lexical baseline; always allowed
	configuredDirs           []string // immutable resolved baseline; always allowed
	allowedRequestedDirs     []string
	allowedDirs              []string
	editPreviews             *editPreviewStore
	replaceFile              func(string, []byte, filesystem.ReplaceOptions) error
	patchPackageAfterPrepare func() error
	mu                       sync.RWMutex
}

// Option is a functional option for configuring Handler
type Option func(*Handler)

// WithConfig sets the configuration for the handler.
func WithConfig(cfg *config.Config) Option {
	return func(h *Handler) {
		if cfg != nil {
			h.config = cfg
		}
	}
}

// WithExecutionPolicy sets an immutable transport-specific execution policy.
func WithExecutionPolicy(policy ExecutionPolicy) Option {
	return func(h *Handler) {
		policyCopy := policy
		h.executionPolicy = &policyCopy
	}
}

// NewHandler creates a new Handler with allowed directories and optional configuration.
// If no config is provided via WithConfig, default configuration is used.
func NewHandler(allowedDirs []string, opts ...Option) *Handler {
	// Retain the configured spelling for lexical containment while exposing and
	// traversing through the resolved representation. This preserves legitimate
	// directory aliases without allowing an external link to become an entry point.
	configuredRequestedDirs, configuredDirs := normalizeAllowedDirectorySets(allowedDirs)

	h := &Handler{
		configuredRequestedDirs: configuredRequestedDirs,
		configuredDirs:          configuredDirs,
		allowedRequestedDirs:    mergeUniqueDirectories(configuredRequestedDirs, configuredDirs),
		allowedDirs:             append([]string(nil), configuredDirs...),
	}

	for _, opt := range opts {
		opt(h)
	}
	if h.config == nil {
		h.config = config.Load()
	}
	h.editPreviews = newEditPreviewStore(
		h.maxEditPreviews(),
		h.maxEditPreviewBytes(),
		time.Duration(h.editPreviewTTLSeconds())*time.Second,
	)
	h.replaceFile = filesystem.ReplaceFile

	return h
}

// GetAllowedDirectories returns a copy of the allowed directories.
func (h *Handler) GetAllowedDirectories() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	dirs := make([]string, len(h.allowedDirs))
	copy(dirs, h.allowedDirs)
	return dirs
}

// ResolvedAllowedDirs returns allowed directories with symlinks resolved.
func (h *Handler) ResolvedAllowedDirs() []string {
	return security.ResolveAllowedDirs(h.GetAllowedDirectories())
}

// UpdateAllowedDirectories updates the allowed directories (for MCP Roots protocol)
func (h *Handler) UpdateAllowedDirectories(newDirs []string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	requested, resolved := normalizeAllowedDirectorySets(newDirs)
	h.allowedRequestedDirs = mergeUniqueDirectories(requested, resolved)
	h.allowedDirs = resolved
}

// HasConfiguredDirectories reports whether this process started with an
// authoritative directory baseline.
func (h *Handler) HasConfiguredDirectories() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.configuredDirs) > 0
}

// MergeAllowedDirectories sets the allowed directories to the deduped union of
// the process baseline and newDirs, so MCP roots never replace configured access.
func (h *Handler) MergeAllowedDirectories(newDirs []string) []string {
	h.mu.Lock()
	defer h.mu.Unlock()

	normalizedRequestedDirs, normalizedNewDirs := normalizeAllowedDirectorySets(newDirs)
	h.allowedRequestedDirs = mergeUniqueDirectories(
		h.configuredRequestedDirs,
		h.configuredDirs,
		normalizedRequestedDirs,
		normalizedNewDirs,
	)
	h.allowedDirs = mergeUniqueDirectories(h.configuredDirs, normalizedNewDirs)

	result := make([]string, len(h.allowedDirs))
	copy(result, h.allowedDirs)
	return result
}

func normalizeAllowedDirectories(dirs []string) []string {
	_, resolved := normalizeAllowedDirectorySets(dirs)
	return resolved
}

func normalizeAllowedDirectorySets(dirs []string) (requested, resolved []string) {
	requested = make([]string, 0, len(dirs))
	resolved = make([]string, 0, len(dirs))
	for _, dir := range dirs {
		set, err := security.NormalizeAllowedDirectorySet([]string{dir})
		if err == nil && len(set.Requested) == 1 && len(set.Resolved) == 1 {
			requested = append(requested, set.Requested[0])
			resolved = append(resolved, set.Resolved[0])
			continue
		}
		requested = append(requested, dir)
		resolved = append(resolved, dir)
	}
	return requested, resolved
}

func mergeUniqueDirectories(groups ...[]string) []string {
	total := 0
	for _, group := range groups {
		total += len(group)
	}
	seen := make(map[string]struct{}, total)
	merged := make([]string, 0, total)
	for _, group := range groups {
		for _, dir := range group {
			if _, ok := seen[dir]; ok {
				continue
			}
			seen[dir] = struct{}{}
			merged = append(merged, dir)
		}
	}
	return merged
}

// validatePath validates a path against allowed directories
func (h *Handler) validatePath(path string) (string, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return security.ValidatePathWithAllowedDirectories(path, h.allowedRequestedDirs, h.allowedDirs)
}

// getFileMode returns the file's current permissions, or DefaultFileMode if file doesn't exist.
func getFileMode(path string) os.FileMode {
	info, err := os.Stat(path)
	if err != nil {
		return DefaultFileMode
	}
	return info.Mode().Perm()
}
