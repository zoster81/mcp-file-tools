package handler

import (
	"context"
	"fmt"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zoster81/scripthold/internal/filesystem"
)

// HandleCopyFile copies a file to a new location.
func (h *Handler) HandleCopyFile(ctx context.Context, req *mcp.CallToolRequest, input CopyFileInput) (*mcp.CallToolResult, CopyFileOutput, error) {
	src, dst := h.ValidateSourceDest(input.Source, input.Destination)
	if !src.Ok() {
		return src.Result, CopyFileOutput{}, nil
	}
	if !dst.Ok() {
		return dst.Result, CopyFileOutput{}, nil
	}

	srcInfo, err := os.Stat(src.Path)
	if os.IsNotExist(err) {
		return errorResult(fmt.Sprintf("source does not exist: %s", input.Source)), CopyFileOutput{}, nil
	}
	if err != nil {
		return errorResult(fmt.Sprintf("failed to access source: %v", err)), CopyFileOutput{}, nil
	}

	if srcInfo.IsDir() {
		return errorResult("source is a directory, only files can be copied"), CopyFileOutput{}, nil
	}

	if _, err := os.Stat(dst.Path); err == nil {
		return errorResult(fmt.Sprintf("destination already exists: %s", input.Destination)), CopyFileOutput{}, nil
	}

	select {
	case <-ctx.Done():
		return errorResult(ctx.Err().Error()), CopyFileOutput{}, nil
	default:
	}
	commitSrc, commitDst := h.ValidateSourceDest(input.Source, input.Destination)
	if !commitSrc.Ok() {
		return commitSrc.Result, CopyFileOutput{}, nil
	}
	if !commitDst.Ok() {
		return commitDst.Result, CopyFileOutput{}, nil
	}
	if commitSrc.Path != src.Path || commitDst.Path != dst.Path {
		return errorResult("source or destination changed while preparing copy"), CopyFileOutput{}, nil
	}

	// Copy through the shared durable no-replace mutation layer.
	if err := filesystem.CopyFile(commitSrc.Path, commitDst.Path); err != nil {
		return errorResult(fmt.Sprintf("failed to copy file: %v", err)), CopyFileOutput{}, nil
	}

	message := fmt.Sprintf("Successfully copied %s to %s", input.Source, input.Destination)
	return &mcp.CallToolResult{}, CopyFileOutput{Message: message}, nil
}
