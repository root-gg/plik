package common

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSessionAuthenticator(t *testing.T) {
	setting := GenerateAuthenticationSignatureKey()
	sa := &SessionAuthenticator{SignatureKey: setting.Value}

	user := NewUser("local", "user")

	sessionCookie, xsrfCookie, err := sa.GenAuthCookies(user)
	require.NoError(t, err, "unable to generate cookies")
	require.NotNil(t, sessionCookie, "missing session cookie")
	require.NotNil(t, xsrfCookie, "missing xsrf cookie")

	require.NotNil(t, sessionCookie, "missing session cookies")
	require.NotEqual(t, -1, sessionCookie.MaxAge, "invalid session cookies")

	require.NotNil(t, xsrfCookie, "missing xsrf cookies")
	require.NotEqual(t, -1, xsrfCookie.MaxAge, "invalid xsrf cookies")

	uid, xsrf, err := sa.ParseSessionCookie(sessionCookie.Value)
	require.NoError(t, err, "unable to parse session cookie")
	require.Equal(t, user.ID, uid, "invalid user id")
	require.Equal(t, xsrfCookie.Value, xsrf, "invalid xsrf token")
}

func TestLogout(t *testing.T) {
	rr := httptest.NewRecorder()
	Logout(rr)
	require.Equal(t, 2, len(rr.Result().Cookies()), "missing response cookies")

	var sessionCookie *http.Cookie
	var xsrfCookie *http.Cookie

	for _, cookie := range rr.Result().Cookies() {
		if cookie.Name == "plik-session" {
			sessionCookie = cookie
		}
		if cookie.Name == "plik-xsrf" {
			xsrfCookie = cookie
		}
	}

	require.NotNil(t, sessionCookie, "missing session cookies")
	require.Equal(t, -1, sessionCookie.MaxAge, "invalid session cookies")

	require.NotNil(t, xsrfCookie, "missing xsrf cookies")
	require.Equal(t, -1, xsrfCookie.MaxAge, "invalid xsrf cookies")
}

func TestHashPassword(t *testing.T) {
	hash, err := HashPassword("password")
	require.NoError(t, err, "hash password error")

	ok := CheckPasswordHash("password", hash)
	require.True(t, ok)

	ok = CheckPasswordHash("invalid", hash)
	require.False(t, ok)
}
