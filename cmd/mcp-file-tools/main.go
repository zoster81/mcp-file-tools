package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/zoster81/mcp-file-tools/filetoolsserver"
	"github.com/zoster81/mcp-file-tools/internal/config"
	"github.com/zoster81/mcp-file-tools/internal/security"
)

// version is set at build time via ldflags.
var version = "dev"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	os.Exit(runCommand(ctx, os.Args[1:], os.Stdout, os.Stderr, os.Getenv))
}

func runCommand(ctx context.Context, args []string, stdout, stderr io.Writer, getenv func(string) string) int {
	configureLogging(stderr, getenv)

	// Keep the legacy exported version synchronized for existing embedders while
	// the explicit server options remain authoritative for this process.
	filetoolsserver.Version = version

	if len(args) == 1 && (args[0] == "--version" || args[0] == "-v") {
		fmt.Fprintln(stdout, version)
		return 0
	}

	options, err := parseCommandOptions(args, loadCommandDefaults(getenv))
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return 1
	}

	normalized, err := security.NormalizeAllowedDirs(options.allowedDirectories)
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return 1
	}
	if len(normalized) > 0 {
		slog.Debug("normalized allowed directories", "dirs", normalized)
	}

	runner, err := newRunner(options.transport)
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return 1
	}
	server := filetoolsserver.BuildServer(filetoolsserver.ServerOptions{
		Version:            version,
		AllowedDirectories: normalized,
		Config:             config.Load(),
		EnableClientRoots:  options.transport == transportStdio,
		LifecycleContext:   ctx,
	})

	if err := runner.Run(ctx, server); err != nil {
		if ctx.Err() != nil && errors.Is(err, ctx.Err()) {
			return 0
		}
		fmt.Fprintf(stderr, "Server error: %v\n", err)
		return 1
	}
	return 0
}

func configureLogging(stderr io.Writer, getenv func(string) string) {
	// Protocol output remains reserved for stdout; all logs use stderr.
	level := slog.LevelInfo
	switch strings.ToLower(strings.TrimSpace(getenv("MCP_LOG_LEVEL"))) {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(stderr, &slog.HandlerOptions{Level: level})))
}
