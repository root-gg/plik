package middleware

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
)

func TestAuthenticateTokenNoUser(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	ctx.GetConfig().Authentication = true

	req, err := http.NewRequest("GET", "", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	req.Header.Set("X-PlikToken", "token")

	rr := ctx.NewRecorder(req)
	Authenticate(true)(ctx, common.DummyHandler).ServeHTTP(rr, req)

	context.TestForbidden(t, rr, "invalid token")
}

func TestAuthenticateToken(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	ctx.GetConfig().Authentication = true

	user := common.NewUser(common.ProviderLocal, "user")
	token := user.NewToken()

	err := ctx.GetMetadataBackend().CreateUser(user)
	require.NoError(t, err, "unable to save user to impersonate : %s", err)

	req, err := http.NewRequest("GET", "", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	req.Header.Set("X-PlikToken", token.Token)

	rr := ctx.NewRecorder(req)
	Authenticate(true)(ctx, common.DummyHandler).ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code, "invalid handler response status code")

	userFromContext := ctx.GetUser()
	tokenFromContext := ctx.GetToken()
	require.Equal(t, user.ID, userFromContext.ID, "missing user from context")
	require.Equal(t, token.Token, tokenFromContext.Token, "invalid token from context")
}

func TestAuthenticateInvalidSessionCookie(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	ctx.GetConfig().Authentication = true
	ctx.SetAuthenticator(&common.SessionAuthenticator{SignatureKey: "secret_key"})

	req, err := http.NewRequest("GET", "", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	cookie := &http.Cookie{}
	cookie.Name = "plik-session"
	cookie.Value = "invalid_value"
	req.AddCookie(cookie)

	rr := ctx.NewRecorder(req)
	Authenticate(false)(ctx, common.DummyHandler).ServeHTTP(rr, req)

	context.TestForbidden(t, rr, "invalid session")
}

func TestAuthenticateMissingXSRFHeader(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	ctx.GetConfig().Authentication = true
	ctx.SetAuthenticator(&common.SessionAuthenticator{SignatureKey: "secret_key"})

	user := common.NewUser(common.ProviderLocal, "user")

	req, err := http.NewRequest("POST", "", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	// Generate session cookie
	sessionCookie, _, err := ctx.GetAuthenticator().GenAuthCookies(user)
	require.NoError(t, err, "unable to create new request")
	req.AddCookie(sessionCookie)

	rr := ctx.NewRecorder(req)
	Authenticate(false)(ctx, common.DummyHandler).ServeHTTP(rr, req)

	context.TestForbidden(t, rr, "missing xsrf header")
}

func TestAuthenticateInvalidXSRFHeader(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	ctx.GetConfig().Authentication = true
	ctx.SetAuthenticator(&common.SessionAuthenticator{SignatureKey: "secret_key"})

	user := common.NewUser(common.ProviderLocal, "user")

	req, err := http.NewRequest("POST", "", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	// Generate session cookie
	sessionCookie, _, err := ctx.GetAuthenticator().GenAuthCookies(user)
	require.NoError(t, err, "unable to create new request")
	req.AddCookie(sessionCookie)

	req.Header.Set("X-XSRFToken", "invalid_header_value")

	rr := ctx.NewRecorder(req)
	Authenticate(false)(ctx, common.DummyHandler).ServeHTTP(rr, req)

	context.TestForbidden(t, rr, "invalid xsrf header")
}

func TestAuthenticateNoUser(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	ctx.GetConfig().Authentication = true
	ctx.SetAuthenticator(&common.SessionAuthenticator{SignatureKey: "secret_key"})

	user := common.NewUser(common.ProviderLocal, "user")

	req, err := http.NewRequest("GET", "", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	// Generate session cookie
	sessionCookie, _, err := ctx.GetAuthenticator().GenAuthCookies(user)
	require.NoError(t, err, "unable to create new request")
	req.AddCookie(sessionCookie)

	rr := ctx.NewRecorder(req)
	Authenticate(false)(ctx, common.DummyHandler).ServeHTTP(rr, req)
	context.TestForbidden(t, rr, "invalid session : user does not exists")

}

func TestAuthenticate(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	ctx.GetConfig().Authentication = true
	ctx.SetAuthenticator(&common.SessionAuthenticator{SignatureKey: "secret_key"})

	user := common.NewUser(common.ProviderLocal, "user")

	err := ctx.GetMetadataBackend().CreateUser(user)
	require.NoError(t, err, "unable to save user")

	req, err := http.NewRequest("GET", "", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	// Generate session cookie
	sessionCookie, _, err := ctx.GetAuthenticator().GenAuthCookies(user)
	require.NoError(t, err, "unable to create new request")
	req.AddCookie(sessionCookie)

	rr := ctx.NewRecorder(req)
	Authenticate(false)(ctx, common.DummyHandler).ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code, "invalid handler response status code")
	require.Equal(t, user.ID, ctx.GetUser().ID, "invalid user from context")
}

func TestAuthenticateAdminUser(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	ctx.GetConfig().Authentication = true
	ctx.SetAuthenticator(&common.SessionAuthenticator{SignatureKey: "secret_key"})

	user := common.NewUser(common.ProviderLocal, "user")
	user.IsAdmin = true
	err := ctx.GetMetadataBackend().CreateUser(user)
	require.NoError(t, err, "unable to save user")

	req, err := http.NewRequest("GET", "", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	// Generate session cookie
	sessionCookie, _, err := ctx.GetAuthenticator().GenAuthCookies(user)
	require.NoError(t, err, "unable to create new request")
	req.AddCookie(sessionCookie)

	rr := ctx.NewRecorder(req)
	Authenticate(false)(ctx, common.DummyHandler).ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code, "invalid handler response status code")
	require.Equal(t, user.ID, ctx.GetUser().ID, "invalid user from context")
	require.True(t, ctx.IsAdmin(), "context is not admin")
}
