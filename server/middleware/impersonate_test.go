package middleware

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
)

func TestImpersonateNotAdmin(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	req, err := http.NewRequest("GET", "", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	req.Header.Set("X-Plik-Impersonate", "user")

	rr := ctx.NewRecorder(req)
	Impersonate(ctx, common.DummyHandler).ServeHTTP(rr, req)

	context.TestForbidden(t, rr, "you need administrator privileges")
}

func TestImpersonateUserNotFound(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	user := &common.User{}
	user.IsAdmin = true
	ctx.SetUser(user)

	req, err := http.NewRequest("GET", "", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	req.Header.Set("X-Plik-Impersonate", "user")

	rr := ctx.NewRecorder(req)
	Impersonate(ctx, common.DummyHandler).ServeHTTP(rr, req)

	context.TestForbidden(t, rr, "user to impersonate does not exists")
}

func TestImpersonate(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	user := &common.User{}
	user.IsAdmin = true
	ctx.SetUser(user)

	userToImpersonate := &common.User{}
	userToImpersonate.ID = "user"
	err := ctx.GetMetadataBackend().CreateUser(userToImpersonate)
	require.NoError(t, err, "unable to save user to impersonate")

	req, err := http.NewRequest("GET", "", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	req.Header.Set("X-Plik-Impersonate", "user")

	rr := ctx.NewRecorder(req)
	Impersonate(ctx, common.DummyHandler).ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code, "invalid handler response status code")

	userFromContext := ctx.GetUser()
	require.NotNil(t, userFromContext, "missing user from context")
	require.Equal(t, userToImpersonate.ID, userFromContext.ID, "invalid user from context")
}
