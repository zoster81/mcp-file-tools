package handler

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	fileEncoding "github.com/dimitar-grigorov/mcp-file-tools/internal/encoding"
)

func TestHandleChangeLineEndings_CRLFtoLF(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})
	testFile := filepath.Join(tempDir, "test.txt")

	if err := os.WriteFile(testFile, []byte("line1\r\nline2\r\nline3\r\n"), 0644); err != nil {
		t.Fatal(err)
	}

	input := ChangeLineEndingsInput{Path: testFile, Style: "lf"}
	result, output, err := h.HandleChangeLineEndings(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("expected success, got error")
	}

	if output.OriginalStyle != "crlf" {
		t.Errorf("expected originalStyle=crlf, got %s", output.OriginalStyle)
	}
	if output.NewStyle != "lf" {
		t.Errorf("expected newStyle=lf, got %s", output.NewStyle)
	}
	if output.LinesChanged != 3 {
		t.Errorf("expected linesChanged=3, got %d", output.LinesChanged)
	}

	// Verify file content
	data, _ := os.ReadFile(testFile)
	if string(data) != "line1\nline2\nline3\n" {
		t.Errorf("unexpected file content: %q", string(data))
	}
}

func TestHandleChangeLineEndings_LFtoCRLF(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})
	testFile := filepath.Join(tempDir, "test.txt")

	if err := os.WriteFile(testFile, []byte("line1\nline2\nline3\n"), 0644); err != nil {
		t.Fatal(err)
	}

	input := ChangeLineEndingsInput{Path: testFile, Style: "crlf"}
	result, output, err := h.HandleChangeLineEndings(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("expected success, got error")
	}

	if output.OriginalStyle != "lf" {
		t.Errorf("expected originalStyle=lf, got %s", output.OriginalStyle)
	}
	if output.LinesChanged != 3 {
		t.Errorf("expected linesChanged=3, got %d", output.LinesChanged)
	}

	data, _ := os.ReadFile(testFile)
	if string(data) != "line1\r\nline2\r\nline3\r\n" {
		t.Errorf("unexpected file content: %q", string(data))
	}
}

func TestHandleChangeLineEndings_MixedToLF(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})
	testFile := filepath.Join(tempDir, "test.txt")

	// Mix of CRLF and LF
	if err := os.WriteFile(testFile, []byte("line1\r\nline2\nline3\r\n"), 0644); err != nil {
		t.Fatal(err)
	}

	input := ChangeLineEndingsInput{Path: testFile, Style: "lf"}
	result, output, err := h.HandleChangeLineEndings(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("expected success, got error")
	}

	if output.OriginalStyle != "mixed" {
		t.Errorf("expected originalStyle=mixed, got %s", output.OriginalStyle)
	}
	if output.LinesChanged != 2 {
		t.Errorf("expected linesChanged=2 (CRLF lines), got %d", output.LinesChanged)
	}

	data, _ := os.ReadFile(testFile)
	if string(data) != "line1\nline2\nline3\n" {
		t.Errorf("unexpected file content: %q", string(data))
	}
}

func TestHandleChangeLineEndings_AlreadyCorrect(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})
	testFile := filepath.Join(tempDir, "test.txt")

	if err := os.WriteFile(testFile, []byte("line1\nline2\n"), 0644); err != nil {
		t.Fatal(err)
	}

	input := ChangeLineEndingsInput{Path: testFile, Style: "lf"}
	result, output, err := h.HandleChangeLineEndings(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("expected success, got error")
	}

	if output.LinesChanged != 0 {
		t.Errorf("expected linesChanged=0 for no-op, got %d", output.LinesChanged)
	}
	if output.OriginalStyle != "lf" {
		t.Errorf("expected originalStyle=lf, got %s", output.OriginalStyle)
	}
}

func representativeTextForEncoding(t *testing.T, encodingName string) string {
	t.Helper()

	switch encodingName {
	case "utf-8", "utf-16-le", "utf-16-be":
		return "MQL © 中文 Привет"
	case "windows-1251", "koi8-r", "ibm866", "iso-8859-5":
		return "Привет"
	case "koi8-u":
		return "Привіт"
	case "windows-1252", "iso-8859-1":
		return "café"
	case "iso-8859-15":
		return "café €"
	case "windows-1250", "iso-8859-2":
		return "Český"
	case "windows-1253", "iso-8859-7":
		return "Ελλάδα"
	case "windows-1254", "iso-8859-9":
		return "Türkçe"
	case "windows-1255":
		return "שלום"
	case "windows-1256":
		return "مرحبا"
	case "windows-1257":
		return "Āžuolas"
	case "windows-1258":
		return "Viêt Nam"
	case "windows-874":
		return "ไทย"
	case "gbk", "gb18030":
		return "中文"
	default:
		t.Fatalf("missing representative text for encoding %q", encodingName)
		return ""
	}
}

func decodeLineEndingFixture(t *testing.T, encodingName string, data []byte) string {
	t.Helper()

	if result, found := fileEncoding.DetectBOM(data); found {
		if result.Charset != canonicalBOMEncoding(encodingName) {
			t.Fatalf("BOM = %s, want %s", result.Charset, canonicalBOMEncoding(encodingName))
		}
		data = data[fileEncoding.BOMSize(result.Charset):]
	}
	if fileEncoding.IsUTF8(encodingName) {
		return string(data)
	}

	enc, ok := fileEncoding.Get(encodingName)
	if !ok {
		t.Fatalf("encoding %q is not registered", encodingName)
	}
	decoded, err := enc.NewDecoder().Bytes(data)
	if err != nil {
		t.Fatalf("failed to decode %s fixture: %v", encodingName, err)
	}
	return string(decoded)
}

func TestHandleChangeLineEndings_AllSupportedEncodings(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	tests := []struct {
		name         string
		input        func(string) string
		target       string
		want         func(string) string
		wantOriginal string
		wantChanged  int
	}{
		{
			name:         "crlf to lf",
			input:        func(s string) string { return s + "\r\n" + s + "\r\n" },
			target:       LineEndingLF,
			want:         func(s string) string { return s + "\n" + s + "\n" },
			wantOriginal: LineEndingCRLF,
			wantChanged:  2,
		},
		{
			name:         "lf to crlf",
			input:        func(s string) string { return s + "\n" + s + "\n" },
			target:       LineEndingCRLF,
			want:         func(s string) string { return s + "\r\n" + s + "\r\n" },
			wantOriginal: LineEndingLF,
			wantChanged:  2,
		},
		{
			name:         "mixed to lf",
			input:        func(s string) string { return s + "\r\n" + s + "\n" + s + "\r\n" },
			target:       LineEndingLF,
			want:         func(s string) string { return s + "\n" + s + "\n" + s + "\n" },
			wantOriginal: LineEndingMixed,
			wantChanged:  2,
		},
		{
			name:         "mixed to crlf",
			input:        func(s string) string { return s + "\r\n" + s + "\n" + s + "\r\n" },
			target:       LineEndingCRLF,
			want:         func(s string) string { return s + "\r\n" + s + "\r\n" + s + "\r\n" },
			wantOriginal: LineEndingMixed,
			wantChanged:  1,
		},
	}

	for _, encodingInfo := range fileEncoding.ListEncodings() {
		encodingInfo := encodingInfo
		t.Run(encodingInfo.Name, func(t *testing.T) {
			representative := representativeTextForEncoding(t, encodingInfo.Name)
			for _, testCase := range tests {
				testCase := testCase
				t.Run(testCase.name, func(t *testing.T) {
					testFile := filepath.Join(tempDir, encodingInfo.Name+"_"+testCase.name+".txt")
					inputText := testCase.input(representative)
					data := encodeLineEndingFixture(t, encodingInfo.Name, inputText, false)
					if err := os.WriteFile(testFile, data, 0644); err != nil {
						t.Fatal(err)
					}

					result, output, err := h.HandleChangeLineEndings(context.Background(), nil, ChangeLineEndingsInput{
						Path:     testFile,
						Style:    testCase.target,
						Encoding: encodingInfo.Name,
					})
					if err != nil {
						t.Fatal(err)
					}
					if result.IsError {
						t.Fatalf("expected success for %s", encodingInfo.Name)
					}
					if output.OriginalStyle != testCase.wantOriginal {
						t.Errorf("OriginalStyle = %q, want %q", output.OriginalStyle, testCase.wantOriginal)
					}
					if output.NewStyle != testCase.target {
						t.Errorf("NewStyle = %q, want %q", output.NewStyle, testCase.target)
					}
					if output.LinesChanged != testCase.wantChanged {
						t.Errorf("LinesChanged = %d, want %d", output.LinesChanged, testCase.wantChanged)
					}

					converted, err := os.ReadFile(testFile)
					if err != nil {
						t.Fatal(err)
					}
					if got, want := decodeLineEndingFixture(t, encodingInfo.Name, converted), testCase.want(representative); got != want {
						t.Errorf("decoded content = %q, want %q", got, want)
					}
				})
			}
		})
	}
}

func TestHandleChangeLineEndings_AllSupportedEncodingsNoOpIsByteIdentical(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	for _, encodingInfo := range fileEncoding.ListEncodings() {
		encodingInfo := encodingInfo
		t.Run(encodingInfo.Name, func(t *testing.T) {
			testFile := filepath.Join(tempDir, encodingInfo.Name+"_noop.txt")
			representative := representativeTextForEncoding(t, encodingInfo.Name)
			original := encodeLineEndingFixture(t, encodingInfo.Name, representative+"\r\n", false)
			if err := os.WriteFile(testFile, original, 0644); err != nil {
				t.Fatal(err)
			}

			result, output, err := h.HandleChangeLineEndings(context.Background(), nil, ChangeLineEndingsInput{
				Path:     testFile,
				Style:    LineEndingCRLF,
				Encoding: encodingInfo.Name,
			})
			if err != nil {
				t.Fatal(err)
			}
			if result.IsError {
				t.Fatal("expected success")
			}
			if output.LinesChanged != 0 {
				t.Errorf("LinesChanged = %d, want 0", output.LinesChanged)
			}

			actual, err := os.ReadFile(testFile)
			if err != nil {
				t.Fatal(err)
			}
			if string(actual) != string(original) {
				t.Fatal("no-op changed file bytes")
			}
		})
	}
}

func TestHandleChangeLineEndings_PreservesUnicodeBOM(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	for _, encodingName := range []string{"utf-8", "utf-16-le", "utf-16-be"} {
		encodingName := encodingName
		t.Run(encodingName, func(t *testing.T) {
			testFile := filepath.Join(tempDir, encodingName+"_bom.txt")
			text := representativeTextForEncoding(t, encodingName) + "\r\n"
			original := encodeLineEndingFixture(t, encodingName, text, true)
			if err := os.WriteFile(testFile, original, 0644); err != nil {
				t.Fatal(err)
			}

			result, _, err := h.HandleChangeLineEndings(context.Background(), nil, ChangeLineEndingsInput{
				Path:  testFile,
				Style: LineEndingLF,
			})
			if err != nil {
				t.Fatal(err)
			}
			if result.IsError {
				t.Fatal("expected success")
			}

			actual, err := os.ReadFile(testFile)
			if err != nil {
				t.Fatal(err)
			}
			resultBOM, found := fileEncoding.DetectBOM(actual)
			if !found || resultBOM.Charset != encodingName {
				t.Fatalf("BOM = %v, found=%v; want %s", resultBOM, found, encodingName)
			}
			if got, want := decodeLineEndingFixture(t, encodingName, actual), representativeTextForEncoding(t, encodingName)+"\n"; got != want {
				t.Errorf("decoded content = %q, want %q", got, want)
			}
		})
	}
}

func TestHandleChangeLineEndings_PreservesUnmappedLegacyBytes(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})
	testFile := filepath.Join(tempDir, "unmapped-cp1252.txt")
	original := []byte{0x81, '\r', '\n', 0x8D, '\r', '\n'}
	if err := os.WriteFile(testFile, original, 0644); err != nil {
		t.Fatal(err)
	}

	result, _, err := h.HandleChangeLineEndings(context.Background(), nil, ChangeLineEndingsInput{
		Path:     testFile,
		Style:    LineEndingLF,
		Encoding: "windows-1252",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatal("expected success")
	}

	actual, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}
	want := []byte{0x81, '\n', 0x8D, '\n'}
	if string(actual) != string(want) {
		t.Fatalf("bytes = %x, want %x", actual, want)
	}
}

func TestHandleChangeLineEndings_BOMEncodingConflictLeavesFileUnchanged(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})
	testFile := filepath.Join(tempDir, "conflict.txt")
	original := encodeLineEndingFixture(t, "utf-16-le", "line1\r\nline2", true)
	if err := os.WriteFile(testFile, original, 0644); err != nil {
		t.Fatal(err)
	}

	result, _, err := h.HandleChangeLineEndings(context.Background(), nil, ChangeLineEndingsInput{
		Path:     testFile,
		Style:    LineEndingLF,
		Encoding: "utf-16-be",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatal("expected BOM/encoding conflict error")
	}

	actual, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(actual) != string(original) {
		t.Fatal("conflict changed file bytes")
	}
}

func TestHandleChangeLineEndings_UnsupportedExplicitEncoding(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})
	testFile := filepath.Join(tempDir, "unsupported.txt")
	if err := os.WriteFile(testFile, []byte("line1\r\nline2"), 0644); err != nil {
		t.Fatal(err)
	}

	result, _, err := h.HandleChangeLineEndings(context.Background(), nil, ChangeLineEndingsInput{
		Path:     testFile,
		Style:    LineEndingLF,
		Encoding: "not-an-encoding",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatal("expected unsupported encoding error")
	}
}

func TestHandleChangeLineEndings_UTF16LEWithBOM_CRLFToLF(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})
	testFile := filepath.Join(tempDir, "metaeditor.mqh")
	originalText := "// Copyright © MetaQuotes\r\nstring message = \"caffè\";\r\n"
	originalData := encodeLineEndingFixture(t, "utf-16-le", originalText, true)
	if err := os.WriteFile(testFile, originalData, 0644); err != nil {
		t.Fatal(err)
	}

	result, output, err := h.HandleChangeLineEndings(context.Background(), nil, ChangeLineEndingsInput{
		Path:     testFile,
		Style:    LineEndingLF,
		Encoding: "utf-16-le",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatal("expected success")
	}
	if output.OriginalStyle != LineEndingCRLF {
		t.Errorf("OriginalStyle = %q, want %q", output.OriginalStyle, LineEndingCRLF)
	}
	if output.LinesChanged != 2 {
		t.Errorf("LinesChanged = %d, want 2", output.LinesChanged)
	}

	data, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) < 2 || data[0] != 0xFF || data[1] != 0xFE {
		t.Fatalf("UTF-16 LE BOM was not preserved: %x", data[:min(len(data), 4)])
	}
	enc, _ := fileEncoding.Get("utf-16-le")
	decoded, err := enc.NewDecoder().Bytes(data[2:])
	if err != nil {
		t.Fatal(err)
	}
	wantText := "// Copyright © MetaQuotes\nstring message = \"caffè\";\n"
	if string(decoded) != wantText {
		t.Errorf("decoded content = %q, want %q", string(decoded), wantText)
	}
}

func TestHandleChangeLineEndings_InvalidStyle(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})
	testFile := filepath.Join(tempDir, "test.txt")

	if err := os.WriteFile(testFile, []byte("test\n"), 0644); err != nil {
		t.Fatal(err)
	}

	input := ChangeLineEndingsInput{Path: testFile, Style: "mac"}
	result, _, err := h.HandleChangeLineEndings(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Errorf("expected error for invalid style")
	}
}

func TestHandleChangeLineEndings_EmptyFile(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})
	testFile := filepath.Join(tempDir, "test.txt")

	if err := os.WriteFile(testFile, []byte("no newlines here"), 0644); err != nil {
		t.Fatal(err)
	}

	input := ChangeLineEndingsInput{Path: testFile, Style: "lf"}
	result, output, err := h.HandleChangeLineEndings(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("expected success, got error")
	}

	if output.LinesChanged != 0 {
		t.Errorf("expected linesChanged=0 for file with no line endings, got %d", output.LinesChanged)
	}
}
