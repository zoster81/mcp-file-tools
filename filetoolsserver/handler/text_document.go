package handler

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	fileEncoding "github.com/dimitar-grigorov/mcp-file-tools/internal/encoding"
)

type bomInfo struct {
	HasBOM bool
	Type   string
	Bytes  []byte
}

type textDocument struct {
	Text               string
	Charset            string
	DetectedEncoding   string
	EncodingConfidence int
	AutoDetected       bool
	BOM                bomInfo
	LineEndings        LineEndingInfo
	FileSizeBytes      int64
	Mode               os.FileMode
}

type bomPolicy string

const bomPreserve bomPolicy = "preserve"

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

func splitTransportBOM(data []byte, encodingName string) ([]byte, bomInfo, error) {
	result, found := fileEncoding.DetectBOM(data)
	if !found {
		return data, bomInfo{}, nil
	}
	if canonicalBOMEncoding(encodingName) != result.Charset {
		return nil, bomInfo{}, fmt.Errorf("%w: file BOM indicates %s but selected encoding is %s", ErrBOMEncodingConflict, result.Charset, encodingName)
	}

	bomSize := fileEncoding.BOMSize(result.Charset)
	bomBytes := append([]byte(nil), data[:bomSize]...)
	return data[bomSize:], bomInfo{
		HasBOM: true,
		Type:   result.Charset,
		Bytes:  bomBytes,
	}, nil
}

func (h *Handler) resolveEncodingFromDataDetailed(inputEncoding string, data []byte, filePath string) (encodingResult, error) {
	result := encodingResult{}

	if inputEncoding != "" {
		result.name = strings.ToLower(inputEncoding)
		enc, ok := fileEncoding.Get(result.name)
		if !ok {
			return result, fmt.Errorf("%w: %s. Use list_encodings to see available encodings", ErrEncodingUnsupported, result.name)
		}
		result.encoder = enc
		return result, nil
	}

	result.autoDetected = true
	detection, _ := fileEncoding.DetectSample(data)
	result.detectedEncoding = detection.Charset
	result.encodingConfidence = detection.Confidence

	if detection.Confidence >= fileEncoding.MinConfidenceThreshold && detection.Charset != "" {
		result.name = detection.Charset
	} else {
		result.name = "utf-8"
		if detection.Charset != "" {
			result.detectedEncoding = detection.Charset + " (low confidence, using utf-8)"
		}
	}

	enc, ok := fileEncoding.Get(result.name)
	if !ok {
		result.encoder = nil
		result.name = "utf-8"
		if result.detectedEncoding == "" {
			result.detectedEncoding = detection.Charset
		}
		result.detectedEncoding += " (unsupported, using utf-8)"
	} else {
		result.encoder = enc
	}

	slog.Debug("resolved encoding from loaded data",
		"path", filePath,
		"encoding", result.name,
		"detected", result.detectedEncoding,
		"confidence", result.encodingConfidence,
	)
	return result, nil
}

func encodeTextDocument(document textDocument, content string, policy bomPolicy) ([]byte, error) {
	content = restoreDocumentLineEndings(content, document.LineEndings.Style)

	var encoded []byte
	if fileEncoding.IsUTF8(document.Charset) {
		encoded = []byte(content)
	} else {
		enc, ok := fileEncoding.Get(document.Charset)
		if !ok {
			return nil, fmt.Errorf("%w: unsupported encoding %s", ErrEncodingEncode, document.Charset)
		}
		var err error
		encoded, err = enc.NewEncoder().Bytes([]byte(content))
		if err != nil {
			return nil, fmt.Errorf("%w: failed to encode content to %s: %v", ErrEncodingEncode, document.Charset, err)
		}
	}

	bom, err := documentBOMBytes(document, policy)
	if err != nil {
		return nil, err
	}
	if len(bom) == 0 {
		return encoded, nil
	}

	result := make([]byte, 0, len(bom)+len(encoded))
	result = append(result, bom...)
	result = append(result, encoded...)
	return result, nil
}

func restoreDocumentLineEndings(content, style string) string {
	switch style {
	case LineEndingCRLF:
		return ConvertLineEndings(content, LineEndingCRLF)
	case LineEndingLF, LineEndingMixed, LineEndingNone:
		// Mixed files historically normalize to LF during edit_file writes.
		return ConvertLineEndings(content, LineEndingLF)
	default:
		return ConvertLineEndings(content, LineEndingLF)
	}
}

func documentBOMBytes(document textDocument, policy bomPolicy) ([]byte, error) {
	if policy != bomPreserve {
		return nil, fmt.Errorf("unsupported BOM policy: %s", policy)
	}
	if !document.BOM.HasBOM {
		return nil, nil
	}
	if canonicalBOMEncoding(document.Charset) != document.BOM.Type {
		return nil, fmt.Errorf("%w: document BOM is %s but encoding is %s", ErrBOMEncodingConflict, document.BOM.Type, document.Charset)
	}
	if len(document.BOM.Bytes) > 0 {
		return append([]byte(nil), document.BOM.Bytes...), nil
	}
	bom := fileEncoding.BOMBytesFor(document.BOM.Type)
	if len(bom) == 0 {
		return nil, fmt.Errorf("%w: no BOM bytes registered for %s", ErrEncodingEncode, document.BOM.Type)
	}
	return bom, nil
}

func (h *Handler) readTextDocument(ctx context.Context, path, requestedEncoding string) (textDocument, error) {
	select {
	case <-ctx.Done():
		return textDocument{}, ctx.Err()
	default:
	}

	info, err := os.Stat(path)
	if err != nil {
		return textDocument{}, fmt.Errorf("failed to stat file: %w", err)
	}
	if info.Size() > h.config.MemoryThreshold {
		slog.Warn("loading large file into memory", "path", path, "size", info.Size(), "threshold", h.config.MemoryThreshold)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return textDocument{}, fmt.Errorf("failed to read file: %w", err)
	}

	select {
	case <-ctx.Done():
		return textDocument{}, ctx.Err()
	default:
	}

	encResult, err := h.resolveEncodingFromDataDetailed(requestedEncoding, data, path)
	if err != nil {
		return textDocument{}, err
	}

	payload, bom, err := splitTransportBOM(data, encResult.name)
	if err != nil {
		return textDocument{}, err
	}

	content, err := decodeContent(payload, encResult)
	if err != nil {
		return textDocument{}, fmt.Errorf("%w: failed to decode file content with %s: %v", ErrEncodingDecode, encResult.name, err)
	}

	return textDocument{
		Text:               content,
		Charset:            encResult.name,
		DetectedEncoding:   encResult.detectedEncoding,
		EncodingConfidence: encResult.encodingConfidence,
		AutoDetected:       encResult.autoDetected,
		BOM:                bom,
		LineEndings:        DetectLineEndings([]byte(content)),
		FileSizeBytes:      info.Size(),
		Mode:               info.Mode().Perm(),
	}, nil
}
