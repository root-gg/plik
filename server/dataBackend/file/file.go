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
	"io"
	"os"

	"github.com/root-gg/plik/server/common"
)

// Backend object
type Backend struct {
	Config *BackendConfig
}

// NewFileBackend instantiate a new File Data Backend
// from configuration passed as argument
func NewFileBackend(config map[string]interface{}) (fb *Backend) {
	fb = new(Backend)
	fb.Config = NewFileBackendConfig(config)
	return
}

// GetFile implementation for file data backend will search
// on filesystem the asked file and return its reading filehandle
func (fb *Backend) GetFile(ctx *common.PlikContext, upload *common.Upload, id string) (file io.ReadCloser, err error) {
	defer ctx.Finalize(err)

	// Get file path
	directory := fb.getDirectoryFromUploadID(upload.ID)
	fullPath := directory + "/" + id

	// The file content will be piped directly
	// to the client response body
	file, err = os.Open(fullPath)
	if err != nil {
		err = ctx.EWarningf("Unable to open file %s : %s", fullPath, err)
		return
	}

	return
}

// AddFile implementation for file data backend will creates a new file for the given upload
// and save it on filesystem with the given file reader
func (fb *Backend) AddFile(ctx *common.PlikContext, upload *common.Upload, file *common.File, fileReader io.Reader) (backendDetails map[string]interface{}, err error) {
	defer ctx.Finalize(err)

	// Get file path
	directory := fb.getDirectoryFromUploadID(upload.ID)
	fullPath := directory + "/" + file.ID

	// Create directory
	_, err = os.Stat(directory)
	if err != nil {
		err = os.MkdirAll(directory, 0777)
		if err != nil {
			err = ctx.EWarningf("Unable to create upload directory %s : %s", directory, err)
			return
		}
		ctx.Infof("Folder %s successfully created", directory)
	}

	// Create file
	out, err := os.Create(fullPath)
	if err != nil {
		err = ctx.EWarningf("Unable to create file %s : %s", fullPath, err)
		return
	}

	// Copy file data from the client request body
	// to the file system
	_, err = io.Copy(out, fileReader)
	if err != nil {
		err = ctx.EWarningf("Unable to save file %s : %s", fullPath, err)
		return
	}
	ctx.Infof("File %s successfully saved", fullPath)

	return
}

// RemoveFile implementation for file data backend will delete the given
// file from filesystem
func (fb *Backend) RemoveFile(ctx *common.PlikContext, upload *common.Upload, id string) (err error) {
	defer ctx.Finalize(err)

	// Get file path
	fullPath := fb.getDirectoryFromUploadID(upload.ID) + "/" + id

	// Remove file
	err = os.Remove(fullPath)
	if err != nil {
		err = ctx.EWarningf("Unable to remove %s : %s", fullPath, err)
		return
	}
	ctx.Infof("File %s successfully removed", fullPath)

	return
}

// RemoveUpload implementation for file data backend will
// delete the whole upload. Given that an upload is a directory,
// we remove the whole directory at once.
func (fb *Backend) RemoveUpload(ctx *common.PlikContext, upload *common.Upload) (err error) {
	defer ctx.Finalize(err)

	// Get upload directory
	fullPath := fb.getDirectoryFromUploadID(upload.ID)

	// Remove everything at once
	err = os.RemoveAll(fullPath)
	if err != nil {
		err = ctx.EWarningf("Unable to remove %s : %s", fullPath, err)
		return
	}

	return
}

func (fb *Backend) getDirectoryFromUploadID(uploadID string) string {
	// To avoid too many files in the same directory
	// data directory is splitted in two levels the
	// first level is the 2 first chars from the upload id
	// it gives 3844 possibilities reaching 65535 files per
	// directory at ~250.000.000 files uploaded.

	return fb.Config.Directory + "/" + uploadID[:2] + "/" + uploadID
}
