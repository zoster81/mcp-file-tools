package textstream

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/zoster81/mcp-file-tools/internal/operation"
)

type collectedLine struct {
	number int
	text   string
	ending string
}

func TestScanLinesPreservesEndingsAcrossOneByteReads(t *testing.T) {
	reader := &singleByteReader{data: []byte("alpha\r\nbeta\ngamma")}
	var lines []collectedLine
	total, err := ScanLines(context.Background(), reader, 1024, func(line Line) error {
		lines = append(lines, collectedLine{number: line.Number, text: string(line.Data), ending: string(line.Ending)})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []collectedLine{
		{number: 1, text: "alpha", ending: "\r\n"},
		{number: 2, text: "beta", ending: "\n"},
		{number: 3, text: "gamma"},
	}
	if total != len(want) || len(lines) != len(want) {
		t.Fatalf("total=%d lines=%v", total, lines)
	}
	for index := range want {
		if lines[index] != want[index] {
			t.Fatalf("line %d = %+v, want %+v", index, lines[index], want[index])
		}
	}
}

func TestScanLinesEmitsFinalEmptyLine(t *testing.T) {
	for _, testCase := range []struct {
		name  string
		input string
		want  []collectedLine
	}{
		{name: "empty", input: "", want: []collectedLine{{number: 1}}},
		{name: "trailing LF", input: "a\n", want: []collectedLine{{number: 1, text: "a", ending: "\n"}, {number: 2}}},
		{name: "trailing CRLF", input: "a\r\n", want: []collectedLine{{number: 1, text: "a", ending: "\r\n"}, {number: 2}}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			var got []collectedLine
			total, err := ScanLines(context.Background(), strings.NewReader(testCase.input), 1024, func(line Line) error {
				got = append(got, collectedLine{number: line.Number, text: string(line.Data), ending: string(line.Ending)})
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
			if total != len(testCase.want) || len(got) != len(testCase.want) {
				t.Fatalf("total=%d got=%v want=%v", total, got, testCase.want)
			}
			for index := range got {
				if got[index] != testCase.want[index] {
					t.Fatalf("line %d = %+v, want %+v", index, got[index], testCase.want[index])
				}
			}
		})
	}
}

func TestScanLinesRejectsOversizedLine(t *testing.T) {
	_, err := ScanLines(context.Background(), bytes.NewReader(bytes.Repeat([]byte{'x'}, 65)), 64, func(Line) error { return nil })
	if operation.KindOf(err) != operation.KindLimit {
		t.Fatalf("error = %v, kind=%v; want limit", err, operation.KindOf(err))
	}
}

func TestScanLinesHonorsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	seen := 0
	_, err := ScanLines(ctx, strings.NewReader("a\nb\nc"), 1024, func(Line) error {
		seen++
		cancel()
		return nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
	if seen != 1 {
		t.Fatalf("seen = %d, want 1", seen)
	}
}

func FuzzScanLinesRoundTrip(f *testing.F) {
	for _, seed := range []struct {
		data  []byte
		limit uint16
	}{
		{data: nil, limit: 64},
		{data: []byte("alpha\r\nbeta\ngamma"), limit: 64},
		{data: []byte("a\n"), limit: 1},
		{data: []byte("a\rb\rc"), limit: 8},
		{data: bytes.Repeat([]byte{'x'}, 65), limit: 64},
	} {
		f.Add(seed.data, seed.limit)
	}

	f.Fuzz(func(t *testing.T, data []byte, rawLimit uint16) {
		if len(data) > 64*1024 {
			t.Skip()
		}
		limit := int(rawLimit%1024) + 1
		var reconstructed bytes.Buffer
		visited := 0
		total, err := ScanLines(context.Background(), bytes.NewReader(data), limit, func(line Line) error {
			visited++
			if line.Number != visited {
				t.Fatalf("line number = %d, want %d", line.Number, visited)
			}
			if len(line.Data) > limit {
				t.Fatalf("line %d contains %d bytes, limit %d", line.Number, len(line.Data), limit)
			}
			if len(line.Ending) != 0 && !bytes.Equal(line.Ending, lfEnding) && !bytes.Equal(line.Ending, crlfEnding) {
				t.Fatalf("line %d has unsupported ending %q", line.Number, line.Ending)
			}
			_, _ = reconstructed.Write(line.Data)
			_, _ = reconstructed.Write(line.Ending)
			return nil
		})
		if err != nil {
			if operation.KindOf(err) != operation.KindLimit {
				t.Fatalf("unexpected scan error: %v", err)
			}
			return
		}
		if total != visited {
			t.Fatalf("total = %d, visited = %d", total, visited)
		}
		if !bytes.Equal(reconstructed.Bytes(), data) {
			t.Fatalf("line framing changed bytes: %x != %x", reconstructed.Bytes(), data)
		}
	})
}

type singleByteReader struct {
	data []byte
}

func (reader *singleByteReader) Read(buffer []byte) (int, error) {
	if len(reader.data) == 0 {
		return 0, io.EOF
	}
	buffer[0] = reader.data[0]
	reader.data = reader.data[1:]
	return 1, nil
}
