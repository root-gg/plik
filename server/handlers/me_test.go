package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
)

// userActivityPoint decodes the activity-series wire shape (day as "YYYY-MM-DD").
type userActivityPoint struct {
	Day             string `json:"day"`
	Downloads       int64  `json:"downloads"`
	DownloadedBytes int64  `json:"downloadedBytes"`
	Uploads         int64  `json:"uploads"`
	UploadedBytes   int64  `json:"uploadedBytes"`
}

func TestGetUserActivityDaily(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	user := common.NewUser(common.ProviderLocal, "daily-user")
	require.NoError(t, ctx.GetMetadataBackend().CreateUser(user))
	ctx.SetUser(user)

	upload := &common.Upload{User: user.ID}
	file := upload.NewFile()
	file.Size = 10
	file.Status = common.FileUploaded
	createTestUpload(t, ctx, upload)
	require.NoError(t, ctx.GetMetadataBackend().RecordFileDownload(upload, file, 256, true))

	req, err := http.NewRequest("GET", "/me/stats/activity/daily?days=7", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")
	rr := ctx.NewRecorder(req)
	GetUserActivityDaily(ctx, rr, req)
	context.TestOK(t, rr)

	var points []userActivityPoint
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &points))
	require.Len(t, points, 7, "dense series must return exactly days points")
	last := points[len(points)-1]
	require.Equal(t, time.Now().UTC().Format("2006-01-02"), last.Day)
	require.Equal(t, int64(1), last.Downloads)
	require.Equal(t, int64(256), last.DownloadedBytes)
	require.GreaterOrEqual(t, last.Uploads, int64(1), "today's upload creation is recorded")
}

func TestGetUserActivityDailyUnauthorized(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	req, err := http.NewRequest("GET", "/me/stats/activity/daily", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")
	rr := ctx.NewRecorder(req)
	GetUserActivityDaily(ctx, rr, req)
	context.TestUnauthorized(t, rr, "missing user, please login first")
}

func TestGetUserActivityDailyInvalidDays(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	user := common.NewUser(common.ProviderLocal, "daily-invalid-user")
	require.NoError(t, ctx.GetMetadataBackend().CreateUser(user))
	ctx.SetUser(user)

	for _, q := range []string{"days=0", "days=-1", "days=32", "days=abc"} {
		req, err := http.NewRequest("GET", "/me/stats/activity/daily?"+q, bytes.NewBuffer([]byte{}))
		require.NoError(t, err, "unable to create new request")
		rr := ctx.NewRecorder(req)
		GetUserActivityDaily(ctx, rr, req)
		context.TestBadRequest(t, rr, "invalid days")
	}
}

// TestGetUserTrendingUploads pins that the self-scoped endpoint returns the
// same []TrendingItem shape as the admin endpoint, restricted to the caller's
// own uploads.
func TestGetUserTrendingUploads(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	user := common.NewUser(common.ProviderLocal, "trending-me-user")
	require.NoError(t, ctx.GetMetadataBackend().CreateUser(user))
	ctx.SetUser(user)

	upload := &common.Upload{User: user.ID, Comments: "mine"}
	file := upload.NewFile()
	file.Name = "trending.txt"
	file.Status = common.FileUploaded
	createTestUpload(t, ctx, upload)
	require.NoError(t, ctx.GetMetadataBackend().RecordFileDownload(upload, file, 1024, true))

	other := common.NewUser(common.ProviderLocal, "trending-me-other")
	require.NoError(t, ctx.GetMetadataBackend().CreateUser(other))
	otherUpload := &common.Upload{User: other.ID, Comments: "not mine"}
	otherFile := otherUpload.NewFile()
	otherFile.Name = "other.txt"
	otherFile.Status = common.FileUploaded
	otherUpload.InitializeForTests()
	require.NoError(t, ctx.GetMetadataBackend().CreateUpload(otherUpload))
	require.NoError(t, ctx.GetMetadataBackend().RecordFileDownload(otherUpload, otherFile, 2048, true))

	req, err := http.NewRequest("GET", "/me/stats/trending/uploads?window=all", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")
	rr := ctx.NewRecorder(req)
	GetUserTrendingUploads(ctx, rr, req)
	context.TestOK(t, rr)

	var items []*common.TrendingItem
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &items))
	require.Len(t, items, 1, "must return only the caller's own uploads")
	require.Equal(t, upload.ID, items[0].ID)
	require.Equal(t, common.DownloadStatsEntityUpload, items[0].Type)
	require.Equal(t, int64(1), items[0].DownloadCount)
	require.Equal(t, int64(1024), items[0].DownloadedBytes)
}

func TestGetUserTrendingUploadsUnauthorized(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	req, err := http.NewRequest("GET", "/me/stats/trending/uploads", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")
	rr := ctx.NewRecorder(req)
	GetUserTrendingUploads(ctx, rr, req)
	context.TestUnauthorized(t, rr, "missing user, please login first")
}

func TestGetUserTrendingUploadsInvalidWindow(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	user := common.NewUser(common.ProviderLocal, "trending-me-bad-window")
	require.NoError(t, ctx.GetMetadataBackend().CreateUser(user))
	ctx.SetUser(user)

	req, err := http.NewRequest("GET", "/me/stats/trending/uploads?window=bad", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")
	rr := ctx.NewRecorder(req)
	GetUserTrendingUploads(ctx, rr, req)
	context.TestBadRequest(t, rr, "invalid trending window")
}

func TestGetUserTrendingUploadsInvalidSort(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	user := common.NewUser(common.ProviderLocal, "trending-me-bad-sort")
	require.NoError(t, ctx.GetMetadataBackend().CreateUser(user))
	ctx.SetUser(user)

	req, err := http.NewRequest("GET", "/me/stats/trending/uploads?sort=bad", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")
	rr := ctx.NewRecorder(req)
	GetUserTrendingUploads(ctx, rr, req)
	context.TestBadRequest(t, rr, "invalid trending sort")
}

func TestGetUserTrendingUploadsEmpty(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	user := common.NewUser(common.ProviderLocal, "trending-me-empty")
	require.NoError(t, ctx.GetMetadataBackend().CreateUser(user))
	ctx.SetUser(user)

	req, err := http.NewRequest("GET", "/me/stats/trending/uploads", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")
	rr := ctx.NewRecorder(req)
	GetUserTrendingUploads(ctx, rr, req)
	context.TestOK(t, rr)

	respBody := bytes.TrimSpace(rr.Body.Bytes())
	require.Equal(t, "[]", string(respBody), "empty trending uploads should return empty array, not null")
}

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

	respBody, err := io.ReadAll(rr.Body)
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

func TestPatchMe(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	user := common.NewUser(common.ProviderLocal, "user1")
	err := ctx.GetMetadataBackend().CreateUser(user)
	require.NoError(t, err, "unable to create test user")
	ctx.SetUser(user)

	req, err := http.NewRequest("PATCH", "/me", bytes.NewBuffer([]byte(`{"theme":"dark","language":"fr","name":"Name","email":"name@example.com"}`)))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	PatchMe(ctx, rr, req)

	context.TestOK(t, rr)

	respBody, err := io.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")

	var userResult *common.User
	err = json.Unmarshal(respBody, &userResult)
	require.NoError(t, err, "unable to unmarshal response body")
	require.Equal(t, "dark", userResult.Theme)
	require.Equal(t, "fr", userResult.Language)
	require.Equal(t, "Name", userResult.Name)
	require.Equal(t, "name@example.com", userResult.Email)

	persisted, err := ctx.GetMetadataBackend().GetUser(user.ID)
	require.NoError(t, err)
	require.Equal(t, "dark", persisted.Theme)
	require.Equal(t, "fr", persisted.Language)
	require.Equal(t, "Name", persisted.Name)
	require.Equal(t, "name@example.com", persisted.Email)
}

func TestPatchMeNoUser(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	req, err := http.NewRequest("PATCH", "/me", bytes.NewBuffer([]byte(`{"theme":"dark"}`)))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	PatchMe(ctx, rr, req)

	context.TestUnauthorized(t, rr, "missing user, please login first")
}

func TestPatchMeInvalidBody(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	user := common.NewUser(common.ProviderLocal, "user1")
	err := ctx.GetMetadataBackend().CreateUser(user)
	require.NoError(t, err, "unable to create test user")
	ctx.SetUser(user)

	req, err := http.NewRequest("PATCH", "/me", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	PatchMe(ctx, rr, req)
	context.TestBadRequest(t, rr, "missing request body")

	req, err = http.NewRequest("PATCH", "/me", bytes.NewBuffer([]byte(`{"theme":`)))
	require.NoError(t, err, "unable to create new request")

	rr = ctx.NewRecorder(req)
	PatchMe(ctx, rr, req)
	context.TestBadRequest(t, rr, "unable to deserialize request body")
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

	respBody, err := io.ReadAll(rr.Body)
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

func TestDeleteUserDisabled(t *testing.T) {
	config := common.NewConfiguration()
	config.FeatureDeleteAccount = common.FeatureDisabled
	ctx := newTestingContext(config)

	user := common.NewUser(common.ProviderLocal, "user1")

	err := ctx.GetMetadataBackend().CreateUser(user)
	require.NoError(t, err, "unable to create test user")
	ctx.SetUser(user)

	req, err := http.NewRequest("DELETE", "/me", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	DeleteAccount(ctx, rr, req)

	context.TestBadRequest(t, rr, "delete account is not enabled")
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

	respBody, err := io.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")

	var response common.PagingResponse
	err = json.Unmarshal(respBody, &response)
	require.NoError(t, err, "unable to unmarshal response body %s", respBody)
	require.Equal(t, 2, len(response.Results), "invalid upload count")
}

func TestGetUserUploadsSortedByDownloads(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	user := common.NewUser(common.ProviderLocal, "user1")
	err := ctx.GetMetadataBackend().CreateUser(user)
	require.NoError(t, err, "unable to create test user")

	ctx.SetUser(user)

	upload1 := &common.Upload{User: user.ID, Comments: "1", DownloadCount: 3}
	createTestUpload(t, ctx, upload1)

	upload2 := &common.Upload{User: user.ID, Comments: "2", DownloadCount: 1}
	createTestUpload(t, ctx, upload2)

	upload3 := &common.Upload{Comments: "3", DownloadCount: 9}
	createTestUpload(t, ctx, upload3)

	req, err := http.NewRequest("GET", "/me/uploads?sort=downloads", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	ctx.SetPagingQuery(&common.PagingQuery{})

	rr := ctx.NewRecorder(req)
	GetUserUploads(ctx, rr, req)

	context.TestOK(t, rr)

	respBody, err := io.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")

	var response common.PagingResponse
	err = json.Unmarshal(respBody, &response)
	require.NoError(t, err, "unable to unmarshal response body %s", respBody)
	require.Equal(t, 2, len(response.Results), "invalid upload count")
	require.Equal(t, []int{1, 2}, getOrder(t, response))
}

func TestGetUserUploadsSortedByDownloadedBytes(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	user := common.NewUser(common.ProviderLocal, "user1")
	err := ctx.GetMetadataBackend().CreateUser(user)
	require.NoError(t, err, "unable to create test user")

	ctx.SetUser(user)

	upload1 := &common.Upload{User: user.ID, Comments: "1", DownloadedBytes: 300}
	createTestUpload(t, ctx, upload1)

	upload2 := &common.Upload{User: user.ID, Comments: "2", DownloadedBytes: 100}
	createTestUpload(t, ctx, upload2)

	upload3 := &common.Upload{Comments: "3", DownloadedBytes: 900}
	createTestUpload(t, ctx, upload3)

	req, err := http.NewRequest("GET", "/me/uploads?sort=downloadedBytes", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	ctx.SetPagingQuery(&common.PagingQuery{})

	rr := ctx.NewRecorder(req)
	GetUserUploads(ctx, rr, req)

	context.TestOK(t, rr)

	respBody, err := io.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")

	var response common.PagingResponse
	err = json.Unmarshal(respBody, &response)
	require.NoError(t, err, "unable to unmarshal response body %s", respBody)
	require.Equal(t, 2, len(response.Results), "invalid upload count")
	require.Equal(t, []int{1, 2}, getOrder(t, response))
}

func TestGetUserUploadsNoUser(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	req, err := http.NewRequest("GET", "/me/uploads", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	GetUserUploads(ctx, rr, req)

	context.TestUnauthorized(t, rr, "missing user, please login first")
}

func TestGetUserUploadsInvalidSort(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	user := common.NewUser(common.ProviderLocal, "user1")
	err := ctx.GetMetadataBackend().CreateUser(user)
	require.NoError(t, err, "unable to create user")
	ctx.SetUser(user)

	req, err := http.NewRequest("GET", "/me/uploads?sort=wat", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	ctx.SetPagingQuery(&common.PagingQuery{})
	rr := ctx.NewRecorder(req)
	GetUserUploads(ctx, rr, req)

	context.TestBadRequest(t, rr, "invalid sort")
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

	respBody, err := io.ReadAll(rr.Body)
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

	context.TestBadRequest(t, rr, "invalid token format")
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

	respBody, err := io.ReadAll(rr.Body)
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

func TestGetUserTokensApiTokensDisabled(t *testing.T) {
	config := common.NewConfiguration()
	config.FeatureApiTokens = common.FeatureDisabled
	ctx := newTestingContext(config)

	user := common.NewUser(common.ProviderLocal, "user1")
	err := ctx.GetMetadataBackend().CreateUser(user)
	require.NoError(t, err, "unable to create test user")
	ctx.SetUser(user)

	req, err := http.NewRequest("GET", "/me/token", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	ctx.SetPagingQuery(&common.PagingQuery{})
	rr := ctx.NewRecorder(req)
	GetUserTokens(ctx, rr, req)

	context.TestBadRequest(t, rr, "API tokens are disabled")
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

	respBody, err := io.ReadAll(rr.Body)
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

	respBody, err := io.ReadAll(rr.Body)
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

	context.TestBadRequest(t, rr, "invalid token format")
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

	respBody, err := io.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")

	var stats = &common.UserStats{}
	err = json.Unmarshal(respBody, stats)
	require.NoError(t, err, "unable to unmarshal response body")

	require.Equal(t, 2, stats.Uploads, "Invalid upload count")
	require.Equal(t, 2, stats.Files, "Invalid files count")
	require.Equal(t, int64(3), stats.TotalSize, "Invalid total size")
	require.NotNil(t, stats.Usage, "missing nested usage stats")
	require.Equal(t, 2, stats.Usage.Current.Uploads, "invalid nested current upload count")
	require.Equal(t, 2, stats.Usage.Current.Files, "invalid nested current file count")
	require.Equal(t, int64(3), stats.Usage.Current.TotalSize, "invalid nested current total size")
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

	respBody, err := io.ReadAll(rr.Body)
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

	context.TestBadRequest(t, rr, "invalid token format")
}

func TestGetUserStatisticsNoUser(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	req, err := http.NewRequest("GET", "/me/stats", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	GetUserStatistics(ctx, rr, req)

	context.TestUnauthorized(t, rr, "please login first")
}
