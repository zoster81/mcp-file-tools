package handler

import (
	"context"
	"fmt"
	"os"
	"strings"

	fileEncoding "github.com/dimitar-grigorov/mcp-file-tools/internal/encoding"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func canonicalBOMEncoding(name string) string {
	switch strings.ToLower(name) {
	case "utf-8", "utf8", "ascii":
		return "utf-8"
	case "utf-16-le", "utf16le", "utf-16le":
		return "utf-16-le"
	case "utf-16-be", "utf16be", "utf-16be":
		return "utf-16-be"
	default:
		return strings.ToLower(name)
	}
}

func splitPreservedBOM(data []byte, encodingName string) ([]byte, []byte, error) {
	result, found := fileEncoding.DetectBOM(data)
	if !found {
		return data, nil, nil
	}
	if canonicalBOMEncoding(encodingName) != result.Charset {
		return nil, nil, fmt.Errorf("file BOM indicates %s but selected encoding is %s", result.Charset, encodingName)
	}

	bomSize := fileEncoding.BOMSize(result.Charset)
	bom := append([]byte(nil), data[:bomSize]...)
	return data[bomSize:], bom, nil
}

func isUTF16Encoding(name string) bool {
	switch canonicalBOMEncoding(name) {
	case "utf-16-le", "utf-16-be":
		return true
	default:
		return false
	}
}

func detectUTF16LineEndings(data []byte, littleEndian bool) (LineEndingInfo, error) {
	if len(data)%2 != 0 {
		return LineEndingInfo{}, fmt.Errorf("invalid UTF-16 byte length: %d", len(data))
	}

	info := LineEndingInfo{}
	readUnit := func(i int) uint16 {
		if littleEndian {
			return uint16(data[i]) | uint16(data[i+1])<<8
		}
		return uint16(data[i])<<8 | uint16(data[i+1])
	}

	for i := 0; i < len(data); i += 2 {
		unit := readUnit(i)
		if unit == '\r' && i+3 < len(data) && readUnit(i+2) == '\n' {
			info.CRLFCount++
			i += 2
		} else if unit == '\n' {
			info.LFCount++
		}
	}
	info.Style = determineStyle(info.CRLFCount, info.LFCount)
	return info, nil
}

func convertASCIICompatibleLineEndings(data []byte, targetStyle string) ([]byte, LineEndingInfo) {
	info := DetectLineEndings(data)
	if info.Style == targetStyle || info.Style == LineEndingNone {
		return data, info
	}

	capacity := len(data)
	if targetStyle == LineEndingCRLF {
		capacity += info.LFCount
	} else {
		capacity -= info.CRLFCount
	}
	converted := make([]byte, 0, capacity)
	for i := 0; i < len(data); i++ {
		if data[i] == '\r' && i+1 < len(data) && data[i+1] == '\n' {
			if targetStyle == LineEndingCRLF {
				converted = append(converted, '\r', '\n')
			} else {
				converted = append(converted, '\n')
			}
			i++
			continue
		}
		if data[i] == '\n' {
			if targetStyle == LineEndingCRLF {
				converted = append(converted, '\r', '\n')
			} else {
				converted = append(converted, '\n')
			}
			continue
		}
		converted = append(converted, data[i])
	}
	return converted, info
}

func convertUTF16LineEndings(data []byte, targetStyle string, littleEndian bool) ([]byte, LineEndingInfo, error) {
	info, err := detectUTF16LineEndings(data, littleEndian)
	if err != nil {
		return nil, LineEndingInfo{}, err
	}
	if info.Style == targetStyle || info.Style == LineEndingNone {
		return data, info, nil
	}

	cr := []byte{0x00, 0x0D}
	lf := []byte{0x00, 0x0A}
	if littleEndian {
		cr = []byte{0x0D, 0x00}
		lf = []byte{0x0A, 0x00}
	}

	capacity := len(data)
	if targetStyle == LineEndingCRLF {
		capacity += info.LFCount * 2
	} else {
		capacity -= info.CRLFCount * 2
	}
	converted := make([]byte, 0, capacity)
	for i := 0; i < len(data); i += 2 {
		unit := data[i : i+2]
		isCR := unit[0] == cr[0] && unit[1] == cr[1]
		isLF := unit[0] == lf[0] && unit[1] == lf[1]
		if isCR && i+3 < len(data) && data[i+2] == lf[0] && data[i+3] == lf[1] {
			if targetStyle == LineEndingCRLF {
				converted = append(converted, cr...)
			}
			converted = append(converted, lf...)
			i += 2
			continue
		}
		if isLF {
			if targetStyle == LineEndingCRLF {
				converted = append(converted, cr...)
			}
			converted = append(converted, lf...)
			continue
		}
		converted = append(converted, unit...)
	}
	return converted, info, nil
}

// HandleChangeLineEndings converts line endings while preserving encoding and BOM state.
func (h *Handler) HandleChangeLineEndings(ctx context.Context, req *mcp.CallToolRequest, input ChangeLineEndingsInput) (*mcp.CallToolResult, ChangeLineEndingsOutput, error) {
	v := h.ValidatePath(input.Path)
	if !v.Ok() {
		return v.Result, ChangeLineEndingsOutput{}, nil
	}

	style := strings.ToLower(input.Style)
	if style != LineEndingLF && style != LineEndingCRLF {
		return errorResult("style must be \"lf\" or \"crlf\""), ChangeLineEndingsOutput{}, nil
	}

	encResult, err := h.resolveEncoding(input.Encoding, v.Path)
	if err != nil {
		return errorResult(err.Error()), ChangeLineEndingsOutput{}, nil
	}

	data, err := os.ReadFile(v.Path)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to read file: %v", err)), ChangeLineEndingsOutput{}, nil
	}

	payload, bom, err := splitPreservedBOM(data, encResult.name)
	if err != nil {
		return errorResult(err.Error()), ChangeLineEndingsOutput{}, nil
	}

	var converted []byte
	var info LineEndingInfo
	if isUTF16Encoding(encResult.name) {
		converted, info, err = convertUTF16LineEndings(payload, style, canonicalBOMEncoding(encResult.name) == "utf-16-le")
		if err != nil {
			return errorResult(fmt.Sprintf("failed to process %s file: %v", encResult.name, err)), ChangeLineEndingsOutput{}, nil
		}
	} else {
		converted, info = convertASCIICompatibleLineEndings(payload, style)
	}

	originalStyle := info.Style
	if originalStyle == style || originalStyle == LineEndingNone {
		return &mcp.CallToolResult{}, ChangeLineEndingsOutput{
			Message:       fmt.Sprintf("File already uses %s line endings, no changes needed", style),
			OriginalStyle: originalStyle,
			NewStyle:      style,
			LinesChanged:  0,
		}, nil
	}

	linesChanged := info.LFCount
	if style == LineEndingLF {
		linesChanged = info.CRLFCount
	}

	outputData := make([]byte, 0, len(bom)+len(converted))
	outputData = append(outputData, bom...)
	outputData = append(outputData, converted...)

	mode := getFileMode(v.Path)
	if err := atomicWriteFile(v.Path, outputData, mode); err != nil {
		return errorResult(fmt.Sprintf("failed to write file: %v", err)), ChangeLineEndingsOutput{}, nil
	}

	return &mcp.CallToolResult{}, ChangeLineEndingsOutput{
		Message:       fmt.Sprintf("Converted %s from %s to %s (%d lines changed)", input.Path, originalStyle, style, linesChanged),
		OriginalStyle: originalStyle,
		NewStyle:      style,
		LinesChanged:  linesChanged,
	}, nil
}
