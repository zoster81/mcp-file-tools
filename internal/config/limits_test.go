package config

import (
	"os"
	"testing"
)

func TestLoad_DefaultLimits(t *testing.T) {
	clearLimitEnvironment(t)
	cfg := Load()

	if cfg.Limits.MaxFileBytes != DefaultMaxFileBytes {
		t.Fatalf("MaxFileBytes = %d, want %d", cfg.Limits.MaxFileBytes, DefaultMaxFileBytes)
	}
	if cfg.Limits.MaxDecodedCharacters != DefaultMaxDecodedCharacters {
		t.Fatalf("MaxDecodedCharacters = %d, want %d", cfg.Limits.MaxDecodedCharacters, DefaultMaxDecodedCharacters)
	}
	if cfg.Limits.MaxLineBytes != DefaultMaxLineBytes {
		t.Fatalf("MaxLineBytes = %d, want %d", cfg.Limits.MaxLineBytes, DefaultMaxLineBytes)
	}
	if cfg.Limits.MaxBatchFiles != DefaultMaxBatchFiles {
		t.Fatalf("MaxBatchFiles = %d, want %d", cfg.Limits.MaxBatchFiles, DefaultMaxBatchFiles)
	}
	if cfg.Limits.MaxMatches != DefaultMaxMatches {
		t.Fatalf("MaxMatches = %d, want %d", cfg.Limits.MaxMatches, DefaultMaxMatches)
	}
	if cfg.Limits.MaxOutputBytes != DefaultMaxOutputBytes {
		t.Fatalf("MaxOutputBytes = %d, want %d", cfg.Limits.MaxOutputBytes, DefaultMaxOutputBytes)
	}
	if cfg.Limits.MaxFingerprintEntries != DefaultMaxFingerprintEntries {
		t.Fatalf("MaxFingerprintEntries = %d, want %d", cfg.Limits.MaxFingerprintEntries, DefaultMaxFingerprintEntries)
	}
	if cfg.Limits.MaxFingerprintEntryDetails != DefaultMaxFingerprintEntryDetails {
		t.Fatalf("MaxFingerprintEntryDetails = %d, want %d", cfg.Limits.MaxFingerprintEntryDetails, DefaultMaxFingerprintEntryDetails)
	}
	if cfg.Limits.MaxSessions != DefaultMaxSessions {
		t.Fatalf("MaxSessions = %d, want %d", cfg.Limits.MaxSessions, DefaultMaxSessions)
	}
}

func TestLoad_SpecificLimitsOverrideLegacyThreshold(t *testing.T) {
	clearLimitEnvironment(t)
	t.Setenv(EnvMemoryThreshold, "1000")
	t.Setenv(EnvMaxFileBytes, "2000")
	t.Setenv(EnvMaxOutputBytes, "3000")
	t.Setenv(EnvMaxDecodedCharacters, "4000")
	t.Setenv(EnvMaxLineBytes, "5000")
	t.Setenv(EnvMaxBatchFiles, "6")
	t.Setenv(EnvMaxMatches, "7")
	t.Setenv(EnvMaxFingerprintEntries, "8")
	t.Setenv(EnvMaxFingerprintEntryDetails, "9")
	t.Setenv(EnvMaxSessions, "10")

	cfg := Load()
	if cfg.Limits.MaxFileBytes != 2000 || cfg.Limits.MaxOutputBytes != 3000 ||
		cfg.Limits.MaxDecodedCharacters != 4000 || cfg.Limits.MaxLineBytes != 5000 ||
		cfg.Limits.MaxBatchFiles != 6 || cfg.Limits.MaxMatches != 7 ||
		cfg.Limits.MaxFingerprintEntries != 8 || cfg.Limits.MaxFingerprintEntryDetails != 9 || cfg.Limits.MaxSessions != 10 {
		t.Fatalf("unexpected limits: %#v", cfg.Limits)
	}
}

func TestLoad_LegacyThresholdFeedsFileAndOutputLimits(t *testing.T) {
	clearLimitEnvironment(t)
	t.Setenv(EnvMemoryThreshold, "12345")

	cfg := Load()
	if cfg.Limits.MaxFileBytes != 12345 || cfg.Limits.MaxOutputBytes != 12345 {
		t.Fatalf("legacy threshold did not feed file/output limits: %#v", cfg.Limits)
	}
}

func clearLimitEnvironment(t *testing.T) {
	t.Helper()
	for _, name := range []string{
		EnvMemoryThreshold, EnvMaxFileBytes, EnvMaxDecodedCharacters, EnvMaxLineBytes,
		EnvMaxBatchFiles, EnvMaxMatches, EnvMaxOutputBytes, EnvMaxFingerprintEntries,
		EnvMaxFingerprintEntryDetails, EnvMaxSessions,
	} {
		original, existed := os.LookupEnv(name)
		if err := os.Unsetenv(name); err != nil {
			t.Fatal(err)
		}
		name := name
		t.Cleanup(func() {
			if existed {
				_ = os.Setenv(name, original)
			} else {
				_ = os.Unsetenv(name)
			}
		})
	}
}
