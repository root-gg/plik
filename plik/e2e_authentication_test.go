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
	"testing"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/server"
	"github.com/stretchr/testify/require"
)

const TOKEN = "22b2c7f9-dead-dead-dead-ee8edd115e8a"

func defaultUser() *common.User {
	return &common.User{
		ID:     "ovh:gg1-ovh",
		Name:   "plik",
		Email:  "plik@root.gg",
		Tokens: []*common.Token{{Token: TOKEN}},
	}
}

func newServerAndClientWithUser(t *testing.T, user *common.User) (ps *server.PlikServer, pc *Client) {
	ps, pc = newPlikServerAndClient()

	ps.GetConfig().Authentication = true
	ps.GetConfig().NoAnonymousUploads = true

	pc.Token = TOKEN

	err := ps.GetMetadataBackend().SaveUser(ps.NewContext(), user)
	require.NoError(t, err, "unable to create user")

	err = start(ps)
	require.NoError(t, err, "unable to start plik server")

	return ps, pc
}

func TestTokenAuthentication(t *testing.T) {
	user := defaultUser()
	ps, pc := newServerAndClientWithUser(t, user)
	defer ps.ShutdownNow()

	data := "data data data"
	upload, file, err := pc.UploadReader("filename", ioutil.NopCloser(bytes.NewBufferString(data)))
	require.NoError(t, err, "unable to upload file")
	require.Len(t, upload.Details().Files, 1, "invalid file count")

	reader, err := file.Download()
	require.NoError(t, err, "unable to download file")
	content, err := ioutil.ReadAll(reader)
	require.NoError(t, err, "unable to read file")
	require.Equal(t, data, string(content), "invalid file content")

}

// A user authenticated with a token should not be able to control an upload authenticated with another token
func TestTokenMultipleToken(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer ps.ShutdownNow()

	ps.GetConfig().Authentication = true
	ps.GetConfig().NoAnonymousUploads = true

	err := start(ps)
	require.NoError(t, err, "unable to start Plik server")

	user := common.NewUser()
	user.ID = "ovh:gg1-ovh"
	t1 := user.NewToken()
	t2 := user.NewToken()

	err = ps.GetMetadataBackend().SaveUser(ps.NewContext(), user)
	require.NoError(t, err, "unable to create user")

	upload := pc.NewUpload()
	upload.Token = t1.Token
	file := upload.AddFileFromReader("filename", bytes.NewBufferString("data"))
	err = upload.Upload()
	require.NoError(t, err, "unable to upload")

	upload.Details().UploadToken = ""

	// try to add file to upload with the good token
	upload.AddFileFromReader("filename", bytes.NewBufferString("data"))
	err = upload.Upload()
	require.NoError(t, err, "Unable to upload file")

	upload.Token = t2.Token

	// try to add file to upload with the wrong token
	f2 := upload.AddFileFromReader("filename", bytes.NewBufferString("data"))
	err = upload.Upload()
	common.RequireError(t, err, "Failed to upload at least one file")
	common.RequireError(t, f2.Error(), "You are not allowed to add file to this upload")

	// try to remove file to upload with the wrong token
	err = file.Delete()
	common.RequireError(t, err, "You are not allowed to remove file from this upload")

	// try to remove upload with the wrong token
	err = upload.Delete()
	common.RequireError(t, err, "You are not allowed to remove this upload")

	upload.Token = t1.Token

	// try to remove file with the good token
	err = file.Delete()
	require.NoError(t, err, "Unable to remove file")

	// try to remove upload with the good token
	err = upload.Delete()
	require.NoError(t, err, "Unable to remove upload")
}

// An admin user authenticated with a token should not have more power than a classical user authenticated with a token
// This is to lower the impact of the leak of an Admin user token
func TestTokenMultipleTokenAdmin(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer ps.ShutdownNow()

	uid := "ovh:gg1-ovh"
	ps.GetConfig().Authentication = true
	ps.GetConfig().NoAnonymousUploads = true
	ps.GetConfig().Admins = append(ps.GetConfig().Admins, uid)

	err := start(ps)
	require.NoError(t, err, "unable to start Plik server")

	user := common.NewUser()
	user.ID = uid
	t1 := user.NewToken()
	t2 := user.NewToken()

	err = ps.GetMetadataBackend().SaveUser(ps.NewContext(), user)
	require.NoError(t, err, "unable to create user")

	upload := pc.NewUpload()
	upload.Token = t1.Token
	file := upload.AddFileFromReader("filename", bytes.NewBufferString("data"))
	err = upload.Upload()
	require.NoError(t, err, "unable to upload")

	upload.Details().UploadToken = ""

	// try to add file to upload with the good token
	upload.AddFileFromReader("filename", bytes.NewBufferString("data"))
	err = upload.Upload()
	require.NoError(t, err, "Unable to upload file")

	upload.Token = t2.Token

	// try to add file to upload with the wrong token
	f2 := upload.AddFileFromReader("filename", bytes.NewBufferString("data"))
	err = upload.Upload()
	common.RequireError(t, err, "Failed to upload at least one file")
	common.RequireError(t, f2.Error(), "You are not allowed to add file to this upload")

	// try to remove file to upload with the wrong token
	err = file.Delete()
	common.RequireError(t, err, "You are not allowed to remove file from this upload")

	// try to remove upload with the wrong token
	err = upload.Delete()
	common.RequireError(t, err, "You are not allowed to remove this upload")

	upload.Token = t1.Token

	// try to remove file with the good token
	err = file.Delete()
	require.NoError(t, err, "Unable to remove file")

	// try to remove upload with the good token
	err = upload.Delete()
	require.NoError(t, err, "Unable to remove upload")
}
