package middleware

import (
	"bytes"
	"github.com/gorilla/mux"
	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
)

func TestUser_NoUser(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	ctx.SetUser(&common.User{})

	req, err := http.NewRequest("GET", "", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	User(ctx, common.DummyHandler).ServeHTTP(rr, req)
	context.TestBadRequest(t, rr, "missing user")
}

func TestUser_NotAuthenticated(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	req, err := http.NewRequest("GET", "", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	User(ctx, common.DummyHandler).ServeHTTP(rr, req)
	context.TestUnauthorized(t, rr, "please login first")
}

func TestUser_Admin(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	ctx.SetUser(common.NewUser(common.ProviderLocal, "admin"))
	ctx.GetUser().IsAdmin = true

	user := common.NewUser(common.ProviderLocal, "user")
	err := ctx.GetMetadataBackend().CreateUser(user)
	require.NoError(t, err)

	req, err := http.NewRequest("GET", "", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	// Fake gorilla/mux vars
	vars := map[string]string{
		"userID": user.ID,
	}
	req = mux.SetURLVars(req, vars)

	rr := ctx.NewRecorder(req)
	User(ctx, common.DummyHandler).ServeHTTP(rr, req)
	context.TestOK(t, rr)

	require.Equal(t, user.ID, ctx.GetUser().ID)
}

func TestUser_NotFound(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	ctx.SetUser(common.NewUser(common.ProviderLocal, "admin"))
	ctx.GetUser().IsAdmin = true

	req, err := http.NewRequest("GET", "", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	// Fake gorilla/mux vars
	vars := map[string]string{
		"userID": "local:user",
	}
	req = mux.SetURLVars(req, vars)

	rr := ctx.NewRecorder(req)
	User(ctx, common.DummyHandler).ServeHTTP(rr, req)
	context.TestNotFound(t, rr, "user not found")
}

func TestUser_Self(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	user := common.NewUser(common.ProviderLocal, "user")
	ctx.SetUser(user)

	req, err := http.NewRequest("GET", "", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	// Fake gorilla/mux vars
	vars := map[string]string{
		"userID": user.ID,
	}
	req = mux.SetURLVars(req, vars)

	rr := ctx.NewRecorder(req)
	User(ctx, common.DummyHandler).ServeHTTP(rr, req)
	context.TestOK(t, rr)

	require.Equal(t, user.ID, ctx.GetUser().ID)
}

func TestUser_KO(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	ctx.SetUser(common.NewUser(common.ProviderLocal, "local:bad"))

	req, err := http.NewRequest("GET", "", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	// Fake gorilla/mux vars
	vars := map[string]string{
		"userID": "local:user",
	}
	req = mux.SetURLVars(req, vars)

	rr := ctx.NewRecorder(req)
	User(ctx, common.DummyHandler).ServeHTTP(rr, req)
	context.TestForbidden(t, rr, "you need administrator privileges")
}
