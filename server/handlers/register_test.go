package handlers

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
)

func TestRegister(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	ctx.GetConfig().Authentication = true
	ctx.GetConfig().Registration = common.RegistrationOpen

	params := &RegisterParams{
		Login:    "plik",
		Password: "plik",
		Name:     "plik",
		Email:    "plik@root.gg",
		Invite:   "",
	}

	reqBody, err := json.Marshal(params)
	require.NoError(t, err, "unable to marshal request body")

	req, err := http.NewRequest("POST", "/auth/local/register", bytes.NewBuffer(reqBody))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	Register(ctx, rr, req)

	// Check the status code is what we expect.
	context.TestOK(t, rr)

	respBody, err := ioutil.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")

	user := &common.User{}
	err = json.Unmarshal(respBody, user)
	require.NoError(t, err, "unable to unmarshal response body")

	require.Equal(t, common.GetUserID(common.ProviderLocal, "plik"), user.ID, "invalid user id")
	require.Equal(t, params.Login, user.Login, "invalid user login")
	require.Equal(t, params.Name, user.Name, "invalid user name")
	require.Equal(t, params.Email, user.Email, "invalid user email")
	require.True(t, user.Verified)
	require.False(t, user.IsAdmin)
	require.Len(t, rr.Result().Cookies(), 2)
}

func TestRegisterAuthDisabled(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	ctx.GetConfig().Authentication = false

	params := &RegisterParams{
		Login:    "plik",
		Password: "plik",
		Name:     "plik",
		Email:    "plik@root.gg",
		Invite:   "",
	}

	reqBody, err := json.Marshal(params)
	require.NoError(t, err, "unable to marshal request body")

	req, err := http.NewRequest("POST", "/auth/local/register", bytes.NewBuffer(reqBody))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	Register(ctx, rr, req)

	// Check the status code is what we expect.
	context.TestBadRequest(t, rr, "authentication is disabled")
}

func TestVerify(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	ctx.GetConfig().Authentication = true
	ctx.GetConfig().EmailVerification = true
	ctx.GetConfig().Registration = common.RegistrationOpen

	user := validUser()
	user.GenVerificationCode()
	err := ctx.GetMetadataBackend().CreateUser(user)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", user.GetVerifyURL(ctx.GetConfig()), bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	// Fake gorilla/mux vars
	vars := map[string]string{
		"userID": user.Login,
		"code":   user.VerificationCode,
	}
	req = mux.SetURLVars(req, vars)

	rr := ctx.NewRecorder(req)
	Verify(ctx, rr, req)

	require.Equal(t, http.StatusMovedPermanently, rr.Result().StatusCode, rr.Body.String())
	require.Len(t, rr.Result().Cookies(), 2)

	u, err := ctx.GetMetadataBackend().GetUser(user.ID)
	require.NoError(t, err)
	require.NotNil(t, u)
	require.True(t, u.Verified)
}

func TestVerifyMissingUserID(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	ctx.GetConfig().Authentication = true
	ctx.GetConfig().EmailVerification = true
	ctx.GetConfig().Registration = common.RegistrationOpen

	user := validUser()
	user.GenVerificationCode()
	err := ctx.GetMetadataBackend().CreateUser(user)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", user.GetVerifyURL(ctx.GetConfig()), bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	// Fake gorilla/mux vars
	vars := map[string]string{
		"code": user.VerificationCode,
	}
	req = mux.SetURLVars(req, vars)

	rr := ctx.NewRecorder(req)
	Verify(ctx, rr, req)

	context.TestBadRequest(t, rr, "missing user ID")
}

func TestVerifyInvalidUserID(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	ctx.GetConfig().Authentication = true
	ctx.GetConfig().EmailVerification = true
	ctx.GetConfig().Registration = common.RegistrationOpen

	user := validUser()
	user.GenVerificationCode()
	err := ctx.GetMetadataBackend().CreateUser(user)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", user.GetVerifyURL(ctx.GetConfig()), bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	// Fake gorilla/mux vars
	vars := map[string]string{
		"userID": "foo bar",
		"code":   user.VerificationCode,
	}
	req = mux.SetURLVars(req, vars)

	rr := ctx.NewRecorder(req)
	Verify(ctx, rr, req)

	context.TestBadRequest(t, rr, "user does not exists")
}

func TestVerifyMissingConfirmationCode(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	ctx.GetConfig().Authentication = true
	ctx.GetConfig().EmailVerification = true
	ctx.GetConfig().Registration = common.RegistrationOpen

	user := validUser()
	user.GenVerificationCode()
	err := ctx.GetMetadataBackend().CreateUser(user)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", user.GetVerifyURL(ctx.GetConfig()), bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	// Fake gorilla/mux vars
	vars := map[string]string{
		"userID": user.Login,
	}
	req = mux.SetURLVars(req, vars)

	rr := ctx.NewRecorder(req)
	Verify(ctx, rr, req)

	context.TestBadRequest(t, rr, "missing verification code")
}

func TestVerifyInvalidVerificationCode(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	ctx.GetConfig().Authentication = true
	ctx.GetConfig().EmailVerification = true
	ctx.GetConfig().Registration = common.RegistrationOpen

	user := validUser()
	user.GenVerificationCode()
	err := ctx.GetMetadataBackend().CreateUser(user)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", user.GetVerifyURL(ctx.GetConfig()), bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	// Fake gorilla/mux vars
	vars := map[string]string{
		"userID": user.Login,
		"code":   "123",
	}
	req = mux.SetURLVars(req, vars)

	rr := ctx.NewRecorder(req)
	Verify(ctx, rr, req)

	context.TestUnauthorized(t, rr, "invalid verification code")
}
