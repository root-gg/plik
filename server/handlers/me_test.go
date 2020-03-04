package handlers

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"testing"

	"github.com/stretchr/testify/require"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
)

func TestGetUser(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	user := common.NewUser(common.ProviderLocal, "user1")
	user.Email = "user1@root.gg"
	user.Login = "user1"
	user.Password = "password"

	token := user.NewToken()
	token.Comment = "token comment"

	err := ctx.GetMetadataBackend().CreateUser(user)
	require.NoError(t, err, "unable to create test user")
	ctx.SetUser(user)

	req, err := http.NewRequest("GET", "/me", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	UserInfo(ctx, rr, req)

	// Check the status code is what we expect.
	context.TestOK(t, rr)

	respBody, err := ioutil.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")

	var userResult *common.User
	err = json.Unmarshal(respBody, &userResult)
	require.NoError(t, err, "unable to unmarshal response body")

	require.Equal(t, user.ID, userResult.ID, "invalid user id")
	require.Equal(t, user.Name, userResult.Name, "invalid user name")
	require.Equal(t, user.Email, userResult.Email, "invalid user email")
	require.Equal(t, user.Login, userResult.Login, "invalid user login")
	require.Equal(t, userResult.Password, "", "invalid user password")
	require.Len(t, userResult.Tokens, 1, "invalid token length")
	require.Equal(t, user.Tokens[0].Comment, userResult.Tokens[0].Comment, "invalid token comment")
}

func TestGetUserNoUser(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	req, err := http.NewRequest("GET", "/me", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	UserInfo(ctx, rr, req)

	context.TestUnauthorized(t, rr, "missing user, please login first")
}

func TestDeleteUser(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	user := common.NewUser(common.ProviderLocal, "user1")

	err := ctx.GetMetadataBackend().CreateUser(user)
	require.NoError(t, err, "unable to create test user")
	ctx.SetUser(user)

	req, err := http.NewRequest("DELETE", "/me", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	DeleteAccount(ctx, rr, req)

	// Check the status code is what we expect.
	context.TestOK(t, rr)

	respBody, err := ioutil.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")

	require.Equal(t, string(respBody), "ok", "invalid response body")
}

func TestDeleteUserNoUser(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	req, err := http.NewRequest("DELETE", "/me", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	DeleteAccount(ctx, rr, req)

	context.TestUnauthorized(t, rr, "missing user, please login first")
}

func TestGetUserUploads(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	user := common.NewUser(common.ProviderLocal, "user1")
	err := ctx.GetMetadataBackend().CreateUser(user)
	require.NoError(t, err, "unable to create test user")

	ctx.SetUser(user)

	upload1 := &common.Upload{}
	upload1.User = user.ID
	createTestUpload(t, ctx, upload1)

	upload2 := &common.Upload{}
	upload2.User = user.ID
	createTestUpload(t, ctx, upload2)

	upload3 := &common.Upload{}
	createTestUpload(t, ctx, upload3)

	// Create a request
	req, err := http.NewRequest("GET", "/me/uploads", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	// Create paging query
	ctx.SetPagingQuery(&common.PagingQuery{})

	rr := ctx.NewRecorder(req)
	GetUserUploads(ctx, rr, req)

	// Check the status code is what we expect.
	context.TestOK(t, rr)

	respBody, err := ioutil.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")

	var response common.PagingResponse
	err = json.Unmarshal(respBody, &response)
	require.NoError(t, err, "unable to unmarshal response body %s", respBody)
	require.Equal(t, 2, len(response.Results), "invalid upload count")
}

func TestGetUserUploadsNoUser(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	req, err := http.NewRequest("GET", "/me/uploads", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	GetUserUploads(ctx, rr, req)

	context.TestUnauthorized(t, rr, "missing user, please login first")
}

func TestGetUserUploadsWithToken(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	user := common.NewUser(common.ProviderLocal, "user1")

	token := common.NewToken()
	token.Comment = "token comment"
	user.Tokens = append(user.Tokens, token)

	err := ctx.GetMetadataBackend().CreateUser(user)
	require.NoError(t, err, "unable to create test user")

	ctx.SetUser(user)

	upload1 := &common.Upload{}
	upload1.User = user.ID
	createTestUpload(t, ctx, upload1)

	upload2 := &common.Upload{}
	upload2.User = user.ID
	upload2.Token = token.Token
	createTestUpload(t, ctx, upload2)

	upload3 := &common.Upload{}
	createTestUpload(t, ctx, upload3)

	req, err := http.NewRequest("GET", "/me/uploads?token="+token.Token, bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	ctx.SetPagingQuery(&common.PagingQuery{})

	rr := ctx.NewRecorder(req)
	GetUserUploads(ctx, rr, req)

	// Check the status code is what we expect.
	context.TestOK(t, rr)

	respBody, err := ioutil.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")

	var response common.PagingResponse
	err = json.Unmarshal(respBody, &response)
	require.NoError(t, err, "unable to unmarshal response body %s", respBody)

	require.Equal(t, 1, len(response.Results), "invalid upload count")
}

func TestGetUserUploadsInvalidToken(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	user := common.NewUser(common.ProviderLocal, "user1")

	err := ctx.GetMetadataBackend().CreateUser(user)
	require.NoError(t, err, "unable to create user")
	ctx.SetUser(user)

	//Create a request
	req, err := http.NewRequest("GET", "/me/uploads?token=invalid_token", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	GetUserUploads(ctx, rr, req)

	context.TestNotFound(t, rr, "token not found")
}

func TestGetUserTokens(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	user := common.NewUser(common.ProviderLocal, "user1")
	user.NewToken()
	user.NewToken()
	err := ctx.GetMetadataBackend().CreateUser(user)
	require.NoError(t, err, "unable to create test user")

	ctx.SetUser(user)

	// Create a request
	req, err := http.NewRequest("GET", "/me/uploads", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	// Create paging query
	ctx.SetPagingQuery(&common.PagingQuery{})

	rr := ctx.NewRecorder(req)
	GetUserTokens(ctx, rr, req)

	// Check the status code is what we expect.
	context.TestOK(t, rr)

	respBody, err := ioutil.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")

	var response common.PagingResponse
	err = json.Unmarshal(respBody, &response)
	require.NoError(t, err, "unable to unmarshal response body %s", respBody)
	require.Equal(t, 2, len(response.Results), "invalid upload count")
}

func TestGetUserTokensNoUser(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	req, err := http.NewRequest("GET", "/me/uploads", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	GetUserTokens(ctx, rr, req)

	context.TestUnauthorized(t, rr, "missing user, please login first")
}

func TestRemoveUserUploads(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	user := common.NewUser(common.ProviderLocal, "user1")

	err := ctx.GetMetadataBackend().CreateUser(user)
	require.NoError(t, err, "unable to create user")
	ctx.SetUser(user)

	upload1 := &common.Upload{}
	upload1.User = user.ID
	createTestUpload(t, ctx, upload1)

	upload2 := &common.Upload{}
	upload2.User = user.ID
	createTestUpload(t, ctx, upload2)

	//Create a request
	req, err := http.NewRequest("DELETE", "/me/uploads", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	RemoveUserUploads(ctx, rr, req)

	// Check the status code is what we expect.
	context.TestOK(t, rr)

	respBody, err := ioutil.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")

	require.Equal(t, "2 uploads removed", string(respBody), "Invalid result message")
}

func TestRemoveUserUploadsNoUser(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	req, err := http.NewRequest("DELETE", "/me/uploads", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	RemoveUserUploads(ctx, rr, req)

	context.TestUnauthorized(t, rr, "missing user, please login first")
}

func TestRemoveUserUploadsWithToken(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	user := common.NewUser(common.ProviderLocal, "user1")
	token := user.NewToken()
	token.Comment = "token comment"
	err := ctx.GetMetadataBackend().CreateUser(user)
	require.NoError(t, err, "unable to create user")
	ctx.SetUser(user)

	upload1 := &common.Upload{}
	upload1.User = user.ID
	createTestUpload(t, ctx, upload1)

	upload2 := &common.Upload{}
	upload2.User = user.ID
	upload2.Token = token.Token
	createTestUpload(t, ctx, upload2)

	//Create a request
	req, err := http.NewRequest("DELETE", "/me/uploads?token="+token.Token, bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	RemoveUserUploads(ctx, rr, req)

	// Check the status code is what we expect.
	context.TestOK(t, rr)

	respBody, err := ioutil.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")

	require.Equal(t, "1 uploads removed", string(respBody), "Invalid result message")
}

func TestRemoveUserUploadsInvalidToken(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	user := common.NewUser(common.ProviderLocal, "user1")
	err := ctx.GetMetadataBackend().CreateUser(user)
	require.NoError(t, err, "unable to create user")
	ctx.SetUser(user)

	//Create a request
	req, err := http.NewRequest("DELETE", "/me/uploads?token=invalid_token", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	RemoveUserUploads(ctx, rr, req)

	context.TestNotFound(t, rr, "token not found")
}

func TestGetUserStatistics(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	user := common.NewUser(common.ProviderLocal, "user1")
	err := ctx.GetMetadataBackend().CreateUser(user)
	require.NoError(t, err, "unable to create user")
	ctx.SetUser(user)

	upload1 := &common.Upload{}
	upload1.User = user.ID
	file1 := upload1.NewFile()
	file1.Size = 1
	file1.Status = common.FileUploaded
	createTestUpload(t, ctx, upload1)

	upload2 := &common.Upload{}
	upload2.User = user.ID
	file2 := upload2.NewFile()
	file2.Size = 2
	file2.Status = common.FileUploaded
	createTestUpload(t, ctx, upload2)

	//Create a request
	req, err := http.NewRequest("GET", "/me/stats", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	GetUserStatistics(ctx, rr, req)

	// Check the status code is what we expect.
	context.TestOK(t, rr)

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
	ctx := newTestingContext(common.NewConfiguration())

	user := common.NewUser(common.ProviderLocal, "user1")
	token := user.NewToken()
	err := ctx.GetMetadataBackend().CreateUser(user)
	require.NoError(t, err, "unable to create user")
	ctx.SetUser(user)

	upload1 := &common.Upload{}
	upload1.User = user.ID
	upload1.Token = token.Token

	file1 := upload1.NewFile()
	file1.Size = 1
	file1.Status = common.FileUploaded
	createTestUpload(t, ctx, upload1)

	upload2 := &common.Upload{}
	upload2.User = user.ID
	file2 := upload2.NewFile()
	file2.Size = 2
	file2.Status = common.FileUploaded
	createTestUpload(t, ctx, upload2)

	//Create a request
	req, err := http.NewRequest("DELETE", "/me/uploads?token="+token.Token, bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	GetUserStatistics(ctx, rr, req)

	// Check the status code is what we expect.
	context.TestOK(t, rr)

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
	ctx := newTestingContext(common.NewConfiguration())

	user := common.NewUser(common.ProviderLocal, "user1")

	err := ctx.GetMetadataBackend().CreateUser(user)
	require.NoError(t, err, "unable to create user")
	ctx.SetUser(user)

	//Create a request
	req, err := http.NewRequest("DELETE", "/me/uploads?token=invalid_token", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	GetUserStatistics(ctx, rr, req)

	context.TestNotFound(t, rr, "token not found")
}

func TestGetUserStatisticsNoUser(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	req, err := http.NewRequest("GET", "/me/stats", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	GetUserStatistics(ctx, rr, req)

	context.TestUnauthorized(t, rr, "please login first")
}
