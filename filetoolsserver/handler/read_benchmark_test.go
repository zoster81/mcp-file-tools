package handler

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func BenchmarkHandleReadTextFileBoundedOutput(b *testing.B) {
	line := []byte("alpha beta gamma delta epsilon\n")
	for _, size := range []int{1 << 20, 16 << 20, 64 << 20} {
		b.Run(fmt.Sprintf("%dMiB", size>>20), func(b *testing.B) {
			b.StopTimer()
			lineCount := size / len(line)
			data := bytes.Repeat(line, lineCount)
			dir := b.TempDir()
			path := filepath.Join(dir, "large.txt")
			if err := os.WriteFile(path, data, 0o600); err != nil {
				b.Fatal(err)
			}
			handler := NewHandler([]string{dir})
			offset := lineCount
			limit := 1
			input := ReadTextFileInput{Path: path, Offset: &offset, Limit: &limit}
			b.SetBytes(int64(len(data)))
			b.ReportAllocs()
			b.StartTimer()

			for range b.N {
				result, output, err := handler.HandleReadTextFile(context.Background(), nil, input)
				if err != nil || result.IsError {
					b.Fatalf("read failed: result=%v err=%v", result, err)
				}
				if output.Content != "alpha beta gamma delta epsilon" {
					b.Fatalf("unexpected output %q", output.Content)
				}
			}
		})
	}
}
