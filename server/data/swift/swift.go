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

package swift

import (
	"github.com/root-gg/utils"
	"io"

	"github.com/ncw/swift"
	"github.com/root-gg/juliet"
	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
	"github.com/root-gg/plik/server/data"
)

// Ensure Swift Data Backend implements data.Backend interface
var _ data.Backend = (*Backend)(nil)

// Config describes configuration for Swift data backend
type Config struct {
	Username, Password, Host, ProjectName, Container string
}

// NewConfig instantiate a new default configuration
// and override it with configuration passed as argument
func NewConfig(params map[string]interface{}) (config *Config) {
	config = new(Config)
	config.Container = "plik"
	utils.Assign(config, params)
	return
}

// Backend object
type Backend struct {
	config     *Config
	connection *swift.Connection
}

// NewBackend instantiate a new OpenSwift Data Backend
// from configuration passed as argument
func NewBackend(config *Config) (b *Backend) {
	b = new(Backend)
	b.config = config
	return b
}

// GetFile implementation for Swift Data Backend
func (b *Backend) GetFile(ctx *juliet.Context, upload *common.Upload, fileID string) (reader io.ReadCloser, err error) {
	log := context.GetLogger(ctx)

	err = b.auth(ctx)
	if err != nil {
		return
	}

	reader, pipeWriter := io.Pipe()
	uuid := b.getFileID(upload, fileID)
	go func() {
		_, err = b.connection.ObjectGet(b.config.Container, uuid, pipeWriter, true, nil)
		defer pipeWriter.Close()
		if err != nil {
			err = log.EWarningf("Unable to get object %s : %s", uuid, err)
			return
		}
	}()

	return
}

// AddFile implementation for Swift Data Backend
func (b *Backend) AddFile(ctx *juliet.Context, upload *common.Upload, file *common.File, fileReader io.Reader) (backendDetails map[string]interface{}, err error) {
	log := context.GetLogger(ctx)

	err = b.auth(ctx)
	if err != nil {
		return
	}

	uuid := b.getFileID(upload, file.ID)
	object, err := b.connection.ObjectCreate(b.config.Container, uuid, true, "", "", nil)

	_, err = io.Copy(object, fileReader)
	if err != nil {
		err = log.EWarningf("Unable to save object %s : %s", uuid, err)
		return
	}
	object.Close()
	log.Infof("Object %s successfully saved", uuid)

	return
}

// RemoveFile implementation for Swift Data Backend
func (b *Backend) RemoveFile(ctx *juliet.Context, upload *common.Upload, fileID string) (err error) {
	log := context.GetLogger(ctx)

	err = b.auth(ctx)
	if err != nil {
		return
	}

	uuid := b.getFileID(upload, fileID)
	err = b.connection.ObjectDelete(b.config.Container, uuid)
	if err != nil {
		err = log.EWarningf("Unable to remove object %s : %s", uuid, err)
		return
	}

	return
}

// RemoveUpload implementation for Swift Data Backend
// Iterates on each upload file and call removeFile
func (b *Backend) RemoveUpload(ctx *juliet.Context, upload *common.Upload) (err error) {
	log := context.GetLogger(ctx)

	err = b.auth(ctx)
	if err != nil {
		return
	}

	for fileID := range upload.Files {
		uuid := b.getFileID(upload, fileID)
		err = b.connection.ObjectDelete(b.config.Container, uuid)
		if err != nil {
			err = log.EWarningf("Unable to remove object %s : %s", uuid, err)
		}
	}

	return
}

func (b *Backend) getFileID(upload *common.Upload, fileID string) string {
	return upload.ID + "." + fileID
}

func (b *Backend) auth(ctx *juliet.Context) (err error) {
	log := context.GetLogger(ctx)

	if b.connection != nil && b.connection.Authenticated() {
		return
	}

	connection := &swift.Connection{
		UserName: b.config.Username,
		ApiKey:   b.config.Password,
		AuthUrl:  b.config.Host,
		Tenant:   b.config.ProjectName,
	}

	// Authenticate
	err = connection.Authenticate()
	if err != nil {
		err = log.EWarningf("Unable to autenticate : %s", err)
		return err
	}
	b.connection = connection

	// Create container
	b.connection.ContainerCreate(b.config.Container, nil)

	return
}
