// Package config provides configuration management for MCP file tools server.
package config

import (
	"log/slog"
	"os"
	"strconv"

	"github.com/zoster81/mcp-file-tools/internal/encoding"
)

const (
	// Environment variable names
	EnvDefaultEncoding = "MCP_DEFAULT_ENCODING"
	EnvMemoryThreshold = "MCP_MEMORY_THRESHOLD"

	// Default values
	DefaultEncoding = "utf-8"
	DefaultMaxSize  = int64(64 * 1024 * 1024) // 64MB full-document and aggregate-output budget
)

// Config holds server configuration loaded from environment variables.
type Config struct {
	// DefaultEncoding is the default encoding for new files when none is specified.
	// Existing files keep a confidently detected encoding. Set via MCP_DEFAULT_ENCODING.
	// Default: "utf-8"; legacy encodings remain available as explicit overrides.
	DefaultEncoding string

	// MemoryThreshold limits full-document editing and aggregate decoded content
	// returned by read_multiple_files. Streaming single-file operations are
	// bounded by line/result state rather than total source size.
	// Set via MCP_MEMORY_THRESHOLD environment variable.
	// Default: 67108864 (64MB)
	MemoryThreshold int64
}

// Load reads configuration from environment variables with sensible defaults.
func Load() *Config {
	cfg := &Config{
		DefaultEncoding: DefaultEncoding,
		MemoryThreshold: DefaultMaxSize,
	}

	// Load default encoding from environment
	if enc := os.Getenv(EnvDefaultEncoding); enc != "" {
		if _, ok := encoding.Get(enc); ok {
			cfg.DefaultEncoding = enc
		} else {
			slog.Warn("invalid MCP_DEFAULT_ENCODING, using default", "value", enc, "fallback", DefaultEncoding)
		}
	}

	// Load memory threshold from environment
	if sizeStr := os.Getenv(EnvMemoryThreshold); sizeStr != "" {
		if size, err := strconv.ParseInt(sizeStr, 10, 64); err == nil && size > 0 {
			cfg.MemoryThreshold = size
		}
	}

	return cfg
}
