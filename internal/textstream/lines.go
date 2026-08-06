// Package textstream provides bounded incremental text consumers shared by
// encoding-aware file handlers.
package textstream

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/zoster81/scripthold/internal/operation"
)

const (
	DefaultMaxLineBytes = 16 * 1024 * 1024
	lineReadBufferSize  = 64 * 1024
)

var (
	lfEnding   = []byte{'\n'}
	crlfEnding = []byte{'\r', '\n'}
)

// Line contains one decoded UTF-8 line. Data excludes the line ending; Ending
// is nil, "\n", or "\r\n". The slices are valid only until the visitor returns;
// consumers that retain them must copy.
type Line struct {
	Number int
	Data   []byte
	Ending []byte
}

// ScanLines incrementally frames decoded UTF-8 input while bounding the memory
// used by any single line. It preserves the empty final line implied by a
// trailing newline, matching strings.Split(text, "\n") line-count semantics.
func ScanLines(ctx context.Context, reader io.Reader, maxLineBytes int, visit func(Line) error) (int, error) {
	if maxLineBytes <= 0 {
		maxLineBytes = DefaultMaxLineBytes
	}
	buffered := bufio.NewReaderSize(reader, lineReadBufferSize)
	lineBuffer := make([]byte, 0, min(maxLineBytes, lineReadBufferSize))
	lineNumber := 1
	emitted := 0
	endedWithNewline := false

	emit := func(raw []byte) error {
		data := raw
		var ending []byte
		if len(data) > 0 && data[len(data)-1] == '\n' {
			ending = lfEnding
			data = data[:len(data)-1]
			if len(data) > 0 && data[len(data)-1] == '\r' {
				ending = crlfEnding
				data = data[:len(data)-1]
			}
		}
		if len(data) > maxLineBytes {
			return operation.Wrap(
				operation.KindLimit,
				"scan_lines",
				"",
				fmt.Errorf("line %d exceeds the %d-byte limit", lineNumber, maxLineBytes),
			)
		}
		line := Line{Number: lineNumber, Data: data, Ending: ending}
		if err := visit(line); err != nil {
			return err
		}
		emitted++
		lineNumber++
		endedWithNewline = len(ending) > 0
		return nil
	}

	for {
		if err := ctx.Err(); err != nil {
			return emitted, err
		}

		fragment, err := buffered.ReadSlice('\n')
		if len(fragment) > 0 {
			lineBuffer = append(lineBuffer, fragment...)
			// Permit at most CRLF beyond the content limit until emit separates
			// the terminator from the actual line data.
			if len(lineBuffer) > maxLineBytes+2 {
				return emitted, operation.Wrap(
					operation.KindLimit,
					"scan_lines",
					"",
					fmt.Errorf("line %d exceeds the %d-byte limit", lineNumber, maxLineBytes),
				)
			}
		}

		switch {
		case err == nil:
			if emitErr := emit(lineBuffer); emitErr != nil {
				return emitted, emitErr
			}
			lineBuffer = lineBuffer[:0]
		case errors.Is(err, bufio.ErrBufferFull):
			continue
		case errors.Is(err, io.EOF):
			if len(lineBuffer) > 0 {
				if emitErr := emit(lineBuffer); emitErr != nil {
					return emitted, emitErr
				}
			} else if emitted == 0 || endedWithNewline {
				if emitErr := emit(nil); emitErr != nil {
					return emitted, emitErr
				}
			}
			return emitted, nil
		default:
			return emitted, err
		}
	}
}
