package handler

import (
	"os"
	"sync"

	"github.com/zoster81/mcp-file-tools/internal/config"
	"github.com/zoster81/mcp-file-tools/internal/security"
)

// Default permissions for new files and directories
const (
	DefaultFileMode os.FileMode = 0644
	DefaultDirMode  os.FileMode = 0755
)

// Handler handles all file tool operations
type Handler struct {
	config      *config.Config
	cliDirs     []string // immutable baseline from CLI args; always allowed
	allowedDirs []string
	mu          sync.RWMutex
}

// Option is a functional option for configuring Handler
type Option func(*Handler)

// WithConfig sets the configuration for the handler
func WithConfig(cfg *config.Config) Option {
	return func(h *Handler) {
		if cfg != nil {
			h.config = cfg
		}
	}
}

// NewHandler creates a new Handler with allowed directories and optional configuration.
// If no config is provided via WithConfig, default configuration is used.
func NewHandler(allowedDirs []string, opts ...Option) *Handler {
	// Keep both the immutable CLI baseline and active roots in one canonical
	// representation. This prevents Windows 8.3 aliases from diverging from the
	// long paths returned by final-path resolution during later validations.
	cliDirs := normalizeAllowedDirectories(allowedDirs)

	h := &Handler{
		config:      config.Load(), // Load defaults from environment
		cliDirs:     cliDirs,
		allowedDirs: append([]string(nil), cliDirs...),
	}

	for _, opt := range opts {
		opt(h)
	}

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
	h.allowedDirs = normalizeAllowedDirectories(newDirs)
}

// MergeAllowedDirectories sets the allowed directories to the deduped union of the
// CLI baseline and newDirs, so MCP roots augment rather than replace CLI args.
func (h *Handler) MergeAllowedDirectories(newDirs []string) []string {
	h.mu.Lock()
	defer h.mu.Unlock()

	normalizedNewDirs := normalizeAllowedDirectories(newDirs)
	seen := make(map[string]struct{}, len(h.cliDirs)+len(normalizedNewDirs))
	merged := make([]string, 0, len(h.cliDirs)+len(normalizedNewDirs))
	for _, dirs := range [][]string{h.cliDirs, normalizedNewDirs} {
		for _, dir := range dirs {
			if _, ok := seen[dir]; ok {
				continue
			}
			seen[dir] = struct{}{}
			merged = append(merged, dir)
		}
	}
	h.allowedDirs = merged

	result := make([]string, len(merged))
	copy(result, merged)
	return result
}

func normalizeAllowedDirectories(dirs []string) []string {
	normalized := make([]string, 0, len(dirs))
	for _, dir := range dirs {
		current, err := security.NormalizeAllowedDirs([]string{dir})
		if err == nil && len(current) == 1 {
			normalized = append(normalized, current[0])
			continue
		}
		normalized = append(normalized, dir)
	}
	return normalized
}

// validatePath validates a path against allowed directories
func (h *Handler) validatePath(path string) (string, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return security.ValidatePath(path, h.allowedDirs)
}

// getFileMode returns the file's current permissions, or DefaultFileMode if file doesn't exist.
func getFileMode(path string) os.FileMode {
	info, err := os.Stat(path)
	if err != nil {
		return DefaultFileMode
	}
	return info.Mode().Perm()
}
