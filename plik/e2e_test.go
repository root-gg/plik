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

package plik

import (
	"bytes"
	"io/ioutil"
	"sync"
	"testing"
	"time"

	"github.com/root-gg/plik/server/common"
	"github.com/stretchr/testify/require"
)

func TestUploadFileTwice(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer ps.ShutdownNow()

	err := start(ps)
	require.NoError(t, err, "unable to start plik server")

	upload := common.NewUpload()
	uploadParams, err := pc.create(upload)
	require.NoError(t, err, "unable to create upload")
	require.NotNil(t, uploadParams, "invalid nil uploads params")
	require.NotZero(t, uploadParams.ID, "invalid upload id")

	file := &common.File{}
	file.Name = "filename"

	fileParams, err := pc.uploadFile(uploadParams, file, bytes.NewBufferString("data"))
	require.NoError(t, err, "unable to upload file")
	require.NotNil(t, fileParams, "invalid nil file params")
	require.NotZero(t, fileParams.ID, "invalid file id")

	_, err = pc.uploadFile(uploadParams, fileParams, bytes.NewBufferString("data"))
	require.Error(t, err, "missing error")
	require.Contains(t, err.Error(), "403 Forbidden : File has already been uploaded", "invalid error")
}

func TestOneShot(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer ps.ShutdownNow()

	pc.OneShot = true

	err := start(ps)
	require.NoError(t, err, "unable to start plik server")

	data := "data data data"
	upload, file, err := pc.UploadReader("filename", bytes.NewBufferString(data))
	require.NoError(t, err, "unable to upload file")

	require.True(t, upload.Details().OneShot, "invalid upload non oneshot")
	require.Len(t, upload.Details().Files, 1, "invalid file count")

	reader, err := pc.downloadFile(upload.Details(), file.Details())
	require.NoError(t, err, "unable to download file")
	content, err := ioutil.ReadAll(reader)
	require.NoError(t, err, "unable to read file")
	require.Equal(t, data, string(content), "invalid file content")

	_, err = pc.downloadFile(upload.Details(), file.Details())
	require.Error(t, err, "unable to download file")
	require.Contains(t, err.Error(), "not found", "invalid error")
}

func TestRemoveFileWithoutUploadToken(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer ps.ShutdownNow()

	err := start(ps)
	require.NoError(t, err, "unable to start plik server")

	data := "data data data"
	upload, file, err := pc.UploadReader("filename", bytes.NewBufferString(data))
	require.NoError(t, err, "unable to upload file")
	require.Len(t, upload.Files(), 1, "invalid file count")

	upload.Details().UploadToken = ""
	err = pc.removeFile(upload.Details(), file.Details())
	require.Error(t, err, "unable to remove file")
	require.Contains(t, err.Error(), "You are not allowed to remove file from this upload", "invalid error")
}

func TestRemovable(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer ps.ShutdownNow()

	pc.Removable = true

	err := start(ps)
	require.NoError(t, err, "unable to start plik server")

	data := "data data data"
	upload, file, err := pc.UploadReader("filename", bytes.NewBufferString(data))
	require.NoError(t, err, "unable to upload file")
	require.Len(t, upload.Files(), 1, "invalid file count")

	upload.Details().UploadToken = ""
	err = pc.removeFile(upload.Details(), file.Details())
	require.NoError(t, err, "unable to upload file")
}

func TestUploadWithoutUploadToken(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer ps.ShutdownNow()

	err := start(ps)
	require.NoError(t, err, "unable to start plik server")

	upload := pc.NewUpload()
	file := upload.AddFileFromReader("filename", bytes.NewBufferString("data"))

	err = upload.Create()
	require.NoError(t, err, "unable to create upload")
	require.True(t, upload.HasBeenCreated(), "invalid nil uploads params")
	require.NotZero(t, upload.ID(), "invalid upload id")

	uploadToken := upload.Details().UploadToken
	upload.Details().UploadToken = ""
	err = file.Upload()
	require.Error(t, err, "should not be able to upload file to anonymous upload")
	require.Equal(t, "403 Forbidden : You are not allowed to add file to this upload", err.Error(), "invalid error")

	upload.Details().UploadToken = uploadToken
	err = file.Upload()
	require.NoError(t, err, "unable to upload file")
}

func TestStream(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer ps.ShutdownNow()

	pc.Stream = true

	err := start(ps)
	require.NoError(t, err, "unable to start plik server")

	data := "data data data"

	upload := pc.NewUpload()
	file := upload.AddFileFromReader("filename", bytes.NewBufferString(data))

	err = upload.Create()
	require.NoError(t, err, "unable to create upload")
	require.True(t, upload.Stream, "invalid nil error params")

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		time.Sleep(50 * time.Millisecond)
		err := upload.Upload()
		require.NoError(t, err, "unable to upload file")
		wg.Done()
	}()

	f := func() {
		for {
			time.Sleep(20 * time.Millisecond)
			reader, err := pc.downloadFile(upload.Details(), file.Details())
			if err != nil {
				continue
			}
			content, err := ioutil.ReadAll(reader)
			require.NoError(t, err, "unable to read file")
			require.Equal(t, data, string(content), "invalid file content")
			break
		}
		wg.Wait()
	}

	err = common.TestTimeout(f, time.Second)
	require.NoError(t, err, "timeout")

	_, err = pc.downloadFile(upload.Details(), file.Details())
	require.Error(t, err, "unable to download file")
	require.Contains(t, err.Error(), "not found", "invalid error")
}

//func TestStreamBlocking(t *testing.T) {
//	ps, pc := newPlikServerAndClient()
//	defer ps.ShutdownNow()
//
//	pc.Stream = true
//
//	err := start(ps)
//	require.NoError(t, err, "unable to start plik server")
//
//	upload := pc.NewUpload()
//	upload.AddFileFromReader("filename", bytes.NewBufferString("data"))
//
//	err = upload.Create()
//	require.NoError(t, err, "unable to create upload")
//	require.True(t, upload.Stream, "invalid nil error params")
//
//	f := func() {
//		defer func() { recover() }()
//		err := upload.Upload()
//		require.NoError(t, err, "unable to upload file")
//	}
//
//	err = common.TestTimeout(f, time.Second)
//	require.Error(t, err, "missing timeout")
//}

func TestTTL(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer ps.ShutdownNow()

	err := start(ps)
	require.NoError(t, err, "unable to start plik server")

	upload := pc.NewUpload()
	upload.TTL = 1
	err = upload.Create()
	require.NoError(t, err, "unable to upload file")
	require.True(t, upload.HasBeenCreated(), "upload has not been created")

	time.Sleep(2 * time.Second)

	_, err = pc.GetUpload(upload.ID())
	require.Error(t, err, "unable to get upload")
	require.Contains(t, err.Error(), "has expired", "upload has not been created")
}
