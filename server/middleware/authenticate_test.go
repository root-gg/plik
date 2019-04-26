/**

    Plik upload server

The MIT License (MIT)

Copyright (c) <2015>
	- Mathieu Bodjikian <mathieu@bodjikian.fr>
	- Charles-Antoine Mathieu <skatkatt@root.gg>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
**/
package middleware

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
	metadata_test "github.com/root-gg/plik/server/metadata/testing"
	"github.com/stretchr/testify/require"
)

func TestAuthenticateTokenNoUser(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())
	context.GetConfig(ctx).Authentication = true

	req, err := http.NewRequest("GET", "", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	req.Header.Set("X-PlikToken", "token")

	rr := httptest.NewRecorder()
	Authenticate(true)(ctx, common.DummyHandler).ServeHTTP(rr, req)

	context.TestFail(t, rr, http.StatusForbidden, "Invalid token")
}

func TestAuthenticateTokenMetadataBackendError(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())
	context.GetConfig(ctx).Authentication = true
	context.GetMetadataBackend(ctx).(*metadata_test.MetadataBackend).SetError(errors.New("metadata backend error"))

	req, err := http.NewRequest("GET", "", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	req.Header.Set("X-PlikToken", "token")

	rr := httptest.NewRecorder()
	Authenticate(true)(ctx, common.DummyHandler).ServeHTTP(rr, req)

	context.TestFail(t, rr, http.StatusInternalServerError, "Unable to get user")
}

func TestAuthenticateToken(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())
	context.GetConfig(ctx).Authentication = true

	user := common.NewUser()
	token := user.NewToken()

	err := context.GetMetadataBackend(ctx).SaveUser(ctx, user)
	require.NoError(t, err, "unable to save user to impersonate")

	req, err := http.NewRequest("GET", "", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	req.Header.Set("X-PlikToken", token.Token)

	rr := httptest.NewRecorder()
	Authenticate(true)(ctx, common.DummyHandler).ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code, "invalid handler response status code")

	userFromContext := context.GetUser(ctx)
	tokenFromContext := context.GetToken(ctx)
	require.Equal(t, user, userFromContext, "missing user from context")
	require.Equal(t, token, tokenFromContext, "invalid token from context")
}

func TestAuthenticateInvalidSessionCookie(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())
	context.GetConfig(ctx).Authentication = true

	req, err := http.NewRequest("GET", "", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	cookie := &http.Cookie{}
	cookie.Name = "plik-session"
	cookie.Value = "invalid_value"
	req.AddCookie(cookie)

	rr := httptest.NewRecorder()
	Authenticate(false)(ctx, common.DummyHandler).ServeHTTP(rr, req)

	context.TestFail(t, rr, http.StatusForbidden, "Invalid session")
}

func TestAuthenticateMissingXSRFHeader(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())
	context.GetConfig(ctx).Authentication = true
	context.GetConfig(ctx).OvhAuthentication = true

	key := "secret_key"
	context.GetConfig(ctx).OvhAPISecret = key

	user := common.NewUser()
	user.ID = "ovh:user"

	req, err := http.NewRequest("POST", "", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	// Generate session cookie
	sessionCookie, _, err := common.GenAuthCookies(user, context.GetConfig(ctx))
	require.NoError(t, err, "unable to create new request")
	req.AddCookie(sessionCookie)

	rr := httptest.NewRecorder()
	Authenticate(false)(ctx, common.DummyHandler).ServeHTTP(rr, req)

	context.TestFail(t, rr, http.StatusForbidden, "Missing xsrf header")
}

func TestAuthenticateInvalidXSRFHeader(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())
	context.GetConfig(ctx).Authentication = true
	context.GetConfig(ctx).OvhAuthentication = true

	key := "secret_key"
	context.GetConfig(ctx).OvhAPISecret = key

	user := common.NewUser()
	user.ID = "ovh:user"

	req, err := http.NewRequest("POST", "", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	// Generate session cookie
	sessionCookie, _, err := common.GenAuthCookies(user, context.GetConfig(ctx))
	require.NoError(t, err, "unable to create new request")
	req.AddCookie(sessionCookie)

	req.Header.Set("X-XSRFToken", "invalid_header_value")

	rr := httptest.NewRecorder()
	Authenticate(false)(ctx, common.DummyHandler).ServeHTTP(rr, req)

	context.TestFail(t, rr, http.StatusForbidden, "Invalid xsrf header")
}

func TestAuthenticateMetadataBackendError(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())
	context.GetConfig(ctx).OvhAuthentication = true
	context.GetConfig(ctx).Authentication = true

	key := "secret_key"
	context.GetConfig(ctx).OvhAPISecret = key

	user := common.NewUser()
	user.ID = "ovh:user"
	err := context.GetMetadataBackend(ctx).SaveUser(ctx, user)
	require.NoError(t, err, "unable to save user")

	req, err := http.NewRequest("GET", "", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	// Generate session cookie
	sessionCookie, _, err := common.GenAuthCookies(user, context.GetConfig(ctx))
	require.NoError(t, err, "unable to create new request")
	req.AddCookie(sessionCookie)

	context.GetMetadataBackend(ctx).(*metadata_test.MetadataBackend).SetError(errors.New("metadata backend error"))

	rr := httptest.NewRecorder()
	Authenticate(false)(ctx, common.DummyHandler).ServeHTTP(rr, req)
	context.TestFail(t, rr, http.StatusInternalServerError, "Unable to get user")
}

func TestAuthenticateNoUser(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())
	context.GetConfig(ctx).OvhAuthentication = true
	context.GetConfig(ctx).Authentication = true

	key := "secret_key"
	context.GetConfig(ctx).OvhAPISecret = key

	user := common.NewUser()
	user.ID = "ovh:user"

	req, err := http.NewRequest("GET", "", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	// Generate session cookie
	sessionCookie, _, err := common.GenAuthCookies(user, context.GetConfig(ctx))
	require.NoError(t, err, "unable to create new request")
	req.AddCookie(sessionCookie)

	rr := httptest.NewRecorder()
	Authenticate(false)(ctx, common.DummyHandler).ServeHTTP(rr, req)
	context.TestFail(t, rr, http.StatusForbidden, "Invalid session : User does not exists")

}

func TestAuthenticate(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())
	context.GetConfig(ctx).OvhAuthentication = true
	context.GetConfig(ctx).Authentication = true

	key := "secret_key"
	context.GetConfig(ctx).OvhAPISecret = key

	user := common.NewUser()
	user.ID = "ovh:user"
	err := context.GetMetadataBackend(ctx).SaveUser(ctx, user)
	require.NoError(t, err, "unable to save user")

	req, err := http.NewRequest("GET", "", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	// Generate session cookie
	sessionCookie, _, err := common.GenAuthCookies(user, context.GetConfig(ctx))
	require.NoError(t, err, "unable to create new request")
	req.AddCookie(sessionCookie)

	rr := httptest.NewRecorder()
	Authenticate(false)(ctx, common.DummyHandler).ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code, "invalid handler response status code")
	require.Equal(t, user, context.GetUser(ctx), "invalid user from context")
}

func TestAuthenticateAdminUser(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())
	context.GetConfig(ctx).OvhAuthentication = true
	context.GetConfig(ctx).Authentication = true

	key := "secret_key"
	context.GetConfig(ctx).OvhAPISecret = key

	user := common.NewUser()
	user.ID = "ovh:user"
	err := context.GetMetadataBackend(ctx).SaveUser(ctx, user)
	require.NoError(t, err, "unable to save user")

	req, err := http.NewRequest("GET", "", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	// Generate session cookie
	sessionCookie, _, err := common.GenAuthCookies(user, context.GetConfig(ctx))
	require.NoError(t, err, "unable to create new request")
	req.AddCookie(sessionCookie)

	rr := httptest.NewRecorder()
	Authenticate(false)(ctx, common.DummyHandler).ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code, "invalid handler response status code")
	require.Equal(t, user, context.GetUser(ctx), "invalid user from context")
}
