package handler

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zoster81/scripthold/internal/filesystem"
	"github.com/zoster81/scripthold/internal/textstream"
)

func isUTF16Encoding(name string) bool {
	switch canonicalBOMEncoding(name) {
	case "utf-16-le", "utf-16-be":
		return true
	default:
		return false
	}
}

// HandleChangeLineEndings converts line endings through a bounded raw-byte
// pipeline while preserving encoding, BOM state, and every unrelated byte.
func (h *Handler) HandleChangeLineEndings(ctx context.Context, req *mcp.CallToolRequest, input ChangeLineEndingsInput) (*mcp.CallToolResult, ChangeLineEndingsOutput, error) {
	validated := h.ValidatePath(input.Path)
	if !validated.Ok() {
		return validated.Result, ChangeLineEndingsOutput{}, nil
	}

	style := strings.ToLower(input.Style)
	if style != LineEndingLF && style != LineEndingCRLF {
		return errorResult("style must be \"lf\" or \"crlf\""), ChangeLineEndingsOutput{}, nil
	}

	stream, err := h.openDecodedTextStream(ctx, validated.Path, input.Encoding)
	if err != nil {
		return errorResultFromError(err), ChangeLineEndingsOutput{}, nil
	}
	defer stream.Close()

	rawSource := textstream.WithContext(ctx, stream.session)
	var transformed *textstream.LineEndingReader
	if isUTF16Encoding(stream.Charset) {
		transformed, err = textstream.NewUTF16LineEndingReader(
			rawSource,
			style,
			canonicalBOMEncoding(stream.Charset) == "utf-16-le",
		)
	} else {
		transformed, err = textstream.NewByteLineEndingReader(rawSource, style)
	}
	if err != nil {
		return errorResult(err.Error()), ChangeLineEndingsOutput{}, nil
	}

	outputReader := io.MultiReader(bytes.NewReader(stream.BOM.Bytes), transformed)
	staged, err := filesystem.StageReplacement(validated.Path, outputReader, stream.Mode.Perm(), nil)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to prepare line-ending conversion: %v", err)), ChangeLineEndingsOutput{}, nil
	}
	defer staged.Cleanup()

	snapshot, err := stream.Finish()
	if err != nil {
		return errorResultFromError(err), ChangeLineEndingsOutput{}, nil
	}
	if err := stream.Close(); err != nil {
		return errorResult(fmt.Sprintf("failed to close source file before commit: %v", err)), ChangeLineEndingsOutput{}, nil
	}

	stats := transformed.Stats()
	originalStyle := determineStyle(stats.CRLFCount, stats.LFCount)
	linesChanged := stats.LFCount
	if style == LineEndingLF {
		linesChanged = stats.CRLFCount
	}
	if originalStyle == style || originalStyle == LineEndingNone {
		return &mcp.CallToolResult{}, ChangeLineEndingsOutput{
			Message:       fmt.Sprintf("File already uses %s line endings, no changes needed", style),
			OriginalStyle: originalStyle,
			NewStyle:      style,
			LinesChanged:  0,
		}, nil
	}

	commit := h.ValidatePath(input.Path)
	if !commit.Ok() {
		return commit.Result, ChangeLineEndingsOutput{}, nil
	}
	if commit.Path != validated.Path {
		return errorResult("path changed while preparing line-ending conversion"), ChangeLineEndingsOutput{}, nil
	}

	changed, err := staged.Commit(filesystem.ReplaceOptions{
		Expected:      &snapshot,
		SkipIdentical: true,
	})
	if err != nil {
		return errorResult(fmt.Sprintf("failed to write file: %v", err)), ChangeLineEndingsOutput{}, nil
	}
	if !changed {
		linesChanged = 0
	}

	return &mcp.CallToolResult{}, ChangeLineEndingsOutput{
		Message:       fmt.Sprintf("Converted %s from %s to %s (%d lines changed)", input.Path, originalStyle, style, linesChanged),
		OriginalStyle: originalStyle,
		NewStyle:      style,
		LinesChanged:  linesChanged,
	}, nil
}
