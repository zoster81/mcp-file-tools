package handler

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zoster81/mcp-file-tools/internal/filesystem"
	"github.com/zoster81/mcp-file-tools/internal/operation"
	"github.com/zoster81/mcp-file-tools/internal/textstream"
)

// LineEndingStyle constants for line ending types.
const (
	LineEndingCRLF  = "crlf"
	LineEndingLF    = "lf"
	LineEndingMixed = "mixed"
	LineEndingNone  = "none"
)

// LineEndingInfo holds detected line ending information.
type LineEndingInfo struct {
	Style     string // "crlf", "lf", "mixed", or "none"
	CRLFCount int
	LFCount   int // LF not preceded by CR
}

// DetectLineEndings analyzes data and returns line ending information.
// Works on byte slice for in-memory data.
func DetectLineEndings(data []byte) LineEndingInfo {
	info := LineEndingInfo{}

	for i := 0; i < len(data); i++ {
		if data[i] == '\r' && i+1 < len(data) && data[i+1] == '\n' {
			info.CRLFCount++
			i++ // skip the \n
		} else if data[i] == '\n' {
			info.LFCount++
		}
	}

	info.Style = determineStyle(info.CRLFCount, info.LFCount)
	return info
}

// determineStyle returns the line ending style based on counts.
func determineStyle(crlfCount, lfCount int) string {
	switch {
	case crlfCount == 0 && lfCount == 0:
		return LineEndingNone
	case crlfCount > 0 && lfCount == 0:
		return LineEndingCRLF
	case crlfCount == 0 && lfCount > 0:
		return LineEndingLF
	default:
		return LineEndingMixed
	}
}

// ConvertLineEndings converts text to the specified line ending style.
func ConvertLineEndings(text string, targetStyle string) string {
	hasCRLF := strings.Contains(text, "\r\n")

	if targetStyle == LineEndingCRLF {
		if !hasCRLF {
			// Only LF present, single pass: LF -> CRLF
			return strings.ReplaceAll(text, "\n", "\r\n")
		}
		// Has CRLF (might be mixed), normalize then convert
		normalized := strings.ReplaceAll(text, "\r\n", "\n")
		return strings.ReplaceAll(normalized, "\n", "\r\n")
	}

	// Target is LF (or other non-CRLF style)
	if !hasCRLF {
		return text // Already no CRLF
	}
	return strings.ReplaceAll(text, "\r\n", "\n")
}

type lineEndingScan struct {
	TotalLines int
	CRLFCount  int
	LFCount    int
	Collected  []int
}

func scanLineEndings(ctx context.Context, reader io.Reader, collectEnding string, maxCollectedBytes int64) (lineEndingScan, error) {
	scan := lineEndingScan{}
	totalLines, err := textstream.ScanLines(ctx, reader, textstream.DefaultMaxLineBytes, func(line textstream.Line) error {
		ending := string(line.Ending)
		switch ending {
		case "\r\n":
			scan.CRLFCount++
		case "\n":
			scan.LFCount++
		}
		if collectEnding != "" && ending == collectEnding {
			if maxCollectedBytes > 0 && int64(len(scan.Collected)+1)*8 > maxCollectedBytes {
				return operation.Wrap(
					operation.KindLimit,
					"detect_line_endings",
					"",
					fmt.Errorf("inconsistent line list exceeds the %d-byte output budget", maxCollectedBytes),
				)
			}
			scan.Collected = append(scan.Collected, line.Number)
		}
		return nil
	})
	scan.TotalLines = totalLines
	return scan, err
}

// HandleDetectLineEndings detects line ending style and returns inconsistent line numbers.
func (h *Handler) HandleDetectLineEndings(ctx context.Context, req *mcp.CallToolRequest, input DetectLineEndingsInput) (*mcp.CallToolResult, DetectLineEndingsOutput, error) {
	v := h.ValidatePath(input.Path)
	if !v.Ok() {
		return v.Result, DetectLineEndingsOutput{}, nil
	}

	stream, err := h.openDecodedTextStream(ctx, v.Path, input.Encoding)
	if err != nil {
		return errorResultFromError(err), DetectLineEndingsOutput{}, nil
	}
	defer stream.Close()

	first, err := scanLineEndings(ctx, stream.Reader, "", 0)
	if err != nil {
		return errorResultFromError(err), DetectLineEndingsOutput{}, nil
	}
	firstSnapshot, err := stream.Finish()
	if err != nil {
		return errorResultFromError(err), DetectLineEndingsOutput{}, nil
	}
	if err := stream.Close(); err != nil {
		return errorResult(fmt.Sprintf("failed to close file after line-ending scan: %v", err)), DetectLineEndingsOutput{}, nil
	}

	style := determineStyle(first.CRLFCount, first.LFCount)
	var inconsistentLines []int
	if style == LineEndingMixed {
		minorityEnding := "\r\n"
		if first.CRLFCount >= first.LFCount {
			minorityEnding = "\n"
		}

		secondStream, err := h.openDecodedTextStream(ctx, v.Path, stream.Charset)
		if err != nil {
			return errorResultFromError(err), DetectLineEndingsOutput{}, nil
		}
		defer secondStream.Close()
		second, err := scanLineEndings(ctx, secondStream.Reader, minorityEnding, h.memoryBudget())
		if err != nil {
			return errorResultFromError(err), DetectLineEndingsOutput{}, nil
		}
		secondSnapshot, err := secondStream.Finish()
		if err != nil {
			return errorResultFromError(err), DetectLineEndingsOutput{}, nil
		}
		if err := secondStream.Close(); err != nil {
			return errorResult(fmt.Sprintf("failed to close file after line-ending rescan: %v", err)), DetectLineEndingsOutput{}, nil
		}
		if !firstSnapshot.Equal(secondSnapshot) || first.TotalLines != second.TotalLines {
			return errorResultFromError(fmt.Errorf("%w: file changed between line-ending scans", filesystem.ErrConcurrentModification)), DetectLineEndingsOutput{}, nil
		}
		inconsistentLines = second.Collected
	}

	return &mcp.CallToolResult{}, DetectLineEndingsOutput{
		Style:             style,
		TotalLines:        first.TotalLines,
		InconsistentLines: inconsistentLines,
	}, nil
}
