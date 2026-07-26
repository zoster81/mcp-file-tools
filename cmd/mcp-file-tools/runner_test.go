package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zoster81/mcp-file-tools/filetoolsserver"
	"github.com/zoster81/mcp-file-tools/internal/config"
)

func TestSingleSessionRunnerUsesSharedServerAndHonorsCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	runner := singleSessionRunner{transport: serverTransport}
	server := filetoolsserver.BuildServer(filetoolsserver.ServerOptions{
		Version:            "runner-test",
		AllowedDirectories: []string{t.TempDir()},
		Config:             config.Load(),
		EnableClientRoots:  false,
		LifecycleContext:   ctx,
	})

	errCh := make(chan error, 1)
	go func() { errCh <- runner.Run(ctx, server) }()

	client := mcp.NewClient(&mcp.Implementation{Name: "runner-test", Version: "test"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("connect client: %v", err)
	}

	tools, err := clientSession.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	if len(tools.Tools) != 23 {
		t.Fatalf("tool count = %d, want 23", len(tools.Tools))
	}

	cancel()
	_ = clientSession.Close()
	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, context.Canceled) {
			t.Fatalf("runner error = %v, want context cancellation", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("runner did not stop after context cancellation")
	}
}
