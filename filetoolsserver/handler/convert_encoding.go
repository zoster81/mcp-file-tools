package handler

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	fileEncoding "github.com/dimitar-grigorov/mcp-file-tools/internal/encoding"
	"github.com/dimitar-grigorov/mcp-file-tools/internal/filesystem"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// HandleConvertEncoding converts a file from one encoding to another.
func (h *Handler) HandleConvertEncoding(ctx context.Context, req *mcp.CallToolRequest, input ConvertEncodingInput) (*mcp.CallToolResult, ConvertEncodingOutput, error) {
	if input.To == "" {
		return errorResult("target encoding (to) is required"), ConvertEncodingOutput{}, nil
	}

	v := h.ValidatePath(input.Path)
	if !v.Ok() {
		return v.Result, ConvertEncodingOutput{}, nil
	}

	targetEncodingName := strings.ToLower(input.To)
	if _, ok := fileEncoding.Get(targetEncodingName); !ok {
		return errorResult(fmt.Sprintf("unsupported target encoding: %s. Use list_encodings to see available encodings.", input.To)), ConvertEncodingOutput{}, nil
	}

	policy, err := parseBOMPolicy(input.BOM, bomAuto)
	if err != nil {
		return errorResult(err.Error()), ConvertEncodingOutput{}, nil
	}

	document, originalData, err := h.readTextDocumentWithData(ctx, v.Path, input.From)
	if err != nil {
		return errorResult(err.Error()), ConvertEncodingOutput{}, nil
	}
	sourceEncodingName := document.Charset

	targetDocument := document
	targetDocument.Charset = targetEncodingName
	targetData, err := encodeTextDocument(targetDocument, document.Text, policy)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to encode to %s: %v", targetEncodingName, err)), ConvertEncodingOutput{}, nil
	}

	select {
	case <-ctx.Done():
		return errorResult(ctx.Err().Error()), ConvertEncodingOutput{}, nil
	default:
	}

	hasBOM, bomType := outputBOMMetadata(targetData)
	if bytes.Equal(originalData, targetData) {
		return &mcp.CallToolResult{}, ConvertEncodingOutput{
			Message:        fmt.Sprintf("No conversion needed for %s: target bytes are unchanged", input.Path),
			SourceEncoding: sourceEncodingName,
			TargetEncoding: targetEncodingName,
			BOMPolicy:      string(policy),
			HasBOM:         hasBOM,
			BOMType:        bomType,
			Changed:        false,
		}, nil
	}

	var backupPath string
	if input.Backup {
		backup := h.ValidatePath(v.Path + ".bak")
		if !backup.Ok() {
			return backup.Result, ConvertEncodingOutput{}, nil
		}
		backupPath = backup.Path
	}

	select {
	case <-ctx.Done():
		return errorResult(ctx.Err().Error()), ConvertEncodingOutput{}, nil
	default:
	}

	commit := h.ValidatePath(input.Path)
	if !commit.Ok() {
		return commit.Result, ConvertEncodingOutput{}, nil
	}
	if commit.Path != v.Path {
		return errorResult("path changed while preparing encoding conversion"), ConvertEncodingOutput{}, nil
	}
	if backupPath != "" {
		backup := h.ValidatePath(backupPath)
		if !backup.Ok() {
			return backup.Result, ConvertEncodingOutput{}, nil
		}
		if backup.Path != backupPath {
			return errorResult("backup path changed while preparing encoding conversion"), ConvertEncodingOutput{}, nil
		}
	}

	if err := filesystem.ReplaceFile(commit.Path, targetData, filesystem.ReplaceOptions{
		Mode:       document.Mode,
		Expected:   &document.Snapshot,
		BackupPath: backupPath,
	}); err != nil {
		return errorResult(fmt.Sprintf("failed to write converted file: %v", err)), ConvertEncodingOutput{}, nil
	}

	message := fmt.Sprintf("Successfully converted %s from %s to %s (BOM: %s)", input.Path, sourceEncodingName, targetEncodingName, policy)
	if backupPath != "" {
		message += fmt.Sprintf(" (backup: %s)", backupPath)
	}

	return &mcp.CallToolResult{}, ConvertEncodingOutput{
		Message:        message,
		SourceEncoding: sourceEncodingName,
		TargetEncoding: targetEncodingName,
		BackupPath:     backupPath,
		BOMPolicy:      string(policy),
		HasBOM:         hasBOM,
		BOMType:        bomType,
		Changed:        true,
	}, nil
}
