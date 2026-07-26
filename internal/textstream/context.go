package textstream

import (
	"context"
	"io"
)

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

// WithContext returns a reader that stops before the next underlying read when
// ctx is cancelled. It does not take ownership of reader.
func WithContext(ctx context.Context, reader io.Reader) io.Reader {
	return &contextReader{ctx: ctx, reader: reader}
}

func (reader *contextReader) Read(buffer []byte) (int, error) {
	if err := reader.ctx.Err(); err != nil {
		return 0, err
	}
	return reader.reader.Read(buffer)
}
