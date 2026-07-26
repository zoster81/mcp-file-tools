package filetoolsserver

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zoster81/mcp-file-tools/internal/config"
)

func TestBuildServerUsesExplicitVersionAndSharedProcessRoots(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	roots := []string{t.TempDir(), t.TempDir()}
	server := BuildServer(ServerOptions{
		Version:            "architecture-test",
		AllowedDirectories: roots,
		Config:             config.Load(),
		EnableClientRoots:  false,
		LifecycleContext:   ctx,
	})

	first := connectTestClient(t, ctx, server, "first")
	second := connectTestClient(t, ctx, server, "second")

	for name, session := range map[string]*mcp.ClientSession{"first": first, "second": second} {
		init := session.InitializeResult()
		if init == nil || init.ServerInfo == nil || init.ServerInfo.Version != "architecture-test" {
			t.Fatalf("%s session server version = %#v", name, init)
		}
	}

	firstTools := toolNames(t, ctx, first)
	secondTools := toolNames(t, ctx, second)
	if !reflect.DeepEqual(firstTools, secondTools) {
		t.Fatalf("tool catalogs differ: first=%v second=%v", firstTools, secondTools)
	}

	firstRoots := allowedDirectories(t, ctx, first)
	secondRoots := allowedDirectories(t, ctx, second)
	if !reflect.DeepEqual(firstRoots, secondRoots) {
		t.Fatalf("sessions observed different process roots: first=%v second=%v", firstRoots, secondRoots)
	}
	if len(firstRoots) != len(roots) {
		t.Fatalf("allowed root count = %d, want %d", len(firstRoots), len(roots))
	}
}

func TestBuildServerUsesProvidedConfiguration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	root := t.TempDir()
	cfg := config.Load()
	cfg.DefaultEncoding = "cp1251"
	server := BuildServer(ServerOptions{
		Version:            "configuration-test",
		AllowedDirectories: []string{root},
		Config:             cfg,
		EnableClientRoots:  false,
		LifecycleContext:   ctx,
	})
	session := connectTestClient(t, ctx, server, "configuration")

	path := filepath.Join(root, "configured-default.txt")
	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "write_file",
		Arguments: map[string]any{
			"path":    path,
			"content": "Привет",
		},
	})
	if err != nil {
		t.Fatalf("call write_file: %v", err)
	}
	if result.IsError {
		t.Fatalf("write_file returned an error: %#v", result.Content)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read configured output: %v", err)
	}
	want := []byte{0xcf, 0xf0, 0xe8, 0xe2, 0xe5, 0xf2}
	if !bytes.Equal(data, want) {
		t.Fatalf("configured output bytes = %x, want cp1251 %x", data, want)
	}
}

func connectTestClient(t *testing.T, ctx context.Context, server *mcp.Server, name string) *mcp.ClientSession {
	t.Helper()
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("connect server %s: %v", name, err)
	}
	t.Cleanup(func() { _ = serverSession.Close() })

	client := mcp.NewClient(&mcp.Implementation{Name: name, Version: "test"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("connect client %s: %v", name, err)
	}
	t.Cleanup(func() { _ = clientSession.Close() })
	return clientSession
}

func toolNames(t *testing.T, ctx context.Context, session *mcp.ClientSession) []string {
	t.Helper()
	result, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	names := make([]string, 0, len(result.Tools))
	for _, tool := range result.Tools {
		names = append(names, tool.Name)
	}
	sort.Strings(names)
	return names
}

func allowedDirectories(t *testing.T, ctx context.Context, session *mcp.ClientSession) []string {
	t.Helper()
	result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "list_allowed_directories"})
	if err != nil {
		t.Fatalf("call list_allowed_directories: %v", err)
	}
	data, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatalf("marshal structured content: %v", err)
	}
	var output struct {
		Directories []string `json:"directories"`
	}
	if err := json.Unmarshal(data, &output); err != nil {
		t.Fatalf("decode structured content %s: %v", data, err)
	}
	return output.Directories
}
