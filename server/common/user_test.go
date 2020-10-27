package common

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestUserNewToken(t *testing.T) {
	user := &User{}
	token := user.NewToken()
	require.NotNil(t, token, "missing token")
	require.NotZero(t, token.Token, "missing token initialization")
	require.NotZero(t, len(user.Tokens), "missing token")
	require.Equal(t, token, user.Tokens[0], "missing token")
}

func TestUserNewInvite(t *testing.T) {
	user := NewUser(ProviderLocal, "user")
	invite, err := user.NewInvite(24 * 30 * time.Hour)
	require.NoError(t, err)
	require.NotNil(t, invite)
	require.Equal(t, user.ID, *invite.Issuer)
}

func TestUser_String(t *testing.T) {
	user := NewUser(ProviderLocal, "user")
	user.Name = "user"
	user.Login = "user"
	user.Email = "user@root.gg"
	require.NotEmpty(t, user.String())
}

func TestIsValidProvider(t *testing.T) {
	require.True(t, IsValidProvider(ProviderLocal))
	require.True(t, IsValidProvider(ProviderGoogle))
	require.True(t, IsValidProvider(ProviderOVH))
	require.False(t, IsValidProvider("blah"))
}

func validUser() (user *User) {
	user = NewUser(ProviderLocal, "plik")
	user.Login = "plik"
	user.Name = "plik"
	user.Email = "plik@root.gg"
	user.Password = "secret"
	return user
}

func TestUser_PrepareInsert(t *testing.T) {
	config := NewConfiguration()

	user := validUser()
	require.NoError(t, user.PrepareInsert(config))

	for _, provider := range []string{"", "foo"} {
		user = validUser()
		user.Provider = provider
		RequireError(t, user.PrepareInsert(config), "invalid provider", provider)
	}

	for _, login := range []string{"", "no", "foo bar", "pélélé", "&!*%$", "login\n"} {
		user = validUser()
		user.Login = login
		RequireError(t, user.PrepareInsert(config), "invalid login", login)
	}

	for _, login := range []string{"foo_bar@baz.gg"} {
		user = validUser()
		user.Login = login
		require.NoError(t, user.PrepareInsert(config), login)
	}

	for _, name := range []string{"", "name\n"} {
		user = validUser()
		user.Name = name
		RequireError(t, user.PrepareInsert(config), "invalid name", name)
	}

	for _, name := range []string{"foo_bar@baz.gg", "ù µ汉字"} {
		user = validUser()
		user.Name = name
		require.NoError(t, user.PrepareInsert(config), name)
	}

	for _, email := range []string{"", "foo", "foo b@r.gg"} {
		user = validUser()
		user.Email = email
		RequireError(t, user.PrepareInsert(config), "invalid email", email)
	}

	for _, email := range []string{"foo@bar", "foo_bar@baz.gg"} {
		user = validUser()
		user.Email = email
		require.NoError(t, user.PrepareInsert(config), email)
	}

	config = NewConfiguration()
	config.EmailVerification = false
	user = validUser()
	require.NoError(t, user.PrepareInsert(config))
	require.True(t, user.Verified)

	config = NewConfiguration()
	config.EmailVerification = true
	user = validUser()
	require.NoError(t, user.PrepareInsert(config))
	require.False(t, user.Verified)

	config = NewConfiguration()
	user = validUser()
	user.Provider = ProviderGoogle
	user.Password = ""
	require.NoError(t, user.PrepareInsert(config))

	user = validUser()
	user.Provider = ProviderOVH
	user.Password = ""
	require.NoError(t, user.PrepareInsert(config))

	user = validUser()
	user.Password = ""
	RequireError(t, user.PrepareInsert(config), "password")

	user = validUser()
	pwd := user.Password
	require.NoError(t, user.PrepareInsert(config))
	require.NotEqual(t, pwd, user.Password)
}

func TestUser_GenVerificationCode(t *testing.T) {
	user := validUser()
	user.GenVerificationCode()
	require.NotEmpty(t, user.VerificationCode)
}
