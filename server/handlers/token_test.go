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

package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
	metadata_test "github.com/root-gg/plik/server/metadata/testing"
	"github.com/stretchr/testify/require"
)

func TestCreateToken(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	user := common.NewUser()
	user.ID = "user1"

	err := addTestUser(ctx, user)
	require.NoError(t, err, "unable to add user")
	ctx.Set("user", user)

	token := common.NewToken()
	token.Comment = "token comment"

	reqBody, err := json.Marshal(token)
	require.NoError(t, err, "unable to marshal request body")

	req, err := http.NewRequest("POST", "/me/token", bytes.NewBuffer(reqBody))
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	CreateToken(ctx, rr, req)

	// Check the status code is what we expect.
	require.Equal(t, http.StatusOK, rr.Code, "handler returned wrong status code")

	respBody, err := ioutil.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")

	var tokenResult = &common.Token{}
	err = json.Unmarshal(respBody, tokenResult)
	require.NoError(t, err, "unable to unmarshal response body")

	require.NotEqual(t, "", tokenResult.Token, "missing token id")
	require.NotEqual(t, 0, tokenResult.CreationDate, "missing token creation date")
	require.Equal(t, token.Comment, tokenResult.Comment, "invalid token comment")
}

func TestCreateTokenWithForbiddenOptions(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	user := common.NewUser()
	user.ID = "user1"

	err := addTestUser(ctx, user)
	require.NoError(t, err, "unable to add user")
	ctx.Set("user", user)

	token := common.NewToken()
	token.Comment = "token comment"
	token.Token = "invalid"
	token.CreationDate = -1

	reqBody, err := json.Marshal(token)
	require.NoError(t, err, "unable to marshal request body")

	req, err := http.NewRequest("POST", "/me/token", bytes.NewBuffer(reqBody))
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	CreateToken(ctx, rr, req)

	// Check the status code is what we expect.
	require.Equal(t, http.StatusOK, rr.Code, "handler returned wrong status code")

	respBody, err := ioutil.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")

	var tokenResult = &common.Token{}
	err = json.Unmarshal(respBody, tokenResult)
	require.NoError(t, err, "unable to unmarshal response body")

	require.NotEqual(t, token.Token, tokenResult.Token, "invalid token id")
	require.NotEqual(t, token.CreationDate, tokenResult.CreationDate, "invalid token creation date")
	require.Equal(t, token.Comment, tokenResult.Comment, "invalid token comment")
}

func TestCreateTokenMissingUser(t *testing.T) {
	config := common.NewConfiguration()
	ctx := context.NewTestingContext(config)

	req, err := http.NewRequest("GET", "/me/token", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	CreateToken(ctx, rr, req)
	context.TestFail(t, rr, http.StatusUnauthorized, "Please login first")
}

func TestCreateTokenMetadataBackendError(t *testing.T) {
	config := common.NewConfiguration()
	ctx := context.NewTestingContext(config)

	user := common.NewUser()
	user.ID = "user1"

	err := addTestUser(ctx, user)
	require.NoError(t, err, "unable to add user")
	ctx.Set("user", user)

	token := common.NewToken()
	token.Comment = "token comment"

	reqBody, err := json.Marshal(token)
	require.NoError(t, err, "unable to marshal request body")

	req, err := http.NewRequest("POST", "/me/token", bytes.NewBuffer(reqBody))
	require.NoError(t, err, "unable to create new request")

	context.GetMetadataBackend(ctx).(*metadata_test.MetadataBackend).SetError(errors.New("metadata backend error"))

	rr := httptest.NewRecorder()
	CreateToken(ctx, rr, req)

	context.TestFail(t, rr, http.StatusInternalServerError, "Unable to create token")
}

func TestRemoveToken(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	user := common.NewUser()
	user.ID = "user1"

	token := common.NewToken()
	token.Comment = "token comment"
	token.Create()
	user.Tokens = append(user.Tokens, token)

	err := addTestUser(ctx, user)
	require.NoError(t, err, "unable to add user")
	ctx.Set("user", user)

	req, err := http.NewRequest("DELETE", "/me/token/"+token.Token, bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	// Fake gorilla/mux vars
	vars := map[string]string{
		"token": token.Token,
	}
	req = mux.SetURLVars(req, vars)

	rr := httptest.NewRecorder()
	RevokeToken(ctx, rr, req)

	// Check the status code is what we expect.
	require.Equal(t, http.StatusOK, rr.Code, "handler returned wrong status code")

	respBody, err := ioutil.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")
	require.Equal(t, "", string(respBody), "invalid response body")

	user, err = context.GetMetadataBackend(ctx).GetUser(ctx, user.ID, "")
	require.NoError(t, err, "unable to get user")
	require.Equal(t, 0, len(user.Tokens), "invalid user token count")
}

func TestRemoveMissingToken(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	user := common.NewUser()
	user.ID = "user1"

	token := common.NewToken()
	token.Comment = "token comment"
	token.Create()
	user.Tokens = append(user.Tokens, token)

	err := addTestUser(ctx, user)
	require.NoError(t, err, "unable to add user")
	ctx.Set("user", user)

	req, err := http.NewRequest("DELETE", "/me/token/invalid_token_id", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	// Fake gorilla/mux vars
	vars := map[string]string{
		"token": "invalid_token_id",
	}
	req = mux.SetURLVars(req, vars)

	rr := httptest.NewRecorder()
	RevokeToken(ctx, rr, req)

	context.TestFail(t, rr, http.StatusNotFound, "Invalid token")
}

func TestRevokeTokenMissingUser(t *testing.T) {
	config := common.NewConfiguration()
	ctx := context.NewTestingContext(config)

	req, err := http.NewRequest("DELETE", "/me/token", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	RevokeToken(ctx, rr, req)
	context.TestFail(t, rr, http.StatusUnauthorized, "Please login first")
}

func TestRevokeTokenMetadataBackendError(t *testing.T) {
	config := common.NewConfiguration()
	ctx := context.NewTestingContext(config)

	user := common.NewUser()
	user.ID = "user1"

	token := common.NewToken()
	token.Comment = "token comment"
	token.Create()
	user.Tokens = append(user.Tokens, token)

	err := addTestUser(ctx, user)
	require.NoError(t, err, "unable to add user")
	ctx.Set("user", user)

	req, err := http.NewRequest("DELETE", "/me/token", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	// Fake gorilla/mux vars
	vars := map[string]string{
		"token": token.Token,
	}
	req = mux.SetURLVars(req, vars)

	context.GetMetadataBackend(ctx).(*metadata_test.MetadataBackend).SetError(errors.New("metadata backend error"))

	rr := httptest.NewRecorder()
	RevokeToken(ctx, rr, req)
	context.TestFail(t, rr, http.StatusInternalServerError, "Unable to revoke token")
}
