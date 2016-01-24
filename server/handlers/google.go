/* The MIT License (MIT)

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
THE SOFTWARE. */

package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/root-gg/plik/server/Godeps/_workspace/src/github.com/dgrijalva/jwt-go"
	"github.com/root-gg/plik/server/Godeps/_workspace/src/github.com/nu7hatch/gouuid"
	"github.com/root-gg/plik/server/Godeps/_workspace/src/github.com/root-gg/juliet"
	"github.com/root-gg/plik/server/Godeps/_workspace/src/golang.org/x/oauth2"
	"github.com/root-gg/plik/server/Godeps/_workspace/src/golang.org/x/oauth2/google"
	api_oauth2 "github.com/root-gg/plik/server/Godeps/_workspace/src/google.golang.org/api/oauth2/v2"
	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/metadataBackend"
)

// GoogleLogin return google api user consent URL.
func GoogleLogin(ctx *juliet.Context, resp http.ResponseWriter, req *http.Request) {
	log := common.GetLogger(ctx)

	if !common.Config.Authentication {
		log.Warning("Authentication is disabled")
		common.Fail(ctx, req, resp, "Authentication is disabled", 400)
		return
	}

	if !common.Config.GoogleAuthentication {
		log.Warning("Missing google api credentials")
		common.Fail(ctx, req, resp, "Missing google API credentials", 500)
		return
	}

	origin := req.Header.Get("referer")
	if origin == "" {
		log.Warning("Missing referer header")
		common.Fail(ctx, req, resp, "Missing referer herader", 400)
		return
	}

	conf := &oauth2.Config{
		ClientID:     common.Config.GoogleAPIClientID,
		ClientSecret: common.Config.GoogleAPISecret,
		RedirectURL:  origin + "auth/google/callback",
		Scopes: []string{
			api_oauth2.UserinfoEmailScope,
			api_oauth2.UserinfoProfileScope,
		},
		Endpoint: google.Endpoint,
	}

	/* Generate state */
	state := jwt.New(jwt.SigningMethodHS256)
	state.Claims["origin"] = origin
	state.Claims["expire"] = time.Now().Add(time.Minute * 5).Unix()

	/* Sign state */
	b64state, err := state.SignedString([]byte(common.Config.GoogleAPISecret))
	if err != nil {
		log.Warningf("Unable to sign state : %s", err)
		common.Fail(ctx, req, resp, "Unable to sign state", 500)
		return
	}

	// Redirect user to Google's consent page to ask for permission
	// for the scopes specified above.
	url := conf.AuthCodeURL(b64state)

	resp.Write([]byte(url))
}

// GoogleCallback authenticate google user.
func GoogleCallback(ctx *juliet.Context, resp http.ResponseWriter, req *http.Request) {
	log := common.GetLogger(ctx)

	if !common.Config.Authentication {
		log.Warning("Authentication is disabled")
		common.Fail(ctx, req, resp, "Authentication is disabled", 400)
		return
	}

	if common.Config.GoogleAPIClientID == "" || common.Config.GoogleAPISecret == "" {
		log.Warning("Missing google api credentials")
		common.Fail(ctx, req, resp, "Missing google API credentials", 500)
		return
	}

	code := req.URL.Query().Get("code")
	if code == "" {
		log.Warning("Missing oauth2 authorization code")
		common.Fail(ctx, req, resp, "Missing oauth2 authorization code", 400)
		return
	}

	b64state := req.URL.Query().Get("state")
	if b64state == "" {
		log.Warning("Missing oauth2 state")
		common.Fail(ctx, req, resp, "Missing oauth2 state", 400)
		return
	}

	/* Parse state */
	state, err := jwt.Parse(b64state, func(token *jwt.Token) (interface{}, error) {
		// Verify signing algorithm
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected siging method : %v", token.Header["alg"])
		}

		// Verify expiration date
		if expire, ok := token.Claims["expire"]; ok {
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

		return []byte(common.Config.GoogleAPISecret), nil
	})
	if err != nil {
		log.Warning("Invalid oauth2 state : %s")
		common.Fail(ctx, req, resp, "Invalid oauth2 state", 400)
		return
	}

	origin := state.Claims["origin"].(string)

	conf := &oauth2.Config{
		ClientID:     common.Config.GoogleAPIClientID,
		ClientSecret: common.Config.GoogleAPISecret,
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
		common.Fail(ctx, req, resp, "Unable to get user info from google API", 500)
		return
	}

	client, err := api_oauth2.New(conf.Client(oauth2.NoContext, token))
	if err != nil {
		log.Warningf("Unable to create google API client : %s", err)
		common.Fail(ctx, req, resp, "Unable to get user info from google API", 500)
		return
	}

	userInfo, err := client.Userinfo.Get().Do()
	if err != nil {
		log.Warningf("Unable to get userinfo from google API : %s", err)
		common.Fail(ctx, req, resp, "Unable to get user info from google API", 500)
		return
	}
	userID := "google:" + userInfo.Id

	// Get user from metadata backend
	user, err := metadataBackend.GetMetaDataBackend().GetUser(ctx, userID, "")
	if err != nil {
		log.Warningf("Unable to get user : %s", err)
		common.Fail(ctx, req, resp, "Unable to get user", 500)
		return
	}

	if user == nil {
		if common.IsWhitelisted(ctx) {
			// Create new user
			user = common.NewUser()
			user.ID = userID
			user.Login = userInfo.Email
			user.Name = userInfo.Name
			user.Email = userInfo.Email

			// Save user to metadata backend
			err = metadataBackend.GetMetaDataBackend().SaveUser(ctx, user)
			if err != nil {
				log.Warningf("Unable to save user to metadata backend : %s", err)
				common.Fail(ctx, req, resp, "Authentification error", 403)
				return
			}
		} else {
			log.Warning("Unable to create user from untrusted source IP address")
			common.Fail(ctx, req, resp, "Unable to create user from untrusted source IP address", 403)
			return
		}
	}

	// Generate session jwt
	session := jwt.New(jwt.SigningMethodHS256)
	session.Claims["uid"] = user.ID
	session.Claims["provider"] = "google"

	// Generate xsrf token
	xsrfToken, err := uuid.NewV4()
	if err != nil {
		log.Warning("Unable to generate xsrf token")
		common.Fail(ctx, req, resp, "Unable to generate xsrf token", 500)
		return
	}
	session.Claims["xsrf"] = xsrfToken.String()

	sessionString, err := session.SignedString([]byte(common.Config.GoogleAPISecret))
	if err != nil {
		log.Warningf("Unable to sign session cookie : %s", err)
		common.Fail(ctx, req, resp, "Authentification error", 403)
		return
	}

	// Store session jwt in secure cookie
	sessionCookie := &http.Cookie{}
	sessionCookie.HttpOnly = true
	//	secureCookie.Secure = true
	sessionCookie.Name = "plik-session"
	sessionCookie.Value = sessionString
	sessionCookie.MaxAge = int(time.Now().Add(10 * 365 * 24 * time.Hour).Unix())
	sessionCookie.Path = "/"
	http.SetCookie(resp, sessionCookie)

	// Store xsrf token cookie
	xsrfCookie := &http.Cookie{}
	sessionCookie.HttpOnly = false
	//	secureCookie.Secure = true
	xsrfCookie.Name = "plik-xsrf"
	xsrfCookie.Value = xsrfToken.String()
	xsrfCookie.MaxAge = int(time.Now().Add(10 * 365 * 24 * time.Hour).Unix())
	xsrfCookie.Path = "/"
	http.SetCookie(resp, xsrfCookie)

	http.Redirect(resp, req, "/#/login", 301)
}
