package textstream

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

const lineEndingInputBufferSize = 64 * 1024

type LineEndingStats struct {
	CRLFCount int
	LFCount   int
}

type LineEndingReader struct {
	source       io.Reader
	targetCRLF   bool
	utf16        bool
	littleEndian bool
	stats        LineEndingStats

	input          [lineEndingInputBufferSize]byte
	output         []byte
	outputOffset   int
	pendingErr     error
	pendingCR      bool
	pendingByte    byte
	hasPendingByte bool
}

func NewByteLineEndingReader(source io.Reader, target string) (*LineEndingReader, error) {
	targetCRLF, err := parseLineEndingTarget(target)
	if err != nil {
		return nil, err
	}
	return &LineEndingReader{source: source, targetCRLF: targetCRLF}, nil
}

func NewUTF16LineEndingReader(source io.Reader, target string, littleEndian bool) (*LineEndingReader, error) {
	targetCRLF, err := parseLineEndingTarget(target)
	if err != nil {
		return nil, err
	}
	return &LineEndingReader{source: source, targetCRLF: targetCRLF, utf16: true, littleEndian: littleEndian}, nil
}

func parseLineEndingTarget(target string) (bool, error) {
	switch target {
	case "lf":
		return false, nil
	case "crlf":
		return true, nil
	default:
		return false, fmt.Errorf("invalid line-ending target %q", target)
	}
}

func (reader *LineEndingReader) Stats() LineEndingStats {
	return reader.stats
}

func (reader *LineEndingReader) Read(buffer []byte) (int, error) {
	for reader.outputOffset == len(reader.output) && reader.pendingErr == nil {
		reader.output = reader.output[:0]
		reader.outputOffset = 0
		reader.fill()
	}
	if reader.outputOffset < len(reader.output) {
		read := copy(buffer, reader.output[reader.outputOffset:])
		reader.outputOffset += read
		return read, nil
	}
	if reader.pendingErr != nil {
		err := reader.pendingErr
		reader.pendingErr = nil
		return 0, err
	}
	return 0, io.EOF
}

func (reader *LineEndingReader) fill() {
	read, err := reader.source.Read(reader.input[:])
	if read > 0 {
		if reader.utf16 {
			reader.transformUTF16(reader.input[:read])
		} else {
			reader.transformBytes(reader.input[:read])
		}
	}

	if err != nil {
		if errors.Is(err, io.EOF) {
			reader.finish()
			if reader.pendingErr == nil {
				reader.pendingErr = io.EOF
			}
		} else {
			reader.pendingErr = err
		}
		return
	}
	if read == 0 {
		reader.pendingErr = io.ErrNoProgress
	}
}

func (reader *LineEndingReader) transformBytes(data []byte) {
	for _, current := range data {
		if reader.pendingCR {
			if current == '\n' {
				reader.stats.CRLFCount++
				reader.appendTargetEnding()
				reader.pendingCR = false
				continue
			}
			reader.output = append(reader.output, '\r')
			reader.pendingCR = false
		}

		switch current {
		case '\r':
			reader.pendingCR = true
		case '\n':
			reader.stats.LFCount++
			reader.appendTargetEnding()
		default:
			reader.output = append(reader.output, current)
		}
	}
}

func (reader *LineEndingReader) transformUTF16(data []byte) {
	for _, current := range data {
		if !reader.hasPendingByte {
			reader.pendingByte = current
			reader.hasPendingByte = true
			continue
		}
		first := reader.pendingByte
		reader.hasPendingByte = false
		var pair [2]byte
		pair[0], pair[1] = first, current
		var order binary.ByteOrder = binary.BigEndian
		if reader.littleEndian {
			order = binary.LittleEndian
		}
		reader.transformUnit(order.Uint16(pair[:]))
	}
}

func (reader *LineEndingReader) transformUnit(unit uint16) {
	if reader.pendingCR {
		if unit == '\n' {
			reader.stats.CRLFCount++
			reader.appendTargetUTF16Ending()
			reader.pendingCR = false
			return
		}
		reader.appendUTF16Unit('\r')
		reader.pendingCR = false
	}

	switch unit {
	case '\r':
		reader.pendingCR = true
	case '\n':
		reader.stats.LFCount++
		reader.appendTargetUTF16Ending()
	default:
		reader.appendUTF16Unit(unit)
	}
}

func (reader *LineEndingReader) appendTargetEnding() {
	if reader.targetCRLF {
		reader.output = append(reader.output, '\r')
	}
	reader.output = append(reader.output, '\n')
}

func (reader *LineEndingReader) appendTargetUTF16Ending() {
	if reader.targetCRLF {
		reader.appendUTF16Unit('\r')
	}
	reader.appendUTF16Unit('\n')
}

func (reader *LineEndingReader) appendUTF16Unit(unit uint16) {
	var pair [2]byte
	if reader.littleEndian {
		binary.LittleEndian.PutUint16(pair[:], unit)
	} else {
		binary.BigEndian.PutUint16(pair[:], unit)
	}
	reader.output = append(reader.output, pair[:]...)
}

func (reader *LineEndingReader) finish() {
	if reader.utf16 && reader.hasPendingByte {
		reader.pendingErr = fmt.Errorf("invalid UTF-16 byte length: trailing byte")
		reader.hasPendingByte = false
	}
	if reader.pendingCR {
		if reader.utf16 {
			reader.appendUTF16Unit('\r')
		} else {
			reader.output = append(reader.output, '\r')
		}
		reader.pendingCR = false
	}
}
