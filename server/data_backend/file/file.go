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
	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/utils"
	"io"
	"os"
)

type FileBackendConfig struct {
	Directory string
}

func NewFileBackendConfig(config map[string]interface{}) (this *FileBackendConfig) {
	this = new(FileBackendConfig)
	this.Directory = "files" // Default upload directory is ./files
	utils.Assign(this, config)
	return
}

type FileBackend struct {
	Config *FileBackendConfig
}

func NewFileBackend(config map[string]interface{}) (this *FileBackend) {
	this = new(FileBackend)
	this.Config = NewFileBackendConfig(config)
	return
}

func (this *FileBackend) GetFile(ctx *common.PlikContext, upload *common.Upload, id string) (file io.ReadCloser, err error) {
	defer ctx.Finalize(err)

	// Get file path
	directory := this.getDirectoryFromUploadId(upload.Id)
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

func (this *FileBackend) AddFile(ctx *common.PlikContext, upload *common.Upload, file *common.File, fileReader io.Reader) (backendDetails map[string]interface{}, err error) {
	defer ctx.Finalize(err)

	// Get file path
	directory := this.getDirectoryFromUploadId(upload.Id)
	fullPath := directory + "/" + file.Id

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

func (this *FileBackend) RemoveFile(ctx *common.PlikContext, upload *common.Upload, id string) (err error) {
	defer ctx.Finalize(err)

	// Get file path
	fullPath := this.getDirectoryFromUploadId(upload.Id) + "/" + id

	// Remove file
	err = os.Remove(fullPath)
	if err != nil {
		err = ctx.EWarningf("Unable to remove %s : %s", fullPath, err)
		return
	}
	ctx.Infof("File %s successfully removed", fullPath)

	return
}

func (this *FileBackend) RemoveUpload(ctx *common.PlikContext, upload *common.Upload) (err error) {
	defer ctx.Finalize(err)

	// Get upload directory
	fullPath := this.getDirectoryFromUploadId(upload.Id)

	// Remove everything at once
	err = os.RemoveAll(fullPath)
	if err != nil {
		err = ctx.EWarningf("Unable to remove %s : %s", fullPath, err)
		return
	}

	return
}

func (this *FileBackend) getDirectoryFromUploadId(uploadId string) string {
	// To avoid too many files in the same directory
	// data directory is splitted in two levels the
	// first level is the 2 first chars from the upload id
	// it gives 3844 possibilities reaching 65535 files per
	// directory at ~250.000.000 files uploaded.

	return this.Config.Directory + "/" + uploadId[:2] + "/" + uploadId
}
