// Package config provides configuration management for MCP file tools server.
package config

import (
	"log/slog"
	"os"
	"strconv"

	"github.com/zoster81/mcp-file-tools/internal/encoding"
)

const (
	EnvDefaultEncoding               = "MCP_DEFAULT_ENCODING"
	EnvMemoryThreshold               = "MCP_MEMORY_THRESHOLD" // Deprecated 1.x fallback for file/output limits.
	EnvMaxFileBytes                  = "MCP_MAX_FILE_BYTES"
	EnvMaxDecodedCharacters          = "MCP_MAX_DECODED_CHARACTERS"
	EnvMaxLineBytes                  = "MCP_MAX_LINE_BYTES"
	EnvMaxBatchFiles                 = "MCP_MAX_BATCH_FILES"
	EnvMaxMatches                    = "MCP_MAX_MATCHES"
	EnvMaxOutputBytes                = "MCP_MAX_OUTPUT_BYTES"
	EnvMaxFingerprintEntries         = "MCP_MAX_FINGERPRINT_ENTRIES"
	EnvMaxFingerprintEntryDetails    = "MCP_MAX_FINGERPRINT_ENTRY_DETAILS"
	EnvMaxEditPreviews               = "MCP_MAX_EDIT_PREVIEWS"
	EnvMaxEditPreviewBytes           = "MCP_MAX_EDIT_PREVIEW_BYTES"
	EnvEditPreviewTTLSeconds         = "MCP_EDIT_PREVIEW_TTL_SECONDS"
	EnvMaxPatchPackageBytes          = "MCP_MAX_PATCH_PACKAGE_BYTES"
	EnvMaxPatchPackagePreparedBytes  = "MCP_MAX_PATCH_PACKAGE_PREPARED_BYTES"
	EnvMaxPatchPackagePreviews       = "MCP_MAX_PATCH_PACKAGE_PREVIEWS"
	EnvMaxPatchPackagePreviewBytes   = "MCP_MAX_PATCH_PACKAGE_PREVIEW_BYTES"
	EnvPatchPackagePreviewTTLSeconds = "MCP_PATCH_PACKAGE_PREVIEW_TTL_SECONDS"
	EnvMaxSessions                   = "MCP_MAX_SESSIONS" // Maximum live native Streamable HTTP sessions.

	DefaultEncoding                      = "utf-8"
	DefaultMaxFileBytes                  = int64(64 * 1024 * 1024)
	DefaultMaxDecodedCharacters          = 16 * 1024 * 1024
	DefaultMaxLineBytes                  = 16 * 1024 * 1024
	DefaultMaxBatchFiles                 = 256
	DefaultMaxMatches                    = 10_000
	DefaultMaxOutputBytes                = int64(64 * 1024 * 1024)
	DefaultMaxFingerprintEntries         = 100_000
	DefaultMaxFingerprintEntryDetails    = 1_000
	DefaultMaxEditPreviews               = 128
	DefaultMaxEditPreviewBytes           = int64(64 * 1024 * 1024)
	DefaultEditPreviewTTLSeconds         = 15 * 60
	DefaultMaxPatchPackageBytes          = int64(16 * 1024 * 1024)
	DefaultMaxPatchPackagePreparedBytes  = int64(64 * 1024 * 1024)
	DefaultMaxPatchPackagePreviews       = 16
	DefaultMaxPatchPackagePreviewBytes   = int64(128 * 1024 * 1024)
	DefaultPatchPackagePreviewTTLSeconds = 15 * 60
	DefaultMaxSessions                   = 128
)

// Limits contains server-wide hard limits. Request-level limits may be lower
// but must never exceed these values.
type Limits struct {
	MaxFileBytes                  int64
	MaxDecodedCharacters          int
	MaxLineBytes                  int
	MaxBatchFiles                 int
	MaxMatches                    int
	MaxOutputBytes                int64
	MaxFingerprintEntries         int
	MaxFingerprintEntryDetails    int
	MaxEditPreviews               int
	MaxEditPreviewBytes           int64
	EditPreviewTTLSeconds         int
	MaxPatchPackageBytes          int64
	MaxPatchPackagePreparedBytes  int64
	MaxPatchPackagePreviews       int
	MaxPatchPackagePreviewBytes   int64
	PatchPackagePreviewTTLSeconds int
	MaxSessions                   int
}

// Config holds server configuration loaded from environment variables.
type Config struct {
	// DefaultEncoding is used for newly created files when no encoding is supplied.
	DefaultEncoding string
	Limits          Limits
}

// Load reads configuration from environment variables with conservative defaults.
func Load() *Config {
	cfg := &Config{
		DefaultEncoding: DefaultEncoding,
		Limits: Limits{
			MaxFileBytes:                  DefaultMaxFileBytes,
			MaxDecodedCharacters:          DefaultMaxDecodedCharacters,
			MaxLineBytes:                  DefaultMaxLineBytes,
			MaxBatchFiles:                 DefaultMaxBatchFiles,
			MaxMatches:                    DefaultMaxMatches,
			MaxOutputBytes:                DefaultMaxOutputBytes,
			MaxFingerprintEntries:         DefaultMaxFingerprintEntries,
			MaxFingerprintEntryDetails:    DefaultMaxFingerprintEntryDetails,
			MaxEditPreviews:               DefaultMaxEditPreviews,
			MaxEditPreviewBytes:           DefaultMaxEditPreviewBytes,
			EditPreviewTTLSeconds:         DefaultEditPreviewTTLSeconds,
			MaxPatchPackageBytes:          DefaultMaxPatchPackageBytes,
			MaxPatchPackagePreparedBytes:  DefaultMaxPatchPackagePreparedBytes,
			MaxPatchPackagePreviews:       DefaultMaxPatchPackagePreviews,
			MaxPatchPackagePreviewBytes:   DefaultMaxPatchPackagePreviewBytes,
			PatchPackagePreviewTTLSeconds: DefaultPatchPackagePreviewTTLSeconds,
			MaxSessions:                   DefaultMaxSessions,
		},
	}

	if enc := os.Getenv(EnvDefaultEncoding); enc != "" {
		if _, ok := encoding.Get(enc); ok {
			cfg.DefaultEncoding = enc
		} else {
			slog.Warn("invalid MCP_DEFAULT_ENCODING, using default", "value", enc, "fallback", DefaultEncoding)
		}
	}

	// Keep the 1.x threshold as a compatibility fallback. Specific 2.0 limits
	// below take precedence when both are configured.
	if legacy, ok := positiveInt64Environment(EnvMemoryThreshold); ok {
		cfg.Limits.MaxFileBytes = legacy
		cfg.Limits.MaxOutputBytes = legacy
	}

	cfg.Limits.MaxFileBytes = int64Environment(EnvMaxFileBytes, cfg.Limits.MaxFileBytes)
	cfg.Limits.MaxOutputBytes = int64Environment(EnvMaxOutputBytes, cfg.Limits.MaxOutputBytes)
	cfg.Limits.MaxDecodedCharacters = intEnvironment(EnvMaxDecodedCharacters, cfg.Limits.MaxDecodedCharacters)
	cfg.Limits.MaxLineBytes = intEnvironment(EnvMaxLineBytes, cfg.Limits.MaxLineBytes)
	cfg.Limits.MaxBatchFiles = intEnvironment(EnvMaxBatchFiles, cfg.Limits.MaxBatchFiles)
	cfg.Limits.MaxMatches = intEnvironment(EnvMaxMatches, cfg.Limits.MaxMatches)
	cfg.Limits.MaxFingerprintEntries = intEnvironment(EnvMaxFingerprintEntries, cfg.Limits.MaxFingerprintEntries)
	cfg.Limits.MaxFingerprintEntryDetails = intEnvironment(EnvMaxFingerprintEntryDetails, cfg.Limits.MaxFingerprintEntryDetails)
	cfg.Limits.MaxEditPreviews = intEnvironment(EnvMaxEditPreviews, cfg.Limits.MaxEditPreviews)
	cfg.Limits.MaxEditPreviewBytes = int64Environment(EnvMaxEditPreviewBytes, cfg.Limits.MaxEditPreviewBytes)
	cfg.Limits.EditPreviewTTLSeconds = intEnvironment(EnvEditPreviewTTLSeconds, cfg.Limits.EditPreviewTTLSeconds)
	cfg.Limits.MaxPatchPackageBytes = int64Environment(EnvMaxPatchPackageBytes, cfg.Limits.MaxPatchPackageBytes)
	cfg.Limits.MaxPatchPackagePreparedBytes = int64Environment(EnvMaxPatchPackagePreparedBytes, cfg.Limits.MaxPatchPackagePreparedBytes)
	cfg.Limits.MaxPatchPackagePreviews = intEnvironment(EnvMaxPatchPackagePreviews, cfg.Limits.MaxPatchPackagePreviews)
	cfg.Limits.MaxPatchPackagePreviewBytes = int64Environment(EnvMaxPatchPackagePreviewBytes, cfg.Limits.MaxPatchPackagePreviewBytes)
	cfg.Limits.PatchPackagePreviewTTLSeconds = intEnvironment(EnvPatchPackagePreviewTTLSeconds, cfg.Limits.PatchPackagePreviewTTLSeconds)
	cfg.Limits.MaxSessions = intEnvironment(EnvMaxSessions, cfg.Limits.MaxSessions)
	return cfg
}

func int64Environment(name string, fallback int64) int64 {
	if value, ok := positiveInt64Environment(name); ok {
		return value
	}
	return fallback
}

func intEnvironment(name string, fallback int) int {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed <= 0 || int64(int(parsed)) != parsed {
		slog.Warn("invalid positive integer environment value, using fallback", "name", name, "value", value, "fallback", fallback)
		return fallback
	}
	return int(parsed)
}

func positiveInt64Environment(name string) (int64, bool) {
	value := os.Getenv(name)
	if value == "" {
		return 0, false
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed <= 0 {
		slog.Warn("invalid positive integer environment value, using fallback", "name", name, "value", value)
		return 0, false
	}
	return parsed, true
}
