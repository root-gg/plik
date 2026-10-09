package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestCountingResponseWriterCountsWrites verifies that every byte written
// through the wrapper is added to the running total, including across multiple
// writes, and that the bytes still reach the underlying writer unchanged.
func TestCountingResponseWriterCountsWrites(t *testing.T) {
	rr := httptest.NewRecorder()
	cw := newCountingResponseWriter(rr)

	n, err := cw.Write([]byte("hello"))
	require.NoError(t, err)
	require.Equal(t, 5, n)

	n, err = cw.Write([]byte(" world"))
	require.NoError(t, err)
	require.Equal(t, 6, n)

	require.Equal(t, int64(11), cw.written, "counter must sum every byte written")
	require.Equal(t, "hello world", rr.Body.String(), "bytes must pass through unchanged")
}

// TestCountingResponseWriterHeaderPassthrough verifies that Header/WriteHeader
// delegate to the wrapped writer so handlers can set headers and status through
// the wrapper transparently.
func TestCountingResponseWriterHeaderPassthrough(t *testing.T) {
	rr := httptest.NewRecorder()
	cw := newCountingResponseWriter(rr)

	cw.Header().Set("X-Test", "value")
	cw.WriteHeader(http.StatusPartialContent)

	require.Equal(t, "value", rr.Header().Get("X-Test"))
	require.Equal(t, http.StatusPartialContent, rr.Code)
	require.Equal(t, int64(0), cw.written, "WriteHeader must not count body bytes")
}

// TestCountingResponseWriterFlushPassthrough verifies Flush is forwarded to a
// wrapped writer that implements http.Flusher (httptest.ResponseRecorder does).
func TestCountingResponseWriterFlushPassthrough(t *testing.T) {
	rr := httptest.NewRecorder()
	cw := newCountingResponseWriter(rr)

	// The wrapper must advertise http.Flusher so net/http keeps flushing.
	flusher, ok := any(cw).(http.Flusher)
	require.True(t, ok, "countingResponseWriter must implement http.Flusher")

	flusher.Flush()
	require.True(t, rr.Flushed, "Flush must pass through to the underlying writer")
}

// nonFlusherWriter is a minimal ResponseWriter with no Flush method, used to
// prove the wrapper's Flush is a safe no-op when the wrapped writer cannot flush.
type nonFlusherWriter struct {
	header http.Header
}

func (w *nonFlusherWriter) Header() http.Header {
	if w.header == nil {
		w.header = http.Header{}
	}
	return w.header
}
func (w *nonFlusherWriter) Write(p []byte) (int, error) { return len(p), nil }
func (w *nonFlusherWriter) WriteHeader(int)             {}

// TestCountingResponseWriterFlushNoopWhenUnsupported verifies Flush does not
// panic when the wrapped writer is not an http.Flusher.
func TestCountingResponseWriterFlushNoopWhenUnsupported(t *testing.T) {
	cw := newCountingResponseWriter(&nonFlusherWriter{})
	require.NotPanics(t, func() { cw.Flush() })
}

// TestCountingResponseWriterHijackUnsupported verifies Hijack reports
// ErrNotSupported when the wrapped writer cannot be hijacked (the common case
// for httptest and the middleware status wrapper).
func TestCountingResponseWriterHijackUnsupported(t *testing.T) {
	cw := newCountingResponseWriter(httptest.NewRecorder())
	_, _, err := cw.Hijack()
	require.ErrorIs(t, err, http.ErrNotSupported)
}
