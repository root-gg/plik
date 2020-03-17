package common

import (
	"fmt"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	uuid "github.com/nu7hatch/gouuid"
	"golang.org/x/crypto/bcrypt"
)

const sessionCookieName = "plik-session"
const xsrfCookieName = "plik-xsrf"

// GenerateAuthenticationSignatureKey create an new random key
func GenerateAuthenticationSignatureKey() (s *Setting) {
	key := GenerateRandomID(64)
	return &Setting{
		Key:   AuthenticationSignatureKeySettingKey,
		Value: key,
	}
}

// SessionAuthenticator to generate and authenticate session cookies
type SessionAuthenticator struct {
	SignatureKey  string
	SecureCookies bool
}

// GenAuthCookies generate a sign a jwt session cookie to authenticate a user
func (sa *SessionAuthenticator) GenAuthCookies(user *User) (sessionCookie *http.Cookie, xsrfCookie *http.Cookie, err error) {
	// Generate session jwt
	session := jwt.New(jwt.SigningMethodHS512)
	session.Claims.(jwt.MapClaims)["uid"] = user.ID

	// Generate xsrf token
	xsrfToken, err := uuid.NewV4()
	if err != nil {
		return nil, nil, fmt.Errorf("unable to generate xsrf token")
	}
	session.Claims.(jwt.MapClaims)["xsrf"] = xsrfToken.String()

	sessionString, err := session.SignedString([]byte(sa.SignatureKey))
	if err != nil {
		return nil, nil, fmt.Errorf("unable to sign session cookie : %s", err)
	}

	// Store session jwt in secure cookie
	sessionCookie = &http.Cookie{}
	sessionCookie.HttpOnly = true
	sessionCookie.Name = sessionCookieName
	sessionCookie.Value = sessionString
	sessionCookie.MaxAge = int(time.Now().Add(10 * 365 * 24 * time.Hour).Unix())
	sessionCookie.Path = "/"

	// Store xsrf token cookie
	xsrfCookie = &http.Cookie{}
	xsrfCookie.HttpOnly = false
	xsrfCookie.Name = xsrfCookieName
	xsrfCookie.Value = xsrfToken.String()
	xsrfCookie.MaxAge = int(time.Now().Add(10 * 365 * 24 * time.Hour).Unix())
	xsrfCookie.Path = "/"

	if sa.SecureCookies {
		sessionCookie.Secure = true
		xsrfCookie.Secure = true
	}

	return sessionCookie, xsrfCookie, nil
}

// ParseSessionCookie parse and validate the session cookie
func (sa *SessionAuthenticator) ParseSessionCookie(value string) (uid string, xsrf string, err error) {
	session, err := jwt.Parse(value, func(t *jwt.Token) (interface{}, error) {
		// Verify signing algorithm
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected siging method : %v", t.Header["alg"])
		}

		return []byte(sa.SignatureKey), nil
	})
	if err != nil {
		return "", "", err
	}

	// Get the user id
	userValue, ok := session.Claims.(jwt.MapClaims)["uid"]
	if ok {
		uid, ok = userValue.(string)
		if !ok || uid == "" {
			return "", "", fmt.Errorf("missing user from session cookie")
		}
	} else {
		return "", "", fmt.Errorf("missing user from session cookie")
	}

	// Get the xsrf token
	xsrfValue, ok := session.Claims.(jwt.MapClaims)["xsrf"]
	if ok {
		xsrf, ok = xsrfValue.(string)
		if !ok || uid == "" {
			return "", "", fmt.Errorf("missing xsrf token from session cookie")
		}
	} else {
		return "", "", fmt.Errorf("missing xsrf token from session cookie")
	}

	return uid, xsrf, nil
}

// Logout delete session cookies
func Logout(resp http.ResponseWriter, sa *SessionAuthenticator) {
	sessionCookie, xsrfCookie, _ := sa.Logout()
	http.SetCookie(resp, sessionCookie)
	http.SetCookie(resp, xsrfCookie)
}

// Logout generate logout session cookies
func (sa *SessionAuthenticator) Logout() (sessionCookie *http.Cookie, xsrfCookie *http.Cookie, err error) {
	// Delete session cookie
	sessionCookie = &http.Cookie{}
	sessionCookie.HttpOnly = true
	sessionCookie.Name = sessionCookieName
	sessionCookie.Value = ""
	sessionCookie.MaxAge = -1
	sessionCookie.Path = "/"

	// Store xsrf token cookie
	xsrfCookie = &http.Cookie{}
	xsrfCookie.HttpOnly = false
	xsrfCookie.Name = xsrfCookieName
	xsrfCookie.Value = ""
	xsrfCookie.MaxAge = -1
	xsrfCookie.Path = "/"

	if sa.SecureCookies {
		sessionCookie.Secure = true
		xsrfCookie.Secure = true
	}

	return sessionCookie, xsrfCookie, nil
}

// HashPassword return bcrypt password hash ( with salt )
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

// CheckPasswordHash check password against bcrypt password hash
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
