package common

import "net/http"

// User is a plik user
type User struct {
	ID     string   `json:"id,omitempty" bson:"id"`
	Login  string   `json:"login,omitempty" bson:"login"`
	Name   string   `json:"name,omitempty" bson:"name"`
	Email  string   `json:"email,omitempty" bson:"email"`
	Tokens []*Token `json:"tokens,omitempty" bson:"tokens"`
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
