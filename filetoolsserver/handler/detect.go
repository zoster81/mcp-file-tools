package handler

import (
	"context"
	"os"

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

	info, err := os.Stat(v.Path)
	if err != nil {
		return errorResultFromError(err), DetectEncodingOutput{}, nil
	}
	if info.Size() == 0 {
		return &mcp.CallToolResult{}, DetectEncodingOutput{
			Encoding: "utf-8",
			Assumed:  true,
		}, nil
	}

	result, err := encoding.DetectFromFile(v.Path, mode)
	if err != nil {
		return errorResultFromError(err), DetectEncodingOutput{}, nil
	}

	output := DetectEncodingOutput{
		Encoding:   result.Charset,
		Confidence: result.Confidence,
		HasBOM:     result.HasBOM,
		Ambiguous:  result.Charset == "" || result.Confidence < encoding.MinConfidenceThreshold,
	}
	if result.HasBOM {
		output.BOMType = result.Charset
	}
	return &mcp.CallToolResult{}, output, nil
}
