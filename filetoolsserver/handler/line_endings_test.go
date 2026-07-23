package handler

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	fileEncoding "github.com/dimitar-grigorov/mcp-file-tools/internal/encoding"
)

func TestDetectLineEndings(t *testing.T) {
	tests := []struct {
		name      string
		input     []byte
		wantStyle string
		wantCRLF  int
		wantLF    int
	}{
		{
			name:      "CRLF only",
			input:     []byte("line1\r\nline2\r\nline3"),
			wantStyle: LineEndingCRLF,
			wantCRLF:  2,
			wantLF:    0,
		},
		{
			name:      "LF only",
			input:     []byte("line1\nline2\nline3"),
			wantStyle: LineEndingLF,
			wantCRLF:  0,
			wantLF:    2,
		},
		{
			name:      "mixed line endings",
			input:     []byte("line1\r\nline2\nline3"),
			wantStyle: LineEndingMixed,
			wantCRLF:  1,
			wantLF:    1,
		},
		{
			name:      "no line endings",
			input:     []byte("single line"),
			wantStyle: LineEndingNone,
			wantCRLF:  0,
			wantLF:    0,
		},
		{
			name:      "empty file",
			input:     []byte{},
			wantStyle: LineEndingNone,
			wantCRLF:  0,
			wantLF:    0,
		},
		{
			name:      "trailing CRLF",
			input:     []byte("line1\r\n"),
			wantStyle: LineEndingCRLF,
			wantCRLF:  1,
			wantLF:    0,
		},
		{
			name:      "trailing LF",
			input:     []byte("line1\n"),
			wantStyle: LineEndingLF,
			wantCRLF:  0,
			wantLF:    1,
		},
		{
			name:      "standalone CR ignored",
			input:     []byte("line1\rline2\nline3"),
			wantStyle: LineEndingLF,
			wantCRLF:  0,
			wantLF:    1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectLineEndings(tt.input)
			if got.Style != tt.wantStyle {
				t.Errorf("Style = %q, want %q", got.Style, tt.wantStyle)
			}
			if got.CRLFCount != tt.wantCRLF {
				t.Errorf("CRLFCount = %d, want %d", got.CRLFCount, tt.wantCRLF)
			}
			if got.LFCount != tt.wantLF {
				t.Errorf("LFCount = %d, want %d", got.LFCount, tt.wantLF)
			}
		})
	}
}

func TestConvertLineEndings(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		target string
		want   string
	}{
		// To LF
		{"CRLF to LF", "line1\r\nline2\r\n", LineEndingLF, "line1\nline2\n"},
		{"LF to LF (unchanged)", "line1\nline2\n", LineEndingLF, "line1\nline2\n"},
		{"mixed to LF", "line1\r\nline2\nline3", LineEndingLF, "line1\nline2\nline3"},

		// To CRLF
		{"LF to CRLF", "line1\nline2\n", LineEndingCRLF, "line1\r\nline2\r\n"},
		{"CRLF to CRLF (unchanged)", "line1\r\nline2\r\n", LineEndingCRLF, "line1\r\nline2\r\n"},
		{"mixed to CRLF", "line1\r\nline2\nline3", LineEndingCRLF, "line1\r\nline2\r\nline3"},

		// Other styles (treated as LF)
		{"to mixed (becomes LF)", "line1\r\nline2\r\n", LineEndingMixed, "line1\nline2\n"},
		{"to none (becomes LF)", "line1\r\nline2\r\n", LineEndingNone, "line1\nline2\n"},

		// Edge cases
		{"no line endings", "single line", LineEndingCRLF, "single line"},
		{"empty string", "", LineEndingCRLF, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ConvertLineEndings(tt.input, tt.target)
			if got != tt.want {
				t.Errorf("ConvertLineEndings(%q, %q) = %q, want %q", tt.input, tt.target, got, tt.want)
			}
		})
	}
}

func TestHandleDetectLineEndings(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	tests := []struct {
		name                string
		content             []byte
		wantStyle           string
		wantTotalLines      int
		wantInconsistent    []int
		wantInconsistentNil bool
	}{
		{
			name:             "pure CRLF file",
			content:          []byte("line1\r\nline2\r\nline3"),
			wantStyle:        LineEndingCRLF,
			wantTotalLines:   3,
			wantInconsistent: []int{},
		},
		{
			name:             "pure LF file",
			content:          []byte("line1\nline2\nline3"),
			wantStyle:        LineEndingLF,
			wantTotalLines:   3,
			wantInconsistent: []int{},
		},
		{
			name:             "mixed - mostly CRLF with one LF",
			content:          []byte("line1\r\nline2\nline3\r\nline4\r\n"),
			wantStyle:        LineEndingMixed,
			wantTotalLines:   5,
			wantInconsistent: []int{2}, // line 2 has LF when CRLF is dominant
		},
		{
			name:             "mixed - mostly LF with one CRLF",
			content:          []byte("line1\nline2\r\nline3\nline4\n"),
			wantStyle:        LineEndingMixed,
			wantTotalLines:   5,
			wantInconsistent: []int{2}, // line 2 has CRLF when LF is dominant
		},
		{
			name:             "no line endings",
			content:          []byte("single line without newline"),
			wantStyle:        LineEndingNone,
			wantTotalLines:   1,
			wantInconsistent: []int{},
		},
		{
			name:             "empty file",
			content:          []byte{},
			wantStyle:        LineEndingNone,
			wantTotalLines:   1,
			wantInconsistent: []int{},
		},
		{
			name:             "mixed - equal CRLF and LF",
			content:          []byte("line1\r\nline2\n"),
			wantStyle:        LineEndingMixed,
			wantTotalLines:   3,
			wantInconsistent: []int{2}, // LF is minority when counts equal (CRLF >= LF)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFile := filepath.Join(tempDir, tt.name+".txt")
			if err := os.WriteFile(testFile, tt.content, 0644); err != nil {
				t.Fatal(err)
			}

			input := DetectLineEndingsInput{Path: testFile}
			result, output, err := h.HandleDetectLineEndings(context.Background(), nil, input)
			if err != nil {
				t.Fatal(err)
			}
			if result.IsError {
				t.Errorf("expected success, got error")
			}

			if output.Style != tt.wantStyle {
				t.Errorf("Style = %q, want %q", output.Style, tt.wantStyle)
			}
			if output.TotalLines != tt.wantTotalLines {
				t.Errorf("TotalLines = %d, want %d", output.TotalLines, tt.wantTotalLines)
			}
			if len(output.InconsistentLines) != len(tt.wantInconsistent) {
				t.Errorf("InconsistentLines = %v, want %v", output.InconsistentLines, tt.wantInconsistent)
			} else {
				for i, line := range output.InconsistentLines {
					if line != tt.wantInconsistent[i] {
						t.Errorf("InconsistentLines[%d] = %d, want %d", i, line, tt.wantInconsistent[i])
					}
				}
			}
		})
	}
}

func encodeLineEndingFixture(t *testing.T, encodingName, content string, withBOM bool) []byte {
	t.Helper()

	var data []byte
	if fileEncoding.IsUTF8(encodingName) {
		data = []byte(content)
	} else {
		enc, ok := fileEncoding.Get(encodingName)
		if !ok {
			t.Fatalf("encoding %q is not registered", encodingName)
		}
		encoded, err := enc.NewEncoder().Bytes([]byte(content))
		if err != nil {
			t.Fatalf("failed to encode fixture as %s: %v", encodingName, err)
		}
		data = encoded
	}

	if withBOM {
		bom := fileEncoding.BOMBytesFor(encodingName)
		if len(bom) == 0 {
			t.Fatalf("encoding %q does not define a BOM", encodingName)
		}
		withPrefix := make([]byte, 0, len(bom)+len(data))
		withPrefix = append(withPrefix, bom...)
		data = append(withPrefix, data...)
	}

	return data
}

func assertLineEndingOutput(t *testing.T, output DetectLineEndingsOutput, wantStyle string, wantTotalLines int, wantInconsistent []int) {
	t.Helper()

	if output.Style != wantStyle {
		t.Errorf("Style = %q, want %q", output.Style, wantStyle)
	}
	if output.TotalLines != wantTotalLines {
		t.Errorf("TotalLines = %d, want %d", output.TotalLines, wantTotalLines)
	}
	if len(output.InconsistentLines) != len(wantInconsistent) {
		t.Fatalf("InconsistentLines = %v, want %v", output.InconsistentLines, wantInconsistent)
	}
	for i, line := range output.InconsistentLines {
		if line != wantInconsistent[i] {
			t.Errorf("InconsistentLines[%d] = %d, want %d", i, line, wantInconsistent[i])
		}
	}
}

func TestHandleDetectLineEndings_AllSupportedEncodings(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	lineEndingCases := []struct {
		name             string
		content          func(string) string
		wantStyle        string
		wantTotalLines   int
		wantInconsistent []int
	}{
		{
			name:             "crlf",
			content:          func(s string) string { return s + "\r\n" + s + "\r\n" + s },
			wantStyle:        LineEndingCRLF,
			wantTotalLines:   3,
			wantInconsistent: []int{},
		},
		{
			name:             "lf",
			content:          func(s string) string { return s + "\n" + s + "\n" + s },
			wantStyle:        LineEndingLF,
			wantTotalLines:   3,
			wantInconsistent: []int{},
		},
		{
			name:             "mixed",
			content:          func(s string) string { return s + "\r\n" + s + "\n" + s + "\r\n" + s },
			wantStyle:        LineEndingMixed,
			wantTotalLines:   4,
			wantInconsistent: []int{2},
		},
	}

	for _, encodingInfo := range fileEncoding.ListEncodings() {
		encodingInfo := encodingInfo
		t.Run(encodingInfo.Name, func(t *testing.T) {
			representative := representativeTextForEncoding(t, encodingInfo.Name)
			for _, testCase := range lineEndingCases {
				testCase := testCase
				t.Run(testCase.name, func(t *testing.T) {
					testFile := filepath.Join(tempDir, encodingInfo.Name+"_"+testCase.name+".txt")
					content := encodeLineEndingFixture(t, encodingInfo.Name, testCase.content(representative), false)
					if err := os.WriteFile(testFile, content, 0644); err != nil {
						t.Fatal(err)
					}

					input := DetectLineEndingsInput{Path: testFile, Encoding: encodingInfo.Name}
					result, output, err := h.HandleDetectLineEndings(context.Background(), nil, input)
					if err != nil {
						t.Fatal(err)
					}
					if result.IsError {
						t.Fatalf("expected success for %s", encodingInfo.Name)
					}
					assertLineEndingOutput(t, output, testCase.wantStyle, testCase.wantTotalLines, testCase.wantInconsistent)
				})
			}
		})
	}
}

func TestHandleDetectLineEndings_AutoDetectUnicodeBOMs(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	tests := []struct {
		name             string
		encoding         string
		content          string
		wantStyle        string
		wantTotalLines   int
		wantInconsistent []int
	}{
		{
			name:             "utf8 BOM CRLF",
			encoding:         "utf-8",
			content:          representativeTextForEncoding(t, "utf-8") + "\r\n" + representativeTextForEncoding(t, "utf-8"),
			wantStyle:        LineEndingCRLF,
			wantTotalLines:   2,
			wantInconsistent: []int{},
		},
		{
			name:             "utf16 LE BOM CRLF",
			encoding:         "utf-16-le",
			content:          representativeTextForEncoding(t, "utf-16-le") + "\r\n" + representativeTextForEncoding(t, "utf-16-le"),
			wantStyle:        LineEndingCRLF,
			wantTotalLines:   2,
			wantInconsistent: []int{},
		},
		{
			name:             "utf16 BE BOM mixed",
			encoding:         "utf-16-be",
			content:          representativeTextForEncoding(t, "utf-16-be") + "\r\n" + representativeTextForEncoding(t, "utf-16-be") + "\n" + representativeTextForEncoding(t, "utf-16-be") + "\r\n" + representativeTextForEncoding(t, "utf-16-be"),
			wantStyle:        LineEndingMixed,
			wantTotalLines:   4,
			wantInconsistent: []int{2},
		},
	}

	for _, testCase := range tests {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			testFile := filepath.Join(tempDir, testCase.name+".txt")
			content := encodeLineEndingFixture(t, testCase.encoding, testCase.content, true)
			if err := os.WriteFile(testFile, content, 0644); err != nil {
				t.Fatal(err)
			}

			result, output, err := h.HandleDetectLineEndings(context.Background(), nil, DetectLineEndingsInput{Path: testFile})
			if err != nil {
				t.Fatal(err)
			}
			if result.IsError {
				t.Fatal("expected success")
			}
			assertLineEndingOutput(t, output, testCase.wantStyle, testCase.wantTotalLines, testCase.wantInconsistent)
		})
	}
}

func TestHandleDetectLineEndings_UnsupportedExplicitEncoding(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})
	testFile := filepath.Join(tempDir, "unsupported.txt")
	if err := os.WriteFile(testFile, []byte("line1\nline2"), 0644); err != nil {
		t.Fatal(err)
	}

	result, _, err := h.HandleDetectLineEndings(context.Background(), nil, DetectLineEndingsInput{
		Path:     testFile,
		Encoding: "not-an-encoding",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatal("expected unsupported encoding error")
	}
}

func TestHandleDetectLineEndings_UTF16LEWithBOM(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})
	testFile := filepath.Join(tempDir, "metaeditor.mqh")

	// UTF-16 LE BOM followed by "line1\r\nline2\r\nline3".
	content := []byte{
		0xFF, 0xFE,
		'l', 0x00, 'i', 0x00, 'n', 0x00, 'e', 0x00, '1', 0x00, '\r', 0x00, '\n', 0x00,
		'l', 0x00, 'i', 0x00, 'n', 0x00, 'e', 0x00, '2', 0x00, '\r', 0x00, '\n', 0x00,
		'l', 0x00, 'i', 0x00, 'n', 0x00, 'e', 0x00, '3', 0x00,
	}
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatal(err)
	}

	result, output, err := h.HandleDetectLineEndings(context.Background(), nil, DetectLineEndingsInput{Path: testFile})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatal("expected success")
	}
	if output.Style != LineEndingCRLF {
		t.Errorf("Style = %q, want %q", output.Style, LineEndingCRLF)
	}
	if output.TotalLines != 3 {
		t.Errorf("TotalLines = %d, want 3", output.TotalLines)
	}
	if len(output.InconsistentLines) != 0 {
		t.Errorf("InconsistentLines = %v, want []", output.InconsistentLines)
	}
}

func TestHandleDetectLineEndings_PathValidation(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	// Test path outside allowed directory
	input := DetectLineEndingsInput{Path: "/not/allowed/path.txt"}
	result, _, err := h.HandleDetectLineEndings(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Error("expected error for path outside allowed directory")
	}
}
