package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	api_oauth2 "google.golang.org/api/oauth2/v2"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
)

var googleEndpointContextKey = "google_endpoint"

// GoogleLogin return google api user consent URL.
func GoogleLogin(ctx *context.Context, resp http.ResponseWriter, req *http.Request) {
	config := ctx.GetConfig()

	if !config.Authentication {
		ctx.BadRequest("authentication is disabled")
		return
	}

	if !config.GoogleAuthentication {
		ctx.BadRequest("Google authentication is disabled")
		return
	}

	if config.GoogleAPIClientID == "" || config.GoogleAPISecret == "" {
		ctx.InternalServerError("missing Google API credentials", nil)
		return
	}

	// Get redirection URL from the referrer header
	redirectURL, err := getRedirectURL(ctx, "/auth/google/callback")
	if err != nil {
		handleHTTPError(ctx, err)
		return
	}

	conf := &oauth2.Config{
		ClientID:     config.GoogleAPIClientID,
		ClientSecret: config.GoogleAPISecret,
		RedirectURL:  redirectURL,
		Scopes: []string{
			api_oauth2.UserinfoEmailScope,
			api_oauth2.UserinfoProfileScope,
		},
		Endpoint: google.Endpoint,
	}

	/* Generate state */
	state := jwt.New(jwt.SigningMethodHS256)
	state.Claims.(jwt.MapClaims)["redirectURL"] = redirectURL
	state.Claims.(jwt.MapClaims)["expire"] = time.Now().Add(time.Minute * 5).Unix()
	if req.URL.Query().Get("invite") != "" {
		state.Claims.(jwt.MapClaims)["invite"] = req.URL.Query().Get("invite")
	}

	/* Sign state */
	b64state, err := state.SignedString([]byte(config.GoogleAPISecret))
	if err != nil {
		ctx.InternalServerError("unable to sign state", err)
		return
	}

	// Redirect user to Google's consent page to ask for permission
	// for the scopes specified above.
	url := conf.AuthCodeURL(b64state)
	common.WriteStringResponse(resp, url)
}

// GoogleCallback authenticate google user.
func GoogleCallback(ctx *context.Context, resp http.ResponseWriter, req *http.Request) {
	config := ctx.GetConfig()

	if !config.Authentication {
		ctx.BadRequest("authentication is disabled")
		return
	}

	if !config.GoogleAuthentication {
		ctx.BadRequest("Google authentication is disabled")
		return
	}

	if config.GoogleAPIClientID == "" || config.GoogleAPISecret == "" {
		ctx.InternalServerError("missing Google API credentials", nil)
		return
	}

	code := req.URL.Query().Get("code")
	if code == "" {
		ctx.MissingParameter("oauth2 authorization code")
		return
	}

	b64state := req.URL.Query().Get("state")
	if b64state == "" {
		ctx.MissingParameter("oauth2 authorization state")
		return
	}

	/* Parse state */
	state, err := jwt.Parse(b64state, func(token *jwt.Token) (interface{}, error) {
		// Verify signing algorithm
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected siging method : %v", token.Header["alg"])
		}

		// Verify expiration data
		if expire, ok := token.Claims.(jwt.MapClaims)["expire"]; ok {
			if _, ok = expire.(float64); ok {
				if time.Now().Unix() > (int64)(expire.(float64)) {
					return nil, fmt.Errorf("state has expired")
				}
			} else {
				return nil, fmt.Errorf("invalid expiration date")
			}
		} else {
			return nil, fmt.Errorf("missing expiration date")
		}

		return []byte(config.GoogleAPISecret), nil
	})
	if err != nil {
		ctx.InvalidParameter("oauth2 state : %s", err)
		return
	}

	redirectURL := getClaim(state, "redirectURL")
	if redirectURL == "" {
		ctx.MissingParameter("redirectURL")
		return
	}

	conf := &oauth2.Config{
		ClientID:     config.GoogleAPIClientID,
		ClientSecret: config.GoogleAPISecret,
		RedirectURL:  redirectURL,
		Scopes: []string{
			api_oauth2.UserinfoEmailScope,
			api_oauth2.UserinfoProfileScope,
		},
		Endpoint: google.Endpoint,
	}

	// For testing purpose
	if customEndpoint := req.Context().Value(googleEndpointContextKey); customEndpoint != nil {
		conf.Endpoint = customEndpoint.(oauth2.Endpoint)
	}

	token, err := conf.Exchange(oauth2.NoContext, code)
	if err != nil {
		ctx.InternalServerError("unable to get user info from Google API (1)", err)
		return
	}

	client, err := api_oauth2.New(conf.Client(oauth2.NoContext, token))
	if err != nil {
		ctx.InternalServerError("unable to get user info from Google API (2)", err)
		return
	}

	// For testing purpose
	if customEndpoint := req.Context().Value(googleEndpointContextKey); customEndpoint != nil {
		client.BasePath = customEndpoint.(oauth2.Endpoint).AuthURL
	}

	userInfo, err := client.Userinfo.Get().Do()
	if err != nil {
		ctx.InternalServerError("unable to get user info from Google API (3)", err)
		return
	}

	// Create new user
	user := common.NewUser(common.ProviderGoogle, userInfo.Email)
	user.Login = userInfo.Email
	user.Name = userInfo.Name
	user.Email = userInfo.Email

	// Trust user info
	user.Verified = true

	// Get or create user
	err = register(ctx, user, getClaim(state, "invite"))
	if err != nil && err != errUserExists {
		handleHTTPError(ctx, err)
		return
	}

	// Authenticate the HTTP response with auth cookies
	err = setCookies(ctx, resp)
	if err != nil {
		handleHTTPError(ctx, err)
		return
	}

	http.Redirect(resp, req, config.Path+"/#/login", http.StatusMovedPermanently)
}
