/**

    Plik upload server

The MIT License (MIT)

Copyright (c) <2015>
	- Mathieu Bodjikian <mathieu@bodjikian.fr>
	- Charles-Antoine Mathieu <skatkatt@root.gg>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
**/

package file

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"sync"

	"github.com/root-gg/juliet"
	"github.com/root-gg/plik/server/common"
)

// Backend object
type Backend struct {
	files map[string]*bytes.Buffer
	err   error
	mu    sync.Mutex
}

// NewBackend instantiate a new Testing Data Backend
// from configuration passed as argument
func NewBackend() (b *Backend) {
	b = new(Backend)
	b.files = make(map[string]*bytes.Buffer)
	return
}

// GetFile implementation for testing data backend will search
// on filesystem the asked file and return its reading filehandle
func (b *Backend) GetFile(ctx *juliet.Context, upload *common.Upload, id string) (file io.ReadCloser, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.err != nil {
		return nil, b.err
	}

	if file, ok := b.files[id]; ok {
		return ioutil.NopCloser(file), nil
	}

	return nil, errors.New("File not found")
}

// AddFile implementation for testing data backend will creates a new file for the given upload
// and save it on filesystem with the given file reader
func (b *Backend) AddFile(ctx *juliet.Context, upload *common.Upload, file *common.File, fileReader io.Reader) (backendDetails map[string]interface{}, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.err != nil {
		return nil, b.err
	}

	if _, ok := b.files[file.ID]; ok {
		return nil, errors.New("File exists")
	}

	data, err := ioutil.ReadAll(fileReader)
	if err != nil {
		return nil, err
	}

	b.files[file.ID] = bytes.NewBuffer(data)

	return nil, nil
}

// RemoveFile implementation for testing data backend will delete the given
// file from filesystem
func (b *Backend) RemoveFile(ctx *juliet.Context, upload *common.Upload, id string) (err error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.err != nil {
		return b.err
	}

	delete(b.files, id)

	return nil
}

// RemoveUpload implementation for testing data backend will
// delete the whole upload. Given that an upload is a directory,
// we remove the whole directory at once.
func (b *Backend) RemoveUpload(ctx *juliet.Context, upload *common.Upload) (err error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.err != nil {
		return b.err
	}

	for id := range upload.Files {
		delete(b.files, id)
	}

	return nil
}

// SetError set the error that this backend will return on any subsequent method call
func (b *Backend) SetError(err error) {
	b.err = err
}
