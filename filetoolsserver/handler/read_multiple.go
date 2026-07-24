package handler

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"runtime"
	"sync"

	"github.com/dimitar-grigorov/mcp-file-tools/internal/security"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// HandleReadMultipleFiles reads multiple files concurrently.
// Individual file failures don't stop the operation - errors are reported per file.
func (h *Handler) HandleReadMultipleFiles(ctx context.Context, req *mcp.CallToolRequest, input ReadMultipleFilesInput) (*mcp.CallToolResult, ReadMultipleFilesOutput, error) {
	if len(input.Paths) == 0 {
		return errorResult("paths array is required and must contain at least one path"), ReadMultipleFilesOutput{}, nil
	}
	results := make([]FileReadResult, len(input.Paths))

	numWorkers := runtime.NumCPU()
	if numWorkers > len(input.Paths) {
		numWorkers = len(input.Paths)
	}

	type job struct {
		idx      int
		filePath string
	}
	jobs := make(chan job, len(input.Paths))
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				select {
				case <-ctx.Done():
					results[j.idx] = FileReadResult{
						Path:      j.filePath,
						Error:     "operation cancelled",
						ErrorCode: ErrCodeOperationFailed,
					}
				default:
					results[j.idx] = h.readSingleFile(ctx, j.filePath, input.Encoding)
				}
			}
		}()
	}
	for i, path := range input.Paths {
		jobs <- job{idx: i, filePath: path}
	}
	close(jobs)
	wg.Wait()

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
		result.Error = v.Err.Error()
		result.ErrorCode = classifyPathError(v.Err)
		return result
	}

	document, err := h.readTextDocument(ctx, v.Path, requestedEncoding)
	if err != nil {
		result.Error, result.ErrorCode = classifyReadError(err, v.Path)
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

// classifyPathError returns an error code based on the path validation error type.
func classifyPathError(err error) string {
	switch {
	case errors.Is(err, ErrPathRequired):
		return ErrCodeInvalidPath
	case errors.Is(err, security.ErrPathDenied):
		return ErrCodeAccessDenied
	case errors.Is(err, security.ErrSymlinkDenied):
		return ErrCodeSymlinkEscape
	case errors.Is(err, security.ErrNoAllowedDirs):
		return ErrCodeAccessDenied
	case errors.Is(err, security.ErrParentDirDenied):
		return ErrCodeAccessDenied
	default:
		return ErrCodeInvalidPath
	}
}

// classifyReadError returns a descriptive error message and code for file read errors.
func classifyReadError(err error, path string) (string, string) {
	switch {
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return "operation cancelled", ErrCodeOperationFailed
	case errors.Is(err, ErrEncodingUnsupported), errors.Is(err, ErrBOMEncodingConflict), errors.Is(err, ErrEncodingDecode):
		return err.Error(), ErrCodeEncoding
	case errors.Is(err, fs.ErrNotExist):
		return fmt.Sprintf("file not found: %s", path), ErrCodeNotFound
	case errors.Is(err, fs.ErrPermission):
		return fmt.Sprintf("permission denied: %s", path), ErrCodePermission
	default:
		return err.Error(), ErrCodeIO
	}
}
