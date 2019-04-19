/**

    Plik upload server

The MIT License (MIT)

Copyright (c) <2015> Copyright holders list can be found in AUTHORS file
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
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/nu7hatch/gouuid"
	"github.com/root-gg/juliet"
	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	api_oauth2 "google.golang.org/api/oauth2/v2"
)

// GoogleLogin return google api user consent URL.
func GoogleLogin(ctx *juliet.Context, resp http.ResponseWriter, req *http.Request) {
	log := context.GetLogger(ctx)
	config := context.GetConfig(ctx)

	if !config.Authentication {
		log.Warning("Authentication is disabled")
		context.Fail(ctx, req, resp, "Authentication is disabled", 400)
		return
	}

	if !config.GoogleAuthentication {
		log.Warning("Missing google api credentials")
		context.Fail(ctx, req, resp, "Missing google API credentials", 500)
		return
	}

	origin := req.Header.Get("referer")
	if origin == "" {
		log.Warning("Missing referer header")
		context.Fail(ctx, req, resp, "Missing referer header", 400)
		return
	}

	conf := &oauth2.Config{
		ClientID:     config.GoogleAPIClientID,
		ClientSecret: config.GoogleAPISecret,
		RedirectURL:  origin + "auth/google/callback",
		Scopes: []string{
			api_oauth2.UserinfoEmailScope,
			api_oauth2.UserinfoProfileScope,
		},
		Endpoint: google.Endpoint,
	}

	/* Generate state */
	state := jwt.New(jwt.SigningMethodHS256)
	state.Claims.(jwt.MapClaims)["origin"] = origin
	state.Claims.(jwt.MapClaims)["expire"] = time.Now().Add(time.Minute * 5).Unix()

	/* Sign state */
	b64state, err := state.SignedString([]byte(config.GoogleAPISecret))
	if err != nil {
		log.Warningf("Unable to sign state : %s", err)
		context.Fail(ctx, req, resp, "Unable to sign state", 500)
		return
	}

	// Redirect user to Google's consent page to ask for permission
	// for the scopes specified above.
	url := conf.AuthCodeURL(b64state)

	resp.Write([]byte(url))
}

// GoogleCallback authenticate google user.
func GoogleCallback(ctx *juliet.Context, resp http.ResponseWriter, req *http.Request) {
	log := context.GetLogger(ctx)
	config := context.GetConfig(ctx)

	if !config.Authentication {
		log.Warning("Authentication is disabled")
		context.Fail(ctx, req, resp, "Authentication is disabled", 400)
		return
	}

	if config.GoogleAPIClientID == "" || config.GoogleAPISecret == "" {
		log.Warning("Missing google api credentials")
		context.Fail(ctx, req, resp, "Missing google API credentials", 500)
		return
	}

	code := req.URL.Query().Get("code")
	if code == "" {
		log.Warning("Missing oauth2 authorization code")
		context.Fail(ctx, req, resp, "Missing oauth2 authorization code", 400)
		return
	}

	b64state := req.URL.Query().Get("state")
	if b64state == "" {
		log.Warning("Missing oauth2 state")
		context.Fail(ctx, req, resp, "Missing oauth2 state", 400)
		return
	}

	/* Parse state */
	state, err := jwt.Parse(b64state, func(token *jwt.Token) (interface{}, error) {
		// Verify signing algorithm
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected siging method : %v", token.Header["alg"])
		}

		// Verify expiration data
		if expire, ok := token.Claims.(jwt.MapClaims)["expire"]; ok {
			if _, ok = expire.(float64); ok {
				if time.Now().Unix() > (int64)(expire.(float64)) {
					return nil, fmt.Errorf("State has expired")
				}
			} else {
				return nil, fmt.Errorf("Invalid expiration date")
			}
		} else {
			return nil, fmt.Errorf("Missing expiration date")
		}

		return []byte(config.GoogleAPISecret), nil
	})
	if err != nil {
		log.Warning("Invalid oauth2 state : %s")
		context.Fail(ctx, req, resp, "Invalid oauth2 state", 400)
		return
	}

	origin := state.Claims.(jwt.MapClaims)["origin"].(string)

	conf := &oauth2.Config{
		ClientID:     config.GoogleAPIClientID,
		ClientSecret: config.GoogleAPISecret,
		RedirectURL:  origin + "auth/google/callback",
		Scopes: []string{
			api_oauth2.UserinfoEmailScope,
			api_oauth2.UserinfoProfileScope,
		},
		Endpoint: google.Endpoint,
	}

	token, err := conf.Exchange(oauth2.NoContext, code)
	if err != nil {
		log.Warningf("Unable to create google API token : %s", err)
		context.Fail(ctx, req, resp, "Unable to get user info from google API", 500)
		return
	}

	client, err := api_oauth2.New(conf.Client(oauth2.NoContext, token))
	if err != nil {
		log.Warningf("Unable to create google API client : %s", err)
		context.Fail(ctx, req, resp, "Unable to get user info from google API", 500)
		return
	}

	userInfo, err := client.Userinfo.Get().Do()
	if err != nil {
		log.Warningf("Unable to get userinfo from google API : %s", err)
		context.Fail(ctx, req, resp, "Unable to get user info from google API", 500)
		return
	}
	userID := "google:" + userInfo.Id

	// Get user from metadata backend
	user, err := context.GetMetadataBackend(ctx).GetUser(ctx, userID, "")
	if err != nil {
		log.Warningf("Unable to get user : %s", err)
		context.Fail(ctx, req, resp, "Unable to get user", 500)
		return
	}

	if user == nil {
		if context.IsWhitelisted(ctx) {
			// Create new user
			user = common.NewUser()
			user.ID = userID
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
				log.Warningf("Unacceptable user domain : %s", components[1])
				context.Fail(ctx, req, resp, fmt.Sprintf("Authentification error : Unauthorized domain %s", components[1]), 403)
				return
			}

			// Save user to metadata backend
			err = context.GetMetadataBackend(ctx).SaveUser(ctx, user)
			if err != nil {
				log.Warningf("Unable to save user to metadata backend : %s", err)
				context.Fail(ctx, req, resp, "Authentification error", 403)
				return
			}
		} else {
			log.Warning("Unable to create user from untrusted source IP address")
			context.Fail(ctx, req, resp, "Unable to create user from untrusted source IP address", 403)
			return
		}
	}

	// Generate session jwt
	session := jwt.New(jwt.SigningMethodHS256)
	session.Claims.(jwt.MapClaims)["uid"] = user.ID
	session.Claims.(jwt.MapClaims)["provider"] = "google"

	// Generate xsrf token
	xsrfToken, err := uuid.NewV4()
	if err != nil {
		log.Warning("Unable to generate xsrf token")
		context.Fail(ctx, req, resp, "Unable to generate xsrf token", 500)
		return
	}
	session.Claims.(jwt.MapClaims)["xsrf"] = xsrfToken.String()

	sessionString, err := session.SignedString([]byte(config.GoogleAPISecret))
	if err != nil {
		log.Warningf("Unable to sign session cookie : %s", err)
		context.Fail(ctx, req, resp, "Authentification error", 403)
		return
	}

	// Store session jwt in secure cookie
	sessionCookie := &http.Cookie{}
	sessionCookie.HttpOnly = true
	sessionCookie.Secure = true
	sessionCookie.Name = "plik-session"
	sessionCookie.Value = sessionString
	sessionCookie.MaxAge = int(time.Now().Add(10 * 365 * 24 * time.Hour).Unix())
	sessionCookie.Path = "/"
	http.SetCookie(resp, sessionCookie)

	// Store xsrf token cookie
	xsrfCookie := &http.Cookie{}
	xsrfCookie.HttpOnly = false
	xsrfCookie.Secure = true
	xsrfCookie.Name = "plik-xsrf"
	xsrfCookie.Value = xsrfToken.String()
	xsrfCookie.MaxAge = int(time.Now().Add(10 * 365 * 24 * time.Hour).Unix())
	xsrfCookie.Path = "/"
	http.SetCookie(resp, xsrfCookie)

	http.Redirect(resp, req, config.Path+"/#/login", 301)
}
