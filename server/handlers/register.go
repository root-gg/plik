package handlers

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
)

//
// User registration is a 3 steps process
//  - 1 : registration : Create the user ( this might require a valid invite code )
//  - 2 : confirmation : Send a confirmation code by email
//  - 3 : verification : The user input the confirmation code recieved
//

// RegisterParams to be POSTed to register a new user
type RegisterParams struct {
	Login    string `json:"login"`
	Password string `json:"password"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Invite   string `json:"invite"`
}

// Register a new user
func Register(ctx *context.Context, resp http.ResponseWriter, req *http.Request) {
	if !ctx.GetConfig().Authentication {
		ctx.BadRequest("authentication is disabled")
		return
	}

	// Read request body
	defer func() { _ = req.Body.Close() }()
	req.Body = http.MaxBytesReader(resp, req.Body, 1048576)
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		ctx.BadRequest("unable to read request body : %s", err)
		return
	}

	// Deserialize json body
	params := &RegisterParams{}
	err = json.Unmarshal(body, params)
	if err != nil {
		ctx.BadRequest("unable to deserialize request body : %s", err)
		return
	}

	// Create new user
	user := common.NewUser(common.ProviderLocal, params.Login)
	user.Login = params.Login
	user.Name = params.Name
	user.Email = params.Email
	user.Password = params.Password

	// Get or create user
	err = register(ctx, user, params.Invite)
	if err != nil {
		handleHTTPError(ctx, err)
		return
	}

	// Authenticate the HTTP response with auth cookies
	err = setCookies(ctx, resp)
	if err != nil {
		handleHTTPError(ctx, err)
		return
	}

	common.WriteJSONResponse(resp, user)
}

// Confirm send the verification link
func Confirm(ctx *context.Context, resp http.ResponseWriter, req *http.Request) {
	// Get user from context
	user := ctx.GetUser()
	if user == nil {
		ctx.Unauthorized("missing user, please login first")
		return
	}

	user.GenVerificationCode()

	err := ctx.GetMetadataBackend().UpdateUser(user)
	if err != nil {
		ctx.InternalServerError("unable to update user metadata : %s", err)
		return
	}

	verifyURL := user.GetVerifyURL(ctx.GetConfig())
	common.WriteStringResponse(resp, verifyURL)
}

// Verify that the user clicked on the user validation link
func Verify(ctx *context.Context, resp http.ResponseWriter, req *http.Request) {

	// Get user
	// Get the file id from the url params
	vars := mux.Vars(req)
	userID := vars["userID"]
	if userID == "" {
		ctx.MissingParameter("user ID")
		return
	}

	user, err := ctx.GetMetadataBackend().GetUser(common.GetUserID(common.ProviderLocal, userID))
	if err != nil {
		ctx.BadRequest("unable to get user : %s", err)
		return
	}

	if user == nil {
		ctx.BadRequest("user does not exists")
		return
	}

	if user.VerificationCode == "" {
		ctx.BadRequest("missing confirmation code, please send confirmation code first")
		return
	}

	// Get the file id from the url params
	code := vars["code"]
	if code == "" {
		ctx.MissingParameter("verification code")
		return
	}

	if user.VerificationCode != code {
		ctx.Unauthorized("invalid verification code")
		return
	}

	user.VerificationCode = ""
	user.Verified = true

	err = ctx.GetMetadataBackend().UpdateUser(user)
	if err != nil {
		ctx.InternalServerError("unable to update user metadata : %s", err)
		return
	}

	// Authenticate the HTTP response with auth cookies
	ctx.SetUser(user)
	err = setCookies(ctx, resp)
	if err != nil {
		handleHTTPError(ctx, err)
		return
	}

	http.Redirect(resp, req, ctx.GetConfig().Path+"/#/login", http.StatusMovedPermanently)
}
