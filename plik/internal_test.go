/* The MIT License (MIT)

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
THE SOFTWARE. */

package plik

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"os"
	"testing"

	"github.com/root-gg/plik/server/context"

	"github.com/stretchr/testify/require"

	"github.com/root-gg/plik/server/common"
)

func TestCreateUpload(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer shutdown(ps)

	err := start(ps)
	require.NoError(t, err, "unable to start plik server")

	upload := &common.Upload{}
	uploadParams, err := pc.create(upload)
	require.NoError(t, err, "unable to create upload")
	require.NotNil(t, uploadParams, "invalid nil uploads params")
	require.NotZero(t, uploadParams.ID, "invalid nil error params")
}

func TestCreateUploadInvalidParams(t *testing.T) {
	_, pc := newPlikServerAndClient()

	_, err := pc.create(nil)
	common.RequireError(t, err, "missing upload params")

	pc.URL = string([]byte{0})

	_, err = pc.create(&common.Upload{})
	common.RequireError(t, err, "")
}

func TestCreateUploadAPIFail(t *testing.T) {
	_, pc := newPlikServerAndClient()

	_, err := pc.create(&common.Upload{})
	common.RequireError(t, err, "connection refused")

	shutdown, err := common.StartAPIMockServer(common.DummyHandler)
	require.NoError(t, err, "unable to start plik server")
	defer shutdown()

	_, err = pc.create(&common.Upload{})
	common.RequireError(t, err, "")
}

func TestCreateUploadInvalidJSON(t *testing.T) {
	_, pc := newPlikServerAndClient()

	_, err := pc.GetServerVersion()
	common.RequireError(t, err, "connection refused")

	handler := http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		resp.Write([]byte("invalid json"))
	})

	shutdown, err := common.StartAPIMockServer(handler)
	defer shutdown()
	require.NoError(t, err, "unable to start HTTP server server")

	_, err = pc.create(&common.Upload{})
	common.RequireError(t, err, "invalid character 'i' looking for beginning of value")
}

func TestUploadFileNoUpload(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer shutdown(ps)

	err := start(ps)
	require.NoError(t, err, "unable to start plik server")

	err = common.CheckHTTPServer(ps.GetConfig().ListenPort)
	require.NoError(t, err, "server unreachable")

	upload := &common.Upload{}
	file := upload.NewFile()
	file.Name = "filename"
	upload.InitializeForTests()

	_, err = pc.uploadFile(upload, file, bytes.NewBufferString("data"))
	common.RequireError(t, err, "upload "+upload.ID+" not found")
}

func TestUploadFileReaderError(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer shutdown(ps)

	err := start(ps)
	require.NoError(t, err, "unable to start plik server")

	err = common.CheckHTTPServer(ps.GetConfig().ListenPort)
	require.NoError(t, err, "server unreachable")

	upload := &common.Upload{}
	file := upload.NewFile()
	file.Name = "filename"
	upload.InitializeForTests()

	_, err = pc.uploadFile(upload, file, common.NewErrorReaderString("io error"))
	common.RequireError(t, err, "io error")
}

func TestUploadFileInvalidParams(t *testing.T) {
	_, pc := newPlikServerAndClient()

	_, err := pc.uploadFile(nil, nil, nil)
	common.RequireError(t, err, "missing file upload parameter")

	pc.URL = string([]byte{0})

	_, err = pc.uploadFile(&common.Upload{}, common.NewFile(), &bytes.Buffer{})
	common.RequireError(t, err, "")
}

func TestUploadFileAPIFail(t *testing.T) {
	_, pc := newPlikServerAndClient()

	_, err := pc.uploadFile(&common.Upload{}, common.NewFile(), &bytes.Buffer{})
	common.RequireError(t, err, "connection refused")

	shutdown, err := common.StartAPIMockServer(common.DummyHandler)
	defer shutdown()
	require.NoError(t, err, "unable to start HTTP server server")

	_, err = pc.uploadFile(&common.Upload{}, common.NewFile(), &bytes.Buffer{})
	common.RequireError(t, err, "")
}

func TestUploadFileInvalidJSON(t *testing.T) {
	_, pc := newPlikServerAndClient()

	_, err := pc.GetServerVersion()
	common.RequireError(t, err, "connection refused")

	handler := http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		resp.Write([]byte("invalid json"))
	})

	shutdown, err := common.StartAPIMockServer(handler)
	defer shutdown()
	require.NoError(t, err, "unable to start HTTP server server")

	_, err = pc.uploadFile(&common.Upload{}, common.NewFile(), &bytes.Buffer{})
	common.RequireError(t, err, "")
}

func TestMakeRequestDebug(t *testing.T) {
	_, pc := newPlikServerAndClient()
	pc.Debug = true

	handler := http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		resp.Write([]byte("display this response"))
	})

	shutdown, err := common.StartAPIMockServer(handler)
	defer shutdown()
	require.NoError(t, err, "unable to start HTTP server server")

	// Temporarily hijack stdout and stderr
	reader, writer, err := os.Pipe()
	if err != nil {
		panic(err)
	}
	stdout := os.Stdout
	stderr := os.Stderr
	os.Stdout = writer
	os.Stderr = writer
	defer func() {
		os.Stdout = stdout
		os.Stderr = stderr
	}()

	req, err := http.NewRequest("GET", pc.URL+"/", bytes.NewBufferString("display this request"))
	require.NoError(t, err, "unable to create request")

	_, err = pc.MakeRequest(req)
	require.NoError(t, err, "missing error")

	os.Stdout = stdout
	os.Stderr = stderr

	err = writer.Close()
	require.NoError(t, err, "unable to close writer")

	printed, err := ioutil.ReadAll(reader)
	require.NoError(t, err, "unable to read output")

	require.Contains(t, string(printed), "X-Clientapp", "invalid output")
	require.Contains(t, string(printed), "X-Clientversion", "invalid output")
	require.Contains(t, string(printed), "display this request", "invalid output")
	require.Contains(t, string(printed), "display this response", "invalid output")
}

func TestMakeRequestDebugFile(t *testing.T) {
	_, pc := newPlikServerAndClient()
	pc.Debug = true

	handler := http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		resp.Write([]byte("display this response"))
	})

	shutdown, err := common.StartAPIMockServer(handler)
	defer shutdown()
	require.NoError(t, err, "unable to start HTTP server server")

	// Temporarily hijack stdout and stderr
	reader, writer, err := os.Pipe()
	if err != nil {
		panic(err)
	}
	stdout := os.Stdout
	stderr := os.Stderr
	os.Stdout = writer
	os.Stderr = writer
	defer func() {
		os.Stdout = stdout
		os.Stderr = stderr
	}()

	req, err := http.NewRequest("POST", pc.URL+"/file", bytes.NewBufferString("display this request"))
	require.NoError(t, err, "unable to create request")

	_, err = pc.MakeRequest(req)
	require.NoError(t, err, "missing error")

	os.Stdout = stdout
	os.Stderr = stderr

	err = writer.Close()
	require.NoError(t, err, "unable to close writer")

	printed, err := ioutil.ReadAll(reader)
	require.NoError(t, err, "unable to read output")
	require.Contains(t, string(printed), "X-Clientapp", "invalid output")
	require.Contains(t, string(printed), "X-Clientversion", "invalid output")
	require.NotContains(t, string(printed), "display this request", "invalid output")
	require.Contains(t, string(printed), "display this response", "invalid output")

}

func TestMakeRequestDebugStream(t *testing.T) {
	_, pc := newPlikServerAndClient()
	pc.Debug = true

	handler := http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		resp.Write([]byte("display this response"))
	})

	shutdown, err := common.StartAPIMockServer(handler)
	defer shutdown()
	require.NoError(t, err, "unable to start HTTP server server")

	// Temporarily hijack stdout and stderr
	reader, writer, err := os.Pipe()
	if err != nil {
		panic(err)
	}
	stdout := os.Stdout
	stderr := os.Stderr
	os.Stdout = writer
	os.Stderr = writer
	defer func() {
		os.Stdout = stdout
		os.Stderr = stderr
	}()

	req, err := http.NewRequest("POST", pc.URL+"/stream", bytes.NewBufferString("display this request"))
	require.NoError(t, err, "unable to create request")

	_, err = pc.MakeRequest(req)
	require.NoError(t, err, "missing error")

	os.Stdout = stdout
	os.Stderr = stderr

	err = writer.Close()
	require.NoError(t, err, "unable to close writer")

	printed, err := ioutil.ReadAll(reader)
	require.NoError(t, err, "unable to read output")
	require.Contains(t, string(printed), "X-Clientapp", "invalid output")
	require.Contains(t, string(printed), "X-Clientversion", "invalid output")
	require.NotContains(t, string(printed), "display this request", "invalid output")
	require.Contains(t, string(printed), "display this response", "invalid output")
}

func TestMakeRequestDebugGetFile(t *testing.T) {
	_, pc := newPlikServerAndClient()
	pc.Debug = true

	handler := http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		resp.Write([]byte("display this response"))
	})

	shutdown, err := common.StartAPIMockServer(handler)
	defer shutdown()
	require.NoError(t, err, "unable to start HTTP server server")

	// Temporarily hijack stdout and stderr
	reader, writer, err := os.Pipe()
	if err != nil {
		panic(err)
	}
	stdout := os.Stdout
	stderr := os.Stderr
	os.Stdout = writer
	os.Stderr = writer
	defer func() {
		os.Stdout = stdout
		os.Stderr = stderr
	}()

	req, err := http.NewRequest("GET", pc.URL+"/file", bytes.NewBufferString("display this request"))
	require.NoError(t, err, "unable to create request")

	_, err = pc.MakeRequest(req)
	require.NoError(t, err, "missing error")

	os.Stdout = stdout
	os.Stderr = stderr

	err = writer.Close()
	require.NoError(t, err, "unable to close writer")

	printed, err := ioutil.ReadAll(reader)
	require.NoError(t, err, "unable to read output")
	require.Contains(t, string(printed), "X-Clientapp", "invalid output")
	require.Contains(t, string(printed), "X-Clientversion", "invalid output")
	require.Contains(t, string(printed), "display this request", "invalid output")
	require.NotContains(t, string(printed), "display this response", "invalid output")
}

func TestMakeRequestDebugGetArchive(t *testing.T) {
	_, pc := newPlikServerAndClient()
	pc.Debug = true

	handler := http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		resp.Write([]byte("display this response"))
	})

	shutdown, err := common.StartAPIMockServer(handler)
	defer shutdown()
	require.NoError(t, err, "unable to start HTTP server server")

	// Temporarily hijack stdout and stderr
	reader, writer, err := os.Pipe()
	if err != nil {
		panic(err)
	}
	stdout := os.Stdout
	stderr := os.Stderr
	os.Stdout = writer
	os.Stderr = writer
	defer func() {
		os.Stdout = stdout
		os.Stderr = stderr
	}()

	req, err := http.NewRequest("GET", pc.URL+"/file", bytes.NewBufferString("display this request"))
	require.NoError(t, err, "unable to create request")

	_, err = pc.MakeRequest(req)
	require.NoError(t, err, "missing error")

	os.Stdout = stdout
	os.Stderr = stderr

	err = writer.Close()
	require.NoError(t, err, "unable to close writer")

	printed, err := ioutil.ReadAll(reader)
	require.NoError(t, err, "unable to read output")
	require.Contains(t, string(printed), "X-Clientapp", "invalid output")
	require.Contains(t, string(printed), "X-Clientversion", "invalid output")
	require.Contains(t, string(printed), "display this request", "invalid output")
	require.NotContains(t, string(printed), "display this response", "invalid output")
}

func TestMakeRequestErrorParsing(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer shutdown(ps)

	handler := http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		ctx := &context.Context{}
		ctx.SetReq(req)
		ctx.SetResp(resp)
		ctx.Fail("plik_api_error", nil, http.StatusInternalServerError)

	})

	shutdown, err := common.StartAPIMockServer(handler)
	defer shutdown()
	require.NoError(t, err, "unable to start HTTP server server")

	req, err := http.NewRequest("GET", pc.URL+"/", nil)
	require.NoError(t, err, "unable to create request")

	_, err = pc.MakeRequest(req)
	common.RequireError(t, err, "500 Internal Server Error : plik_api_error")
}

func TestMakeRequestErrorParsingInvalidJSON(t *testing.T) {
	_, pc := newPlikServerAndClient()

	handler := http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		resp.WriteHeader(http.StatusInternalServerError)
		resp.Write([]byte("plik_api_error"))
	})

	shutdown, err := common.StartAPIMockServer(handler)
	defer shutdown()
	require.NoError(t, err, "unable to start HTTP server server")

	req, err := http.NewRequest("GET", pc.URL+"/", nil)
	require.NoError(t, err, "unable to create request")

	_, err = pc.MakeRequest(req)
	common.RequireError(t, err, "500 Internal Server Error : plik_api_error")
}

func TestMakeRequestErrorParsingEmpty(t *testing.T) {
	_, pc := newPlikServerAndClient()

	handler := http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		resp.WriteHeader(http.StatusInternalServerError)
	})

	shutdown, err := common.StartAPIMockServer(handler)
	defer shutdown()
	require.NoError(t, err, "unable to start HTTP server server")

	req, err := http.NewRequest("GET", pc.URL+"/", nil)
	require.NoError(t, err, "unable to create request")

	_, err = pc.MakeRequest(req)
	common.RequireError(t, err, "500 Internal Server Error")
}
