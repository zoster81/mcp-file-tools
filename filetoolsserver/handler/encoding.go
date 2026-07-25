package handler

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zoster81/mcp-file-tools/internal/encoding"
)

// HandleListEncodings returns a list of supported encodings
func (h *Handler) HandleListEncodings(ctx context.Context, req *mcp.CallToolRequest, input ListEncodingsInput) (*mcp.CallToolResult, ListEncodingsOutput, error) {
	return &mcp.CallToolResult{}, ListEncodingsOutput{Encodings: encoding.ListEncodings()}, nil
}
