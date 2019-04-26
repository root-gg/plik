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

package stream

import (
	"errors"
	"github.com/root-gg/utils"
	"io"
	"sync"

	"github.com/root-gg/juliet"
	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
	"github.com/root-gg/plik/server/data"
)

// Ensure Stream Data Backend implements data.Backend interface
var _ data.Backend = (*Backend)(nil)

// Config describes configuration for stream data backend
type Config struct {
}

// NewConfig instantiate a new default configuration
// and override it with configuration passed as argument
func NewConfig(params map[string]interface{}) (config *Config) {
	config = new(Config)
	utils.Assign(config, params)
	return
}

// Backend object
type Backend struct {
	Config *Config
	store  map[string]io.ReadCloser
	mu     sync.Mutex
}

// NewBackend instantiate a new Stream Data Backend
// from configuration passed as argument
func NewBackend(config *Config) (b *Backend) {
	b = new(Backend)
	b.Config = config
	b.store = make(map[string]io.ReadCloser)
	return
}

// GetFile implementation for steam data backend will search
// on filesystem the requested steam and return its reading filehandle
func (b *Backend) GetFile(ctx *juliet.Context, upload *common.Upload, id string) (stream io.ReadCloser, err error) {
	log := context.GetLogger(ctx)

	b.mu.Lock()
	defer b.mu.Unlock()

	storeID := upload.ID + "/" + id
	stream, ok := b.store[storeID]
	if !ok {
		err = log.EWarningf("Missing reader")
		return nil, err
	}

	delete(b.store, id)

	return stream, err
}

// AddFile implementation for steam data backend will creates a new steam for the given upload
// and save it on filesystem with the given steam reader
func (b *Backend) AddFile(ctx *juliet.Context, upload *common.Upload, file *common.File, stream io.Reader) (backendDetails map[string]interface{}, err error) {
	log := context.GetLogger(ctx)

	backendDetails = make(map[string]interface{})

	id := upload.ID + "/" + file.ID

	pipeReader, pipeWriter := io.Pipe()

	b.mu.Lock()

	b.store[id] = pipeReader
	defer delete(b.store, id)

	b.mu.Unlock()

	log.Info("Stream data backend waiting for download")

	// This will block until download begins
	_, err = io.Copy(pipeWriter, stream)
	pipeWriter.Close()

	return backendDetails, nil
}

// RemoveFile is not implemented
func (b *Backend) RemoveFile(ctx *juliet.Context, upload *common.Upload, id string) (err error) {
	return errors.New("can't remove stream file")
}

// RemoveUpload is not implemented
func (b *Backend) RemoveUpload(ctx *juliet.Context, upload *common.Upload) (err error) {
	return errors.New("can't remove stream upload")
}
