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
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewUpload(t *testing.T) {
	upload := NewUpload()
	require.NotNil(t, upload, "invalid upload")
	require.NotNil(t, upload.Files, "invalid upload")
}

func TestUploadCreate(t *testing.T) {
	upload := NewUpload()
	upload.Create()
	require.NotZero(t, upload.ID, "missing id")
	require.NotZero(t, upload.Creation, "missing creation date")
}

func TestUploadNewFile(t *testing.T) {
	upload := NewUpload()
	file := upload.NewFile()
	require.NotZero(t, len(upload.Files), "invalid file count")
	require.Equal(t, file, upload.Files[file.ID], "missing file")
}

func TestUploadSanitize(t *testing.T) {
	upload := NewUpload()
	upload.RemoteIP = "ip"
	upload.Login = "login"
	upload.Password = "password"
	upload.Yubikey = "token"
	upload.UploadToken = "token"
	upload.Token = "token"
	upload.User = "user"
	upload.Sanitize()

	require.Zero(t, upload.RemoteIP, "invalid sanitized upload")
	require.Zero(t, upload.Login, "invalid sanitized upload")
	require.Zero(t, upload.Password, "invalid sanitized upload")
	require.Zero(t, upload.Yubikey, "invalid sanitized upload")
	require.Zero(t, upload.UploadToken, "invalid sanitized upload")
	require.Zero(t, upload.Token, "invalid sanitized upload")
	require.Zero(t, upload.UploadToken, "invalid sanitized upload")
}
