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

package common

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	uuid "github.com/nu7hatch/gouuid"
)

// User is a plik user
type User struct {
	ID      string   `json:"id,omitempty" bson:"id"`
	Login   string   `json:"login,omitempty" bson:"login"`
	Name    string   `json:"name,omitempty" bson:"name"`
	Email   string   `json:"email,omitempty" bson:"email"`
	Tokens  []*Token `json:"tokens,omitempty" bson:"tokens"`
	IsAdmin bool     `json:"admin" bson:"-"`
}

// UserStats represents the user statistics
type UserStats struct {
	Uploads   int   `json:"uploads"`
	Files     int   `json:"files"`
	TotalSize int64 `json:"totalSize"`
}

// NewUser create a new user object
func NewUser() (user *User) {
	user = new(User)
	user.Tokens = make([]*Token, 0)
	return
}

// NewToken add a new token to a user
func (user *User) NewToken() (token *Token) {
	token = NewToken()
	token.Create()
	user.Tokens = append(user.Tokens, token)
	return
}

// GenAuthCookies generate a sign a jwt session cookie to authenticate a user
func GenAuthCookies(user *User, config *Configuration) (sessionCookie *http.Cookie, xsrfCookie *http.Cookie, err error) {
	var provider string
	var sig string
	if strings.HasPrefix(user.ID, "ovh:") {
		provider = "ovh"
		sig = config.OvhAPISecret
	} else if strings.HasPrefix(user.ID, "google:") {
		provider = "google"
		sig = config.GoogleAPISecret
	} else {
		return nil, nil, fmt.Errorf("invlid user id from unknown provider")
	}

	// Generate session jwt
	session := jwt.New(jwt.SigningMethodHS256)
	session.Claims.(jwt.MapClaims)["uid"] = user.ID
	session.Claims.(jwt.MapClaims)["provider"] = provider

	// Generate xsrf token
	xsrfToken, err := uuid.NewV4()
	if err != nil {
		return nil, nil, fmt.Errorf("Unable to generate xsrf token")
	}
	session.Claims.(jwt.MapClaims)["xsrf"] = xsrfToken.String()

	sessionString, err := session.SignedString([]byte(sig))
	if err != nil {
		return nil, nil, fmt.Errorf("Unable to sign session cookie : %s", err)
	}

	// Store session jwt in secure cookie
	sessionCookie = &http.Cookie{}
	sessionCookie.HttpOnly = true
	sessionCookie.Secure = true
	sessionCookie.Name = "plik-session"
	sessionCookie.Value = sessionString
	sessionCookie.MaxAge = int(time.Now().Add(10 * 365 * 24 * time.Hour).Unix())
	sessionCookie.Path = "/"

	// Store xsrf token cookie
	xsrfCookie = &http.Cookie{}
	xsrfCookie.HttpOnly = false
	xsrfCookie.Secure = true
	xsrfCookie.Name = "plik-xsrf"
	xsrfCookie.Value = xsrfToken.String()
	xsrfCookie.MaxAge = int(time.Now().Add(10 * 365 * 24 * time.Hour).Unix())
	xsrfCookie.Path = "/"

	return sessionCookie, xsrfCookie, nil
}

// ParseSessionCookie parse and validate the session cookie
func ParseSessionCookie(value string, config *Configuration) (uid string, xsrf string, err error) {
	session, err := jwt.Parse(value, func(t *jwt.Token) (interface{}, error) {
		// Verify signing algorithm
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected siging method : %v", t.Header["alg"])
		}

		// Get authentication provider
		provider, ok := t.Claims.(jwt.MapClaims)["provider"]
		if !ok {
			return nil, fmt.Errorf("Missing authentication provider")
		}

		switch provider {
		case "google":
			if config.GoogleAPISecret == "" {
				return nil, fmt.Errorf("Missing Google API credentials")
			}
			return []byte(config.GoogleAPISecret), nil
		case "ovh":
			if config.OvhAPISecret == "" {
				return nil, fmt.Errorf("Missing OVH API credentials")
			}
			return []byte(config.OvhAPISecret), nil
		default:
			return nil, fmt.Errorf("Invalid authentication provider : %s", provider)
		}
	})
	if err != nil {
		return "", "", err
	}

	// Get the user id
	userValue, ok := session.Claims.(jwt.MapClaims)["uid"]
	if ok {
		uid, ok = userValue.(string)
		if !ok || uid == "" {
			return "", "", fmt.Errorf("Missing user from session cookie")
		}
	} else {
		return "", "", fmt.Errorf("Missing user from session cookie")
	}

	// Get the xsrf token
	xsrfValue, ok := session.Claims.(jwt.MapClaims)["xsrf"]
	if ok {
		xsrf, ok = xsrfValue.(string)
		if !ok || uid == "" {
			return "", "", fmt.Errorf("Missing xsrf token from session cookie")
		}
	} else {
		return "", "", fmt.Errorf("Missing xsrf token from session cookie")
	}

	return uid, xsrf, nil
}

// Logout delete plik session cookies
func Logout(resp http.ResponseWriter) {
	// Delete session cookie
	sessionCookie := &http.Cookie{}
	sessionCookie.HttpOnly = true
	sessionCookie.Secure = true
	sessionCookie.Name = "plik-session"
	sessionCookie.Value = ""
	sessionCookie.MaxAge = -1
	sessionCookie.Path = "/"
	http.SetCookie(resp, sessionCookie)

	// Store xsrf token cookie
	xsrfCookie := &http.Cookie{}
	xsrfCookie.HttpOnly = false
	xsrfCookie.Secure = true
	xsrfCookie.Name = "plik-xsrf"
	xsrfCookie.Value = ""
	xsrfCookie.MaxAge = -1
	xsrfCookie.Path = "/"
	http.SetCookie(resp, xsrfCookie)
}
