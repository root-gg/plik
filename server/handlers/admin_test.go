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

	req, err := http.NewRequest("GET", "/admin/users", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	ctx.SetPagingQuery(&common.PagingQuery{})
	rr := ctx.NewRecorder(req)
	GetUsers(ctx, rr, req)

	context.TestOK(t, rr)

	respBody, err := ioutil.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")

	var response common.PagingResponse
	err = json.Unmarshal(respBody, &response)
	require.NoError(t, err, "unable to unmarshal response body %s", respBody)
	require.Equal(t, 3, len(response.Results), "invalid upload count")
}

func TestGetUsersNoUser(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	req, err := http.NewRequest("GET", "/admin/users", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	GetUsers(ctx, rr, req)

	context.TestForbidden(t, rr, "you need administrator privileges")
}

func TestGetUsersNotAdmin(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	createAdminUser(t, ctx)
	ctx.GetUser().IsAdmin = false

	req, err := http.NewRequest("GET", "/admin/users", bytes.NewBuffer([]byte{}))
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

	req, err := http.NewRequest("GET", "/admin/users", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	GetUsers(ctx, rr, req)

	context.TestInternalServerError(t, rr, "database is closed")
}

func TestGetServerStatistics(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	createAdminUser(t, ctx)

	for i := 0; i < 10; i++ {
		upload := &common.Upload{}
		file := upload.NewFile()
		file.Size = 2
		file.Status = common.FileUploaded
		upload.InitializeForTests()
		err := ctx.GetMetadataBackend().CreateUpload(upload)
		require.NoError(t, err, "create error")
	}

	for i := 0; i < 10; i++ {
		upload := &common.Upload{}
		upload.User = ctx.GetUser().ID
		file := upload.NewFile()
		file.Size = 3
		file.Status = common.FileUploaded
		upload.InitializeForTests()
		err := ctx.GetMetadataBackend().CreateUpload(upload)
		require.NoError(t, err, "create error")
	}

	req, err := http.NewRequest("GET", "/admin/stats", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	GetServerStatistics(ctx, rr, req)

	context.TestOK(t, rr)

	respBody, err := ioutil.ReadAll(rr.Body)
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
