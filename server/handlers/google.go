package handlers

import (
	"fmt"
	"net/http"
	"strings"
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

	/* Sign state */
	b64state, err := state.SignedString([]byte(config.GoogleAPISecret))
	if err != nil {
		ctx.InternalServerError("unable to sign state", err)
		return
	}

	// Redirect user to Google's consent page to ask for permission
	// for the scopes specified above.
	url := conf.AuthCodeURL(b64state)

	_, _ = resp.Write([]byte(url))
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

	if _, ok := state.Claims.(jwt.MapClaims)["redirectURL"]; !ok {
		ctx.InvalidParameter("oauth2 state : missing redirectURL")
		return
	}

	if _, ok := state.Claims.(jwt.MapClaims)["redirectURL"].(string); !ok {
		ctx.InvalidParameter("oauth2 state : invalid redirectURL")
		return
	}

	redirectURL := state.Claims.(jwt.MapClaims)["redirectURL"].(string)

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

	// Get user from metadata backend
	user, err := ctx.GetMetadataBackend().GetUser(common.GetUserID(common.ProviderGoogle, userInfo.Email))
	if err != nil {
		ctx.InternalServerError("unable to get user from metadata backend", err)
		return
	}

	if user == nil {
		if ctx.IsWhitelisted() {
			// Create new user
			user = common.NewUser(common.ProviderGoogle, userInfo.Email)
			user.Login = userInfo.Email
			user.Name = userInfo.Name
			user.Email = userInfo.Email
			components := strings.Split(user.Email, "@")

			// Accepted user domain checking
			goodDomain := false
			if len(config.GoogleValidDomains) > 0 {
				for _, validDomain := range config.GoogleValidDomains {
					if strings.Compare(components[1], validDomain) == 0 {
						goodDomain = true
					}
				}
			} else {
				goodDomain = true
			}

			if !goodDomain {
				// User not from accepted google domains list
				ctx.Forbidden("unauthorized domain name")
				return
			}

			// Save user to metadata backend
			err = ctx.GetMetadataBackend().CreateUser(user)
			if err != nil {
				ctx.InternalServerError("unable to create user : %s", err)
				return
			}
		} else {
			ctx.Forbidden("unable to create user from untrusted source IP address")
			return
		}
	}

	// Set Plik session cookie and xsrf cookie
	sessionCookie, xsrfCookie, err := ctx.GetAuthenticator().GenAuthCookies(user)
	if err != nil {
		ctx.InternalServerError("unable to generate session cookies", err)
	}
	http.SetCookie(resp, sessionCookie)
	http.SetCookie(resp, xsrfCookie)

	http.Redirect(resp, req, config.Path+"/#/login", http.StatusMovedPermanently)
}
