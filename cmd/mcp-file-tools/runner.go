package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zoster81/mcp-file-tools/filetoolsserver/handler"
	"github.com/zoster81/mcp-file-tools/internal/httptransport"
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

type runnerSelection struct {
	runner            serverRunner
	enableClientRoots bool
	executionPolicy   *handler.ExecutionPolicy
}

func selectRunner(
	transport transportName,
	getenv func(string) string,
	maxSessions int,
) (runnerSelection, error) {
	switch transport {
	case transportStdio:
		return runnerSelection{
			runner:            singleSessionRunner{transport: &mcp.StdioTransport{}},
			enableClientRoots: true,
		}, nil
	case transportStreamableHTTP:
		httpConfig, err := httptransport.LoadConfig(getenv, maxSessions)
		if err != nil {
			return runnerSelection{}, err
		}
		httptransport.ClearCredentialEnvironment()
		basePolicy := handler.ExecutionPolicyFromEnvironment(getenv)
		executionPolicy := &handler.ExecutionPolicy{
			AllowRunScript: httpConfig.EnableExecution && basePolicy.AllowRunScript,
			AllowShell:     httpConfig.EnableExecution && basePolicy.AllowShell,
		}
		return runnerSelection{
			runner: httptransport.Runner{
				Config: httpConfig,
				Logger: slog.Default(),
			},
			enableClientRoots: false,
			executionPolicy:   executionPolicy,
		}, nil
	default:
		return runnerSelection{}, fmt.Errorf("unsupported transport %q", transport)
	}
}
