package common

import (
	"fmt"
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
	user.Name = "user"
	user.Login = "user"
	user.Email = "user@root.gg"
	fmt.Println(user.String())
}
