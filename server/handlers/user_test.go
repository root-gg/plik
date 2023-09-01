package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"testing"

	"github.com/stretchr/testify/require"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
	"github.com/root-gg/utils"
)

var userOK = &struct {
	ID       string `json:"id,omitempty"`
	Provider string `json:"provider"`
	Login    string `json:"login,omitempty"`
	Password string `json:"password"` // Needed here because it's json private
	Name     string `json:"name,omitempty"`
	Email    string `json:"email,omitempty"`
	IsAdmin  bool   `json:"admin"`

	MaxFileSize int64 `json:"maxFileSize"`
	MaxTTL      int   `json:"maxTTL"`
}{
	ID:          "nope",
	Provider:    "local",
	Login:       "user",
	Password:    "password",
	Email:       "user@root.gg",
	Name:        "user",
	MaxFileSize: 1234,
	MaxTTL:      1234,
	IsAdmin:     true,
}

func TestCreateUser(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	ctx.SetUser(&common.User{IsAdmin: true})

	user := *userOK
	userJSON, err := utils.ToJsonString(user)
	require.NoError(t, err)

	req, err := http.NewRequest("GET", "/", bytes.NewBufferString(userJSON))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	CreateUser(ctx, rr, req)

	// Check the status code is what we expect.
	context.TestOK(t, rr)

	respBody, err := io.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")

	var userResult *common.User
	err = json.Unmarshal(respBody, &userResult)
	require.NoError(t, err, "unable to unmarshal response body")
	require.NotNil(t, userResult)
	require.Equal(t, "local:user", userResult.ID, "invalid user id")
	require.Equal(t, user.Provider, userResult.Provider, "invalid user provider")
	require.Equal(t, user.Name, userResult.Name, "invalid user name")
	require.Equal(t, user.Email, userResult.Email, "invalid user email")
	require.Equal(t, user.Login, userResult.Login, "invalid user login")
	require.Empty(t, userResult.Password, "user password returned")
	require.Equal(t, user.MaxTTL, userResult.MaxTTL, "invalid user login")
	require.Equal(t, user.MaxFileSize, userResult.MaxFileSize, "invalid user login")

	userResult, err = ctx.GetMetadataBackend().GetUser("local:user")
	require.NoError(t, err)
	require.NotNil(t, userResult)
	require.Equal(t, "local:user", userResult.ID, "invalid user id")
	require.Equal(t, user.Provider, userResult.Provider, "invalid user provider")
	require.Equal(t, user.Name, userResult.Name, "invalid user name")
	require.Equal(t, user.Email, userResult.Email, "invalid user email")
	require.Equal(t, user.Login, userResult.Login, "invalid user login")
	require.True(t, common.CheckPasswordHash(user.Password, userResult.Password), "invalid user password")
	require.Equal(t, user.MaxTTL, userResult.MaxTTL, "invalid user login")
	require.Equal(t, user.MaxFileSize, userResult.MaxFileSize, "invalid user login")
}

func TestCreateUser_Unauthorized(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	req, err := http.NewRequest("GET", "/", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	CreateUser(ctx, rr, req)

	context.TestForbidden(t, rr, "you need administrator privileges")
}

func TestCreateUser_InvaliduserJSON(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	ctx.SetUser(&common.User{IsAdmin: true})

	req, err := http.NewRequest("GET", "/", bytes.NewBufferString(""))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	CreateUser(ctx, rr, req)
	context.TestBadRequest(t, rr, "unable to deserialize user : missing")

	req, err = http.NewRequest("GET", "/", bytes.NewBufferString("invalid"))
	require.NoError(t, err, "unable to create new request")

	rr = ctx.NewRecorder(req)
	CreateUser(ctx, rr, req)
	context.TestBadRequest(t, rr, "unable to deserialize user")

	req, err = http.NewRequest("GET", "/", bytes.NewBufferString("{\"password\": 1}"))
	require.NoError(t, err, "unable to create new request")

	rr = ctx.NewRecorder(req)
	CreateUser(ctx, rr, req)
	context.TestBadRequest(t, rr, "unable to deserialize password")
}

func TestCreateUser_InvalidUserParams(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	ctx.SetUser(&common.User{IsAdmin: true})

	req, err := http.NewRequest("GET", "/", bytes.NewBufferString("{}"))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	CreateUser(ctx, rr, req)
	context.TestBadRequest(t, rr, "unable to create user")
}

func TestCreateUser_DuplicateUser(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	ctx.SetUser(&common.User{IsAdmin: true})

	err := ctx.GetMetadataBackend().CreateUser(&common.User{ID: "local:user"})
	require.NoError(t, err)

	userJSON, err := utils.ToJsonString(userOK)
	require.NoError(t, err)

	req, err := http.NewRequest("GET", "/", bytes.NewBufferString(userJSON))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	CreateUser(ctx, rr, req)
	context.TestInternalServerError(t, rr, "unable to save user")
}

func TestUpdateUser(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	originalUser := &common.User{
		ID:          "local:user",
		Provider:    "local",
		Login:       "user",
		Password:    "password",
		Email:       "user@root.gg",
		Name:        "user",
		MaxFileSize: 1234,
		MaxTTL:      1234,
		IsAdmin:     true,
	}
	ctx.SetUser(originalUser)

	err := ctx.GetMetadataBackend().CreateUser(originalUser)
	require.NoError(t, err)

	var updateParams = &struct {
		ID       string `json:"id,omitempty"`
		Provider string `json:"provider"`
		Login    string `json:"login,omitempty"`
		Password string `json:"password"` // Needed here because
		Name     string `json:"name,omitempty"`
		Email    string `json:"email,omitempty"`
		IsAdmin  bool   `json:"admin"`

		MaxFileSize int64 `json:"maxFileSize"`
		MaxTTL      int   `json:"maxTTL"`
	}{
		ID:          "local:user",
		Provider:    "nope",
		Login:       "nope",
		Password:    "newpassword",
		Email:       "new@email",
		Name:        "newname",
		MaxFileSize: -1,
		MaxTTL:      -1,
		IsAdmin:     false,
	}

	userJSON, err := utils.ToJsonString(updateParams)
	require.NoError(t, err)

	req, err := http.NewRequest("GET", "/", bytes.NewBufferString(userJSON))
	require.NoError(t, err, "unable to update new request")

	rr := ctx.NewRecorder(req)
	UpdateUser(ctx, rr, req)

	// Check the status code is what we expect.
	context.TestOK(t, rr)

	respBody, err := io.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")

	var userResult *common.User
	err = json.Unmarshal(respBody, &userResult)
	require.NoError(t, err, "unable to unmarshal response body")
	require.NotNil(t, userResult)
	require.Equal(t, "local:user", userResult.ID, "invalid user id")
	require.Equal(t, originalUser.Provider, userResult.Provider, "invalid user provider")
	require.Equal(t, updateParams.Name, userResult.Name, "invalid user name")
	require.Equal(t, updateParams.Email, userResult.Email, "invalid user email")
	require.Equal(t, originalUser.Login, userResult.Login, "invalid user login")
	require.Empty(t, userResult.Password, "user password returned")
	require.Equal(t, updateParams.MaxTTL, userResult.MaxTTL, "invalid user MaxTTL")
	require.Equal(t, updateParams.MaxFileSize, userResult.MaxFileSize, "invalid user MaxFileSize")
	require.Equal(t, updateParams.IsAdmin, userResult.IsAdmin, "invalid user IsAdmin")

	userResult, err = ctx.GetMetadataBackend().GetUser("local:user")
	require.NoError(t, err)
	require.NotNil(t, userResult)
	require.Equal(t, "local:user", userResult.ID, "invalid user id")
	require.Equal(t, originalUser.Provider, userResult.Provider, "invalid user provider")
	require.Equal(t, updateParams.Name, userResult.Name, "invalid user name")
	require.Equal(t, updateParams.Email, userResult.Email, "invalid user email")
	require.Equal(t, originalUser.Login, userResult.Login, "invalid user login")
	require.True(t, common.CheckPasswordHash(updateParams.Password, userResult.Password), "invalid user password")
	require.Equal(t, updateParams.MaxTTL, userResult.MaxTTL, "invalid user MaxTTL")
	require.Equal(t, updateParams.MaxFileSize, userResult.MaxFileSize, "invalid user MaxFileSize")
	require.Equal(t, updateParams.IsAdmin, userResult.IsAdmin, "invalid user IsAdmin")
}

func TestUpdateUser_Unauthorized(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	req, err := http.NewRequest("GET", "/", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to update new request")

	rr := ctx.NewRecorder(req)
	UpdateUser(ctx, rr, req)

	context.TestUnauthorized(t, rr, "you need to be authenticated, please login first")
}

func TestUpdateUser_InvaliduserJSON(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	ctx.SetUser(&common.User{IsAdmin: true})

	req, err := http.NewRequest("GET", "/", bytes.NewBufferString(""))
	require.NoError(t, err, "unable to update new request")

	rr := ctx.NewRecorder(req)
	UpdateUser(ctx, rr, req)
	context.TestBadRequest(t, rr, "unable to deserialize user : missing")

	req, err = http.NewRequest("GET", "/", bytes.NewBufferString("invalid"))
	require.NoError(t, err, "unable to update new request")

	rr = ctx.NewRecorder(req)
	UpdateUser(ctx, rr, req)
	context.TestBadRequest(t, rr, "unable to deserialize user")

	req, err = http.NewRequest("GET", "/", bytes.NewBufferString("{\"password\": 1}"))
	require.NoError(t, err, "unable to update new request")

	rr = ctx.NewRecorder(req)
	UpdateUser(ctx, rr, req)
	context.TestBadRequest(t, rr, "unable to deserialize password")
}

func TestUpdateUser_InvalidUserParams(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	ctx.SetUser(&common.User{Provider: common.ProviderLocal, IsAdmin: true})

	req, err := http.NewRequest("GET", "/", bytes.NewBufferString("{\"password\":\"short\"}"))
	require.NoError(t, err, "unable to update new request")

	rr := ctx.NewRecorder(req)
	UpdateUser(ctx, rr, req)
	context.TestBadRequest(t, rr, "unable to update user")
}

func TestUpdateUser_InvalidUserID(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	ctx.SetUser(&common.User{ID: "local:user"})

	req, err := http.NewRequest("GET", "/", bytes.NewBufferString("{\"id\":\"invalid\"}"))
	require.NoError(t, err, "unable to update new request")

	rr := ctx.NewRecorder(req)
	UpdateUser(ctx, rr, req)
	context.TestBadRequest(t, rr, "user id mismatch")
}

func TestUpdateUser_FailGrant(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	originalUser := &common.User{
		ID:          "local:user",
		Provider:    "local",
		Login:       "user",
		Password:    "password",
		Email:       "user@root.gg",
		Name:        "user",
		MaxFileSize: 1234,
		MaxTTL:      1234,
		IsAdmin:     false,
	}
	ctx.SetUser(originalUser)

	err := ctx.GetMetadataBackend().CreateUser(originalUser)
	require.NoError(t, err)

	var updateParams = &struct {
		ID       string `json:"id,omitempty"`
		Provider string `json:"provider"`
		Login    string `json:"login,omitempty"`
		Password string `json:"password"` // Needed here because
		Name     string `json:"name,omitempty"`
		Email    string `json:"email,omitempty"`
		IsAdmin  bool   `json:"admin"`

		MaxFileSize int64 `json:"maxFileSize"`
		MaxTTL      int   `json:"maxTTL"`
	}{
		ID:          "local:user",
		Provider:    "nope",
		Login:       "nope",
		Password:    "newpassword",
		Email:       "new@email",
		Name:        "newname",
		MaxFileSize: -1,
		MaxTTL:      -1,
		IsAdmin:     true,
	}

	userJSON, err := utils.ToJsonString(updateParams)
	require.NoError(t, err)

	req, err := http.NewRequest("GET", "/", bytes.NewBufferString(userJSON))
	require.NoError(t, err, "unable to update new request")

	rr := ctx.NewRecorder(req)
	UpdateUser(ctx, rr, req)
	context.TestForbidden(t, rr, "can't grant yourself admin right")

	updateParams.IsAdmin = false
	userJSON, err = utils.ToJsonString(updateParams)
	require.NoError(t, err)

	req, err = http.NewRequest("GET", "/", bytes.NewBufferString(userJSON))
	require.NoError(t, err, "unable to update new request")

	rr = ctx.NewRecorder(req)
	UpdateUser(ctx, rr, req)

	context.TestForbidden(t, rr, "can't edit your own quota")

	updatedUser, err := ctx.GetMetadataBackend().GetUser(originalUser.ID)
	require.NoError(t, err)
	require.False(t, updatedUser.IsAdmin)
}

func TestUpdateUser_OKGrant(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	originalUser := &common.User{
		ID:          "local:admin",
		Provider:    "local",
		Login:       "admin",
		Password:    "password",
		Email:       "admin@root.gg",
		Name:        "admin",
		MaxFileSize: 1234,
		MaxTTL:      1234,
		IsAdmin:     true,
	}
	ctx.SetUser(originalUser)
	ctx.SaveOriginalUser()

	userToUpdate := &common.User{
		ID:          "local:user",
		Provider:    "local",
		Login:       "user",
		Password:    "password",
		Email:       "user@root.gg",
		Name:        "user",
		MaxFileSize: 1234,
		MaxTTL:      1234,
		IsAdmin:     false,
	}
	ctx.SetUser(userToUpdate)

	err := ctx.GetMetadataBackend().CreateUser(userToUpdate)
	require.NoError(t, err)

	var updateParams = &struct {
		ID       string `json:"id,omitempty"`
		Provider string `json:"provider"`
		Login    string `json:"login,omitempty"`
		Password string `json:"password"` // Needed here because
		Name     string `json:"name,omitempty"`
		Email    string `json:"email,omitempty"`
		IsAdmin  bool   `json:"admin"`

		MaxFileSize int64 `json:"maxFileSize"`
		MaxTTL      int   `json:"maxTTL"`
	}{
		ID:          "local:user",
		Provider:    "nope",
		Login:       "nope",
		Password:    "newpassword",
		Email:       "new@email",
		Name:        "newname",
		MaxFileSize: -1,
		MaxTTL:      -1,
		IsAdmin:     true,
	}

	userJSON, err := utils.ToJsonString(updateParams)
	require.NoError(t, err)

	req, err := http.NewRequest("GET", "/", bytes.NewBufferString(userJSON))
	require.NoError(t, err, "unable to update new request")

	rr := ctx.NewRecorder(req)
	UpdateUser(ctx, rr, req)
	context.TestOK(t, rr)

	updatedUser, err := ctx.GetMetadataBackend().GetUser(userToUpdate.ID)
	require.NoError(t, err)
	require.True(t, updatedUser.IsAdmin)
}
