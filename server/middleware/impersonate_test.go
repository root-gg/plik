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

func TestImpersonateNotAdmin(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	req, err := http.NewRequest("GET", "", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	req.Header.Set("X-Plik-Impersonate", "user")

	rr := httptest.NewRecorder()
	Impersonate(ctx, common.DummyHandler).ServeHTTP(rr, req)

	context.TestFail(t, rr, http.StatusForbidden, "You need administrator privileges")
}

func TestImpersonateMetadataBackendError(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())
	context.GetMetadataBackend(ctx).(*metadata_test.MetadataBackend).SetError(errors.New("metadata backend error"))

	user := common.NewUser()
	ctx.Set("user", user)
	ctx.Set("is_admin", true)

	req, err := http.NewRequest("GET", "", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	req.Header.Set("X-Plik-Impersonate", "user")

	rr := httptest.NewRecorder()
	Impersonate(ctx, common.DummyHandler).ServeHTTP(rr, req)

	context.TestFail(t, rr, http.StatusInternalServerError, "Unable to get user to impersonate")
}

func TestImpersonateUserNotFound(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	user := common.NewUser()
	ctx.Set("user", user)
	ctx.Set("is_admin", true)

	req, err := http.NewRequest("GET", "", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	req.Header.Set("X-Plik-Impersonate", "user")

	rr := httptest.NewRecorder()
	Impersonate(ctx, common.DummyHandler).ServeHTTP(rr, req)

	context.TestFail(t, rr, http.StatusForbidden, "Unable to get user to impersonate : User does not exists")
}

func TestImpersonate(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	user := common.NewUser()
	ctx.Set("user", user)
	ctx.Set("is_admin", true)

	userToImpersonate := common.NewUser()
	userToImpersonate.ID = "user"
	err := context.GetMetadataBackend(ctx).SaveUser(ctx, userToImpersonate)
	require.NoError(t, err, "unable to save user to impersonate")

	req, err := http.NewRequest("GET", "", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	req.Header.Set("X-Plik-Impersonate", "user")

	rr := httptest.NewRecorder()
	Impersonate(ctx, common.DummyHandler).ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code, "invalid handler response status code")

	userFromContext, ok := ctx.Get("user")
	require.True(t, ok, "missing user from context")
	require.Equal(t, userToImpersonate, userFromContext, "invalid user from context")
}
