package handler

import (
	"bufio"
	"context"
	"io"
	"os"
	"strings"

	fileEncoding "github.com/dimitar-grigorov/mcp-file-tools/internal/encoding"
	"github.com/modelcontextprotocol/go-sdk/mcp"
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

type detectedLineEnding struct {
	lineNum int
	isCRLF  bool
}

// analyzeLineEndings reads decoded UTF-8 text and reports its line-ending style.
func analyzeLineEndings(r io.Reader) (string, int, []int, error) {
	var lineEndings []detectedLineEnding
	br := bufio.NewReader(r)
	lineNum := 1
	prevWasCR := false

	for {
		b, err := br.ReadByte()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", 0, nil, err
		}

		if b == '\n' {
			lineEndings = append(lineEndings, detectedLineEnding{lineNum: lineNum, isCRLF: prevWasCR})
			lineNum++
		}
		prevWasCR = b == '\r'
	}

	crlfCount := 0
	lfCount := 0
	for _, ending := range lineEndings {
		if ending.isCRLF {
			crlfCount++
		} else {
			lfCount++
		}
	}

	style := determineStyle(crlfCount, lfCount)
	inconsistentLines := make([]int, 0)
	if style == LineEndingMixed {
		dominantIsCRLF := crlfCount >= lfCount
		for _, ending := range lineEndings {
			if ending.isCRLF != dominantIsCRLF {
				inconsistentLines = append(inconsistentLines, ending.lineNum)
			}
		}
	}

	return style, lineNum, inconsistentLines, nil
}

// HandleDetectLineEndings detects line ending style and returns inconsistent line numbers.
func (h *Handler) HandleDetectLineEndings(ctx context.Context, req *mcp.CallToolRequest, input DetectLineEndingsInput) (*mcp.CallToolResult, DetectLineEndingsOutput, error) {
	v := h.ValidatePath(input.Path)
	if !v.Ok() {
		return v.Result, DetectLineEndingsOutput{}, nil
	}

	encResult, err := h.resolveEncoding(input.Encoding, v.Path)
	if err != nil {
		return errorResult(err.Error()), DetectLineEndingsOutput{}, nil
	}

	f, err := os.Open(v.Path)
	if err != nil {
		return errorResult("failed to open file: " + err.Error()), DetectLineEndingsOutput{}, nil
	}
	defer f.Close()

	var reader io.Reader = f
	if !fileEncoding.IsUTF8(encResult.name) {
		reader = encResult.encoder.NewDecoder().Reader(f)
	}

	style, totalLines, inconsistentLines, err := analyzeLineEndings(reader)
	if err != nil {
		return errorResult("failed to decode or read file: " + err.Error()), DetectLineEndingsOutput{}, nil
	}

	return &mcp.CallToolResult{}, DetectLineEndingsOutput{
		Style:             style,
		TotalLines:        totalLines,
		InconsistentLines: inconsistentLines,
	}, nil
}
