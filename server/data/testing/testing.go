package file

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"sync"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/data"
)

// Ensure Testing Data Backend implements data.Backend interface
var _ data.Backend = (*Backend)(nil)

// Backend object
type Backend struct {
	files map[string][]byte
	err   error
	mu    sync.Mutex
}

// NewBackend instantiate a new Testing Data Backend
// from configuration passed as argument
func NewBackend() (b *Backend) {
	b = new(Backend)
	b.files = make(map[string][]byte)
	return
}

// GetFiles return the content of the backend for testing purposes
func (b *Backend) GetFiles() (files map[string][]byte) {
	return b.files
}

// GetFile implementation for testing data backend will search
// on filesystem the asked file and return its reading filehandle
func (b *Backend) GetFile(file *common.File) (reader io.ReadCloser, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.err != nil {
		return nil, b.err
	}

	if content, ok := b.files[file.ID]; ok {
		return ioutil.NopCloser(bytes.NewBuffer(content)), nil
	}

	return nil, errors.New("file not found")
}

// AddFile implementation for testing data backend will creates a new file for the given upload
// and save it on filesystem with the given file reader
func (b *Backend) AddFile(file *common.File, fileReader io.Reader) (err error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.err != nil {
		return b.err
	}

	if _, ok := b.files[file.ID]; ok {
		return errors.New("file exists")
	}

	content, err := ioutil.ReadAll(fileReader)
	if err != nil {
		return err
	}

	b.files[file.ID] = content

	return nil
}

// RemoveFile implementation for testing data backend will delete the given
// file from filesystem
func (b *Backend) RemoveFile(file *common.File) (err error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.err != nil {
		return b.err
	}

	delete(b.files, file.ID)

	return nil
}

// SetError set the error that this backend will return on any subsequent method call
func (b *Backend) SetError(err error) {
	b.err = err
}
