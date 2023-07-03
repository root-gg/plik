package middleware

import (
	"bytes"
	"net"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
)

func TestCreateUpload(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	ctx.GetConfig().FeatureAuthentication = common.FeatureEnabled
	ctx.GetConfig().DefaultTTL = 60

	ctx.SetSourceIP(net.ParseIP("1.2.3.4"))
	ctx.SetUser(&common.User{ID: "user"})
	ctx.SetToken(&common.Token{Token: "token"})

	req, err := http.NewRequest("GET", "", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	CreateUpload(ctx, common.DummyHandler).ServeHTTP(rr, req)
	context.TestOK(t, rr)

	require.NotNil(t, ctx.GetUpload(), "missing upload")
	require.NotEqual(t, "", ctx.GetUpload().ID, "upload should be created")

	upload, err := ctx.GetMetadataBackend().GetUpload(ctx.GetUpload().ID)
	require.NoError(t, err, "metadata backend error")

	require.Equal(t, ctx.GetConfig().DefaultTTL, upload.TTL, "invalid ttl")
	require.Equal(t, ctx.GetSourceIP().String(), upload.RemoteIP, "invalid source ip")
	require.Equal(t, ctx.GetUser().ID, upload.User, "invalid source ip")
	require.Equal(t, ctx.GetToken().Token, upload.Token, "invalid source ip")

	require.True(t, ctx.GetUpload().IsAdmin, "should be upload admin")
	require.True(t, ctx.IsQuick(), "should be quick")
}

func TestCreateUploadNotWhitelisted(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	ctx.GetConfig().FeatureAuthentication = common.FeatureEnabled
	ctx.SetWhitelisted(false)

	ctx.SetSourceIP(net.ParseIP("1.2.3.4"))
	ctx.SetUser(&common.User{ID: "user"})
	ctx.SetToken(&common.Token{Token: "token"})

	req, err := http.NewRequest("GET", "", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	CreateUpload(ctx, common.DummyHandler).ServeHTTP(rr, req)
	context.TestForbidden(t, rr, "untrusted source IP address")
}

func TestCreateUploadInvalidContext(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	ctx.GetConfig().FeatureAuthentication = common.FeatureForced

	ctx.SetSourceIP(net.ParseIP("1.2.3.4"))
	req, err := http.NewRequest("GET", "", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	CreateUpload(ctx, common.DummyHandler).ServeHTTP(rr, req)
	context.TestBadRequest(t, rr, "anonymous uploads are disabled")
}

func TestCreateUploadMetadataBackendError(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	err := ctx.GetMetadataBackend().Shutdown()
	require.NoError(t, err, "unable to shutdown metadata backend")

	ctx.SetSourceIP(net.ParseIP("1.2.3.4"))
	req, err := http.NewRequest("GET", "", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	CreateUpload(ctx, common.DummyHandler).ServeHTTP(rr, req)
	context.TestInternalServerError(t, rr, "database is closed")
}

func testCreateUploadMaxUserSize(t *testing.T, ok bool) {
	ctx := newTestingContext(common.NewConfiguration())

	user := common.NewUser("local", "test")
	if ok {
		user.MaxUserSize = 100000
	} else {
		user.MaxUserSize = 1000
	}

	err := ctx.GetMetadataBackend().CreateUser(user)
	require.NoError(t, err)

	ctx.SetUser(user)

	upload := common.NewUpload()
	upload.User = user.ID
	f := upload.NewFile()
	f.Size = 1024
	f.Status = common.FileUploaded
	err = ctx.GetMetadataBackend().CreateUpload(upload)
	require.NoError(t, err)

	ctx.SetSourceIP(net.ParseIP("1.2.3.4"))
	req, err := http.NewRequest("GET", "", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	CreateUpload(ctx, common.DummyHandler).ServeHTTP(rr, req)

	if ok {
		context.TestOK(t, rr)
	} else {
		context.TestBadRequest(t, rr, "maximum user upload size reached")
	}
}

func TestCreateUploadMaxUserSizeOK(t *testing.T) {
	testCreateUploadMaxUserSize(t, true)
}

func TestCreateUploadMaxUserSizeKO(t *testing.T) {
	testCreateUploadMaxUserSize(t, false)
}
