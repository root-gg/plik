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
	ctx.GetConfig().Authentication = true

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

	require.True(t, ctx.IsUploadAdmin(), "should be upload admin")
	require.True(t, ctx.IsQuick(), "should be quick")
}

func TestCreateUploadNotWhitelisted(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	ctx.GetConfig().Authentication = true
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
