package encoding

import (
	"bytes"
	"io"
	"testing"
)

type oneByteReader struct {
	data []byte
}

func (r *oneByteReader) Read(buffer []byte) (int, error) {
	if len(r.data) == 0 {
		return 0, io.EOF
	}
	buffer[0] = r.data[0]
	r.data = r.data[1:]
	return 1, nil
}

func TestNewDecoderReaderPreservesMultibyteSequencesAcrossReads(t *testing.T) {
	tests := []struct {
		charset string
		text    string
	}{
		{charset: "utf-8", text: "Città Привет 中文 🌍"},
		{charset: "utf-16-le", text: "Città Привет 中文 🌍"},
		{charset: "utf-16-be", text: "Città Привет 中文 🌍"},
		{charset: "gb18030", text: "中文 🌍"},
	}

	for _, testCase := range tests {
		t.Run(testCase.charset, func(t *testing.T) {
			encoded := []byte(testCase.text)
			if !IsUTF8(testCase.charset) {
				registered, ok := Get(testCase.charset)
				if !ok {
					t.Fatalf("encoding %q is not registered", testCase.charset)
				}
				var err error
				encoded, err = registered.NewEncoder().Bytes(encoded)
				if err != nil {
					t.Fatal(err)
				}
			}

			reader, err := NewDecoderReader(&oneByteReader{data: append([]byte(nil), encoded...)}, testCase.charset)
			if err != nil {
				t.Fatal(err)
			}
			decoded, err := io.ReadAll(reader)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(decoded, []byte(testCase.text)) {
				t.Fatalf("decoded = %q, want %q", decoded, testCase.text)
			}
		})
	}
}

func TestNewDecoderReaderRejectsUnsupportedEncoding(t *testing.T) {
	if _, err := NewDecoderReader(bytes.NewReader(nil), "not-an-encoding"); err == nil {
		t.Fatal("expected unsupported encoding error")
	}
}
