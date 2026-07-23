// Package updater checks for new releases on GitHub and notifies users.
package updater

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	// UpdateCheckURL is the GitHub API endpoint for the latest fork release.
	UpdateCheckURL = "https://api.github.com/repos/zoster81/mcp-file-tools/releases/latest"

	// RepoURL is the custom fork repository URL.
	RepoURL = "https://github.com/zoster81/mcp-file-tools"

	// ReleaseURL is the human-facing latest release page for this fork.
	ReleaseURL = RepoURL + "/releases/latest"

	// CheckInterval is the minimum time between API calls (respects GitHub rate limits)
	CheckInterval = 30 * time.Minute

	// httpTimeout is the timeout for HTTP requests to GitHub API
	httpTimeout = 10 * time.Second
)

// cache stores the last check result to avoid excessive API calls
type cache struct {
	Source        string    `json:"source"`
	LastCheck     time.Time `json:"lastCheck"`
	LatestVersion string    `json:"latestVersion"`
}

// Check checks for updates and returns a notification message if available.
// Returns empty string if: no update, disabled via MCP_NO_UPDATE_CHECK=1, dev version, or error.
// If force is true, the cache is bypassed and a fresh check is performed.
func Check(ctx context.Context, currentVersion string, force bool) string {
	// Skip if disabled or running dev build
	if os.Getenv("MCP_NO_UPDATE_CHECK") == "1" || currentVersion == "dev" || currentVersion == "" {
		return ""
	}

	cacheFile := getCacheFile()
	latestVersion := ""

	// Use cached result if within check interval (unless forced)
	if !force {
		if c := readCache(cacheFile); cacheMatchesSource(c) && time.Since(c.LastCheck) < CheckInterval {
			latestVersion = c.LatestVersion
		}
	}

	if latestVersion == "" {
		var err error
		latestVersion, err = fetchLatestVersion(ctx)
		if err != nil {
			// Cache the failure so we don't hammer GitHub when offline.
			// The empty version means "unknown" — next check after CheckInterval.
			writeCache(cacheFile, "")
			return ""
		}
		writeCache(cacheFile, latestVersion)
	}

	if isNewerVersion(latestVersion, currentVersion) {
		return updateMessage(currentVersion, latestVersion)
	}
	return ""
}

// fetchLatestVersion queries GitHub API for the latest release tag
func fetchLatestVersion(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, UpdateCheckURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "mcp-file-tools-update-checker")

	client := &http.Client{Timeout: httpTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status %d", resp.StatusCode)
	}

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}
	return strings.TrimPrefix(release.TagName, "v"), nil
}

// getCacheFile returns the path to the cache file in user's cache directory
func getCacheFile() string {
	if dir, err := os.UserCacheDir(); err == nil {
		return filepath.Join(dir, "mcp-file-tools", "update-check.json")
	}
	return ""
}

func readCache(path string) *cache {
	if path == "" {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var c cache
	if json.Unmarshal(data, &c) != nil {
		return nil
	}
	return &c
}

// CachedLatestVersion returns the latest version from the cache file, if available.
func CachedLatestVersion() string {
	if c := readCache(getCacheFile()); cacheMatchesSource(c) {
		return c.LatestVersion
	}
	return ""
}

func cacheMatchesSource(c *cache) bool {
	return c != nil && c.Source == UpdateCheckURL
}

func updateMessage(currentVersion, latestVersion string) string {
	return fmt.Sprintf(
		"mcp-file-tools fork update available: %s → %s\n"+
			"Stop the tunnel or MCP client using the binary before replacing it.\n"+
			"Release: %s",
		currentVersion, latestVersion, ReleaseURL)
}

func writeCache(path, version string) {
	if path == "" {
		return
	}
	_ = os.MkdirAll(filepath.Dir(path), 0755)
	data, _ := json.Marshal(cache{Source: UpdateCheckURL, LastCheck: time.Now(), LatestVersion: version})
	_ = os.WriteFile(path, data, 0644)
}

// isNewerVersion compares semver versions (major.minor.patch)
func isNewerVersion(latest, current string) bool {
	l, c := parseVersion(latest), parseVersion(current)
	for i := 0; i < 3; i++ {
		if l[i] > c[i] {
			return true
		}
		if l[i] < c[i] {
			return false
		}
	}
	return false
}

// parseVersion extracts [major, minor, patch] from version string
// Handles: "1.2.3", "v1.2.3", "1.2.3-beta", "1.2", "1"
func parseVersion(v string) [3]int {
	v = strings.TrimPrefix(v, "v")
	parts := strings.Split(v, ".")
	var r [3]int
	for i := 0; i < 3 && i < len(parts); i++ {
		r[i], _ = strconv.Atoi(strings.Split(parts[i], "-")[0])
	}
	return r
}
