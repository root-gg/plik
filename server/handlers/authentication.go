package handlers

import (
	"net/http"

	"github.com/dgrijalva/jwt-go"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
)

// Is your name on the list ?
func bounce(ctx *context.Context, inviteID string) (invite *common.Invite, err error) {

	switch ctx.GetConfig().Registration {
	case common.RegistrationOpen:
	case common.RegistrationClosed:
		return nil, common.NewHTTPError("user registration is disabled", nil, http.StatusForbidden)
	case common.RegistrationInvite:
		if inviteID == "" {
			return nil, common.NewHTTPError("an invite is required to register", nil, http.StatusForbidden)
		}
		invite, err = ctx.GetMetadataBackend().GetInvite(inviteID)
		if err != nil {
			return nil, common.NewHTTPError("error getting invite", err, http.StatusInternalServerError)
		}
		if invite == nil {
			return nil, common.NewHTTPError("invite not found", nil, http.StatusForbidden)
		}
		if invite.HasExpired() {
			return nil, common.NewHTTPError("invite has expired", nil, http.StatusForbidden)

		}
	}

	return invite, err
}

// After calling register the user will be available in the request context
// Validation checks are performed to ensure inputs are valid regarding the server configuration
// If an invite is required to register and one is provided it will be consumed during the registration process
var errUserExists = common.NewHTTPError("user already exists", nil, http.StatusForbidden)

func register(ctx *context.Context, user *common.User, inviteID string) (err error) {

	// Check if user already exists
	// TODO Gorm does not currently provide a way to handle unique constraint violation as a specific error
	// See : https://github.com/go-gorm/gorm/pull/3512
	// So we'll do a quick check to return a nice error message but there is obviously a small race condition

	// Get user from metadata backend
	u, err := ctx.GetMetadataBackend().GetUser(user.ID)
	if err != nil {
		return common.NewHTTPError("unable to get user from metadata backend", err, http.StatusInternalServerError)
	}
	if u != nil {
		ctx.SetUser(u)
		return errUserExists
	}

	// Check invite
	invite, err := bounce(ctx, inviteID)
	if err != nil {
		return err
	}

	if invite != nil {
		if invite.Email != "" && user.Email != invite.Email {
			return common.NewHTTPError("email does not match invite email", nil, http.StatusForbidden)
		}

		if invite.Verified {
			user.Verified = true
		}

		if invite.Admin {
			user.IsAdmin = true
		}
	}

	// Check inputs, hash password,...
	err = user.PrepareInsert(ctx.GetConfig())
	if err != nil {
		return common.NewHTTPError("invalid user", err, http.StatusBadRequest)
	}

	if invite == nil && !ctx.IsWhitelisted() {
		return common.NewHTTPError("unable to create user from untrusted source IP address", nil, http.StatusForbidden)
	}

	// Save user to metadata backend
	err = ctx.GetMetadataBackend().CreateUserWithInvite(user, invite)
	if err != nil {
		return common.NewHTTPError("unable to create user", err, http.StatusInternalServerError)
	}

	ctx.SetUser(user)
	return nil
}

// Generate session cookies and set them in the HTTP response
func setCookies(ctx *context.Context, resp http.ResponseWriter) (err error) {
	if ctx.GetUser() == nil {
		return common.NewHTTPError("missing user from context", err, http.StatusInternalServerError)
	}

	// Set Plik session cookie and xsrf cookie
	sessionCookie, xsrfCookie, err := ctx.GetAuthenticator().GenAuthCookies(ctx.GetUser())
	if err != nil {
		return common.NewHTTPError("unable to generate session cookies", err, http.StatusInternalServerError)
	}

	http.SetCookie(resp, sessionCookie)
	http.SetCookie(resp, xsrfCookie)
	return nil
}

// get a string item from a JWT token, if the item is not found or not a string an empty string is returned
func getClaim(state *jwt.Token, key string) (value string) {
	if _, ok := state.Claims.(jwt.MapClaims)[key]; ok {
		if _, ok := state.Claims.(jwt.MapClaims)[key].(string); ok {
			return state.Claims.(jwt.MapClaims)[key].(string)
		}
	}

	return ""
}
