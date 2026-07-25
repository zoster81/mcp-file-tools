package handler

import (
	"context"
	"fmt"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zoster81/mcp-file-tools/internal/filesystem"
)

// HandleMoveFile moves or renames a file or directory.
func (h *Handler) HandleMoveFile(ctx context.Context, req *mcp.CallToolRequest, input MoveFileInput) (*mcp.CallToolResult, MoveFileOutput, error) {
	src, dst := h.ValidateSourceDest(input.Source, input.Destination)
	if !src.Ok() {
		return src.Result, MoveFileOutput{}, nil
	}
	if !dst.Ok() {
		return dst.Result, MoveFileOutput{}, nil
	}

	// Check if source exists.
	if _, err := os.Stat(src.Path); err != nil {
		if os.IsNotExist(err) {
			return errorResult(fmt.Sprintf("source does not exist: %s", input.Source)), MoveFileOutput{}, nil
		}
		return errorResult(fmt.Sprintf("failed to access source: %v", err)), MoveFileOutput{}, nil
	}

	// Check if destination already exists.
	if _, err := os.Lstat(dst.Path); err == nil {
		return errorResult(fmt.Sprintf("destination already exists: %s", input.Destination)), MoveFileOutput{}, nil
	} else if !os.IsNotExist(err) {
		return errorResult(fmt.Sprintf("failed to inspect destination: %v", err)), MoveFileOutput{}, nil
	}

	select {
	case <-ctx.Done():
		return errorResult(ctx.Err().Error()), MoveFileOutput{}, nil
	default:
	}
	commitSrc, commitDst := h.ValidateSourceDest(input.Source, input.Destination)
	if !commitSrc.Ok() {
		return commitSrc.Result, MoveFileOutput{}, nil
	}
	if !commitDst.Ok() {
		return commitDst.Result, MoveFileOutput{}, nil
	}
	if commitSrc.Path != src.Path || commitDst.Path != dst.Path {
		return errorResult("source or destination changed while preparing move"), MoveFileOutput{}, nil
	}

	if err := filesystem.MoveNoReplace(commitSrc.Path, commitDst.Path); err != nil {
		return errorResult(fmt.Sprintf("failed to move file: %v", err)), MoveFileOutput{}, nil
	}

	message := fmt.Sprintf("Successfully moved %s to %s", input.Source, input.Destination)
	return &mcp.CallToolResult{}, MoveFileOutput{Message: message}, nil
}
