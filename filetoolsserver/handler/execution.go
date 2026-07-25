package handler

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	internalexecution "github.com/zoster81/mcp-file-tools/internal/execution"
	"github.com/zoster81/mcp-file-tools/internal/filesystem"
)

// RunScriptInput executes a script located inside an allowed directory.
type RunScriptInput struct {
	Path           string   `json:"path"`
	Args           []string `json:"args,omitempty"`
	Cwd            string   `json:"cwd,omitempty"`
	TimeoutSeconds int      `json:"timeoutSeconds,omitempty"`
}

// ShellInput executes an arbitrary shell command. The cwd is validated against
// the allowed directories, but the command itself is intentionally unrestricted.
type ShellInput struct {
	Command        string `json:"command"`
	Cwd            string `json:"cwd,omitempty"`
	Shell          string `json:"shell,omitempty"`
	TimeoutSeconds int    `json:"timeoutSeconds,omitempty"`
}

// ExecutionOutput is returned by run_script and shell.
type ExecutionOutput = internalexecution.Result

// HandleRunScript executes a supported script whose path is inside an allowed directory.
func (h *Handler) HandleRunScript(ctx context.Context, req *mcp.CallToolRequest, input RunScriptInput) (*mcp.CallToolResult, ExecutionOutput, error) {
	if !executionFeatureEnabled("MCP_ENABLE_RUN_SCRIPT") {
		return errorResult("run_script is disabled; set MCP_ENABLE_RUN_SCRIPT=1 or MCP_ENABLE_EXECUTION=1 before starting the server"), ExecutionOutput{}, nil
	}

	validatedScript := h.ValidatePath(input.Path)
	if !validatedScript.Ok() {
		return validatedScript.Result, ExecutionOutput{}, nil
	}
	scriptInfo, err := inspectScriptFile(validatedScript.Path)
	if err != nil {
		return errorResultFromError(err), ExecutionOutput{}, nil
	}

	cwd := input.Cwd
	if strings.TrimSpace(cwd) == "" {
		cwd = filepath.Dir(validatedScript.Path)
	}
	validatedCwd := h.ValidatePath(cwd)
	if !validatedCwd.Ok() {
		return validatedCwd.Result, ExecutionOutput{}, nil
	}
	if _, err := internalexecution.ValidateWorkingDirectory(validatedCwd.Path); err != nil {
		return errorResultFromError(err), ExecutionOutput{}, nil
	}
	if err := internalexecution.ValidateTimeoutSeconds(input.TimeoutSeconds); err != nil {
		return errorResultFromError(err), ExecutionOutput{}, nil
	}

	program, args, err := buildScriptCommand(validatedScript.Path, input.Args)
	if err != nil {
		return errorResultFromError(err), ExecutionOutput{}, nil
	}
	plan, err := internalexecution.Prepare(internalexecution.Request{
		Program:          program,
		Args:             args,
		WorkingDirectory: validatedCwd.Path,
		TimeoutSeconds:   input.TimeoutSeconds,
	})
	if err != nil {
		return errorResultFromError(err), ExecutionOutput{}, nil
	}

	output, err := plan.Run(ctx, func() error {
		if err := h.revalidateWorkingDirectory(validatedCwd.Path); err != nil {
			return err
		}
		return h.revalidateScript(validatedScript.Path, scriptInfo)
	})
	if err != nil {
		return errorResultFromError(err), ExecutionOutput{}, nil
	}
	return executionResult(output), output, nil
}

// HandleShell executes an arbitrary command through the selected shell.
func (h *Handler) HandleShell(ctx context.Context, req *mcp.CallToolRequest, input ShellInput) (*mcp.CallToolResult, ExecutionOutput, error) {
	if !executionFeatureEnabled("MCP_ENABLE_SHELL") {
		return errorResult("shell is disabled; set MCP_ENABLE_SHELL=1 or MCP_ENABLE_EXECUTION=1 before starting the server"), ExecutionOutput{}, nil
	}
	if strings.TrimSpace(input.Command) == "" {
		return errorResult("command is required and must be a non-empty string"), ExecutionOutput{}, nil
	}

	cwd := strings.TrimSpace(input.Cwd)
	if cwd == "" {
		allowedDirs := h.GetAllowedDirectories()
		if len(allowedDirs) == 0 {
			return errorResult("no allowed directories are configured; pass at least one directory when starting the server"), ExecutionOutput{}, nil
		}
		cwd = allowedDirs[0]
	}
	validatedCwd := h.ValidatePath(cwd)
	if !validatedCwd.Ok() {
		return validatedCwd.Result, ExecutionOutput{}, nil
	}
	if _, err := internalexecution.ValidateWorkingDirectory(validatedCwd.Path); err != nil {
		return errorResultFromError(err), ExecutionOutput{}, nil
	}
	if err := internalexecution.ValidateTimeoutSeconds(input.TimeoutSeconds); err != nil {
		return errorResultFromError(err), ExecutionOutput{}, nil
	}

	program, args, err := buildShellCommand(input.Shell, input.Command)
	if err != nil {
		return errorResultFromError(err), ExecutionOutput{}, nil
	}
	plan, err := internalexecution.Prepare(internalexecution.Request{
		Program:          program,
		Args:             args,
		WorkingDirectory: validatedCwd.Path,
		TimeoutSeconds:   input.TimeoutSeconds,
	})
	if err != nil {
		return errorResultFromError(err), ExecutionOutput{}, nil
	}

	output, err := plan.Run(ctx, func() error {
		return h.revalidateWorkingDirectory(validatedCwd.Path)
	})
	if err != nil {
		return errorResultFromError(err), ExecutionOutput{}, nil
	}
	return executionResult(output), output, nil
}

func (h *Handler) revalidateWorkingDirectory(path string) error {
	validated, err := h.validatePath(path)
	if err != nil {
		return err
	}
	_, err = internalexecution.ValidateWorkingDirectory(validated)
	return err
}

func (h *Handler) revalidateScript(path string, original filesystem.FileSnapshot) error {
	validated, err := h.validatePath(path)
	if err != nil {
		return err
	}
	if err := original.Verify(validated); err != nil {
		return fmt.Errorf("script changed before execution: %s: %w", validated, err)
	}
	return nil
}

func inspectScriptFile(path string) (filesystem.FileSnapshot, error) {
	info, err := os.Stat(path)
	if err != nil {
		return filesystem.FileSnapshot{}, fmt.Errorf("failed to inspect script: %w", err)
	}
	if info.IsDir() {
		return filesystem.FileSnapshot{}, fmt.Errorf("path must refer to a script file, not a directory")
	}
	snapshot, err := filesystem.CaptureSnapshotWithDigest(path)
	if err != nil {
		return filesystem.FileSnapshot{}, fmt.Errorf("failed to inspect script: %w", err)
	}
	return snapshot, nil
}

func executionFeatureEnabled(specificVariable string) bool {
	return environmentFlagEnabled("MCP_ENABLE_EXECUTION") || environmentFlagEnabled(specificVariable)
}

func environmentFlagEnabled(name string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(name))) {
	case "1", "true", "yes", "on", "enabled":
		return true
	default:
		return false
	}
}

func buildScriptCommand(scriptPath string, scriptArgs []string) (string, []string, error) {
	extension := strings.ToLower(filepath.Ext(scriptPath))

	switch extension {
	case ".ps1":
		program, err := firstExecutable("pwsh.exe", "pwsh", "powershell.exe", "powershell")
		if err != nil {
			return "", nil, fmt.Errorf("PowerShell was not found: %w", err)
		}
		args := []string{"-NoLogo", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-File", scriptPath}
		return program, append(args, scriptArgs...), nil

	case ".bat", ".cmd":
		if runtime.GOOS != "windows" {
			return "", nil, fmt.Errorf("%s scripts are supported only on Windows", extension)
		}
		program, err := firstExecutable("cmd.exe", "cmd")
		if err != nil {
			return "", nil, fmt.Errorf("cmd.exe was not found: %w", err)
		}
		args := []string{"/d", "/s", "/c", scriptPath}
		return program, append(args, scriptArgs...), nil

	case ".py":
		if program, err := firstExecutable("py.exe", "py"); err == nil {
			args := []string{"-3", scriptPath}
			return program, append(args, scriptArgs...), nil
		}
		program, err := firstExecutable("python.exe", "python3", "python")
		if err != nil {
			return "", nil, fmt.Errorf("python was not found: %w", err)
		}
		return program, append([]string{scriptPath}, scriptArgs...), nil

	case ".js", ".mjs", ".cjs":
		program, err := firstExecutable("node.exe", "node")
		if err != nil {
			return "", nil, fmt.Errorf("node.js was not found: %w", err)
		}
		return program, append([]string{scriptPath}, scriptArgs...), nil

	case ".sh":
		program, err := firstExecutable("bash.exe", "bash")
		if err != nil {
			return "", nil, fmt.Errorf("bash was not found: %w", err)
		}
		return program, append([]string{scriptPath}, scriptArgs...), nil

	case ".exe", ".com":
		return scriptPath, append([]string(nil), scriptArgs...), nil

	default:
		return "", nil, fmt.Errorf("unsupported script type %q; supported extensions: .ps1, .bat, .cmd, .py, .js, .mjs, .cjs, .sh, .exe, .com", extension)
	}
}

func buildShellCommand(requestedShell, command string) (string, []string, error) {
	shell := strings.ToLower(strings.TrimSpace(requestedShell))

	if runtime.GOOS == "windows" {
		if shell == "" {
			shell = "powershell"
		}
		switch shell {
		case "powershell", "windows-powershell":
			program, err := firstExecutable("powershell.exe", "powershell")
			if err != nil {
				return "", nil, fmt.Errorf("windows powershell was not found: %w", err)
			}
			return program, []string{"-NoLogo", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", command}, nil
		case "pwsh", "powershell-core":
			program, err := firstExecutable("pwsh.exe", "pwsh")
			if err != nil {
				return "", nil, fmt.Errorf("PowerShell 7 was not found: %w", err)
			}
			return program, []string{"-NoLogo", "-NoProfile", "-NonInteractive", "-Command", command}, nil
		case "cmd":
			program, err := firstExecutable("cmd.exe", "cmd")
			if err != nil {
				return "", nil, fmt.Errorf("cmd.exe was not found: %w", err)
			}
			return program, []string{"/d", "/s", "/c", command}, nil
		default:
			return "", nil, fmt.Errorf("unsupported shell %q on Windows; use powershell, pwsh, or cmd", requestedShell)
		}
	}

	if shell == "" {
		shell = "sh"
	}
	switch shell {
	case "sh":
		program, err := firstExecutable("sh")
		if err != nil {
			return "", nil, fmt.Errorf("sh was not found: %w", err)
		}
		return program, []string{"-c", command}, nil
	case "bash":
		program, err := firstExecutable("bash")
		if err != nil {
			return "", nil, fmt.Errorf("bash was not found: %w", err)
		}
		return program, []string{"-c", command}, nil
	case "pwsh", "powershell":
		program, err := firstExecutable("pwsh", "powershell")
		if err != nil {
			return "", nil, fmt.Errorf("PowerShell was not found: %w", err)
		}
		return program, []string{"-NoLogo", "-NoProfile", "-NonInteractive", "-Command", command}, nil
	default:
		return "", nil, fmt.Errorf("unsupported shell %q; use sh, bash, or pwsh", requestedShell)
	}
}

func firstExecutable(candidates ...string) (string, error) {
	var lastErr error
	for _, candidate := range candidates {
		path, err := exec.LookPath(candidate)
		if err == nil {
			return path, nil
		}
		lastErr = err
	}
	return "", lastErr
}

func executionResult(output ExecutionOutput) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: output.ExitCode != 0 || output.TimedOut || output.ExecutionCancelled,
	}
}
