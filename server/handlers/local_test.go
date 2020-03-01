package handlers

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/root-gg/utils"
	"github.com/stretchr/testify/require"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
)

func TestLocalLogin(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	ctx.GetConfig().Authentication = true

	user := common.NewUser(common.ProviderLocal, "user")
	user.Name = "user"
	user.Login = "user"
	user.Password, _ = common.HashPassword("password")
	err := ctx.GetMetadataBackend().CreateUser(user)
	require.NoError(t, err, "create user error")

	credentials, _ := utils.ToJson(struct{ Login, Password string }{"user", "password"})
	req, err := http.NewRequest("GET", "/auth/local/login", bytes.NewBuffer(credentials))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	LocalLogin(ctx, rr, req)

	context.TestOK(t, rr)

	var sessionCookie string
	var xsrfCookie string
	a := rr.Result().Cookies()
	require.NotEmpty(t, a)
	for _, cookie := range rr.Result().Cookies() {
		if cookie.Name == "plik-session" {
			sessionCookie = cookie.Value
		}
		if cookie.Name == "plik-xsrf" {
			xsrfCookie = cookie.Value
		}
	}

	require.NotEqual(t, "", sessionCookie, "missing plik session cookie")
	require.NotEqual(t, "", xsrfCookie, "missing plik xsrf cookie")
}

func TestLocalLoginAuthDisabled(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	ctx.GetConfig().Authentication = false

	req, err := http.NewRequest("GET", "/auth/local/login", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	LocalLogin(ctx, rr, req)

	context.TestBadRequest(t, rr, "authentication is disabled")
}

func TestLocalLoginInvalidJSON(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	ctx.GetConfig().Authentication = true

	req, err := http.NewRequest("GET", "/auth/local/login", bytes.NewBuffer([]byte("blah")))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	LocalLogin(ctx, rr, req)

	context.TestBadRequest(t, rr, "unable to deserialize request body")
}

func TestLocalLoginMissingLogin(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	ctx.GetConfig().Authentication = true

	credentials, _ := utils.ToJson(struct{ Password string }{"password"})
	req, err := http.NewRequest("GET", "/auth/local/login", bytes.NewBuffer(credentials))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	LocalLogin(ctx, rr, req)

	context.TestBadRequest(t, rr, "missing login")
}

func TestLocalLoginMissingPassword(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	ctx.GetConfig().Authentication = true

	credentials, _ := utils.ToJson(struct{ Login string }{"user"})
	req, err := http.NewRequest("GET", "/auth/local/login", bytes.NewBuffer(credentials))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	LocalLogin(ctx, rr, req)

	context.TestBadRequest(t, rr, "missing password")
}

func TestLocalLoginMissingUser(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	ctx.GetConfig().Authentication = true

	credentials, _ := utils.ToJson(struct{ Login, Password string }{"user", "invalid"})
	req, err := http.NewRequest("GET", "/auth/local/login", bytes.NewBuffer(credentials))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	LocalLogin(ctx, rr, req)

	context.TestForbidden(t, rr, "invalid credentials")
}

func TestLocalLoginInvalidPassword(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	ctx.GetConfig().Authentication = true

	user := common.NewUser(common.ProviderLocal, "user")
	user.Name = "user"
	user.Login = "user"
	user.Password, _ = common.HashPassword("password")
	err := ctx.GetMetadataBackend().CreateUser(user)
	require.NoError(t, err, "create user error")

	credentials, _ := utils.ToJson(struct{ Login, Password string }{"user", "invalid"})
	req, err := http.NewRequest("GET", "/auth/local/login", bytes.NewBuffer(credentials))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	LocalLogin(ctx, rr, req)

	context.TestForbidden(t, rr, "invalid credentials")
}
