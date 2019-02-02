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

import "net/http"

// User is a plik user
type User struct {
	ID     string   `json:"id,omitempty" bson:"id"`
	Login  string   `json:"login,omitempty" bson:"login"`
	Name   string   `json:"name,omitempty" bson:"name"`
	Email  string   `json:"email,omitempty" bson:"email"`
	Tokens []*Token `json:"tokens,omitempty" bson:"tokens"`
	Admin  bool     `json:"admin" bson:"-"`
}

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

// IsAdmin check if the user is a Plik server administrator
func (user *User) IsAdmin() bool {
	if user.Admin == true {
		return true
	}

	// Check if user is admin
	for _, id := range Config.Admins {
		if user.ID == id {
			user.Admin = true
			return true
		}
	}

	return false
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
