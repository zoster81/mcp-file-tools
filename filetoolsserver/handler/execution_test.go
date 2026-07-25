package handler

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestRevalidateScriptRejectsReplacement(t *testing.T) {
	root := t.TempDir()
	script := filepath.Join(root, "script.exe")
	if err := os.WriteFile(script, []byte("first"), 0o700); err != nil {
		t.Fatal(err)
	}

	h := NewHandler([]string{root})
	original, err := inspectScriptFile(script)
	if err != nil {
		t.Fatal(err)
	}

	replacement := filepath.Join(root, "replacement.exe")
	if err := os.WriteFile(replacement, []byte("second"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(script); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(replacement, script); err != nil {
		t.Fatal(err)
	}

	err = h.revalidateScript(script, original)
	if err == nil || !strings.Contains(err.Error(), "script changed before execution") {
		t.Fatalf("revalidateScript() error = %v, want replacement rejection", err)
	}
}

func TestRevalidateWorkingDirectoryRejectsOutsideRoot(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	h := NewHandler([]string{root})

	if err := h.revalidateWorkingDirectory(outside); err == nil {
		t.Fatal("revalidateWorkingDirectory() accepted an outside directory")
	}
}

func TestBuildScriptCommandKeepsArgumentsSeparate(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("direct executable behavior is platform-independent; Windows avoids executable permission setup in this test")
	}

	script := `C:\allowed\tool.exe`
	inputArgs := []string{"value with spaces", `$(untrusted)`, `& whoami`}
	program, args, err := buildScriptCommand(script, inputArgs)
	if err != nil {
		t.Fatalf("buildScriptCommand() error = %v", err)
	}
	if program != script {
		t.Fatalf("program = %q, want %q", program, script)
	}
	if len(args) != len(inputArgs) {
		t.Fatalf("args length = %d, want %d", len(args), len(inputArgs))
	}
	for index := range inputArgs {
		if args[index] != inputArgs[index] {
			t.Fatalf("arg %d = %q, want %q", index, args[index], inputArgs[index])
		}
	}
	inputArgs[0] = "modified"
	if args[0] == "modified" {
		t.Fatal("buildScriptCommand() retained the caller's argument slice")
	}
}

func TestHandleRunScriptValidatesTimeoutBeforeCommandSelection(t *testing.T) {
	root := t.TempDir()
	script := filepath.Join(root, "script.unsupported")
	if err := os.WriteFile(script, []byte("content"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MCP_ENABLE_EXECUTION", "")
	t.Setenv("MCP_ENABLE_RUN_SCRIPT", "1")

	h := NewHandler([]string{root})
	result, _, err := h.HandleRunScript(context.Background(), nil, RunScriptInput{
		Path:           script,
		TimeoutSeconds: 601,
	})
	if err != nil {
		t.Fatalf("HandleRunScript() error = %v", err)
	}
	if text := callToolResultText(t, result); !strings.Contains(text, "timeoutSeconds must be between") {
		t.Fatalf("result = %q, want timeout validation before command selection", text)
	}
}

func TestHandleShellValidatesTimeoutBeforeShellSelection(t *testing.T) {
	root := t.TempDir()
	t.Setenv("MCP_ENABLE_EXECUTION", "")
	t.Setenv("MCP_ENABLE_SHELL", "1")

	h := NewHandler([]string{root})
	result, _, err := h.HandleShell(context.Background(), nil, ShellInput{
		Command:        "echo test",
		Shell:          "unsupported-shell",
		TimeoutSeconds: 601,
	})
	if err != nil {
		t.Fatalf("HandleShell() error = %v", err)
	}
	if text := callToolResultText(t, result); !strings.Contains(text, "timeoutSeconds must be between") {
		t.Fatalf("result = %q, want timeout validation before shell selection", text)
	}
}

func callToolResultText(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()
	if result == nil || len(result.Content) == 0 {
		t.Fatal("tool result has no content")
	}
	text, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("tool result content type = %T, want *mcp.TextContent", result.Content[0])
	}
	return text.Text
}

func TestExecutionFeatureFlagsRemainIndependent(t *testing.T) {
	t.Setenv("MCP_ENABLE_EXECUTION", "")
	t.Setenv("MCP_ENABLE_RUN_SCRIPT", "1")
	t.Setenv("MCP_ENABLE_SHELL", "")

	if !executionFeatureEnabled("MCP_ENABLE_RUN_SCRIPT") {
		t.Fatal("run_script flag did not enable run_script")
	}
	if executionFeatureEnabled("MCP_ENABLE_SHELL") {
		t.Fatal("run_script flag incorrectly enabled shell")
	}
}
