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
	"github.com/root-gg/plik/server/context"
	"io"

	"github.com/root-gg/juliet"
	"github.com/root-gg/plik/server/common"
)

// Backend object
type Backend struct {
	Config *BackendConfig
	Store  map[string]io.ReadCloser
}

// NewStreamBackend instantiate a new Stream Data Backend
// from configuration passed as argument
func NewStreamBackend(config *BackendConfig) (sb *Backend) {
	sb = new(Backend)
	sb.Config = config
	sb.Store = make(map[string]io.ReadCloser)
	return
}

// GetFile implementation for steam data backend will search
// on filesystem the requested steam and return its reading filehandle
func (sb *Backend) GetFile(ctx *juliet.Context, upload *common.Upload, id string) (stream io.ReadCloser, err error) {
	log := context.GetLogger(ctx)
	storeID := upload.ID + "/" + id
	stream, ok := sb.Store[storeID]
	if !ok {
		err = log.EWarningf("Missing reader")
	}
	delete(sb.Store, id)
	return
}

// AddFile implementation for steam data backend will creates a new steam for the given upload
// and save it on filesystem with the given steam reader
func (sb *Backend) AddFile(ctx *juliet.Context, upload *common.Upload, file *common.File, stream io.Reader) (backendDetails map[string]interface{}, err error) {
	log := context.GetLogger(ctx)
	backendDetails = make(map[string]interface{})
	id := upload.ID + "/" + file.ID
	pipeReader, pipeWriter := io.Pipe()
	sb.Store[id] = pipeReader
	defer delete(sb.Store, id)
	log.Info("Stream data backend waiting for download")
	// This will block until download begins
	_, err = io.Copy(pipeWriter, stream)
	pipeWriter.Close()
	return
}

// RemoveFile is not implemented
func (sb *Backend) RemoveFile(ctx *juliet.Context, upload *common.Upload, id string) (err error) {
	return
}

// RemoveUpload is not implemented
func (sb *Backend) RemoveUpload(ctx *juliet.Context, upload *common.Upload) (err error) {
	return
}
