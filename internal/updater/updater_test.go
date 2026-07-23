package updater

import (
	"context"
	"strings"
	"testing"
)

func TestIsNewerVersion(t *testing.T) {
	tests := []struct {
		latest, current string
		want            bool
	}{
		{"2.0.0", "1.0.0", true},
		{"1.1.0", "1.0.0", true},
		{"1.0.1", "1.0.0", true},
		{"1.0.0", "1.0.0", false},
		{"1.0.0", "2.0.0", false},
		{"v1.1.0", "1.0.0", true},
		{"1.1.0", "v1.0.0", true},
		{"1.1.0-beta", "1.0.0", true},
		{"1.1", "1.0.0", true},
		{"2", "1.0.0", true},
	}

	for _, tt := range tests {
		if got := isNewerVersion(tt.latest, tt.current); got != tt.want {
			t.Errorf("isNewerVersion(%q, %q) = %v, want %v", tt.latest, tt.current, got, tt.want)
		}
	}
}

func TestParseVersion(t *testing.T) {
	tests := []struct {
		input string
		want  [3]int
	}{
		{"1.2.3", [3]int{1, 2, 3}},
		{"v1.2.3", [3]int{1, 2, 3}},
		{"1.2.3-beta", [3]int{1, 2, 3}},
		{"1.2", [3]int{1, 2, 0}},
		{"1", [3]int{1, 0, 0}},
		{"", [3]int{0, 0, 0}},
	}

	for _, tt := range tests {
		if got := parseVersion(tt.input); got != tt.want {
			t.Errorf("parseVersion(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestCheckDisabled(t *testing.T) {
	t.Setenv("MCP_NO_UPDATE_CHECK", "1")
	if msg := Check(context.Background(), "1.0.0", false); msg != "" {
		t.Errorf("Check with disabled should return empty, got %q", msg)
	}
}

func TestCheckDevVersion(t *testing.T) {
	if msg := Check(context.Background(), "dev", false); msg != "" {
		t.Errorf("Check with dev version should return empty, got %q", msg)
	}
	if msg := Check(context.Background(), "", false); msg != "" {
		t.Errorf("Check with empty version should return empty, got %q", msg)
	}
}

func TestForkUpdateSource(t *testing.T) {
	const expectedAPI = "https://api.github.com/repos/zoster81/mcp-file-tools/releases/latest"
	const expectedRepo = "https://github.com/zoster81/mcp-file-tools"

	if UpdateCheckURL != expectedAPI {
		t.Fatalf("UpdateCheckURL = %q, want %q", UpdateCheckURL, expectedAPI)
	}
	if RepoURL != expectedRepo {
		t.Fatalf("RepoURL = %q, want %q", RepoURL, expectedRepo)
	}
	if ReleaseURL != expectedRepo+"/releases/latest" {
		t.Fatalf("ReleaseURL = %q", ReleaseURL)
	}
}

func TestCacheMatchesCurrentSource(t *testing.T) {
	if cacheMatchesSource(nil) {
		t.Fatal("nil cache must not match")
	}
	if cacheMatchesSource(&cache{Source: "https://example.com/releases/latest"}) {
		t.Fatal("cache from another release source must not match")
	}
	if !cacheMatchesSource(&cache{Source: UpdateCheckURL}) {
		t.Fatal("cache from the configured fork must match")
	}
}

func TestUpdateMessageFormat(t *testing.T) {
	msg := updateMessage("1.0.0", "1.1.0")
	for _, expected := range []string{"1.0.0", "1.1.0", ReleaseURL, "tunnel or MCP client"} {
		if !strings.Contains(msg, expected) {
			t.Errorf("message %q does not contain %q", msg, expected)
		}
	}
	if strings.Contains(strings.ToLower(msg), "claude") {
		t.Errorf("message must be client-neutral: %q", msg)
	}
}
