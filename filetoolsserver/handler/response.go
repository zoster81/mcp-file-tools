package handler

import "github.com/modelcontextprotocol/go-sdk/mcp"

// errorResult creates a generic stable MCP tool error.
func errorResult(message string) *mcp.CallToolResult {
	return errorResultWithCode(ErrCodeOperationFailed, message)
}

// errorResultWithCode creates an MCP tool error with a stable machine-readable
// code in _meta while preserving the human-readable text content.
func errorResultWithCode(code, message string) *mcp.CallToolResult {
	if code == "" {
		code = ErrCodeOperationFailed
	}
	return &mcp.CallToolResult{
		Meta:    mcp.Meta{ErrorCodeMetaKey: code},
		Content: []mcp.Content{&mcp.TextContent{Text: message}},
		IsError: true,
	}
}
