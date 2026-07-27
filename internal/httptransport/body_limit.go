package httptransport

import (
	"errors"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
)

var errBodyTooLarge = errors.New("HTTP request body exceeds configured limit")

type byteBudget struct {
	mu       sync.Mutex
	capacity int64
	used     int64
}

func newByteBudget(capacity int64) *byteBudget {
	return &byteBudget{capacity: capacity}
}

func (budget *byteBudget) tryAcquire(amount int64) bool {
	if amount <= 0 {
		return true
	}
	budget.mu.Lock()
	defer budget.mu.Unlock()
	if amount > budget.capacity-budget.used {
		return false
	}
	budget.used += amount
	return true
}

func (budget *byteBudget) release(amount int64) {
	if amount <= 0 {
		return
	}
	budget.mu.Lock()
	defer budget.mu.Unlock()
	budget.used -= amount
	if budget.used < 0 {
		budget.used = 0
	}
}

type boundedReadCloser struct {
	source    io.ReadCloser
	remaining int64
	exceeded  atomic.Bool
}

func newBoundedReadCloser(source io.ReadCloser, limit int64) *boundedReadCloser {
	if source == nil {
		source = http.NoBody
	}
	return &boundedReadCloser{source: source, remaining: limit}
}

func (body *boundedReadCloser) Read(buffer []byte) (int, error) {
	if body.remaining == 0 {
		var probe [1]byte
		count, err := body.source.Read(probe[:])
		if count > 0 {
			body.exceeded.Store(true)
			return 0, errBodyTooLarge
		}
		return 0, err
	}
	if int64(len(buffer)) > body.remaining {
		buffer = buffer[:body.remaining]
	}
	count, err := body.source.Read(buffer)
	body.remaining -= int64(count)
	return count, err
}

func (body *boundedReadCloser) Close() error {
	return body.source.Close()
}

type bodyLimitResponseWriter struct {
	http.ResponseWriter
	body               *boundedReadCloser
	translated         bool
	replacementWritten bool
}

func newBodyLimitResponseWriter(writer http.ResponseWriter, body *boundedReadCloser) *bodyLimitResponseWriter {
	return &bodyLimitResponseWriter{ResponseWriter: writer, body: body}
}

func (writer *bodyLimitResponseWriter) WriteHeader(status int) {
	if status == http.StatusBadRequest && writer.body.exceeded.Load() {
		writer.translated = true
		writer.Header().Del("Content-Length")
		writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
		writer.ResponseWriter.WriteHeader(http.StatusRequestEntityTooLarge)
		return
	}
	writer.ResponseWriter.WriteHeader(status)
}

func (writer *bodyLimitResponseWriter) Write(payload []byte) (int, error) {
	if writer.body.exceeded.Load() && !writer.translated {
		writer.WriteHeader(http.StatusBadRequest)
	}
	if writer.translated {
		if !writer.replacementWritten {
			writer.replacementWritten = true
			_, err := io.WriteString(writer.ResponseWriter, "Request Entity Too Large\n")
			return len(payload), err
		}
		return len(payload), nil
	}
	return writer.ResponseWriter.Write(payload)
}

func (writer *bodyLimitResponseWriter) Flush() {
	_ = http.NewResponseController(writer.ResponseWriter).Flush()
}

func (writer *bodyLimitResponseWriter) Unwrap() http.ResponseWriter {
	return writer.ResponseWriter
}
