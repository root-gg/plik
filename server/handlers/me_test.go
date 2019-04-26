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
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
	metadata_test "github.com/root-gg/plik/server/metadata/testing"
	"github.com/stretchr/testify/require"
)

func TestGetUser(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	user := common.NewUser()
	user.ID = "user1"
	user.Email = "user1@root.gg"
	user.Login = "user1"

	token := common.NewToken()
	token.Comment = "token comment"
	user.Tokens = append(user.Tokens, token)

	err := addTestUser(ctx, user)
	require.NoError(t, err, "unable to add user")
	ctx.Set("user", user)

	req, err := http.NewRequest("GET", "/me", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	UserInfo(ctx, rr, req)

	// Check the status code is what we expect.
	require.Equal(t, http.StatusOK, rr.Code, "handler returned wrong status code")

	respBody, err := ioutil.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")

	var userResult *common.User
	err = json.Unmarshal(respBody, &userResult)
	require.NoError(t, err, "unable to unmarshal response body")

	require.EqualValues(t, user, userResult, "invalid user")
}

func TestGetUserNoUser(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	req, err := http.NewRequest("GET", "/me", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	UserInfo(ctx, rr, req)

	context.TestFail(t, rr, http.StatusUnauthorized, "Please login first")
}

func TestDeleteUser(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	user := common.NewUser()
	user.ID = "user1"

	err := addTestUser(ctx, user)
	require.NoError(t, err, "unable to add user")
	ctx.Set("user", user)

	req, err := http.NewRequest("DELETE", "/me", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	DeleteAccount(ctx, rr, req)

	// Check the status code is what we expect.
	require.Equal(t, http.StatusOK, rr.Code, "handler returned wrong status code")

	respBody, err := ioutil.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")

	require.Equal(t, string(respBody), "", "invalid response body")
}

func TestDeleteUserNoUser(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	req, err := http.NewRequest("DELETE", "/me", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	DeleteAccount(ctx, rr, req)

	context.TestFail(t, rr, http.StatusUnauthorized, "Please login first")
}

func TestGetUserUploads(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	user := common.NewUser()
	user.ID = "user1"

	err := addTestUser(ctx, user)
	require.NoError(t, err, "unable to add user")
	ctx.Set("user", user)

	upload1 := common.NewUpload()
	upload1.User = user.ID
	upload1.Create()
	createTestUpload(ctx, upload1)

	upload2 := common.NewUpload()
	upload2.User = user.ID
	upload2.Create()
	createTestUpload(ctx, upload2)

	upload3 := common.NewUpload()
	upload3.Create()
	createTestUpload(ctx, upload3)

	//Create a request
	req, err := http.NewRequest("GET", "/me/uploads", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	GetUserUploads(ctx, rr, req)

	// Check the status code is what we expect.
	require.Equal(t, http.StatusOK, rr.Code, "handler returned wrong status code")

	respBody, err := ioutil.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")

	var uploads []*common.Upload
	err = json.Unmarshal(respBody, &uploads)
	require.NoError(t, err, "unable to unmarshal response body")

	require.Equal(t, 2, len(uploads), "invalid upload count")
}

func TestGetUserUploadsNoUser(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	req, err := http.NewRequest("GET", "/me/uploads", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	GetUserUploads(ctx, rr, req)

	context.TestFail(t, rr, http.StatusUnauthorized, "Please login first")
}

func TestGetUserUploadsWithToken(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	user := common.NewUser()
	user.ID = "user1"

	token := common.NewToken()
	token.Create()
	token.Comment = "token comment"
	user.Tokens = append(user.Tokens, token)

	err := addTestUser(ctx, user)
	require.NoError(t, err, "unable to add user")
	ctx.Set("user", user)

	upload1 := common.NewUpload()
	upload1.User = user.ID
	upload1.Create()
	createTestUpload(ctx, upload1)

	upload2 := common.NewUpload()
	upload2.User = user.ID
	upload2.Token = token.Token
	upload2.Create()
	createTestUpload(ctx, upload2)

	upload3 := common.NewUpload()
	upload3.Create()
	createTestUpload(ctx, upload3)

	req, err := http.NewRequest("GET", "/me/uploads?token="+token.Token, bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	GetUserUploads(ctx, rr, req)

	// Check the status code is what we expect.
	require.Equal(t, http.StatusOK, rr.Code, "handler returned wrong status code")

	respBody, err := ioutil.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")

	var uploads []*common.Upload
	err = json.Unmarshal(respBody, &uploads)
	require.NoError(t, err, "unable to unmarshal response body")

	require.Equal(t, 1, len(uploads), "invalid upload count")
}

func TestGetUserUploadsInvalidToken(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	user := common.NewUser()
	user.ID = "user1"

	err := addTestUser(ctx, user)
	require.NoError(t, err, "unable to add user")
	ctx.Set("user", user)

	//Create a request
	req, err := http.NewRequest("GET", "/me/uploads?token=invalid_token", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	GetUserUploads(ctx, rr, req)

	context.TestFail(t, rr, http.StatusBadRequest, "Invalid token")
}

func TestGetUserUploadsWithSizeAndOffset(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	user := common.NewUser()
	user.ID = "user1"

	err := addTestUser(ctx, user)
	require.NoError(t, err, "unable to add user")
	ctx.Set("user", user)

	upload1 := common.NewUpload()
	upload1.User = user.ID
	upload1.Create()
	createTestUpload(ctx, upload1)

	upload2 := common.NewUpload()
	upload2.User = user.ID
	upload2.Create()
	createTestUpload(ctx, upload2)

	upload3 := common.NewUpload()
	upload3.User = user.ID
	upload3.Create()
	createTestUpload(ctx, upload3)

	//Create a request
	req, err := http.NewRequest("GET", "/me/uploads?size=1&offset=1", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	GetUserUploads(ctx, rr, req)

	// Check the status code is what we expect.
	require.Equal(t, http.StatusOK, rr.Code, "handler returned wrong status code")

	respBody, err := ioutil.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")

	var uploads []*common.Upload
	err = json.Unmarshal(respBody, &uploads)
	require.NoError(t, err, "unable to unmarshal response body")

	require.Equal(t, 1, len(uploads), "invalid upload count")
}

func TestGetUserUploadsWithInvalidSize(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	user := common.NewUser()
	user.ID = "user1"

	err := addTestUser(ctx, user)
	require.NoError(t, err, "unable to add user")
	ctx.Set("user", user)

	//Create a request
	req, err := http.NewRequest("GET", "/me/uploads?size=-1", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	GetUserUploads(ctx, rr, req)

	context.TestFail(t, rr, http.StatusBadRequest, "Invalid size parameter")
}

func TestGetUserUploadsWithInvalidOffset(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	user := common.NewUser()
	user.ID = "user1"

	err := addTestUser(ctx, user)
	require.NoError(t, err, "unable to add user")
	ctx.Set("user", user)

	//Create a request
	req, err := http.NewRequest("GET", "/me/uploads?offset=-1", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	GetUserUploads(ctx, rr, req)

	context.TestFail(t, rr, http.StatusBadRequest, "Invalid offset parameter")
}

func TestGetUserStatisticsMetadataBackendError(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	user := common.NewUser()
	user.ID = "user1"

	err := addTestUser(ctx, user)
	require.NoError(t, err, "unable to add user")
	ctx.Set("user", user)

	upload1 := common.NewUpload()
	upload1.User = user.ID
	upload1.Create()
	createTestUpload(ctx, upload1)

	//Create a request
	req, err := http.NewRequest("GET", "/me/uploads", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	context.GetMetadataBackend(ctx).(*metadata_test.MetadataBackend).SetError(errors.New("metadata backend error"))

	rr := httptest.NewRecorder()
	GetUserStatistics(ctx, rr, req)

	context.TestFail(t, rr, http.StatusInternalServerError, "Unable to get user statistics")
}

func TestRemoveUserUploads(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	user := common.NewUser()
	user.ID = "user1"

	err := addTestUser(ctx, user)
	require.NoError(t, err, "unable to add user")
	ctx.Set("user", user)

	upload1 := common.NewUpload()
	upload1.User = user.ID
	upload1.Create()
	createTestUpload(ctx, upload1)

	upload2 := common.NewUpload()
	upload2.User = user.ID
	upload2.Create()
	createTestUpload(ctx, upload2)

	//Create a request
	req, err := http.NewRequest("DELETE", "/me/uploads", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	RemoveUserUploads(ctx, rr, req)

	// Check the status code is what we expect.
	require.Equal(t, http.StatusOK, rr.Code, "handler returned wrong status code")

	respBody, err := ioutil.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")

	var result = &common.Result{}
	err = json.Unmarshal(respBody, result)
	require.NoError(t, err, "unable to unmarshal response body")

	require.Equal(t, "2 uploads removed", result.Message, "Invalid result message")
}

func TestRemoveUserUploadsNoUser(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	req, err := http.NewRequest("DELETE", "/me/uploads", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	RemoveUserUploads(ctx, rr, req)

	context.TestFail(t, rr, http.StatusUnauthorized, "Please login first")
}

func TestRemoveUserUploadsWithToken(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	user := common.NewUser()
	user.ID = "user1"

	token := common.NewToken()
	token.Create()
	token.Comment = "token comment"
	user.Tokens = append(user.Tokens, token)

	err := addTestUser(ctx, user)
	require.NoError(t, err, "unable to add user")
	ctx.Set("user", user)

	upload1 := common.NewUpload()
	upload1.User = user.ID
	upload1.Create()
	createTestUpload(ctx, upload1)

	upload2 := common.NewUpload()
	upload2.User = user.ID
	upload2.Token = token.Token
	upload2.Create()
	createTestUpload(ctx, upload2)

	//Create a request
	req, err := http.NewRequest("DELETE", "/me/uploads?token="+token.Token, bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	RemoveUserUploads(ctx, rr, req)

	// Check the status code is what we expect.
	require.Equal(t, http.StatusOK, rr.Code, "handler returned wrong status code")

	respBody, err := ioutil.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")

	var result = &common.Result{}
	err = json.Unmarshal(respBody, result)
	require.NoError(t, err, "unable to unmarshal response body")

	require.Equal(t, "1 uploads removed", result.Message, "Invalid result message")
}

func TestRemoveUserUploadsInvalidToken(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	user := common.NewUser()
	user.ID = "user1"

	err := addTestUser(ctx, user)
	require.NoError(t, err, "unable to add user")
	ctx.Set("user", user)

	//Create a request
	req, err := http.NewRequest("DELETE", "/me/uploads?token=invalid_token", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	RemoveUserUploads(ctx, rr, req)

	context.TestFail(t, rr, http.StatusBadRequest, "Unable to remove uploads : Invalid token")
}

func TestRemoveUserUploadsMetadataBackendError(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	user := common.NewUser()
	user.ID = "user1"

	err := addTestUser(ctx, user)
	require.NoError(t, err, "unable to add user")
	ctx.Set("user", user)

	upload1 := common.NewUpload()
	upload1.User = user.ID
	upload1.Create()
	createTestUpload(ctx, upload1)

	//Create a request
	req, err := http.NewRequest("DELETE", "/me/uploads", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	context.GetMetadataBackend(ctx).(*metadata_test.MetadataBackend).SetError(errors.New("metadata backend error"))

	rr := httptest.NewRecorder()
	RemoveUserUploads(ctx, rr, req)

	context.TestFail(t, rr, http.StatusInternalServerError, "Unable to get uploads")
}

func TestGetUserStatistics(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	user := common.NewUser()
	user.ID = "user1"

	err := addTestUser(ctx, user)
	require.NoError(t, err, "unable to add user")
	ctx.Set("user", user)

	upload1 := common.NewUpload()
	upload1.User = user.ID
	upload1.Create()

	file1 := common.NewFile()
	file1.CurrentSize = 1
	upload1.Files[file1.ID] = file1

	createTestUpload(ctx, upload1)

	upload2 := common.NewUpload()
	upload2.User = user.ID
	upload2.Create()

	file2 := common.NewFile()
	file2.CurrentSize = 2
	upload2.Files[file2.ID] = file2

	createTestUpload(ctx, upload2)

	//Create a request
	req, err := http.NewRequest("DELETE", "/me/uploads", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	GetUserStatistics(ctx, rr, req)

	// Check the status code is what we expect.
	require.Equal(t, http.StatusOK, rr.Code, "handler returned wrong status code")

	respBody, err := ioutil.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")

	var stats = &common.UserStats{}
	err = json.Unmarshal(respBody, stats)
	require.NoError(t, err, "unable to unmarshal response body")

	require.Equal(t, 2, stats.Uploads, "Invalid upload count")
	require.Equal(t, 2, stats.Files, "Invalid files count")
	require.Equal(t, int64(3), stats.TotalSize, "Invalid total size")
}

func TestGetUserStatisticsToken(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	token := common.NewToken()
	token.Create()

	user := common.NewUser()
	user.ID = "user1"
	user.Tokens = append(user.Tokens, token)

	err := addTestUser(ctx, user)
	require.NoError(t, err, "unable to add user")
	ctx.Set("user", user)

	upload1 := common.NewUpload()
	upload1.User = user.ID
	upload1.Token = token.Token
	upload1.Create()

	file1 := common.NewFile()
	file1.CurrentSize = 1
	upload1.Files[file1.ID] = file1

	createTestUpload(ctx, upload1)

	upload2 := common.NewUpload()
	upload2.User = user.ID
	upload2.Create()

	file2 := common.NewFile()
	file2.CurrentSize = 2
	upload2.Files[file2.ID] = file2

	createTestUpload(ctx, upload2)

	//Create a request
	req, err := http.NewRequest("DELETE", "/me/uploads?token="+token.Token, bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	GetUserStatistics(ctx, rr, req)

	// Check the status code is what we expect.
	require.Equal(t, http.StatusOK, rr.Code, "handler returned wrong status code")

	respBody, err := ioutil.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")

	var stats = &common.UserStats{}
	err = json.Unmarshal(respBody, stats)
	require.NoError(t, err, "unable to unmarshal response body")

	require.Equal(t, 1, stats.Uploads, "Invalid upload count")
	require.Equal(t, 1, stats.Files, "Invalid files count")
	require.Equal(t, int64(1), stats.TotalSize, "Invalid total size")
}

func TestGetUserStatisticsInvalidToken(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	user := common.NewUser()
	user.ID = "user1"

	err := addTestUser(ctx, user)
	require.NoError(t, err, "unable to add user")
	ctx.Set("user", user)

	//Create a request
	req, err := http.NewRequest("DELETE", "/me/uploads?token=invalid_token", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	GetUserStatistics(ctx, rr, req)

	context.TestFail(t, rr, http.StatusBadRequest, "Unable to get uploads : Invalid token")
}

func TestGetUserStatisticsNoUser(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	req, err := http.NewRequest("GET", "/me/stats", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	GetUserStatistics(ctx, rr, req)

	context.TestFail(t, rr, http.StatusUnauthorized, "Please login first")
}

func TestRemoveUserStatisticsMetadataBackendError(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	user := common.NewUser()
	user.ID = "user1"

	err := addTestUser(ctx, user)
	require.NoError(t, err, "unable to add user")
	ctx.Set("user", user)

	upload1 := common.NewUpload()
	upload1.User = user.ID
	upload1.Create()
	createTestUpload(ctx, upload1)

	//Create a request
	req, err := http.NewRequest("GET", "/me/stats", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	context.GetMetadataBackend(ctx).(*metadata_test.MetadataBackend).SetError(errors.New("metadata backend error"))

	rr := httptest.NewRecorder()
	GetUserStatistics(ctx, rr, req)

	context.TestFail(t, rr, http.StatusInternalServerError, "Unable to get user statistics")
}
