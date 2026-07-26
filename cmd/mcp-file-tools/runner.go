package main

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type serverRunner interface {
	Run(context.Context, *mcp.Server) error
}

type singleSessionRunner struct {
	transport mcp.Transport
}

func (runner singleSessionRunner) Run(ctx context.Context, server *mcp.Server) error {
	if runner.transport == nil {
		return fmt.Errorf("transport is required")
	}
	return server.Run(ctx, runner.transport)
}

func newRunner(transport transportName) (serverRunner, error) {
	switch transport {
	case transportStdio:
		return singleSessionRunner{transport: &mcp.StdioTransport{}}, nil
	default:
		return nil, fmt.Errorf("unsupported transport %q", transport)
	}
}
