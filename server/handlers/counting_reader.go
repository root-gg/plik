package handlers

import (
	"io"
	"sync/atomic"
)

// countingReader wraps an io.Reader and counts the number of bytes actually read
// through it. AddFile uses it to measure upload wire bytes (ingress): it wraps the
// multipart file "part" reader — which already strips multipart framing — so the
// count is exactly the file content bytes that flow to the preprocessor and on to
// the data backend, excluding framing overhead. Because the count reflects what
// Read actually returned, a client that disconnects mid-transfer records exactly
// the bytes that reached the server — the correct partial ingress figure,
// symmetric with countingResponseWriter's egress count.
//
// The count is stored in an atomic int64: the reader is consumed by the
// preprocessor goroutine while AddFile reads the count at its various return
// points (including the backend-error path where the preprocessor may still be
// running against an abandoned pipe), so the load must be race-free.
type countingReader struct {
	r     io.Reader
	count atomic.Int64
}

// newCountingReader wraps r so bytes read through it are counted.
func newCountingReader(r io.Reader) *countingReader {
	return &countingReader{r: r}
}

// Read reads from the wrapped reader and adds the bytes actually read to the
// running total. Partial reads count only what was returned.
func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	if n > 0 {
		c.count.Add(int64(n))
	}
	return n, err
}

// bytesRead returns the number of bytes read so far, race-free.
func (c *countingReader) bytesRead() int64 {
	return c.count.Load()
}
