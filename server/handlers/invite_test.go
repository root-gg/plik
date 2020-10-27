package handlers

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
)

func TestCreateInvite(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	user := common.NewUser(common.ProviderLocal, "user1")
	user.IsAdmin = true
	err := ctx.GetMetadataBackend().CreateUser(user)
	require.NoError(t, err, "unable to add user")
	ctx.SetUser(user)

	invite, err := common.NewInvite(user, time.Hour)
	require.NoError(t, err)
	invite.Email = "plik@root.gg"
	invite.Admin = true

	reqBody, err := json.Marshal(invite)
	require.NoError(t, err, "unable to marshal request body")

	req, err := http.NewRequest("POST", "/me/invite", bytes.NewBuffer(reqBody))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	CreateInvite(ctx, rr, req)

	// Check the status code is what we expect.
	context.TestOK(t, rr)

	respBody, err := ioutil.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")

	var inviteResult = &common.Invite{}
	err = json.Unmarshal(respBody, inviteResult)
	require.NoError(t, err, "unable to unmarshal response body")

	require.NotEqual(t, "", inviteResult.ID, "missing invite id")
	require.NotEqual(t, time.Time{}, inviteResult.CreatedAt, "missing invite creation date")
	require.Equal(t, invite.Email, inviteResult.Email, "invalid invite email")
	require.Equal(t, invite.Admin, inviteResult.Admin, "invalid invite admin")
}

func TestCreateInviteInvalid(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	user := common.NewUser(common.ProviderLocal, "user1")
	user.IsAdmin = true
	err := ctx.GetMetadataBackend().CreateUser(user)
	require.NoError(t, err, "unable to add user")
	ctx.SetUser(user)

	invite, err := common.NewInvite(user, time.Hour)
	require.NoError(t, err)
	invite.Email = "plik.root.gg"
	invite.Admin = true

	reqBody, err := json.Marshal(invite)
	require.NoError(t, err, "unable to marshal request body")

	req, err := http.NewRequest("POST", "/me/invite", bytes.NewBuffer(reqBody))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	CreateInvite(ctx, rr, req)

	// Check the status code is what we expect.
	context.TestBadRequest(t, rr, "invalid email")
}

func TestRemoveInvite(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	user := common.NewUser(common.ProviderLocal, "user1")
	user.IsAdmin = true

	invite, err := user.NewInvite(0)
	require.NoError(t, err)

	err = ctx.GetMetadataBackend().CreateUser(user)
	require.NoError(t, err, "unable to add user")

	err = ctx.GetMetadataBackend().CreateInvite(invite)
	require.NoError(t, err, "unable to add invite")

	ctx.SetUser(user)

	req, err := http.NewRequest("DELETE", "/me/invite/"+invite.ID, bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	// Fake gorilla/mux vars
	vars := map[string]string{
		"invite": invite.ID,
	}
	req = mux.SetURLVars(req, vars)

	rr := ctx.NewRecorder(req)
	RevokeInvite(ctx, rr, req)

	// Check the status code is what we expect.
	context.TestOK(t, rr)

	respBody, err := ioutil.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")
	require.Equal(t, "ok", string(respBody), "invalid response body")

	i, err := ctx.GetMetadataBackend().GetInvite(invite.ID)
	require.NoError(t, err, "unable to get invite")
	require.Nil(t, i)
}

func TestRemoveInviteNotFound(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	user := common.NewUser(common.ProviderLocal, "user1")
	user.IsAdmin = true

	invite, err := user.NewInvite(0)
	require.NoError(t, err)

	err = ctx.GetMetadataBackend().CreateUser(user)
	require.NoError(t, err, "unable to add user")

	ctx.SetUser(user)

	req, err := http.NewRequest("DELETE", "/me/invite/"+invite.ID, bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	// Fake gorilla/mux vars
	vars := map[string]string{
		"invite": invite.ID,
	}
	req = mux.SetURLVars(req, vars)

	rr := ctx.NewRecorder(req)
	RevokeInvite(ctx, rr, req)

	// Check the status code is what we expect.
	context.TestNotFound(t, rr, "invite not found")
}

func TestRemoveInviteNotOwned(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	user := common.NewUser(common.ProviderLocal, "user1")
	user.IsAdmin = true

	user2 := common.NewUser(common.ProviderLocal, "user2")
	user.IsAdmin = true

	invite, err := user2.NewInvite(0)
	require.NoError(t, err)

	err = ctx.GetMetadataBackend().CreateUser(user)
	require.NoError(t, err, "unable to add user")

	err = ctx.GetMetadataBackend().CreateUser(user2)
	require.NoError(t, err, "unable to add user")

	err = ctx.GetMetadataBackend().CreateInvite(invite)
	require.NoError(t, err, "unable to add invite")

	ctx.SetUser(user)

	req, err := http.NewRequest("DELETE", "/me/invite/"+invite.ID, bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	// Fake gorilla/mux vars
	vars := map[string]string{
		"invite": invite.ID,
	}
	req = mux.SetURLVars(req, vars)

	rr := ctx.NewRecorder(req)
	RevokeInvite(ctx, rr, req)

	// Check the status code is what we expect.
	context.TestNotFound(t, rr, "invite not found")
}

func TestRevokeInviteMissingUser(t *testing.T) {
	config := common.NewConfiguration()
	ctx := newTestingContext(config)

	req, err := http.NewRequest("DELETE", "/me/invite", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	RevokeInvite(ctx, rr, req)
	context.TestUnauthorized(t, rr, "missing user, please login first")
}

func TestGetUserInvites(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	user := common.NewUser(common.ProviderLocal, "user1")
	i1, err := user.NewInvite(0)
	i2, err := user.NewInvite(0)

	err = ctx.GetMetadataBackend().CreateUser(user)
	require.NoError(t, err, "unable to create test user")

	err = ctx.GetMetadataBackend().CreateInvite(i1)
	require.NoError(t, err, "unable to create test invite 1")

	err = ctx.GetMetadataBackend().CreateInvite(i2)
	require.NoError(t, err, "unable to create test invite 2")

	ctx.SetUser(user)

	// Create a request
	req, err := http.NewRequest("GET", "/me/uploads", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	// Create paging query
	ctx.SetPagingQuery(&common.PagingQuery{})

	rr := ctx.NewRecorder(req)
	GetUserInvites(ctx, rr, req)

	// Check the status code is what we expect.
	context.TestOK(t, rr)

	respBody, err := ioutil.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")

	var response common.PagingResponse
	err = json.Unmarshal(respBody, &response)
	require.NoError(t, err, "unable to unmarshal response body %s", respBody)
	require.Equal(t, 2, len(response.Results), "invalid invite count")
}

func TestGetUserInvitesNoUser(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	req, err := http.NewRequest("GET", "/me/uploads", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	GetUserInvites(ctx, rr, req)

	context.TestUnauthorized(t, rr, "missing user, please login first")
}
