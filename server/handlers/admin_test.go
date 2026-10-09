package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
)

// activityPoint decodes the activity-series wire shape: day is a "YYYY-MM-DD"
// string, which is the public contract (ActivityDailyPoint.MarshalJSON), not an
// RFC3339 timestamp.
type activityPoint struct {
	Day             string `json:"day"`
	Downloads       int64  `json:"downloads"`
	DownloadedBytes int64  `json:"downloadedBytes"`
	Uploads         int64  `json:"uploads"`
	UploadedBytes   int64  `json:"uploadedBytes"`
}

func TestGetServerActivityDaily(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	createAdminUser(t, ctx)

	upload := &common.Upload{}
	file := upload.NewFile()
	file.Size = 10
	file.Status = common.FileUploaded
	upload.InitializeForTests()
	require.NoError(t, ctx.GetMetadataBackend().CreateUpload(upload))
	require.NoError(t, ctx.GetMetadataBackend().RecordFileDownload(upload, file, 512, true))

	req, err := http.NewRequest("GET", "/stats/activity/daily?days=7", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")
	rr := ctx.NewRecorder(req)
	GetServerActivityDaily(ctx, rr, req)
	context.TestOK(t, rr)

	require.True(t, bytes.HasPrefix(bytes.TrimSpace(rr.Body.Bytes()), []byte("[")), "response must be a JSON array, never null")

	var points []activityPoint
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &points))
	require.Len(t, points, 7, "dense series must return exactly days points")
	require.Equal(t, int64(0), points[0].Downloads, "older days are zero-filled")

	last := points[len(points)-1]
	require.Equal(t, time.Now().UTC().Format("2006-01-02"), last.Day, "last point is today (YYYY-MM-DD)")
	require.Equal(t, int64(1), last.Downloads, "today's download event")
	require.Equal(t, int64(512), last.DownloadedBytes, "today's egress bytes")
	// The handlers-package test DB accumulates uploads across tests, so assert the
	// server-wide count includes at least our upload rather than an exact figure.
	require.GreaterOrEqual(t, last.Uploads, int64(1), "today's upload creation is recorded in the CreateUpload transaction")
}

func TestGetServerActivityDailyDefaultDays(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	createAdminUser(t, ctx)

	req, err := http.NewRequest("GET", "/stats/activity/daily", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")
	rr := ctx.NewRecorder(req)
	GetServerActivityDaily(ctx, rr, req)
	context.TestOK(t, rr)

	var points []activityPoint
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &points))
	require.Len(t, points, 30, "default days is 30")
}

func TestGetServerActivityDailyForbidden(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	createAdminUser(t, ctx)
	ctx.GetUser().IsAdmin = false

	req, err := http.NewRequest("GET", "/stats/activity/daily", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")
	rr := ctx.NewRecorder(req)
	GetServerActivityDaily(ctx, rr, req)
	context.TestForbidden(t, rr, "you need administrator privileges")
}

func TestGetServerActivityDailyInvalidDays(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	createAdminUser(t, ctx)

	for _, q := range []string{"days=0", "days=-1", "days=32", "days=100"} {
		req, err := http.NewRequest("GET", "/stats/activity/daily?"+q, bytes.NewBuffer([]byte{}))
		require.NoError(t, err, "unable to create new request")
		rr := ctx.NewRecorder(req)
		GetServerActivityDaily(ctx, rr, req)
		context.TestBadRequest(t, rr, "invalid days")
	}

	// Non-integer days must 400 with a curated message that hides strconv internals.
	req, err := http.NewRequest("GET", "/stats/activity/daily?days=abc", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")
	rr := ctx.NewRecorder(req)
	GetServerActivityDaily(ctx, rr, req)
	context.TestBadRequest(t, rr, "invalid days")
	respBody, err := io.ReadAll(rr.Body)
	require.NoError(t, err)
	require.NotContains(t, string(respBody), "strconv", "error message should not expose strconv internals")
}

func TestGetServerActivityDailyBoundaryDays(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	createAdminUser(t, ctx)

	for days, want := range map[string]int{"days=1": 1, "days=31": 31} {
		req, err := http.NewRequest("GET", "/stats/activity/daily?"+days, bytes.NewBuffer([]byte{}))
		require.NoError(t, err, "unable to create new request")
		rr := ctx.NewRecorder(req)
		GetServerActivityDaily(ctx, rr, req)
		context.TestOK(t, rr)
		var points []activityPoint
		require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &points))
		require.Len(t, points, want, "%s", days)
	}
}

func TestGetServerActivityDailyMetadataBackendError(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	createAdminUser(t, ctx)
	require.NoError(t, ctx.GetMetadataBackend().Shutdown())

	req, err := http.NewRequest("GET", "/stats/activity/daily?days=7", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")
	rr := ctx.NewRecorder(req)
	GetServerActivityDaily(ctx, rr, req)
	context.TestInternalServerError(t, rr, "database is closed")
}

func createAdminUser(t *testing.T, ctx *context.Context) (user *common.User) {
	user = common.NewUser(common.ProviderLocal, "admin")
	user.IsAdmin = true
	user.Email = "admin@root.gg"
	user.Login = "admin"
	user.Password = "passwords"
	ctx.SetUser(user)

	err := ctx.GetMetadataBackend().CreateUser(user)
	require.NoError(t, err, "create admin user error")
	return user
}

func TestGetUsers(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	createAdminUser(t, ctx)

	user1 := common.NewUser(common.ProviderLocal, "user1")
	user1.Email = "user1@root.gg"
	user1.Login = "user1"
	user1.Password = "pass"

	user2 := common.NewUser(common.ProviderLocal, "user2")
	user2.Email = "user2@root.gg"
	user2.Login = "user2"
	user2.Password = "pass"

	err := ctx.GetMetadataBackend().CreateUser(user1)
	require.NoError(t, err, "unable to create user1")

	err = ctx.GetMetadataBackend().CreateUser(user2)
	require.NoError(t, err, "unable to create user2")

	req, err := http.NewRequest("GET", "/users", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	ctx.SetPagingQuery(&common.PagingQuery{})
	rr := ctx.NewRecorder(req)
	GetUsers(ctx, rr, req)

	context.TestOK(t, rr)

	respBody, err := io.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")

	var response common.PagingResponse
	err = json.Unmarshal(respBody, &response)
	require.NoError(t, err, "unable to unmarshal response body %s", respBody)
	require.Equal(t, 3, len(response.Results), "invalid upload count")
}

// TestGetUsersSortedByDownloadedBytes exercises the handler wiring for
// sort=downloadedBytes end-to-end (query param -> isUserSort -> GetUsers ->
// getUsersSortedByUsage); TestBackend_GetUsers_SortByDownloadedBytes in
// server/metadata covers the sort order itself in depth.
func TestGetUsersSortedByDownloadedBytes(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	createAdminUser(t, ctx)

	user := common.NewUser(common.ProviderLocal, "downloaded-bytes-user")
	err := ctx.GetMetadataBackend().CreateUser(user)
	require.NoError(t, err, "unable to create user")

	upload := common.NewUpload()
	upload.User = user.ID
	file := upload.NewFile()
	file.Status = common.FileUploaded
	file.Size = 42
	err = ctx.GetMetadataBackend().CreateUpload(upload)
	require.NoError(t, err, "unable to create upload")
	require.NoError(t, ctx.GetMetadataBackend().RecordFileDownload(upload, file, 42, true))

	req, err := http.NewRequest("GET", "/users?sort=downloadedBytes", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	ctx.SetPagingQuery(&common.PagingQuery{})
	rr := ctx.NewRecorder(req)
	GetUsers(ctx, rr, req)

	context.TestOK(t, rr)
}

func TestGetUsersFilterByProvider(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	createAdminUser(t, ctx)

	user1 := common.NewUser(common.ProviderGoogle, "guser")
	user1.Login = "guser"
	err := ctx.GetMetadataBackend().CreateUser(user1)
	require.NoError(t, err, "unable to create google user")

	req, err := http.NewRequest("GET", "/users?provider=google", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	ctx.SetPagingQuery(&common.PagingQuery{})
	rr := ctx.NewRecorder(req)
	GetUsers(ctx, rr, req)

	context.TestOK(t, rr)

	respBody, err := io.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")

	var response common.PagingResponse
	err = json.Unmarshal(respBody, &response)
	require.NoError(t, err, "unable to unmarshal response body %s", respBody)
	require.Equal(t, 1, len(response.Results), "should only return google users")
}

func TestGetUsersFilterByAdmin(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	createAdminUser(t, ctx) // 1 admin

	user1 := common.NewUser(common.ProviderLocal, "regular")
	user1.Login = "regular"
	user1.Password = "pass"
	err := ctx.GetMetadataBackend().CreateUser(user1)
	require.NoError(t, err, "unable to create regular user")

	// Filter admin=true
	req, err := http.NewRequest("GET", "/users?admin=true", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	ctx.SetPagingQuery(&common.PagingQuery{})
	rr := ctx.NewRecorder(req)
	GetUsers(ctx, rr, req)

	context.TestOK(t, rr)

	respBody, err := io.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")

	var response common.PagingResponse
	err = json.Unmarshal(respBody, &response)
	require.NoError(t, err, "unable to unmarshal response body %s", respBody)
	require.Equal(t, 1, len(response.Results), "should only return admin users")

	// Filter admin=false
	req, err = http.NewRequest("GET", "/users?admin=false", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	ctx.SetPagingQuery(&common.PagingQuery{})
	rr = ctx.NewRecorder(req)
	GetUsers(ctx, rr, req)

	context.TestOK(t, rr)

	respBody, err = io.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")

	err = json.Unmarshal(respBody, &response)
	require.NoError(t, err, "unable to unmarshal response body %s", respBody)
	require.Equal(t, 1, len(response.Results), "should only return non-admin users")
}

func TestGetUsersNoUser(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	req, err := http.NewRequest("GET", "/users", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	GetUsers(ctx, rr, req)

	context.TestForbidden(t, rr, "you need administrator privileges")
}

func TestGetUsersNotAdmin(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	createAdminUser(t, ctx)
	ctx.GetUser().IsAdmin = false

	req, err := http.NewRequest("GET", "/users", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	GetUsers(ctx, rr, req)

	context.TestForbidden(t, rr, "you need administrator privileges")
}

func TestGetUsersMetadataBackendError(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	createAdminUser(t, ctx)
	ctx.GetUser().IsAdmin = true
	ctx.SetPagingQuery(&common.PagingQuery{})

	err := ctx.GetMetadataBackend().Shutdown()
	require.NoError(t, err, "unable to shutdown metadata backend")

	req, err := http.NewRequest("GET", "/users", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	GetUsers(ctx, rr, req)

	context.TestInternalServerError(t, rr, "database is closed")
}

func TestSearchUsers(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	createAdminUser(t, ctx)

	user1 := common.NewUser(common.ProviderLocal, "alice")
	user1.Login = "alice"
	user1.Name = "Alice Wonderland"
	err := ctx.GetMetadataBackend().CreateUser(user1)
	require.NoError(t, err)

	user2 := common.NewUser(common.ProviderLocal, "bob")
	user2.Login = "bob"
	err = ctx.GetMetadataBackend().CreateUser(user2)
	require.NoError(t, err)

	req, err := http.NewRequest("GET", "/users/search?q=ali", bytes.NewBuffer([]byte{}))
	require.NoError(t, err)

	rr := ctx.NewRecorder(req)
	SearchUsers(ctx, rr, req)

	context.TestOK(t, rr)

	respBody, err := io.ReadAll(rr.Body)
	require.NoError(t, err)

	var users []*common.User
	err = json.Unmarshal(respBody, &users)
	require.NoError(t, err)
	require.Len(t, users, 1)
	require.Equal(t, "alice", users[0].Login)
}

func TestSearchUsersEmptyQuery(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	createAdminUser(t, ctx)

	req, err := http.NewRequest("GET", "/users/search", bytes.NewBuffer([]byte{}))
	require.NoError(t, err)

	rr := ctx.NewRecorder(req)
	SearchUsers(ctx, rr, req)

	context.TestBadRequest(t, rr, "search query must be at least 2 characters")
}

func TestSearchUsersShortQuery(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	createAdminUser(t, ctx)

	req, err := http.NewRequest("GET", "/users/search?q=a", bytes.NewBuffer([]byte{}))
	require.NoError(t, err)

	rr := ctx.NewRecorder(req)
	SearchUsers(ctx, rr, req)

	context.TestBadRequest(t, rr, "search query must be at least 2 characters")
}

func TestSearchUsersWithProvider(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	createAdminUser(t, ctx)

	user1 := common.NewUser(common.ProviderLocal, "alice")
	user1.Login = "alice"
	err := ctx.GetMetadataBackend().CreateUser(user1)
	require.NoError(t, err)

	user2 := common.NewUser(common.ProviderGoogle, "alicia")
	user2.Login = "alicia"
	err = ctx.GetMetadataBackend().CreateUser(user2)
	require.NoError(t, err)

	req, err := http.NewRequest("GET", "/users/search?q=ali&provider=local", bytes.NewBuffer([]byte{}))
	require.NoError(t, err)

	rr := ctx.NewRecorder(req)
	SearchUsers(ctx, rr, req)

	context.TestOK(t, rr)

	respBody, err := io.ReadAll(rr.Body)
	require.NoError(t, err)

	var users []*common.User
	err = json.Unmarshal(respBody, &users)
	require.NoError(t, err)
	require.Len(t, users, 1)
	require.Equal(t, "alice", users[0].Login)
}

func TestSearchUsersNotAdmin(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	createAdminUser(t, ctx)
	ctx.GetUser().IsAdmin = false

	req, err := http.NewRequest("GET", "/users/search?q=test", bytes.NewBuffer([]byte{}))
	require.NoError(t, err)

	rr := ctx.NewRecorder(req)
	SearchUsers(ctx, rr, req)

	context.TestForbidden(t, rr, "you need administrator privileges")
}

func createTestUploads(t *testing.T, ctx *context.Context) {
	upload1 := common.NewUpload()
	upload1.Comments = "1"
	upload1.DownloadCount = 30
	f1 := upload1.NewFile()
	f1.Status = common.FileUploaded
	f1.Size = 1
	err := ctx.GetMetadataBackend().CreateUpload(upload1)
	require.NoError(t, err, "unable to create upload1")

	upload2 := common.NewUpload()
	upload2.Comments = "2"
	upload2.DownloadCount = 10
	f2 := upload2.NewFile()
	f2.Status = common.FileUploaded
	f2.Size = 3
	upload2.User = "user"
	err = ctx.GetMetadataBackend().CreateUpload(upload2)
	require.NoError(t, err, "unable to create upload2")

	upload3 := common.NewUpload()
	upload3.Comments = "3"
	upload3.DownloadCount = 20
	f3 := upload3.NewFile()
	f3.Status = common.FileUploaded
	f3.Size = 2
	upload3.User = "user"
	upload3.Token = "token"
	err = ctx.GetMetadataBackend().CreateUpload(upload3)
	require.NoError(t, err, "unable to create upload3")
}

// createTestUploadsWithDownloadedBytes mirrors createTestUploads but varies
// DownloadedBytes instead of DownloadCount, for sort=downloadedBytes tests.
func createTestUploadsWithDownloadedBytes(t *testing.T, ctx *context.Context) {
	upload1 := common.NewUpload()
	upload1.Comments = "1"
	upload1.DownloadedBytes = 30
	f1 := upload1.NewFile()
	f1.Status = common.FileUploaded
	f1.Size = 1
	err := ctx.GetMetadataBackend().CreateUpload(upload1)
	require.NoError(t, err, "unable to create upload1")

	upload2 := common.NewUpload()
	upload2.Comments = "2"
	upload2.DownloadedBytes = 10
	f2 := upload2.NewFile()
	f2.Status = common.FileUploaded
	f2.Size = 3
	upload2.User = "user"
	err = ctx.GetMetadataBackend().CreateUpload(upload2)
	require.NoError(t, err, "unable to create upload2")

	upload3 := common.NewUpload()
	upload3.Comments = "3"
	upload3.DownloadedBytes = 20
	f3 := upload3.NewFile()
	f3.Status = common.FileUploaded
	f3.Size = 2
	upload3.User = "user"
	upload3.Token = "token"
	err = ctx.GetMetadataBackend().CreateUpload(upload3)
	require.NoError(t, err, "unable to create upload3")
}

func getOrder(t *testing.T, response common.PagingResponse) []int {
	order := make([]int, len(response.Results))
	for idx, u := range response.Results {
		upload := u.(map[string]any)
		i, err := strconv.Atoi(upload["comments"].(string))
		require.NoError(t, err)
		order[idx] = i
	}
	return order
}

func TestGetUploads(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	createAdminUser(t, ctx)
	createTestUploads(t, ctx)

	req, err := http.NewRequest("GET", "/uploads", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	ctx.SetPagingQuery(&common.PagingQuery{})
	rr := ctx.NewRecorder(req)
	GetUploads(ctx, rr, req)

	context.TestOK(t, rr)

	respBody, err := io.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")

	var response common.PagingResponse
	err = json.Unmarshal(respBody, &response)
	require.NoError(t, err, "unable to unmarshal response body %s", respBody)
	require.Equal(t, 3, len(response.Results), "invalid upload count")
	require.Equal(t, []int{3, 2, 1}, getOrder(t, response))
}

func TestGetUploadsAsc(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	createAdminUser(t, ctx)
	createTestUploads(t, ctx)

	req, err := http.NewRequest("GET", "/uploads", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	ctx.SetPagingQuery(&common.PagingQuery{})
	rr := ctx.NewRecorder(req)
	GetUploads(ctx, rr, req)

	context.TestOK(t, rr)

	respBody, err := io.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")

	var response common.PagingResponse
	err = json.Unmarshal(respBody, &response)
	require.NoError(t, err, "unable to unmarshal response body %s", respBody)
	require.Equal(t, 3, len(response.Results), "invalid upload count")
	require.Equal(t, []int{3, 2, 1}, getOrder(t, response))
}

func TestGetUploadsUser(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	createAdminUser(t, ctx)
	createTestUploads(t, ctx)

	req, err := http.NewRequest("GET", "/uploads", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	query := req.URL.Query()
	query.Add("user", "user")
	req.URL.RawQuery = query.Encode()

	ctx.SetPagingQuery(&common.PagingQuery{})
	rr := ctx.NewRecorder(req)
	GetUploads(ctx, rr, req)

	context.TestOK(t, rr)

	respBody, err := io.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")

	var response common.PagingResponse
	err = json.Unmarshal(respBody, &response)
	require.NoError(t, err, "unable to unmarshal response body %s", respBody)
	require.Equal(t, 2, len(response.Results), "invalid upload count")
	require.Equal(t, []int{3, 2}, getOrder(t, response))
}

func TestGetUploadsUserToken(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	createAdminUser(t, ctx)
	createTestUploads(t, ctx)

	req, err := http.NewRequest("GET", "/uploads", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	query := req.URL.Query()
	query.Add("user", "user")
	query.Add("token", "token")
	req.URL.RawQuery = query.Encode()

	ctx.SetPagingQuery(&common.PagingQuery{})
	rr := ctx.NewRecorder(req)
	GetUploads(ctx, rr, req)

	context.TestOK(t, rr)

	respBody, err := io.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")

	var response common.PagingResponse
	err = json.Unmarshal(respBody, &response)
	require.NoError(t, err, "unable to unmarshal response body %s", respBody)
	require.Equal(t, 1, len(response.Results), "invalid upload count")
	require.Equal(t, []int{3}, getOrder(t, response))
}

func TestGetUploadsNotAdmin(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	createAdminUser(t, ctx)
	ctx.GetUser().IsAdmin = false

	req, err := http.NewRequest("GET", "/admin/users", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	GetUsers(ctx, rr, req)

	context.TestForbidden(t, rr, "you need administrator privileges")
}

func TestGetUploadsSortedBySize(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	createAdminUser(t, ctx)
	createTestUploads(t, ctx)

	req, err := http.NewRequest("GET", "/uploads", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	query := req.URL.Query()
	query.Add("sort", "size")
	req.URL.RawQuery = query.Encode()

	ctx.SetPagingQuery(&common.PagingQuery{})
	rr := ctx.NewRecorder(req)
	GetUploads(ctx, rr, req)

	context.TestOK(t, rr)

	respBody, err := io.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")

	var response common.PagingResponse
	err = json.Unmarshal(respBody, &response)
	require.NoError(t, err, "unable to unmarshal response body %s", respBody)
	require.Equal(t, 3, len(response.Results), "invalid upload count")
	require.Equal(t, []int{2, 3, 1}, getOrder(t, response))
}

func TestGetUploadsSortedByDownloads(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	createAdminUser(t, ctx)
	createTestUploads(t, ctx)

	req, err := http.NewRequest("GET", "/uploads", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	query := req.URL.Query()
	query.Add("sort", "downloads")
	req.URL.RawQuery = query.Encode()

	ctx.SetPagingQuery(&common.PagingQuery{})
	rr := ctx.NewRecorder(req)
	GetUploads(ctx, rr, req)

	context.TestOK(t, rr)

	respBody, err := io.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")

	var response common.PagingResponse
	err = json.Unmarshal(respBody, &response)
	require.NoError(t, err, "unable to unmarshal response body %s", respBody)
	require.Equal(t, 3, len(response.Results), "invalid upload count")
	require.Equal(t, []int{1, 3, 2}, getOrder(t, response))
}

func TestGetUploadsSortedByDownloadedBytes(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	createAdminUser(t, ctx)
	createTestUploadsWithDownloadedBytes(t, ctx)

	req, err := http.NewRequest("GET", "/uploads", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	query := req.URL.Query()
	query.Add("sort", "downloadedBytes")
	req.URL.RawQuery = query.Encode()

	ctx.SetPagingQuery(&common.PagingQuery{})
	rr := ctx.NewRecorder(req)
	GetUploads(ctx, rr, req)

	context.TestOK(t, rr)

	respBody, err := io.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")

	var response common.PagingResponse
	err = json.Unmarshal(respBody, &response)
	require.NoError(t, err, "unable to unmarshal response body %s", respBody)
	require.Equal(t, 3, len(response.Results), "invalid upload count")
	require.Equal(t, []int{1, 3, 2}, getOrder(t, response))
}

func TestGetUploadsInvalidSort(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	createAdminUser(t, ctx)

	req, err := http.NewRequest("GET", "/uploads?sort=wat", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	ctx.SetPagingQuery(&common.PagingQuery{})
	rr := ctx.NewRecorder(req)
	GetUploads(ctx, rr, req)

	context.TestBadRequest(t, rr, "invalid sort")
}

func TestGetUploadsMetadataBackendError(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	createAdminUser(t, ctx)
	ctx.GetUser().IsAdmin = true
	ctx.SetPagingQuery(&common.PagingQuery{})

	err := ctx.GetMetadataBackend().Shutdown()
	require.NoError(t, err, "unable to shutdown metadata backend")

	req, err := http.NewRequest("GET", "/admin/users", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	GetUsers(ctx, rr, req)

	context.TestInternalServerError(t, rr, "database is closed")
}

func TestGetServerStatistics(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	createAdminUser(t, ctx)

	var downloadedUpload *common.Upload
	var downloadedFile *common.File
	for range 10 {
		upload := &common.Upload{}
		file := upload.NewFile()
		file.Size = 2
		file.Status = common.FileUploaded
		upload.InitializeForTests()
		err := ctx.GetMetadataBackend().CreateUpload(upload)
		require.NoError(t, err, "create error")
		if downloadedUpload == nil {
			downloadedUpload = upload
			downloadedFile = file
		}
	}

	for range 10 {
		upload := &common.Upload{}
		upload.User = ctx.GetUser().ID
		file := upload.NewFile()
		file.Size = 3
		file.Status = common.FileUploaded
		upload.InitializeForTests()
		err := ctx.GetMetadataBackend().CreateUpload(upload)
		require.NoError(t, err, "create error")
	}
	require.NoError(t, ctx.GetMetadataBackend().RecordFileDownload(downloadedUpload, downloadedFile, 1024, true))

	req, err := http.NewRequest("GET", "/admin/stats", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	GetServerStatistics(ctx, rr, req)

	context.TestOK(t, rr)

	respBody, err := io.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")

	var stats *common.ServerStats
	err = json.Unmarshal(respBody, &stats)
	require.NoError(t, err, "unable to unmarshal response body")

	require.NotNil(t, stats, "invalid server statistics")
	require.Equal(t, 20, stats.Uploads, "invalid upload count")
	require.Equal(t, 20, stats.Files, "invalid files count")
	require.Equal(t, int64(50), stats.TotalSize, "invalid total file size")
	require.Equal(t, 10, stats.AnonymousUploads, "invalid anonymous upload count")
	require.Equal(t, int64(20), stats.AnonymousSize, "invalid anonymous total file size")
	require.NotNil(t, stats.Usage, "missing nested usage stats")
	require.NotNil(t, stats.AnonymousUsage, "missing nested anonymous usage stats")
	require.Equal(t, int64(1), stats.Usage.Downloads.Total, "invalid download count")
	require.NotNil(t, stats.Usage.Downloads.Today, "missing 1d download window")
	require.Equal(t, int64(1), *stats.Usage.Downloads.Today, "invalid 1d download count")
	require.Equal(t, int64(1), *stats.Usage.Downloads.Last7Days, "invalid 7d download count")
	require.Equal(t, int64(1), *stats.Usage.Downloads.Last30Days, "invalid 30d download count")
	require.Equal(t, 20, stats.Usage.Current.Uploads, "invalid nested current upload count")
	require.Equal(t, 10, stats.AnonymousUsage.Current.Uploads, "invalid nested anonymous upload count")
	require.Nil(t, stats.AnonymousUsage.Downloads.Today, "anonymous usage must omit download windows")
}

func TestGetServerStatisticsNoUser(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	req, err := http.NewRequest("GET", "/admin/users", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")
	ctx.SetUser(nil)

	rr := ctx.NewRecorder(req)
	GetServerStatistics(ctx, rr, req)

	context.TestForbidden(t, rr, "you need administrator privileges")
}

func TestGetServerStatisticsNotAdmin(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	createAdminUser(t, ctx)
	ctx.GetUser().IsAdmin = false

	req, err := http.NewRequest("GET", "/admin/users", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	GetServerStatistics(ctx, rr, req)

	context.TestForbidden(t, rr, "you need administrator privileges")
}

func TestGetServerStatisticsMetadataBackendError(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	createAdminUser(t, ctx)
	ctx.GetUser().IsAdmin = true

	err := ctx.GetMetadataBackend().Shutdown()
	require.NoError(t, err, "unable to shutdown metadata backend")

	req, err := http.NewRequest("GET", "/admin/users", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	GetServerStatistics(ctx, rr, req)

	context.TestInternalServerError(t, rr, "database is closed")
}

func TestGetTrendingUploads(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	createAdminUser(t, ctx)

	upload := &common.Upload{Comments: "trending upload"}
	file := upload.NewFile()
	file.Name = "trending.txt"
	file.Status = common.FileUploaded
	createTestUpload(t, ctx, upload)
	require.NoError(t, ctx.GetMetadataBackend().RecordFileDownload(upload, file, 1024, true))

	req, err := http.NewRequest("GET", "/stats/trending/uploads?window=all&limit=1", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	GetTrendingUploads(ctx, rr, req)

	context.TestOK(t, rr)

	var items []*common.TrendingItem
	err = json.Unmarshal(rr.Body.Bytes(), &items)
	require.NoError(t, err, "unable to unmarshal response body")
	require.Len(t, items, 1)
	require.Equal(t, upload.ID, items[0].ID)
	require.Equal(t, common.DownloadStatsEntityUpload, items[0].Type)
	require.Equal(t, int64(1), items[0].DownloadCount)
}

func TestGetTrendingFiles(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	createAdminUser(t, ctx)

	upload := &common.Upload{}
	file := upload.NewFile()
	file.Name = "trending.txt"
	file.Status = common.FileUploaded
	createTestUpload(t, ctx, upload)
	require.NoError(t, ctx.GetMetadataBackend().RecordFileDownload(upload, file, 1024, true))

	req, err := http.NewRequest("GET", "/stats/trending/files?window=1d&limit=1", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	GetTrendingFiles(ctx, rr, req)

	context.TestOK(t, rr)

	var items []*common.TrendingItem
	err = json.Unmarshal(rr.Body.Bytes(), &items)
	require.NoError(t, err, "unable to unmarshal response body")
	require.Len(t, items, 1)
	require.Equal(t, file.ID, items[0].ID)
	require.Equal(t, upload.ID, items[0].UploadID)
	require.Equal(t, common.DownloadStatsEntityFile, items[0].Type)
	require.Equal(t, int64(1), items[0].DownloadCount)
}

func TestGetTrendingFilesForbidden(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	createAdminUser(t, ctx)
	ctx.GetUser().IsAdmin = false

	req, err := http.NewRequest("GET", "/stats/trending/files", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	GetTrendingFiles(ctx, rr, req)

	context.TestForbidden(t, rr, "you need administrator privileges")
}

func TestGetTrendingFilesInvalidLimit(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	createAdminUser(t, ctx)

	req, err := http.NewRequest("GET", "/stats/trending/files?limit=0", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	GetTrendingFiles(ctx, rr, req)

	context.TestBadRequest(t, rr, "limit must be positive")

	// Test non-integer limit
	req, err = http.NewRequest("GET", "/stats/trending/files?limit=abc", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr = ctx.NewRecorder(req)
	GetTrendingFiles(ctx, rr, req)

	context.TestBadRequest(t, rr, "invalid limit")
	respBody, err := io.ReadAll(rr.Body)
	require.NoError(t, err)
	require.NotContains(t, string(respBody), "strconv", "error message should not expose strconv internals")
}

func TestGetTrendingFilesInvalidWindow(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	createAdminUser(t, ctx)

	req, err := http.NewRequest("GET", "/stats/trending/files?window=bad", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	GetTrendingFiles(ctx, rr, req)

	context.TestBadRequest(t, rr, "invalid trending window")
}

func TestGetTrendingFilesMetadataBackendError(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	createAdminUser(t, ctx)

	err := ctx.GetMetadataBackend().Shutdown()
	require.NoError(t, err, "unable to shutdown metadata backend")

	req, err := http.NewRequest("GET", "/stats/trending/files?window=1d", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	GetTrendingFiles(ctx, rr, req)

	context.TestInternalServerError(t, rr, "database is closed")
}

func TestGetTrendingUploadsForbidden(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	createAdminUser(t, ctx)
	ctx.GetUser().IsAdmin = false

	req, err := http.NewRequest("GET", "/stats/trending/uploads", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	GetTrendingUploads(ctx, rr, req)

	context.TestForbidden(t, rr, "you need administrator privileges")
}

func TestIsUsageStatsSort(t *testing.T) {
	require.True(t, isUsageStatsSort(""))
	require.True(t, isUsageStatsSort("date"))
	require.True(t, isUsageStatsSort("size"))
	require.True(t, isUsageStatsSort("lifetimeSize"))
	require.False(t, isUsageStatsSort("downloads"))
	require.False(t, isUsageStatsSort("downloadedBytes"), "downloadedBytes is GET /users-only; GET /me/token must keep rejecting it")
	require.False(t, isUsageStatsSort("wat"))
}

func TestIsUserSort(t *testing.T) {
	require.True(t, isUserSort(""))
	require.True(t, isUserSort("date"))
	require.True(t, isUserSort("size"))
	require.True(t, isUserSort("lifetimeSize"))
	require.True(t, isUserSort("downloadedBytes"))
	require.False(t, isUserSort("downloads"))
	require.False(t, isUserSort("wat"))
}

func TestGetTrendingUploadsEmpty(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	createAdminUser(t, ctx)

	req, err := http.NewRequest("GET", "/stats/trending/uploads", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	GetTrendingUploads(ctx, rr, req)

	context.TestOK(t, rr)

	respBody := bytes.TrimSpace(rr.Body.Bytes())
	require.Equal(t, "[]", string(respBody), "empty trending uploads should return empty array, not null")
}

func TestGetTrendingFilesEmpty(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	createAdminUser(t, ctx)

	req, err := http.NewRequest("GET", "/stats/trending/files", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	GetTrendingFiles(ctx, rr, req)

	context.TestOK(t, rr)

	respBody := bytes.TrimSpace(rr.Body.Bytes())
	require.Equal(t, "[]", string(respBody), "empty trending files should return empty array, not null")
}

func TestGetTrendingUploadsInvalidWindow(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	createAdminUser(t, ctx)

	req, err := http.NewRequest("GET", "/stats/trending/uploads?window=bad", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	GetTrendingUploads(ctx, rr, req)

	context.TestBadRequest(t, rr, "invalid trending window")
}

func TestGetTrendingUploadsMetadataBackendError(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	createAdminUser(t, ctx)

	err := ctx.GetMetadataBackend().Shutdown()
	require.NoError(t, err, "unable to shutdown metadata backend")

	req, err := http.NewRequest("GET", "/stats/trending/uploads?window=all", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	GetTrendingUploads(ctx, rr, req)

	context.TestInternalServerError(t, rr, "database is closed")
}

func TestGetTrendingUploadsSortByDownloadedBytes(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	createAdminUser(t, ctx)

	upload := &common.Upload{Comments: "trending upload"}
	file := upload.NewFile()
	file.Name = "trending.txt"
	file.Status = common.FileUploaded
	createTestUpload(t, ctx, upload)
	require.NoError(t, ctx.GetMetadataBackend().RecordFileDownload(upload, file, 4096, true))

	req, err := http.NewRequest("GET", "/stats/trending/uploads?window=all&sort=downloadedBytes", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	GetTrendingUploads(ctx, rr, req)

	context.TestOK(t, rr)

	var items []*common.TrendingItem
	err = json.Unmarshal(rr.Body.Bytes(), &items)
	require.NoError(t, err, "unable to unmarshal response body")
	require.Len(t, items, 1)
	require.Equal(t, upload.ID, items[0].ID)
	require.Equal(t, int64(1), items[0].DownloadCount, "downloadCount stays populated when sorting by bytes")
	require.Equal(t, int64(4096), items[0].DownloadedBytes)
}

func TestGetTrendingUploadsInvalidSort(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	createAdminUser(t, ctx)

	req, err := http.NewRequest("GET", "/stats/trending/uploads?sort=bad", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	GetTrendingUploads(ctx, rr, req)

	context.TestBadRequest(t, rr, "invalid trending sort")
}
