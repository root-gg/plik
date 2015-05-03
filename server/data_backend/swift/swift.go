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
	"github.com/ncw/swift"
	"github.com/root-gg/logger"
	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/utils"
	"io"
)

type SwiftBackend struct {
	config     *configInfo
	connection swift.Connection
}

type configInfo struct {
	Username, Password, Host, ProjectName, Container string
}

var log *logger.Logger

func NewSwiftBackend(config map[string]interface{}) (this *SwiftBackend) {
	this = new(SwiftBackend)
	this.config = new(configInfo)
	this.config.Container = "PlickData"
	utils.Assign(this.config, config)
	return this
}

func (this *SwiftBackend) auth(ctx *common.PlikContext) (err error) {
	timer := ctx.Time("auth")
	defer timer.Stop()

	if this.connection.Authenticated() {
		return
	}

	connection := swift.Connection{
		UserName: this.config.Username,
		ApiKey:   this.config.Password,
		AuthUrl:  this.config.Host,
		Tenant:   this.config.ProjectName,
	}

	// Authenticate
	err = connection.Authenticate()
	if err != nil {
		err = log.EWarningf("Unable to autenticate : %s", err)
		return err
	}
	this.connection = connection

	// Create container
	this.connection.ContainerCreate(this.config.Container, nil)

	return
}

func (this *SwiftBackend) GetFile(ctx *common.PlikContext, upload *common.Upload, fileId string) (reader io.ReadCloser, err error) {
	defer func() {
		if err != nil {
			ctx.Finalize(err)
		}
	}() // Finalize the context only if error, else let it be finalized by the download goroutine

	err = this.auth(ctx)
	if err != nil {
		return
	}

	reader, pipeWriter := io.Pipe()
	uuid := this.getFileId(upload, fileId)
	go func() {
		defer ctx.Finalize(err)
		_, err = this.connection.ObjectGet(this.config.Container, uuid, pipeWriter, true, nil)
		defer pipeWriter.Close()
		if err != nil {
			err = ctx.EWarningf("Unable to get object %s : %s", uuid, err)
			return
		}
	}()

	return
}

func (this *SwiftBackend) AddFile(ctx *common.PlikContext, upload *common.Upload, file *common.File, fileReader io.Reader) (backendDetails map[string]interface{}, err error) {
	defer ctx.Finalize(err)

	err = this.auth(ctx)
	if err != nil {
		return
	}

	uuid := this.getFileId(upload, file.Id)
	object, err := this.connection.ObjectCreate(this.config.Container, uuid, true, "", "", nil)

	_, err = io.Copy(object, fileReader)
	if err != nil {
		err = ctx.EWarningf("Unable to save object %s : %s", uuid, err)
		return
	}
	object.Close()
	ctx.Infof("Object %s successfully saved", uuid)

	return
}

func (this *SwiftBackend) RemoveFile(ctx *common.PlikContext, upload *common.Upload, fileId string) (err error) {
	defer ctx.Finalize(err)

	err = this.auth(ctx)
	if err != nil {
		return
	}

	uuid := this.getFileId(upload, fileId)
	err = this.connection.ObjectDelete(this.config.Container, uuid)
	if err != nil {
		err = ctx.EWarningf("Unable to remove object %s : %s", uuid, err)
		return
	}

	return
}

func (this *SwiftBackend) RemoveUpload(ctx *common.PlikContext, upload *common.Upload) (err error) {
	defer ctx.Finalize(err)

	err = this.auth(ctx)
	if err != nil {
		return
	}

	for fileId, _ := range upload.Files {
		uuid := this.getFileId(upload, fileId)
		err = this.connection.ObjectDelete(this.config.Container, uuid)
		if err != nil {
			err = ctx.EWarningf("Unable to remove object %s : %s", uuid, err)
		}
	}

	return
}

func (bf *SwiftBackend) getFileId(upload *common.Upload, fileId string) string {
	return upload.Id + "." + fileId
}
