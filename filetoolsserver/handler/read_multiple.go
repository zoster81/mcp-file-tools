package handler

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zoster81/mcp-file-tools/internal/concurrency"
)

// HandleReadMultipleFiles reads multiple files concurrently.
// Individual file failures don't stop the operation - errors are reported per file.
func (h *Handler) HandleReadMultipleFiles(ctx context.Context, req *mcp.CallToolRequest, input ReadMultipleFilesInput) (*mcp.CallToolResult, ReadMultipleFilesOutput, error) {
	if len(input.Paths) == 0 {
		return errorResult("paths array is required and must contain at least one path"), ReadMultipleFilesOutput{}, nil
	}
	results := make([]FileReadResult, len(input.Paths))
	concurrency.ProcessOrdered(ctx, input.Paths, concurrency.Options{
		ContinueOnCancellation: true,
	}, func(ctx context.Context, _ int, filePath string) FileReadResult {
		if err := ctx.Err(); err != nil {
			mapped := mapOperationError(err, filePath)
			return FileReadResult{
				Path:      filePath,
				Error:     mapped.Message,
				ErrorCode: mapped.BatchCode,
			}
		}
		return h.readSingleFile(ctx, filePath, input.Encoding)
	}, func(index int, result FileReadResult) bool {
		results[index] = result
		return true
	})

	var successCount, errorCount int
	var errorSummary []string
	for _, r := range results {
		if r.Error != "" {
			errorCount++
			errorSummary = append(errorSummary, fmt.Sprintf("%s: %s", r.Path, r.Error))
		} else {
			successCount++
		}
	}

	return &mcp.CallToolResult{}, ReadMultipleFilesOutput{
		Results:      results,
		SuccessCount: successCount,
		ErrorCount:   errorCount,
		Errors:       errorSummary,
	}, nil
}

// readSingleFile maps the shared text-document pipeline into a batch result.
func (h *Handler) readSingleFile(ctx context.Context, path, requestedEncoding string) FileReadResult {
	result := FileReadResult{Path: path}

	v := h.ValidatePath(path)
	if !v.Ok() {
		mapped := mapOperationError(v.Err, path)
		result.Error = mapped.Message
		result.ErrorCode = mapped.BatchCode
		return result
	}

	document, err := h.readTextDocument(ctx, v.Path, requestedEncoding)
	if err != nil {
		mapped := mapOperationError(err, v.Path)
		result.Error = mapped.Message
		result.ErrorCode = mapped.BatchCode
		return result
	}

	result.Content = document.Text
	result.HasBOM = document.BOM.HasBOM
	result.BOMType = document.BOM.Type
	if document.AutoDetected {
		result.DetectedEncoding = document.DetectedEncoding
		result.EncodingConfidence = document.EncodingConfidence
	}

	return result
}
