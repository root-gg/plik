/**

    Plik upload server

The MIT License (MIT)

Copyright (c) <2015> Copyright holders list can be found in AUTHORS file
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

package common

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/root-gg/plik/server/Godeps/_workspace/src/github.com/root-gg/context"
	"github.com/root-gg/plik/server/Godeps/_workspace/src/github.com/root-gg/logger"
)

var rootLogger = logger.NewLogger()
var rootContext = newRootContext()

// RootContext is a shortcut to get rootContext
func RootContext() *PlikContext {
	return rootContext
}

// Log is a shortcut to get rootLogger
func Log() *logger.Logger {
	return rootLogger
}

// PlikContext is a root-gg logger && logger object
type PlikContext struct {
	*context.Context
	*logger.Logger
}

func newRootContext() (ctx *PlikContext) {
	ctx = new(PlikContext)
	ctx.Context = context.NewContext("ROOT")
	ctx.Logger = rootLogger
	return
}

// NewPlikContext creates a new plik context forked from root logger/context
func NewPlikContext(name string, req *http.Request) (ctx *PlikContext) {
	ctx = new(PlikContext)
	ctx.Context = rootContext.Context.Fork(name).AutoDetach()
	ctx.Logger = rootContext.Logger.Copy()

	// TODO X-FORWARDED-FOR
	remoteAddr := strings.Split(req.RemoteAddr, ":")
	if len(remoteAddr) > 0 {
		ctx.Set("RemoteIp", remoteAddr[0])
	}

	ctx.UpdateLoggerPrefix("")
	return
}

// Fork context and copy logger
func (ctx *PlikContext) Fork(name string) (fork *PlikContext) {
	fork = new(PlikContext)
	fork.Context = ctx.Context.Fork(name)
	fork.Logger = ctx.Logger.Copy()
	return fork
}

// SetUpload is used to display upload id in logger prefix and set it in context
func (ctx *PlikContext) SetUpload(uploadID string) *PlikContext {
	ctx.Set("UploadId", uploadID)
	ctx.UpdateLoggerPrefix("")
	return ctx
}

// SetFile is used to display file id in logger prefix and set it in context
func (ctx *PlikContext) SetFile(fileName string) *PlikContext {
	ctx.Set("FileName", fileName)
	ctx.UpdateLoggerPrefix("")
	return ctx
}

// UpdateLoggerPrefix sets a new prefix for the context logger
func (ctx *PlikContext) UpdateLoggerPrefix(prefix string) {
	str := ""
	if ip, ok := ctx.Get("RemoteIp"); ok {
		str += fmt.Sprintf("[%s]", ip)
	}
	if uploadID, ok := ctx.Get("UploadId"); ok {
		str += fmt.Sprintf("[%s]", uploadID)
	}
	if fileName, ok := ctx.Get("FileName"); ok {
		str += fmt.Sprintf("[%s]", fileName)
	}
	ctx.SetPrefix(str + prefix)
}
