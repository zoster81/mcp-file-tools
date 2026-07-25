package handler

import (
	"context"
	"fmt"
	"os"

	fileEncoding "github.com/dimitar-grigorov/mcp-file-tools/internal/encoding"
	"github.com/dimitar-grigorov/mcp-file-tools/internal/filesystem"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func (h *Handler) HandleWriteFile(ctx context.Context, req *mcp.CallToolRequest, input WriteFileInput) (*mcp.CallToolResult, WriteFileOutput, error) {
	v := h.ValidatePath(input.Path)
	if !v.Ok() {
		return v.Result, WriteFileOutput{}, nil
	}
	expected, err := filesystem.CaptureSnapshot(v.Path)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to inspect target: %v", err)), WriteFileOutput{}, nil
	}

	policy, err := parseBOMPolicy(input.BOM, bomAuto)
	if err != nil {
		return errorResultFromError(err), WriteFileOutput{}, nil
	}

	// Resolve encoding: explicit > preserve existing > configured default.
	encodingName, err := h.resolveWriteEncoding(input.Encoding, v.Path)
	if err != nil {
		return errorResultFromError(err), WriteFileOutput{}, nil
	}

	document := textDocument{Charset: encodingName}
	if policy == bomPreserve {
		head, readErr := readFileHead(v.Path, 4)
		switch {
		case readErr == nil:
			if detected, found := fileEncoding.DetectBOM(head); found {
				document.BOM = bomInfo{HasBOM: true, Type: detected.Charset}
			}
		case !os.IsNotExist(readErr):
			return errorResult(fmt.Sprintf("failed to inspect existing BOM: %v", readErr)), WriteFileOutput{}, nil
		}
	}

	contentToWrite, err := encodeTextDocument(document, input.Content, policy)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to encode content: %v", err)), WriteFileOutput{}, nil
	}

	select {
	case <-ctx.Done():
		return errorResult(ctx.Err().Error()), WriteFileOutput{}, nil
	default:
	}

	commit := h.ValidatePath(input.Path)
	if !commit.Ok() {
		return commit.Result, WriteFileOutput{}, nil
	}
	if commit.Path != v.Path {
		return errorResult("path changed while preparing write"), WriteFileOutput{}, nil
	}

	mode := getFileMode(commit.Path)
	if err := filesystem.ReplaceFile(commit.Path, contentToWrite, filesystem.ReplaceOptions{
		Mode:     mode,
		Expected: &expected,
	}); err != nil {
		return errorResult(fmt.Sprintf("failed to write file: %v", err)), WriteFileOutput{}, nil
	}

	hasBOM, bomType := outputBOMMetadata(contentToWrite)
	message := fmt.Sprintf("Successfully wrote %d bytes to %s (encoding: %s, BOM: %s)", len(contentToWrite), input.Path, encodingName, policy)
	return &mcp.CallToolResult{}, WriteFileOutput{
		Message:   message,
		Encoding:  encodingName,
		BOMPolicy: string(policy),
		HasBOM:    hasBOM,
		BOMType:   bomType,
	}, nil
}
