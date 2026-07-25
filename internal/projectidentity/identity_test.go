package projectidentity

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

const (
	forkModule       = "github.com/zoster81/mcp-file-tools"
	forkRegistryName = "io.github.zoster81/mcp-file-tools"
	forkRepository   = "https://github.com/zoster81/mcp-file-tools"
)

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not resolve test file path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
}

func TestGoModuleTargetsFork(t *testing.T) {
	root := repositoryRoot(t)
	data, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(data), "module "+forkModule+"\n") {
		t.Fatalf("go.mod must declare module %q", forkModule)
	}
}

func TestUpstreamReferencesAreDocumentationOnly(t *testing.T) {
	root := repositoryRoot(t)
	upstreamOwner := "dimitar" + "-grigorov"
	allowedCounts := map[string]int{
		"README.md":                              1,
		"CHANGELOG.md":                           1,
		filepath.FromSlash("docs/PUBLISHING.md"): 2,
	}
	actualCounts := make(map[string]int)

	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if entry.Name() == ".git" || entry.Name() == "dist" {
				return filepath.SkipDir
			}
			return nil
		}
		if !isIdentityTextFile(entry.Name()) {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		count := strings.Count(string(data), upstreamOwner)
		if count == 0 {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		allowed, ok := allowedCounts[relative]
		if !ok {
			t.Errorf("operational file %s contains %d upstream repository reference(s)", relative, count)
			return nil
		}
		actualCounts[relative] = count
		if count != allowed {
			t.Errorf("documentation file %s contains %d upstream references, want %d", relative, count, allowed)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	for path, expected := range allowedCounts {
		if actualCounts[path] != expected {
			t.Errorf("documentation file %s contains %d upstream references, want %d", path, actualCounts[path], expected)
		}
	}
}

func TestOperationalMetadataTargetsFork(t *testing.T) {
	root := repositoryRoot(t)
	assertFileContains(t, root, ".goreleaser.yml", "-X "+forkModule+"/filetoolsserver.Version={{.Version}}")
	assertFileContains(t, root, filepath.FromSlash(".github/workflows/publish-registry.yml"), "github.repository == 'zoster81/mcp-file-tools'")
	assertFileContains(t, root, filepath.FromSlash(".github/workflows/release.yml"), "uses: ./.github/workflows/publish-registry.yml")
	assertFileContains(t, root, filepath.FromSlash("scripts/generate-server-json.js"), "const forkRepository = '"+forkRepository+"'")
}

func TestValidationToolVersionsArePinned(t *testing.T) {
	root := repositoryRoot(t)
	assertFileContains(t, root, filepath.FromSlash("scripts/validate-workflows.sh"), "ACTIONLINT_VERSION=1.7.12")
	assertFileContains(t, root, filepath.FromSlash("scripts/validate-workflows.sh"), "SHELLCHECK_VERSION=0.11.0")
	assertFileContains(t, root, filepath.FromSlash(".github/workflows/test.yml"), "staticcheck@v0.7.0")
	assertFileContains(t, root, filepath.FromSlash(".github/workflows/test.yml"), "govulncheck@v1.1.4")
	assertFileContains(t, root, filepath.FromSlash(".github/workflows/release.yml"), "version: 'v2.17.0'")
	assertFileContains(t, root, filepath.FromSlash(".github/workflows/publish-registry.yml"), "MCP_PUBLISHER_VERSION: 1.8.0")
}

func assertFileContains(t *testing.T, root, relativePath, expected string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, relativePath))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), expected) {
		t.Errorf("%s must contain %q", relativePath, expected)
	}
}
func TestRegistryTemplateTargetsFork(t *testing.T) {
	root := repositoryRoot(t)
	data, err := os.ReadFile(filepath.Join(root, "server.template.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Homepage    string `json:"homepage"`
		Repository struct {
			URL string `json:"url"`
		} `json:"repository"`
		Packages []struct {
			Identifier string `json:"identifier"`
		} `json:"packages"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.Name != forkRegistryName {
		t.Errorf("registry name = %q, want %q", manifest.Name, forkRegistryName)
	}
	if len(manifest.Description) == 0 || len(manifest.Description) > 100 {
		t.Errorf("registry description length = %d, want 1..100", len(manifest.Description))
	}
	if manifest.Homepage != forkRepository || manifest.Repository.URL != forkRepository {
		t.Errorf("registry repository metadata must target %s", forkRepository)
	}
	for _, pkg := range manifest.Packages {
		if !strings.HasPrefix(pkg.Identifier, forkRepository+"/releases/download/") {
			t.Errorf("package identifier %q does not target fork releases", pkg.Identifier)
		}
	}
}

func isIdentityTextFile(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".go", ".mod", ".sum", ".json", ".yml", ".yaml", ".js", ".mjs", ".cjs", ".ps1", ".sh", ".bat", ".cmd", ".md", ".txt", ".toml", ".xml", ".ini", ".conf":
		return true
	}
	return name == "Makefile" || name == "Dockerfile"
}
