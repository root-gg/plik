package stream

import (
	"fmt"
	"io"
	"sync"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/data"
)

// Ensure Stream Data Backend implements data.Backend interface
var _ data.Backend = (*Backend)(nil)

// Backend object
type Backend struct {
	store map[string]io.ReadCloser
	mu    sync.Mutex
}

// NewBackend instantiate a new Stream Data Backend
// from configuration passed as argument
func NewBackend() (b *Backend) {
	b = new(Backend)
	b.store = make(map[string]io.ReadCloser)
	return
}

// GetFile implementation for steam data backend will search
// on filesystem the requested steam and return its reading filehandle
func (b *Backend) GetFile(file *common.File) (stream io.ReadCloser, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	storeID := file.UploadID + "/" + file.ID
	stream, ok := b.store[storeID]
	if !ok {
		return nil, fmt.Errorf("missing reader")
	}

	delete(b.store, storeID)

	return stream, err
}

// AddFile implementation for steam data backend will creates a new steam for the given upload
// and save it on filesystem with the given steam reader
func (b *Backend) AddFile(file *common.File, stream io.Reader) (err error) {
	storeID := file.UploadID + "/" + file.ID

	pipeReader, pipeWriter := io.Pipe()

	b.mu.Lock()

	b.store[storeID] = pipeReader
	defer delete(b.store, storeID)

	b.mu.Unlock()

	// This will block until download begins
	_, err = io.Copy(pipeWriter, stream)
	_ = pipeWriter.Close()

	return nil
}

// RemoveFile does not need to be implemented cleaning occurs in AddFile's defer delete
func (b *Backend) RemoveFile(file *common.File) (err error) {
	return nil
}
