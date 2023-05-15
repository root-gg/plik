package middleware

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
)

func getTestSessionAuthenticator() *common.SessionAuthenticator {
	return &common.SessionAuthenticator{
		SignatureKey:   "secret_key",
		SecureCookies:  true,
		SessionTimeout: 3600,
		Path:           "/",
	}
}

func TestAuthenticateTokenNoUser(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	ctx.GetConfig().FeatureAuthentication = common.FeatureEnabled

	req, err := http.NewRequest("GET", "", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	req.Header.Set("X-PlikToken", "token")

	rr := ctx.NewRecorder(req)
	Authenticate(true)(ctx, common.DummyHandler).ServeHTTP(rr, req)

	context.TestForbidden(t, rr, "invalid token")
}

func TestAuthenticateToken(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	ctx.GetConfig().FeatureAuthentication = common.FeatureEnabled

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
	ctx.GetConfig().FeatureAuthentication = common.FeatureEnabled
	ctx.SetAuthenticator(getTestSessionAuthenticator())

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
	ctx.GetConfig().FeatureAuthentication = common.FeatureEnabled
	ctx.SetAuthenticator(getTestSessionAuthenticator())

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
	ctx.GetConfig().FeatureAuthentication = common.FeatureEnabled
	ctx.SetAuthenticator(getTestSessionAuthenticator())

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
	ctx.GetConfig().FeatureAuthentication = common.FeatureEnabled
	ctx.SetAuthenticator(getTestSessionAuthenticator())

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
	ctx.GetConfig().FeatureAuthentication = common.FeatureEnabled
	ctx.SetAuthenticator(getTestSessionAuthenticator())

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
	context.TestOK(t, rr)
	require.Equal(t, user.ID, ctx.GetUser().ID, "invalid user from context")
}

func TestAuthenticateAdminUser(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	ctx.GetConfig().FeatureAuthentication = common.FeatureEnabled
	ctx.SetAuthenticator(getTestSessionAuthenticator())

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
	context.TestOK(t, rr)
	require.Equal(t, user.ID, ctx.GetUser().ID, "invalid user from context")
	require.True(t, ctx.IsAdmin(), "context is not admin")
}

func TestAuthenticatedOnly_OK(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	user := common.NewUser(common.ProviderLocal, "user")
	ctx.SetUser(user)

	req, err := http.NewRequest("GET", "", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	AuthenticatedOnly(ctx, common.DummyHandler).ServeHTTP(rr, req)
	context.TestOK(t, rr)
}

func TestAuthenticatedOnly_NoUser(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	req, err := http.NewRequest("GET", "", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	AuthenticatedOnly(ctx, common.DummyHandler).ServeHTTP(rr, req)
	context.TestUnauthorized(t, rr, "please login first")
}

func TestAdminOnly_OK(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	user := common.NewUser(common.ProviderLocal, "user")
	user.IsAdmin = true
	ctx.SetUser(user)

	req, err := http.NewRequest("GET", "", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	AdminOnly(ctx, common.DummyHandler).ServeHTTP(rr, req)
	context.TestOK(t, rr)
	require.Equal(t, user.ID, ctx.GetUser().ID, "invalid user from context")
}

func TestAdminOnly_NoUser(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	req, err := http.NewRequest("GET", "", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	AdminOnly(ctx, common.DummyHandler).ServeHTTP(rr, req)
	context.TestUnauthorized(t, rr, "please login first")
}

func TestAdminOnly_NotAdmin(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	user := common.NewUser(common.ProviderLocal, "user")
	ctx.SetUser(user)

	req, err := http.NewRequest("GET", "", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	AdminOnly(ctx, common.DummyHandler).ServeHTTP(rr, req)
	context.TestForbidden(t, rr, "you need administrator privileges")
}
