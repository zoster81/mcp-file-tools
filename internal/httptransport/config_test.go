package httptransport

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoadConfigDefaults(t *testing.T) {
	cfg, err := LoadConfig(environment(map[string]string{
		EnvToken: strings.Repeat("a", 32),
	}), 128)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if cfg.Address != DefaultAddress || cfg.Path != DefaultPath {
		t.Fatalf("address/path = %q %q", cfg.Address, cfg.Path)
	}
	if cfg.MaxBodyBytes != DefaultMaxBodyBytes || cfg.MaxInFlightBodyBytes != DefaultMaxInFlightBodyBytes || cfg.MaxConcurrentRequests != DefaultMaxConcurrentRequests {
		t.Fatalf("limits = %#v", cfg)
	}
	if cfg.SessionTimeout != DefaultSessionTimeout || cfg.MaxSessions != 128 {
		t.Fatalf("session config = timeout %v max %d", cfg.SessionTimeout, cfg.MaxSessions)
	}
	if cfg.AllowNonLoopback || cfg.EnableExecution || cfg.UseTLS() {
		t.Fatalf("insecure default flags = %#v", cfg)
	}
	if !cfg.AllowsHost("127.0.0.1:8765") || !cfg.AllowsHost("localhost:8765") {
		t.Fatalf("listener-derived hosts = %v", cfg.AllowedHosts)
	}
}

func TestLoadConfigRequiresExactlyOneTokenSource(t *testing.T) {
	if _, err := LoadConfig(environment(nil), 128); err == nil {
		t.Fatal("missing token accepted")
	}
	if _, err := LoadConfig(environment(map[string]string{
		EnvToken:     strings.Repeat("a", 32),
		EnvTokenFile: "token.txt",
	}), 128); err == nil {
		t.Fatal("conflicting token sources accepted")
	}
}

func TestLoadConfigTokenFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "token.txt")
	if err := os.WriteFile(path, []byte(strings.Repeat("b", 32)+"\r\n"), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig(environment(map[string]string{EnvTokenFile: path}), 4)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if cfg.PrincipalID == "" || cfg.TokenDigest == [32]byte{} {
		t.Fatalf("token material not derived: %#v", cfg)
	}
}

func TestLoadConfigRejectsMalformedTokens(t *testing.T) {
	for name, token := range map[string]string{
		"short":   "short",
		"space":   strings.Repeat("a", 31) + " ",
		"newline": strings.Repeat("a", 32) + "\nother",
		"nul":     strings.Repeat("a", 32) + "\x00",
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := LoadConfig(environment(map[string]string{EnvToken: token}), 128); err == nil {
				t.Fatalf("malformed token %q accepted", name)
			}
		})
	}
}

func TestLoadConfigRejectsInsecureNonLoopback(t *testing.T) {
	base := map[string]string{
		EnvToken:            strings.Repeat("a", 32),
		EnvAddress:          "192.0.2.10:8765",
		EnvAllowNonLoopback: "1",
	}
	if _, err := LoadConfig(environment(base), 128); err == nil {
		t.Fatal("plaintext non-loopback without proxy accepted")
	}

	base[EnvTrustedProxyCIDRs] = "192.0.2.0/24"
	cfg, err := LoadConfig(environment(base), 128)
	if err != nil {
		t.Fatalf("trusted proxy profile rejected: %v", err)
	}
	if !cfg.ProxyOnlyPlaintext {
		t.Fatal("trusted proxy plaintext profile not marked proxy-only")
	}
}

func TestLoadConfigAllowsDirectNonLoopbackTLS(t *testing.T) {
	cfg, err := LoadConfig(environment(map[string]string{
		EnvToken:            strings.Repeat("a", 32),
		EnvAddress:          "192.0.2.10:8765",
		EnvAllowNonLoopback: "true",
		EnvTLSCertFile:      "server.crt",
		EnvTLSKeyFile:       "server.key",
	}), 128)
	if err != nil {
		t.Fatalf("TLS profile rejected: %v", err)
	}
	if !cfg.UseTLS() || cfg.ProxyOnlyPlaintext {
		t.Fatalf("TLS flags = %#v", cfg)
	}
}

func TestLoadConfigValidatesPathHostsOriginsAndLimits(t *testing.T) {
	valid := map[string]string{
		EnvToken:                 strings.Repeat("a", 32),
		EnvPath:                  "/custom-mcp",
		EnvAllowedHosts:          "mcp.example.test, mcp.example.test:443",
		EnvAllowedOrigins:        "https://app.example.test",
		EnvMaxBodyBytes:          "1048576",
		EnvMaxInFlightBodyBytes:  "2097152",
		EnvMaxConcurrentRequests: "8",
		EnvSessionTimeout:        "2m",
		EnvEnableExecution:       "1",
	}
	cfg, err := LoadConfig(environment(valid), 7)
	if err != nil {
		t.Fatalf("valid config rejected: %v", err)
	}
	if cfg.Path != "/custom-mcp" || cfg.MaxBodyBytes != 1048576 || cfg.MaxInFlightBodyBytes != 2097152 || cfg.MaxConcurrentRequests != 8 || cfg.SessionTimeout != 2*time.Minute || !cfg.EnableExecution {
		t.Fatalf("config = %#v", cfg)
	}
	if !cfg.AllowsHost("mcp.example.test") || !cfg.AllowsOrigin("https://app.example.test") || !cfg.AllowsOrigin("https://app.example.test:443") {
		t.Fatalf("allowlists = hosts %v origins %v", cfg.AllowedHosts, cfg.AllowedOrigins)
	}

	for name, update := range map[string]map[string]string{
		"path conflict":        {EnvPath: "/healthz"},
		"wildcard host":        {EnvAllowedHosts: "*.example.test"},
		"invalid port":         {EnvAllowedHosts: "mcp.example.test:not-a-port"},
		"host userinfo":        {EnvAllowedHosts: "user@mcp.example.test"},
		"origin path":          {EnvAllowedOrigins: "https://app.example.test/path"},
		"body zero":            {EnvMaxBodyBytes: "0"},
		"aggregate below body": {EnvMaxInFlightBodyBytes: "524288"},
		"timeout short":        {EnvSessionTimeout: "1s"},
	} {
		t.Run(name, func(t *testing.T) {
			env := cloneMap(valid)
			for key, value := range update {
				env[key] = value
			}
			if _, err := LoadConfig(environment(env), 7); err == nil {
				t.Fatalf("invalid %s accepted", name)
			}
		})
	}
}

func environment(values map[string]string) func(string) string {
	return func(name string) string { return values[name] }
}

func cloneMap(source map[string]string) map[string]string {
	result := make(map[string]string, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}
