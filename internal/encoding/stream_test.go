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

func FuzzDecoderReaderChunkInvariance(f *testing.F) {
	charsets := []string{
		"utf-8",
		"utf-16-le",
		"utf-16-be",
		"windows-1251",
		"windows-1252",
		"gbk",
		"gb18030",
	}
	for _, seed := range []struct {
		selector byte
		data     []byte
	}{
		{selector: 0, data: []byte("Città Привет 中文 🌍")},
		{selector: 1, data: []byte{'A', 0, 'B', 0}},
		{selector: 2, data: []byte{0, 'A', 0, 'B'}},
		{selector: 5, data: []byte{0xd6, 0xd0, 0xce, 0xc4}},
		{selector: 6, data: []byte{0x94, 0x39, 0xfc, 0x36}},
		{selector: 1, data: []byte{0xd8}},
	} {
		f.Add(seed.selector, seed.data)
	}

	f.Fuzz(func(t *testing.T, selector byte, data []byte) {
		if len(data) > 64*1024 {
			t.Skip()
		}
		charset := charsets[int(selector)%len(charsets)]

		directReader, err := NewDecoderReader(bytes.NewReader(data), charset)
		if err != nil {
			t.Fatalf("construct direct decoder for %q: %v", charset, err)
		}
		direct, directErr := io.ReadAll(directReader)

		chunkedReader, err := NewDecoderReader(&oneByteReader{data: append([]byte(nil), data...)}, charset)
		if err != nil {
			t.Fatalf("construct chunked decoder for %q: %v", charset, err)
		}
		chunked, chunkedErr := io.ReadAll(chunkedReader)

		if (directErr == nil) != (chunkedErr == nil) {
			t.Fatalf("decoder error differs by chunking for %q: direct=%v chunked=%v", charset, directErr, chunkedErr)
		}
		if !bytes.Equal(direct, chunked) {
			t.Fatalf("decoder output differs by chunking for %q: %x != %x", charset, direct, chunked)
		}
		if directErr == nil && len(direct) > len(data)*4+4 {
			t.Fatalf("decoder output expansion is unexpectedly large for %q: input=%d output=%d", charset, len(data), len(direct))
		}
	})
}
