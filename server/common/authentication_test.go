package common

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSessionAuthenticator(t *testing.T) {
	setting := GenerateAuthenticationSignatureKey()
	sa := &SessionAuthenticator{SignatureKey: setting.Value, SecureCookies: true}

	user := NewUser("local", "user")

	sessionCookie, xsrfCookie, err := sa.GenAuthCookies(user)
	require.NoError(t, err, "unable to generate cookies")
	require.NotNil(t, sessionCookie, "missing session cookie")
	require.NotNil(t, xsrfCookie, "missing xsrf cookie")

	require.NotNil(t, sessionCookie, "missing session cookie")
	require.NotEqual(t, -1, sessionCookie.MaxAge, "invalid session cookie")
	require.True(t, sessionCookie.Secure, "invalid session cookies not secure")

	require.NotNil(t, xsrfCookie, "missing xsrf cookie")
	require.NotEqual(t, -1, xsrfCookie.MaxAge, "invalid xsrf cookie")
	require.NotEqual(t, -1, xsrfCookie.MaxAge, "invalid xsrf cookie")
	require.True(t, xsrfCookie.Secure, "invalid xsrf cookie not secure")

	uid, xsrf, err := sa.ParseSessionCookie(sessionCookie.Value)
	require.NoError(t, err, "unable to parse session cookie")
	require.Equal(t, user.ID, uid, "invalid user id")
	require.Equal(t, xsrfCookie.Value, xsrf, "invalid xsrf token")
}

func TestLogout(t *testing.T) {
	rr := httptest.NewRecorder()
	Logout(rr, &SessionAuthenticator{SecureCookies: true})
	require.Equal(t, 2, len(rr.Result().Cookies()), "missing response cookies")

	var sessionCookie *http.Cookie
	var xsrfCookie *http.Cookie

	for _, cookie := range rr.Result().Cookies() {
		if cookie.Name == sessionCookieName {
			sessionCookie = cookie
		}
		if cookie.Name == xsrfCookieName {
			xsrfCookie = cookie
		}
	}

	require.NotNil(t, sessionCookie, "missing session cookie")
	require.Equal(t, -1, sessionCookie.MaxAge, "invalid session cookie")
	require.True(t, sessionCookie.Secure, "invalid session cookie not secure")

	require.NotNil(t, xsrfCookie, "missing xsrf cookie")
	require.Equal(t, -1, xsrfCookie.MaxAge, "invalid xsrf cookie")
	require.True(t, xsrfCookie.Secure, "invalid xsrf cookie not secure")
}

func TestHashPassword(t *testing.T) {
	hash, err := HashPassword("password")
	require.NoError(t, err, "hash password error")

	ok := CheckPasswordHash("password", hash)
	require.True(t, ok)

	ok = CheckPasswordHash("invalid", hash)
	require.False(t, ok)
}
