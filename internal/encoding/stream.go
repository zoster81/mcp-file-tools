package encoding

import (
	"fmt"
	"io"

	"golang.org/x/text/transform"
)

// NewDecoderReader returns a UTF-8 reader for source encoded as charset.
// UTF-8-compatible inputs are passed through byte-for-byte; stateful and
// multibyte decoders preserve incomplete sequences across underlying reads.
func NewDecoderReader(source io.Reader, charset string) (io.Reader, error) {
	registered, ok := Get(charset)
	if !ok {
		return nil, fmt.Errorf("unsupported encoding: %s", charset)
	}
	if IsUTF8(charset) {
		return source, nil
	}
	return transform.NewReader(source, registered.NewDecoder()), nil
}

// NewEncoderReader returns a reader that transforms UTF-8 source bytes into
// charset incrementally. Encoding failures are surfaced by Read.
func NewEncoderReader(source io.Reader, charset string) (io.Reader, error) {
	registered, ok := Get(charset)
	if !ok {
		return nil, fmt.Errorf("unsupported encoding: %s", charset)
	}
	if IsUTF8(charset) {
		return source, nil
	}
	return transform.NewReader(source, registered.NewEncoder()), nil
}
