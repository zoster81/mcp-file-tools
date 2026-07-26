package main

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/zoster81/mcp-file-tools/filetoolsserver"
)

func TestRunCommandVersionWritesOnlyVersion(t *testing.T) {
	originalVersion := version
	originalServerVersion := filetoolsserver.Version
	version = "test-version"
	t.Cleanup(func() {
		version = originalVersion
		filetoolsserver.Version = originalServerVersion
	})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runCommand(context.Background(), []string{"--version"}, &stdout, &stderr, func(string) string { return "" })
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if stdout.String() != "test-version\n" {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunCommandRejectsUnsupportedTransportBeforeStartup(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runCommand(context.Background(), nil, &stdout, &stderr, func(name string) string {
		if name == envTransport {
			return "streamable-http"
		}
		return ""
	})
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "unsupported transport") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}
