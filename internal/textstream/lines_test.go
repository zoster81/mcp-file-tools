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
