package common

import (
	"testing"

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

func TestUser_String(t *testing.T) {
	user := NewUser(ProviderLocal, "user")
	user.Name = "John Doe"
	user.Login = "user"
	user.Email = "user@root.gg"
	require.Equal(t, "local:user John Doe user@root.gg", user.String())
}

func TestUser_String_OIDC(t *testing.T) {
	// OIDC users are keyed by the sub claim, the login holds preferred_username.
	// The list output must show the real ID so it can be used with --login.
	user := NewUser(ProviderOIDC, "8f9d056a-1dab-3192-f1ae-552e024d948e")
	user.Login = "test_user"
	user.Name = "Test User"
	user.Email = "primary@email.com"
	require.Equal(t, "oidc:8f9d056a-1dab-3192-f1ae-552e024d948e test_user Test User primary@email.com", user.String())
}

func TestUser_String_EmptyID(t *testing.T) {
	// A user built without an ID (e.g. constructed directly) must still fall
	// back to provider:login rather than emitting a leading space.
	user := &User{Provider: ProviderLocal, Login: "user"}
	require.Equal(t, "local:user", user.String())
}

func TestIsValidProvider(t *testing.T) {
	require.True(t, IsValidProvider(ProviderLocal))
	require.True(t, IsValidProvider(ProviderGoogle))
	require.True(t, IsValidProvider(ProviderGitHub))
	require.True(t, IsValidProvider(ProviderOVH))
	require.True(t, IsValidProvider(ProviderOIDC))
	require.False(t, IsValidProvider(""))
	require.False(t, IsValidProvider("foo"))
}

func TestCreateUserFromParams(t *testing.T) {
	params := &User{}
	user, err := CreateUserFromParams(params)
	require.Error(t, err)
	require.Nil(t, user)

	userOK := &User{
		Provider:    "local",
		Login:       "user",
		Password:    "password",
		Email:       "user@root.gg",
		Name:        "user",
		MaxFileSize: 1234,
		MaxTTL:      1234,
		MaxUserSize: 1234,
		IsAdmin:     true,
	}

	user, err = CreateUserFromParams(userOK)
	require.NoError(t, err)
	require.NotNil(t, user)
	require.Equal(t, userOK.Provider+":"+userOK.Login, user.ID)
	require.Equal(t, userOK.Provider, user.Provider)
	require.Equal(t, userOK.Login, user.Login)
	require.True(t, CheckPasswordHash(userOK.Password, user.Password))
	require.Equal(t, userOK.Name, user.Name)
	require.Equal(t, userOK.Email, user.Email)
	require.Equal(t, userOK.MaxFileSize, user.MaxFileSize)
	require.Equal(t, userOK.MaxUserSize, user.MaxUserSize)
	require.Equal(t, userOK.MaxTTL, user.MaxTTL)
	require.Equal(t, userOK.IsAdmin, user.IsAdmin)

	userKO := *userOK
	userKO.Provider = ""
	user, err = CreateUserFromParams(&userKO)
	require.Error(t, err)
	require.Nil(t, user)

	userKO = *userOK
	userKO.Login = ""
	user, err = CreateUserFromParams(&userKO)
	require.Error(t, err)
	require.Nil(t, user)

	userKO = *userOK
	userKO.Login = "bad"
	user, err = CreateUserFromParams(&userKO)
	require.Error(t, err)
	require.Nil(t, user)

	userKO = *userOK
	userKO.Password = ""
	user, err = CreateUserFromParams(&userKO)
	require.Error(t, err)
	require.Nil(t, user)

	userGoogle := *userOK
	userGoogle.Provider = ProviderGoogle
	userGoogle.Password = ""
	user, err = CreateUserFromParams(&userGoogle)
	require.NoError(t, err)
	require.Empty(t, user.Password)
}

func TestUpdateUser(t *testing.T) {
	userOK := &User{
		Provider:    "local",
		Login:       "user",
		Password:    "password",
		Email:       "user@root.gg",
		Name:        "user",
		MaxFileSize: 1234,
		MaxUserSize: 1234,
		MaxTTL:      1234,
		IsAdmin:     true,
		Theme:       "nord",
	}

	params := *userOK
	params.Login = "notupdated"
	params.Provider = "notupdated"
	params.ID = "notupdated"
	params.Email = "updated@root.gg"
	params.Name = "updated"
	params.MaxFileSize = 0
	params.MaxUserSize = 0
	params.MaxTTL = 0
	params.IsAdmin = false
	params.Theme = "catppuccin-mocha"

	user := *userOK
	err := UpdateUser(&user, &params)
	require.NoError(t, err)

	require.Equal(t, userOK.Provider, user.Provider)
	require.Equal(t, userOK.ID, user.ID)
	require.Equal(t, userOK.Login, user.Login)
	require.True(t, CheckPasswordHash(params.Password, user.Password))
	require.Equal(t, params.Name, user.Name)
	require.Equal(t, params.Email, user.Email)
	require.Equal(t, params.MaxFileSize, user.MaxFileSize)
	require.Equal(t, params.MaxUserSize, user.MaxUserSize)
	require.Equal(t, params.MaxTTL, user.MaxTTL)
	require.Equal(t, params.IsAdmin, user.IsAdmin)
	require.Equal(t, userOK.Theme, user.Theme) // Theme is NOT copied by UpdateUser (managed via PATCH /me)

	params = *userOK
	params.Password = "newpassword"

	user = *userOK
	err = UpdateUser(&user, &params)
	require.NoError(t, err)
	require.NotEqual(t, userOK.Password, user.Password)

	require.Equal(t, params.Name, user.Name)

	params = *userOK
	params.Password = "short"

	user = *userOK
	err = UpdateUser(&user, &params)
	require.Error(t, err)

	params = *userOK
	user = *userOK
	user.Provider = ProviderGoogle
	user.Password = ""
	params.Password = "abcdefgh"

	err = UpdateUser(&user, &params)
	require.NoError(t, err)
	require.Equal(t, "", user.Password)

}
