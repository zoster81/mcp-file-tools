package handler

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zoster81/mcp-file-tools/internal/encoding"
)

// HandleDetectEncoding detects the encoding of a file
func (h *Handler) HandleDetectEncoding(ctx context.Context, req *mcp.CallToolRequest, input DetectEncodingInput) (*mcp.CallToolResult, DetectEncodingOutput, error) {
	v := h.ValidatePath(input.Path)
	if !v.Ok() {
		return v.Result, DetectEncodingOutput{}, nil
	}

	mode := input.Mode
	if mode == "" {
		mode = "sample"
	}

	result, err := encoding.DetectFromFile(v.Path, mode)
	if err != nil {
		return errorResultFromError(err), DetectEncodingOutput{}, nil
	}

	if result.Charset == "" {
		return errorResult("could not detect encoding"), DetectEncodingOutput{}, nil
	}

	return &mcp.CallToolResult{}, DetectEncodingOutput{
		Encoding:   result.Charset,
		Confidence: result.Confidence,
		HasBOM:     result.HasBOM,
	}, nil
}
