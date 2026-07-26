package handler

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	fileEncoding "github.com/zoster81/mcp-file-tools/internal/encoding"
	"github.com/zoster81/mcp-file-tools/internal/filesystem"
	"github.com/zoster81/mcp-file-tools/internal/operation"
)

type encodingOutputReader struct {
	reader io.Reader
	target string
}

func (reader *encodingOutputReader) Read(buffer []byte) (int, error) {
	read, err := reader.reader.Read(buffer)
	if err != nil && err != io.EOF {
		err = operation.Wrap(
			operation.KindEncodingOutput,
			"encode_stream",
			"",
			fmt.Errorf("%w: failed to encode content to %s: %v", ErrEncodingEncode, reader.target, err),
		)
	}
	return read, err
}

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
		return errorResultFromError(err), ConvertEncodingOutput{}, nil
	}

	stream, err := h.openDecodedTextStream(ctx, v.Path, input.From)
	if err != nil {
		return errorResultFromError(err), ConvertEncodingOutput{}, nil
	}
	defer stream.Close()
	sourceEncodingName := stream.Charset

	targetDocument := textDocument{Charset: targetEncodingName, BOM: stream.BOM}
	targetBOM, err := documentBOMBytes(targetDocument, policy)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to encode to %s: %v", targetEncodingName, err)), ConvertEncodingOutput{}, nil
	}
	encoded, err := fileEncoding.NewEncoderReader(stream.Reader, targetEncodingName)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to encode to %s: %v", targetEncodingName, err)), ConvertEncodingOutput{}, nil
	}
	outputReader := io.MultiReader(bytes.NewReader(targetBOM), &encodingOutputReader{reader: encoded, target: targetEncodingName})
	staged, err := filesystem.StageReplacement(v.Path, outputReader, stream.Mode.Perm(), nil)
	if err != nil {
		if operation.KindOf(err) == operation.KindEncodingOutput {
			return errorResult(fmt.Sprintf("failed to encode to %s: %v", targetEncodingName, err)), ConvertEncodingOutput{}, nil
		}
		return errorResult(fmt.Sprintf("failed to prepare converted file: %v", err)), ConvertEncodingOutput{}, nil
	}
	defer staged.Cleanup()

	snapshot, err := stream.Finish()
	if err != nil {
		return errorResultFromError(err), ConvertEncodingOutput{}, nil
	}
	if err := stream.Close(); err != nil {
		return errorResult(fmt.Sprintf("failed to close source file before commit: %v", err)), ConvertEncodingOutput{}, nil
	}

	hasBOM := len(targetBOM) > 0
	bomType := ""
	if hasBOM {
		bomType = canonicalBOMEncoding(targetEncodingName)
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

	changed, err := staged.Commit(filesystem.ReplaceOptions{
		Expected:      &snapshot,
		BackupPath:    backupPath,
		SkipIdentical: true,
	})
	if err != nil {
		return errorResult(fmt.Sprintf("failed to write converted file: %v", err)), ConvertEncodingOutput{}, nil
	}
	if !changed {
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
