package main

import "testing"

func TestLoadCommandDefaultsUsesStdio(t *testing.T) {
	opts := loadCommandDefaults(func(string) string { return "" })
	if opts.transport != transportStdio {
		t.Fatalf("transport = %q, want %q", opts.transport, transportStdio)
	}
}

func TestParseCommandOptionsDefaultsToStdio(t *testing.T) {
	opts, err := parseCommandOptions(nil, commandOptions{transport: transportStdio})
	if err != nil {
		t.Fatalf("parseCommandOptions() error = %v", err)
	}
	if opts.transport != transportStdio {
		t.Fatalf("transport = %q, want %q", opts.transport, transportStdio)
	}
	if len(opts.allowedDirectories) != 0 {
		t.Fatalf("allowed directories = %v, want none", opts.allowedDirectories)
	}
}

func TestParseCommandOptionsCLIOverridesEnvironmentDefault(t *testing.T) {
	defaults := loadCommandDefaults(func(name string) string {
		if name == envTransport {
			return "unsupported"
		}
		return ""
	})
	opts, err := parseCommandOptions([]string{"--transport=stdio", "project"}, defaults)
	if err != nil {
		t.Fatalf("parseCommandOptions() error = %v", err)
	}
	if opts.transport != transportStdio {
		t.Fatalf("transport = %q, want %q", opts.transport, transportStdio)
	}
	if got, want := opts.allowedDirectories, []string{"project"}; len(got) != 1 || got[0] != want[0] {
		t.Fatalf("allowed directories = %v, want %v", got, want)
	}
}

func TestParseCommandOptionsSupportsSeparatedTransportValue(t *testing.T) {
	opts, err := parseCommandOptions([]string{"--transport", "stdio", "first", "second"}, commandOptions{transport: transportStdio})
	if err != nil {
		t.Fatalf("parseCommandOptions() error = %v", err)
	}
	if opts.transport != transportStdio {
		t.Fatalf("transport = %q, want %q", opts.transport, transportStdio)
	}
	if got := opts.allowedDirectories; len(got) != 2 || got[0] != "first" || got[1] != "second" {
		t.Fatalf("allowed directories = %v", got)
	}
}

func TestParseCommandOptionsSeparatorPreservesFlagLikeDirectory(t *testing.T) {
	opts, err := parseCommandOptions([]string{"--", "--workspace"}, commandOptions{transport: transportStdio})
	if err != nil {
		t.Fatalf("parseCommandOptions() error = %v", err)
	}
	if got := opts.allowedDirectories; len(got) != 1 || got[0] != "--workspace" {
		t.Fatalf("allowed directories = %v, want [--workspace]", got)
	}
}

func TestParseCommandOptionsRejectsUnsupportedTransport(t *testing.T) {
	defaults := loadCommandDefaults(func(name string) string {
		if name == envTransport {
			return "streamable-http"
		}
		return ""
	})
	_, err := parseCommandOptions(nil, defaults)
	if err == nil {
		t.Fatal("parseCommandOptions() accepted an unsupported transport")
	}
}

func TestParseCommandOptionsRequiresTransportValue(t *testing.T) {
	if _, err := parseCommandOptions([]string{"--transport"}, commandOptions{transport: transportStdio}); err == nil {
		t.Fatal("parseCommandOptions() accepted --transport without a value")
	}
}
