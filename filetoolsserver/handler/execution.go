package handler

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	defaultExecutionTimeoutSeconds = 60
	maximumExecutionTimeoutSeconds = 600
	maximumExecutionOutputBytes    = 256 * 1024
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
type ExecutionOutput struct {
	WorkingDirectory   string `json:"workingDirectory"`
	ExitCode           int    `json:"exitCode"`
	Stdout             string `json:"stdout,omitempty"`
	Stderr             string `json:"stderr,omitempty"`
	TimedOut           bool   `json:"timedOut,omitempty"`
	OutputTruncated    bool   `json:"outputTruncated,omitempty"`
	DurationMillis     int64  `json:"durationMillis"`
	ExecutionCancelled bool   `json:"executionCancelled,omitempty"`
}

// HandleRunScript executes a supported script whose path is inside an allowed directory.
func (h *Handler) HandleRunScript(ctx context.Context, req *mcp.CallToolRequest, input RunScriptInput) (*mcp.CallToolResult, ExecutionOutput, error) {
	if !executionFeatureEnabled("MCP_ENABLE_RUN_SCRIPT") {
		return errorResult("run_script is disabled; set MCP_ENABLE_RUN_SCRIPT=1 or MCP_ENABLE_EXECUTION=1 before starting the server"), ExecutionOutput{}, nil
	}

	validatedScript := h.ValidatePath(input.Path)
	if !validatedScript.Ok() {
		return validatedScript.Result, ExecutionOutput{}, nil
	}

	info, err := os.Stat(validatedScript.Path)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to inspect script: %v", err)), ExecutionOutput{}, nil
	}
	if info.IsDir() {
		return errorResult("path must refer to a script file, not a directory"), ExecutionOutput{}, nil
	}

	cwd := input.Cwd
	if strings.TrimSpace(cwd) == "" {
		cwd = filepath.Dir(validatedScript.Path)
	}
	validatedCwd := h.ValidatePath(cwd)
	if !validatedCwd.Ok() {
		return validatedCwd.Result, ExecutionOutput{}, nil
	}
	if err := requireDirectory(validatedCwd.Path); err != nil {
		return errorResult(err.Error()), ExecutionOutput{}, nil
	}

	timeout, err := executionTimeout(input.TimeoutSeconds)
	if err != nil {
		return errorResult(err.Error()), ExecutionOutput{}, nil
	}

	program, args, err := buildScriptCommand(validatedScript.Path, input.Args)
	if err != nil {
		return errorResult(err.Error()), ExecutionOutput{}, nil
	}

	output, err := executeProcess(ctx, program, args, validatedCwd.Path, timeout)
	if err != nil {
		return errorResult(err.Error()), ExecutionOutput{}, nil
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
	if err := requireDirectory(validatedCwd.Path); err != nil {
		return errorResult(err.Error()), ExecutionOutput{}, nil
	}

	timeout, err := executionTimeout(input.TimeoutSeconds)
	if err != nil {
		return errorResult(err.Error()), ExecutionOutput{}, nil
	}

	program, args, err := buildShellCommand(input.Shell, input.Command)
	if err != nil {
		return errorResult(err.Error()), ExecutionOutput{}, nil
	}

	output, err := executeProcess(ctx, program, args, validatedCwd.Path, timeout)
	if err != nil {
		return errorResult(err.Error()), ExecutionOutput{}, nil
	}
	return executionResult(output), output, nil
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

func executionTimeout(seconds int) (time.Duration, error) {
	if seconds == 0 {
		seconds = defaultExecutionTimeoutSeconds
	}
	if seconds < 1 || seconds > maximumExecutionTimeoutSeconds {
		return 0, fmt.Errorf("timeoutSeconds must be between 1 and %d", maximumExecutionTimeoutSeconds)
	}
	return time.Duration(seconds) * time.Second, nil
}

func requireDirectory(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to inspect working directory: %v", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("working directory is not a directory: %s", path)
	}
	return nil
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
			return "", nil, fmt.Errorf("Python was not found: %w", err)
		}
		return program, append([]string{scriptPath}, scriptArgs...), nil

	case ".js", ".mjs", ".cjs":
		program, err := firstExecutable("node.exe", "node")
		if err != nil {
			return "", nil, fmt.Errorf("Node.js was not found: %w", err)
		}
		return program, append([]string{scriptPath}, scriptArgs...), nil

	case ".sh":
		program, err := firstExecutable("bash.exe", "bash")
		if err != nil {
			return "", nil, fmt.Errorf("Bash was not found: %w", err)
		}
		return program, append([]string{scriptPath}, scriptArgs...), nil

	case ".exe", ".com":
		return scriptPath, scriptArgs, nil

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
				return "", nil, fmt.Errorf("Windows PowerShell was not found: %w", err)
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

func executeProcess(parent context.Context, program string, args []string, cwd string, timeout time.Duration) (ExecutionOutput, error) {
	started := time.Now()
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()

	stdout := newLimitedBuffer(maximumExecutionOutputBytes)
	stderr := newLimitedBuffer(maximumExecutionOutputBytes)

	cmd := exec.Command(program, args...)
	cmd.Dir = cwd
	cmd.Stdin = nil
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	if err := cmd.Start(); err != nil {
		return ExecutionOutput{}, fmt.Errorf("failed to start process: %w", err)
	}

	waitResult := make(chan error, 1)
	go func() {
		waitResult <- cmd.Wait()
	}()

	var runErr error
	timedOut := false
	cancelled := false

	select {
	case runErr = <-waitResult:
	case <-ctx.Done():
		timedOut = errors.Is(ctx.Err(), context.DeadlineExceeded)
		cancelled = !timedOut
		terminateProcessTree(cmd)
		runErr = <-waitResult
	}

	exitCode := 0
	if runErr != nil {
		var exitError *exec.ExitError
		if errors.As(runErr, &exitError) {
			exitCode = exitError.ExitCode()
		} else if timedOut || cancelled {
			exitCode = -1
		} else {
			return ExecutionOutput{}, fmt.Errorf("failed while waiting for process: %w", runErr)
		}
	}

	return ExecutionOutput{
		WorkingDirectory:   cwd,
		ExitCode:           exitCode,
		Stdout:             stdout.String(),
		Stderr:             stderr.String(),
		TimedOut:           timedOut,
		OutputTruncated:    stdout.Truncated() || stderr.Truncated(),
		DurationMillis:     time.Since(started).Milliseconds(),
		ExecutionCancelled: cancelled,
	}, nil
}

func executionResult(output ExecutionOutput) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: output.ExitCode != 0 || output.TimedOut || output.ExecutionCancelled,
	}
}

func terminateProcessTree(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}

	if runtime.GOOS == "windows" {
		killer := exec.Command("taskkill", "/PID", strconv.Itoa(cmd.Process.Pid), "/T", "/F")
		killer.Stdout = io.Discard
		killer.Stderr = io.Discard
		_ = killer.Run()
	}

	_ = cmd.Process.Kill()
}

type limitedBuffer struct {
	mu        sync.Mutex
	buffer    bytes.Buffer
	limit     int
	truncated bool
}

func newLimitedBuffer(limit int) *limitedBuffer {
	return &limitedBuffer{limit: limit}
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	originalLength := len(p)
	remaining := b.limit - b.buffer.Len()
	if remaining > 0 {
		writeLength := len(p)
		if writeLength > remaining {
			writeLength = remaining
		}
		_, _ = b.buffer.Write(p[:writeLength])
	}
	if originalLength > remaining {
		b.truncated = true
	}
	return originalLength, nil
}

func (b *limitedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buffer.String()
}

func (b *limitedBuffer) Truncated() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.truncated
}
