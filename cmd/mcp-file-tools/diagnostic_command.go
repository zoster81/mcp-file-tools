package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/zoster81/mcp-file-tools/internal/backupstore"
	"github.com/zoster81/mcp-file-tools/internal/config"
)

type backupDiagnosticCommandOptions struct {
	store      string
	mode       backupstore.AuditMode
	maxObjects int
	maxBytes   int64
	pretty     bool
}

func parseBackupDiagnosticCommand(args []string) (backupDiagnosticCommandOptions, bool, error) {
	if len(args) < 2 || args[0] != "backup-store" || args[1] != "diagnose" {
		return backupDiagnosticCommandOptions{}, false, nil
	}
	options := backupDiagnosticCommandOptions{mode: backupstore.AuditQuick}
	seen := make(map[string]bool)
	for index := 2; index < len(args); index++ {
		argument := args[index]
		switch {
		case argument == "--store":
			value, next, err := diagnosticOptionValue(args, index, "--store")
			if err != nil {
				return backupDiagnosticCommandOptions{}, true, err
			}
			index = next
			if err := setDiagnosticStoreOption(&options, seen, value); err != nil {
				return backupDiagnosticCommandOptions{}, true, err
			}
		case strings.HasPrefix(argument, "--store="):
			if err := setDiagnosticStoreOption(&options, seen, strings.TrimPrefix(argument, "--store=")); err != nil {
				return backupDiagnosticCommandOptions{}, true, err
			}
		case argument == "--mode":
			value, next, err := diagnosticOptionValue(args, index, "--mode")
			if err != nil {
				return backupDiagnosticCommandOptions{}, true, err
			}
			index = next
			if err := setDiagnosticModeOption(&options, seen, value); err != nil {
				return backupDiagnosticCommandOptions{}, true, err
			}
		case strings.HasPrefix(argument, "--mode="):
			if err := setDiagnosticModeOption(&options, seen, strings.TrimPrefix(argument, "--mode=")); err != nil {
				return backupDiagnosticCommandOptions{}, true, err
			}
		case argument == "--max-objects":
			value, next, err := diagnosticOptionValue(args, index, "--max-objects")
			if err != nil {
				return backupDiagnosticCommandOptions{}, true, err
			}
			index = next
			if err := setDiagnosticMaxObjectsOption(&options, seen, value); err != nil {
				return backupDiagnosticCommandOptions{}, true, err
			}
		case strings.HasPrefix(argument, "--max-objects="):
			if err := setDiagnosticMaxObjectsOption(&options, seen, strings.TrimPrefix(argument, "--max-objects=")); err != nil {
				return backupDiagnosticCommandOptions{}, true, err
			}
		case argument == "--max-bytes":
			value, next, err := diagnosticOptionValue(args, index, "--max-bytes")
			if err != nil {
				return backupDiagnosticCommandOptions{}, true, err
			}
			index = next
			if err := setDiagnosticMaxBytesOption(&options, seen, value); err != nil {
				return backupDiagnosticCommandOptions{}, true, err
			}
		case strings.HasPrefix(argument, "--max-bytes="):
			if err := setDiagnosticMaxBytesOption(&options, seen, strings.TrimPrefix(argument, "--max-bytes=")); err != nil {
				return backupDiagnosticCommandOptions{}, true, err
			}
		case argument == "--pretty":
			if seen["pretty"] {
				return backupDiagnosticCommandOptions{}, true, errors.New("--pretty may be specified only once")
			}
			seen["pretty"] = true
			options.pretty = true
		default:
			return backupDiagnosticCommandOptions{}, true, errors.New("unsupported backup diagnostic argument")
		}
	}
	if options.store == "" {
		return backupDiagnosticCommandOptions{}, true, errors.New("--store is required")
	}
	return options, true, nil
}

func diagnosticOptionValue(args []string, index int, name string) (string, int, error) {
	if index+1 >= len(args) {
		return "", index, fmt.Errorf("%s requires a value", name)
	}
	value := strings.TrimSpace(args[index+1])
	if value == "" || strings.HasPrefix(value, "--") {
		return "", index, fmt.Errorf("%s requires a non-empty value", name)
	}
	return value, index + 1, nil
}

func setDiagnosticStoreOption(options *backupDiagnosticCommandOptions, seen map[string]bool, value string) error {
	if seen["store"] {
		return errors.New("--store may be specified only once")
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return errors.New("--store requires a non-empty value")
	}
	seen["store"] = true
	options.store = value
	return nil
}

func setDiagnosticModeOption(options *backupDiagnosticCommandOptions, seen map[string]bool, value string) error {
	if seen["mode"] {
		return errors.New("--mode may be specified only once")
	}
	mode := backupstore.AuditMode(strings.ToLower(strings.TrimSpace(value)))
	if mode != backupstore.AuditQuick && mode != backupstore.AuditFull {
		return errors.New("--mode must be quick or full")
	}
	seen["mode"] = true
	options.mode = mode
	return nil
}

func setDiagnosticMaxObjectsOption(options *backupDiagnosticCommandOptions, seen map[string]bool, value string) error {
	if seen["max-objects"] {
		return errors.New("--max-objects may be specified only once")
	}
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || parsed <= 0 {
		return errors.New("--max-objects must be a positive integer")
	}
	seen["max-objects"] = true
	options.maxObjects = parsed
	return nil
}

func setDiagnosticMaxBytesOption(options *backupDiagnosticCommandOptions, seen map[string]bool, value string) error {
	if seen["max-bytes"] {
		return errors.New("--max-bytes may be specified only once")
	}
	parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil || parsed <= 0 {
		return errors.New("--max-bytes must be a positive integer")
	}
	seen["max-bytes"] = true
	options.maxBytes = parsed
	return nil
}

func runBackupDiagnosticCommand(
	ctx context.Context,
	options backupDiagnosticCommandOptions,
	stdout io.Writer,
	stderr io.Writer,
	getenv func(string) string,
) int {
	store, err := backupstore.OpenExistingForDiagnosis(backupstore.DiagnosticOpenOptions{Directory: options.store})
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return 1
	}
	report, diagnoseErr := store.Diagnose(ctx, backupstore.DiagnosticOptions{
		Mode:       options.mode,
		MaxObjects: options.maxObjects,
		MaxBytes:   options.maxBytes,
	})
	closeErr := store.Close()
	if diagnoseErr != nil {
		fmt.Fprintf(stderr, "Error: %v\n", diagnoseErr)
		return 1
	}
	if closeErr != nil {
		fmt.Fprintln(stderr, "Error: backup diagnostic lock could not be released")
		return 1
	}

	encoded, err := encodeBackupDiagnosticReport(report, options.pretty)
	if err != nil {
		fmt.Fprintln(stderr, "Error: backup diagnostic report could not be encoded")
		return 1
	}
	if int64(len(encoded)) > backupDiagnosticOutputLimit(getenv) {
		fmt.Fprintln(stderr, "Error: backup diagnostic output exceeds the configured limit")
		return 1
	}
	written, writeErr := stdout.Write(encoded)
	if writeErr != nil || written != len(encoded) {
		fmt.Fprintln(stderr, "Error: backup diagnostic output could not be written")
		return 1
	}
	if report.SafeForNormalOpen && len(report.Issues) == 0 {
		return 0
	}
	return 2
}

func encodeBackupDiagnosticReport(report backupstore.DiagnosticReport, pretty bool) ([]byte, error) {
	var encoded []byte
	var err error
	if pretty {
		encoded, err = json.MarshalIndent(report, "", "  ")
	} else {
		encoded, err = json.Marshal(report)
	}
	if err != nil {
		return nil, err
	}
	return append(encoded, '\n'), nil
}

func backupDiagnosticOutputLimit(getenv func(string) string) int64 {
	limit := config.DefaultMaxOutputBytes
	if getenv == nil {
		return limit
	}
	parsed, err := strconv.ParseInt(strings.TrimSpace(getenv(config.EnvMaxOutputBytes)), 10, 64)
	if err == nil && parsed > 0 {
		return parsed
	}
	return limit
}
