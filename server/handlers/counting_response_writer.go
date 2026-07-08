package handlers

import (
	"bufio"
	"net"
	"net/http"
)

// countingResponseWriter wraps an http.ResponseWriter and counts the number of
// body bytes actually written to the client. The download handlers use it to
// measure egress (bytes served) independently of the logical download-event
// count: a full GET, a byte-0 range, and a mid-range GET all serve bytes, but
// only the first two are download events (see the Download Counting Policy in
// server/ARCHITECTURE.md). Because the count reflects what Write actually wrote,
// a client that disconnects mid-stream records exactly the bytes that reached
// it — the correct egress figure.
//
// It intentionally does NOT implement io.ReaderFrom. Go's io.Copy /
// http.ServeContent fall back to the Write path when the destination is not an
// io.ReaderFrom, which is what lets us count every byte. This is correct on
// every code path; it is not a performance regression here because the writer
// this wraps (the middleware's statusCodeResponseWriter) does not expose the
// underlying socket's ReadFrom either, so there is no sendfile fast path to
// preserve. Flush and Hijack are passed through when the wrapped writer supports
// them so streaming (chunked) responses and connection hijacking keep working.
type countingResponseWriter struct {
	http.ResponseWriter
	written int64
}

// newCountingResponseWriter wraps resp so body bytes written through it are counted.
func newCountingResponseWriter(resp http.ResponseWriter) *countingResponseWriter {
	return &countingResponseWriter{ResponseWriter: resp}
}

// Write writes p to the underlying writer and adds the bytes actually written to
// the running total. Partial writes (n < len(p)) count only what was written.
func (w *countingResponseWriter) Write(p []byte) (int, error) {
	n, err := w.ResponseWriter.Write(p)
	w.written += int64(n)
	return n, err
}

// Flush passes through to the underlying writer when it supports http.Flusher so
// streamed/chunked responses keep flushing. It is a no-op otherwise.
func (w *countingResponseWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Hijack passes through to the underlying writer when it supports http.Hijacker.
// It returns http.ErrNotSupported otherwise, matching the standard library's
// contract for a non-hijackable ResponseWriter.
func (w *countingResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := w.ResponseWriter.(http.Hijacker); ok {
		return h.Hijack()
	}
	return nil, nil, http.ErrNotSupported
}
