package textstream

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"testing"
)

func BenchmarkScanLines(b *testing.B) {
	for _, size := range []int{1 << 20, 16 << 20, 64 << 20} {
		data := repeatedBenchmarkData(size)
		b.Run(fmt.Sprintf("%dMiB", size>>20), func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(data)))
			for range b.N {
				lines, err := ScanLines(context.Background(), bytes.NewReader(data), DefaultMaxLineBytes, func(Line) error { return nil })
				if err != nil {
					b.Fatal(err)
				}
				if lines == 0 {
					b.Fatal("expected lines")
				}
			}
		})
	}
}

func BenchmarkByteLineEndingReader(b *testing.B) {
	for _, size := range []int{1 << 20, 16 << 20, 64 << 20} {
		b.StopTimer()
		data := repeatedBenchmarkData(size)
		b.StartTimer()
		b.Run(fmt.Sprintf("%dMiB", size>>20), func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			b.ReportAllocs()
			for range b.N {
				reader, err := NewByteLineEndingReader(bytes.NewReader(data), "crlf")
				if err != nil {
					b.Fatal(err)
				}
				if _, err := io.Copy(io.Discard, reader); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func repeatedBenchmarkData(size int) []byte {
	line := []byte("alpha beta gamma delta epsilon\n")
	data := bytes.Repeat(line, size/len(line)+1)
	return data[:size]
}
