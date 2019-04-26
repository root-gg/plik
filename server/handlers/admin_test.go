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

package handlers

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/root-gg/juliet"
	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
	"github.com/stretchr/testify/require"
)

func addTestUser(ctx *juliet.Context, user *common.User) (err error) {
	metadataBackend := context.GetMetadataBackend(ctx)
	return metadataBackend.SaveUser(ctx, user)
}

func addTestUserAdmin(ctx *juliet.Context) (user *common.User, err error) {
	user = common.NewUser()
	user.ID = "admin"
	user.Email = "admin@root.gg"
	user.Login = "admin"
	ctx.Set("user", user)
	ctx.Set("is_admin", true)
	return user, addTestUser(ctx, user)
}

func TestGetUsers(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	_, err := addTestUserAdmin(ctx)
	require.NoError(t, err, "unable to add user admin")

	user1 := common.NewUser()
	user1.ID = "user1"
	user1.Email = "user1@root.gg"
	user1.Login = "user1"

	user2 := common.NewUser()
	user2.ID = "user2"
	user2.Email = "user2@root.gg"
	user2.Login = "user2"

	err = addTestUser(ctx, user1)
	require.NoError(t, err, "unable to add user1")

	err = addTestUser(ctx, user2)
	require.NoError(t, err, "unable to add user2")

	req, err := http.NewRequest("GET", "/admin/users", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	GetUsers(ctx, rr, req)

	require.Equal(t, http.StatusOK, rr.Code, "handler returned wrong status code")

	respBody, err := ioutil.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")

	var users []*common.User
	err = json.Unmarshal(respBody, &users)
	require.NoError(t, err, "unable to unmarshal response body")

	require.Equal(t, 3, len(users), "invalid user count")
}

func TestGetUsersNoUser(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	req, err := http.NewRequest("GET", "/admin/users", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	GetUsers(ctx, rr, req)

	context.TestFail(t, rr, http.StatusUnauthorized, "Missing user, Please login first")
}

func TestGetUsersNotAdmin(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	_, err := addTestUserAdmin(ctx)
	require.NoError(t, err, "unable to add admin")
	ctx.Set("is_admin", false)

	req, err := http.NewRequest("GET", "/admin/users", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	GetUsers(ctx, rr, req)

	context.TestFail(t, rr, http.StatusForbidden, "You need administrator privileges")
}

func TestGetServerStatistics(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	_, err := addTestUserAdmin(ctx)
	require.NoError(t, err, "unable to add user admin")

	type pair struct {
		typ   string
		size  int64
		count int
	}

	plan := []pair{
		{"type1", 1, 1},
		{"type2", 1000, 5},
		{"type3", 1000 * 1000, 10},
		{"type4", 1000 * 1000 * 1000, 15},
	}

	for _, item := range plan {
		for i := 0; i < item.count; i++ {
			upload := common.NewUpload()
			upload.Create()
			file := upload.NewFile()
			file.Type = item.typ
			file.CurrentSize = item.size

			err := context.GetMetadataBackend(ctx).Upsert(ctx, upload)
			require.NoError(t, err, "create error")
		}
	}

	req, err := http.NewRequest("GET", "/admin/stats", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	GetServerStatistics(ctx, rr, req)

	require.Equal(t, http.StatusOK, rr.Code, "handler returned wrong status code")

	respBody, err := ioutil.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")

	var stats *common.ServerStats
	err = json.Unmarshal(respBody, &stats)
	require.NoError(t, err, "unable to unmarshal response body")

	require.NotNil(t, stats, "invalid server statistics")
	require.Equal(t, 31, stats.Uploads, "invalid upload count")
	require.Equal(t, 31, stats.Files, "invalid files count")
	require.Equal(t, int64(15010005001), stats.TotalSize, "invalid total file size")
	require.Equal(t, 31, stats.AnonymousUploads, "invalid anonymous upload count")
	require.Equal(t, int64(15010005001), stats.AnonymousSize, "invalid anonymous total file size")
	require.Equal(t, 10, len(stats.FileTypeByCount), "invalid file type by count length")
	require.Equal(t, "type4", stats.FileTypeByCount[0].Type, "invalid file type by count type")
	require.Equal(t, 10, len(stats.FileTypeBySize), "invalid file type by size length")
	require.Equal(t, "type4", stats.FileTypeBySize[0].Type, "invalid file type by size type")

}

func TestGetServerStatisticsNoUser(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	req, err := http.NewRequest("GET", "/admin/users", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")
	ctx.Delete("user")

	rr := httptest.NewRecorder()
	GetServerStatistics(ctx, rr, req)

	context.TestFail(t, rr, http.StatusUnauthorized, "Missing user, Please login first")
}

func TestGetServerStatisticsNotAdmin(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	_, err := addTestUserAdmin(ctx)
	require.NoError(t, err, "unable to add admin")
	ctx.Set("is_admin", false)

	req, err := http.NewRequest("GET", "/admin/users", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	GetServerStatistics(ctx, rr, req)

	context.TestFail(t, rr, http.StatusForbidden, "You need administrator privileges")
}
