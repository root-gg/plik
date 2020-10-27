package handlers

import (
	"bytes"
	"net/http"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"

	"github.com/stretchr/testify/require"

	"github.com/root-gg/plik/server/common"
)

func TestBounce(t *testing.T) {
	config := common.NewConfiguration()
	config.Registration = common.RegistrationOpen
	ctx := newTestingContext(config)

	invite, err := common.NewInvite(nil, 0)
	require.NoError(t, err)
	err = ctx.GetMetadataBackend().CreateInvite(invite)
	require.NoError(t, err)

	i, err := bounce(ctx, "")
	require.NoError(t, err)
	require.Nil(t, i)

	i, err = bounce(ctx, invite.ID)
	require.NoError(t, err)
	require.Nil(t, i)

	config.Registration = common.RegistrationClosed
	i, err = bounce(ctx, invite.ID)
	common.RequireHTTPError(t, err, http.StatusForbidden, "user registration is disabled")

	config.Registration = common.RegistrationInvite

	i, err = bounce(ctx, invite.ID)
	require.NoError(t, err)
	require.Equal(t, invite.ID, i.ID)

	i, err = bounce(ctx, "")
	common.RequireHTTPError(t, err, http.StatusForbidden, "an invite is required to register")

	config.Registration = common.RegistrationInvite
	i, err = bounce(ctx, "invalid invite id")
	common.RequireHTTPError(t, err, http.StatusForbidden, "invite not found")

	invite, err = common.NewInvite(nil, time.Millisecond)
	require.NoError(t, err)
	err = ctx.GetMetadataBackend().CreateInvite(invite)
	require.NoError(t, err)
	time.Sleep(100 * time.Millisecond)

	config.Registration = common.RegistrationInvite
	invite, err = bounce(ctx, invite.ID)
	common.RequireHTTPError(t, err, http.StatusForbidden, "invite has expired")
}

func validUser() (user *common.User) {
	user = common.NewUser(common.ProviderLocal, "plik")
	user.Login = "plik"
	user.Name = "plik"
	user.Email = "plik@root.gg"
	user.Password = "secret"
	return user
}

func TestRegisterOpen(t *testing.T) {
	config := common.NewConfiguration()
	config.Registration = common.RegistrationOpen
	ctx := newTestingContext(config)

	user := validUser()
	err := register(ctx, user, "")
	require.NoError(t, err)

	require.Equal(t, user, ctx.GetUser())
	u, err := ctx.GetMetadataBackend().GetUser(user.ID)
	require.NoError(t, err)
	require.Equal(t, user.ID, u.ID)

	require.False(t, u.IsAdmin)
	require.True(t, u.Verified)
	require.NotNil(t, ctx.GetUser())
	require.Equal(t, user.ID, ctx.GetUser().ID)

	err = register(ctx, user, "")
	common.RequireHTTPError(t, err, http.StatusForbidden, "user already exists")
}

func TestRegisterInvite(t *testing.T) {
	config := common.NewConfiguration()
	config.Registration = common.RegistrationInvite
	config.EmailVerification = true
	ctx := newTestingContext(config)

	user := validUser()

	err := register(ctx, user, "")
	common.RequireHTTPError(t, err, http.StatusForbidden, "invite is required")

	invite, err := common.NewInvite(nil, 0)
	require.NoError(t, err)
	err = ctx.GetMetadataBackend().CreateInvite(invite)
	require.NoError(t, err)

	err = register(ctx, user, invite.ID)
	require.NoError(t, err)

	require.Equal(t, user, ctx.GetUser())
	u, err := ctx.GetMetadataBackend().GetUser(user.ID)
	require.NoError(t, err)
	require.Equal(t, user.ID, u.ID)

	require.False(t, u.IsAdmin)
	require.False(t, u.Verified)
	require.NotNil(t, ctx.GetUser())
	require.Equal(t, user.ID, ctx.GetUser().ID)

	err = register(ctx, user, invite.ID)
	common.RequireHTTPError(t, err, http.StatusForbidden, "user already exists")

	user2 := common.NewUser(common.ProviderLocal, "user2")
	err = register(ctx, user2, invite.ID)
	common.RequireHTTPError(t, err, http.StatusForbidden, "invite not found")
}

func TestRegisterNoEmailVerification(t *testing.T) {
	config := common.NewConfiguration()
	config.Registration = common.RegistrationInvite
	config.EmailVerification = false
	ctx := newTestingContext(config)

	user := validUser()

	err := register(ctx, user, "")
	common.RequireHTTPError(t, err, http.StatusForbidden, "invite is required")

	invite, err := common.NewInvite(nil, 0)
	require.NoError(t, err)
	err = ctx.GetMetadataBackend().CreateInvite(invite)
	require.NoError(t, err)

	err = register(ctx, user, invite.ID)
	require.NoError(t, err)

	require.Equal(t, user, ctx.GetUser())
	u, err := ctx.GetMetadataBackend().GetUser(user.ID)
	require.NoError(t, err)
	require.Equal(t, user.ID, u.ID)

	require.False(t, u.IsAdmin)
	require.True(t, u.Verified)
	require.NotNil(t, ctx.GetUser())
	require.Equal(t, user.ID, ctx.GetUser().ID)
}

func TestRegisterInviteVerifiedAdmin(t *testing.T) {
	config := common.NewConfiguration()
	config.Registration = common.RegistrationInvite
	config.EmailVerification = true
	ctx := newTestingContext(config)

	user := validUser()

	invite, err := common.NewInvite(nil, 0)
	require.NoError(t, err)
	invite.Admin = true
	invite.Verified = true
	err = ctx.GetMetadataBackend().CreateInvite(invite)
	require.NoError(t, err)

	err = register(ctx, user, invite.ID)
	require.NoError(t, err)

	require.Equal(t, user, ctx.GetUser())
	u, err := ctx.GetMetadataBackend().GetUser(user.ID)
	require.NoError(t, err)
	require.Equal(t, user.ID, u.ID)

	require.True(t, u.IsAdmin)
	require.True(t, u.Verified)
	require.NotNil(t, ctx.GetUser())
	require.Equal(t, user.ID, ctx.GetUser().ID)
}

func TestRegisterClosed(t *testing.T) {
	config := common.NewConfiguration()
	config.Registration = common.RegistrationClosed
	ctx := newTestingContext(config)

	user := validUser()
	err := register(ctx, user, "")
	common.RequireHTTPError(t, err, http.StatusForbidden, "")
}

func TestRegisterInvalidEmail(t *testing.T) {
	config := common.NewConfiguration()
	config.Registration = common.RegistrationInvite
	ctx := newTestingContext(config)

	invite, err := common.NewInvite(nil, 0)
	require.NoError(t, err)
	invite.Email = "invite@root.gg"
	require.NoError(t, err)
	err = ctx.GetMetadataBackend().CreateInvite(invite)
	require.NoError(t, err)

	user := validUser()
	err = register(ctx, user, invite.ID)
	common.RequireHTTPError(t, err, http.StatusForbidden, "email does not match invite email")
}

func TestRegisterInvalidUser(t *testing.T) {
	config := common.NewConfiguration()
	config.Registration = common.RegistrationOpen
	ctx := newTestingContext(config)

	user := validUser()
	user.Login = "foo bar"
	err := register(ctx, user, "")
	common.RequireHTTPError(t, err, http.StatusBadRequest, "invalid login")
}

func TestSetCookies(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	req, err := http.NewRequest("GET", "/auth/callback", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")
	rr := ctx.NewRecorder(req)

	err = setCookies(ctx, rr)
	common.RequireHTTPError(t, err, http.StatusInternalServerError, "missing user from context")

	ctx.SetUser(common.NewUser(common.ProviderLocal, "plik"))
	err = setCookies(ctx, rr)
	require.NoError(t, err)

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

func TestGetClaim(t *testing.T) {
	state := jwt.New(jwt.SigningMethodHS256)
	state.Claims.(jwt.MapClaims)["key"] = "value"
	require.Equal(t, "value", getClaim(state, "key"))
	require.Equal(t, "", getClaim(state, "foo"))

	state.Claims.(jwt.MapClaims)["key"] = 1
	require.Equal(t, "", getClaim(state, "key"))
}
