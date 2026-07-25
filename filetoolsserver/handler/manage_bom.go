package handler

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zoster81/mcp-file-tools/internal/encoding"
	"github.com/zoster81/mcp-file-tools/internal/filesystem"
)

// validBOMEncodings lists encodings that have a defined BOM.
var validBOMEncodings = map[string]bool{
	"utf-8":     true,
	"utf-16-le": true,
	"utf-16-be": true,
	"utf-32-le": true,
	"utf-32-be": true,
}

// HandleManageBom detects, strips, or adds a Unicode BOM (Byte Order Mark).
func (h *Handler) HandleManageBom(ctx context.Context, req *mcp.CallToolRequest, input ManageBomInput) (*mcp.CallToolResult, ManageBomOutput, error) {
	v := h.ValidatePath(input.Path)
	if !v.Ok() {
		return v.Result, ManageBomOutput{}, nil
	}

	action := strings.ToLower(input.Action)
	if action != "detect" && action != "strip" && action != "add" {
		return errorResult(`action must be "detect", "strip", or "add"`), ManageBomOutput{}, nil
	}

	switch action {
	case "detect":
		return h.bomDetect(v.Path)
	case "strip":
		return h.bomStrip(ctx, input.Path, v.Path)
	case "add":
		enc := strings.ToLower(input.Encoding)
		if enc == "" {
			return errorResult(`encoding is required for "add" action (utf-8, utf-16-le, utf-16-be, utf-32-le, utf-32-be)`), ManageBomOutput{}, nil
		}
		if !validBOMEncodings[enc] {
			return errorResult(fmt.Sprintf("unsupported BOM encoding %q — valid: utf-8, utf-16-le, utf-16-be, utf-32-le, utf-32-be", enc)), ManageBomOutput{}, nil
		}
		return h.bomAdd(ctx, input.Path, v.Path, enc)
	}
	// unreachable
	return errorResult("unexpected action"), ManageBomOutput{}, nil
}

// bomDetect reads the first 4 bytes and checks for a BOM.
func (h *Handler) bomDetect(path string) (*mcp.CallToolResult, ManageBomOutput, error) {
	data, err := readFileHead(path, 4)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to read file: %v", err)), ManageBomOutput{}, nil
	}

	result, found := encoding.DetectBOM(data)
	if !found {
		return &mcp.CallToolResult{}, ManageBomOutput{
			Message: "No BOM detected",
			HasBOM:  false,
			Changed: false,
		}, nil
	}

	return &mcp.CallToolResult{}, ManageBomOutput{
		Message:  fmt.Sprintf("BOM detected: %s (%d bytes)", result.Charset, encoding.BOMSize(result.Charset)),
		HasBOM:   true,
		BOMType:  result.Charset,
		BOMBytes: encoding.BOMSize(result.Charset),
		Changed:  false,
	}, nil
}

// bomStrip removes the BOM if present, otherwise returns a no-op result.
func (h *Handler) bomStrip(ctx context.Context, requestedPath, path string) (*mcp.CallToolResult, ManageBomOutput, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to read file: %v", err)), ManageBomOutput{}, nil
	}

	result, found := encoding.DetectBOM(data)
	if !found {
		return &mcp.CallToolResult{}, ManageBomOutput{
			Message: "No BOM found, nothing to strip",
			HasBOM:  false,
			Changed: false,
		}, nil
	}

	bomSize := encoding.BOMSize(result.Charset)
	stripped := data[bomSize:]
	expected, err := filesystem.CaptureSnapshotWithData(path, data)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to snapshot file: %v", err)), ManageBomOutput{}, nil
	}
	select {
	case <-ctx.Done():
		return errorResult(ctx.Err().Error()), ManageBomOutput{}, nil
	default:
	}
	commit := h.ValidatePath(requestedPath)
	if !commit.Ok() {
		return commit.Result, ManageBomOutput{}, nil
	}
	if commit.Path != path {
		return errorResult("path changed while preparing BOM removal"), ManageBomOutput{}, nil
	}
	if err := filesystem.ReplaceFile(commit.Path, stripped, filesystem.ReplaceOptions{
		Mode:     expected.Mode.Perm(),
		Expected: &expected,
	}); err != nil {
		return errorResult(fmt.Sprintf("failed to write file: %v", err)), ManageBomOutput{}, nil
	}

	return &mcp.CallToolResult{}, ManageBomOutput{
		Message:  fmt.Sprintf("Stripped %s BOM (%d bytes) from %s", result.Charset, bomSize, path),
		HasBOM:   false,
		BOMType:  result.Charset,
		BOMBytes: bomSize,
		Changed:  true,
	}, nil
}

// bomAdd prepends a BOM for the given encoding. Fails if a BOM already exists.
func (h *Handler) bomAdd(ctx context.Context, requestedPath, path, enc string) (*mcp.CallToolResult, ManageBomOutput, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to read file: %v", err)), ManageBomOutput{}, nil
	}

	// Check if file already has a BOM
	if existingResult, found := encoding.DetectBOM(data); found {
		return errorResult(fmt.Sprintf("file already has a %s BOM — strip it first if you want to change it", existingResult.Charset)), ManageBomOutput{}, nil
	}

	bomBytes := encoding.BOMBytesFor(enc)
	// Prepend BOM
	withBOM := make([]byte, len(bomBytes)+len(data))
	copy(withBOM, bomBytes)
	copy(withBOM[len(bomBytes):], data)
	expected, err := filesystem.CaptureSnapshotWithData(path, data)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to snapshot file: %v", err)), ManageBomOutput{}, nil
	}
	select {
	case <-ctx.Done():
		return errorResult(ctx.Err().Error()), ManageBomOutput{}, nil
	default:
	}
	commit := h.ValidatePath(requestedPath)
	if !commit.Ok() {
		return commit.Result, ManageBomOutput{}, nil
	}
	if commit.Path != path {
		return errorResult("path changed while preparing BOM addition"), ManageBomOutput{}, nil
	}
	if err := filesystem.ReplaceFile(commit.Path, withBOM, filesystem.ReplaceOptions{
		Mode:     expected.Mode.Perm(),
		Expected: &expected,
	}); err != nil {
		return errorResult(fmt.Sprintf("failed to write file: %v", err)), ManageBomOutput{}, nil
	}

	return &mcp.CallToolResult{}, ManageBomOutput{
		Message:  fmt.Sprintf("Added %s BOM (%d bytes) to %s", enc, len(bomBytes), path),
		HasBOM:   true,
		BOMType:  enc,
		BOMBytes: len(bomBytes),
		Changed:  true,
	}, nil
}

// readFileHead reads up to n bytes from the beginning of a file.
func readFileHead(path string, n int) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	buf := make([]byte, n)
	read, err := f.Read(buf)
	if err != nil && err != io.EOF {
		return nil, err
	}
	return buf[:read], nil
}
