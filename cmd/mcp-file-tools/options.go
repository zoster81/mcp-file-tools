package main

import (
	"fmt"
	"strings"
)

const envTransport = "MCP_TRANSPORT"

type transportName string

const transportStdio transportName = "stdio"

type commandOptions struct {
	transport          transportName
	allowedDirectories []string
}

func loadCommandDefaults(getenv func(string) string) commandOptions {
	transport := transportName(strings.ToLower(strings.TrimSpace(getenv(envTransport))))
	if transport == "" {
		transport = transportStdio
	}
	return commandOptions{transport: transport}
}

func parseCommandOptions(args []string, defaults commandOptions) (commandOptions, error) {
	transport := defaults.transport
	directories := append([]string(nil), defaults.allowedDirectories...)
	parseOptions := true
	for index := 0; index < len(args); index++ {
		argument := args[index]
		if parseOptions && argument == "--" {
			parseOptions = false
			continue
		}
		if parseOptions && argument == "--transport" {
			if index+1 >= len(args) {
				return commandOptions{}, fmt.Errorf("--transport requires a value")
			}
			index++
			transport = transportName(strings.ToLower(strings.TrimSpace(args[index])))
			continue
		}
		if parseOptions && strings.HasPrefix(argument, "--transport=") {
			transport = transportName(strings.ToLower(strings.TrimSpace(strings.TrimPrefix(argument, "--transport="))))
			continue
		}
		directories = append(directories, argument)
	}

	if transport != transportStdio {
		return commandOptions{}, fmt.Errorf("unsupported transport %q; supported transports: %s", transport, transportStdio)
	}
	return commandOptions{transport: transport, allowedDirectories: directories}, nil
}
