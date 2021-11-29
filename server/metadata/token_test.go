package metadata

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/root-gg/plik/server/common"
)

func TestBackend_CreateToken(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	user := common.NewUser(common.ProviderLocal, "user")
	createUser(t, b, user)
	require.NotZero(t, user.ID, "missing user id")
	require.NotZero(t, user.CreatedAt, "missing creation date")

	token := user.NewToken()
	err := b.CreateToken(token)
	require.NoError(t, err, "create token error")

	err = b.CreateToken(token)
	require.Error(t, err, "create token error expected")
}

func TestBackend_GetToken(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	token, err := b.GetToken("token")
	require.NoError(t, err, "get token error")
	require.Nil(t, token, "non nil token")

	user := common.NewUser(common.ProviderLocal, "user")
	token = user.NewToken()
	token.Comment = "blah"
	createUser(t, b, user)

	tokenResult, err := b.GetToken(token.Token)
	require.NoError(t, err, "get token error")
	require.NotNil(t, tokenResult, "nil token")
	require.Equal(t, token.Token, tokenResult.Token, "invalid token token")
	require.Equal(t, token.UserID, tokenResult.UserID, "invalid token user id")
	require.Equal(t, token.Comment, tokenResult.Comment, "invalid token user id")
}

func TestBackend_GetTokens(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	user := common.NewUser(common.ProviderLocal, "user")
	for i := 0; i < 10; i++ {
		user.NewToken()
	}
	createUser(t, b, user)

	tokens, cursor, err := b.GetTokens(user.ID, common.NewPagingQuery().WithLimit(5))
	require.NoError(t, err, "get tokens error")
	require.Len(t, tokens, 5, "invalid token count")
	require.NotNil(t, cursor, "invalid nil cursor")
}

func TestBackend_DeleteToken(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	deleted, err := b.DeleteToken("token")
	require.NoError(t, err, "get token error")
	require.False(t, deleted, "invalid deleted value")

	user := common.NewUser(common.ProviderLocal, "user")
	token := user.NewToken()
	createUser(t, b, user)

	tokenResult, err := b.GetToken(token.Token)
	require.NoError(t, err, "get token error")
	require.NotNil(t, tokenResult, "nil token")

	deleted, err = b.DeleteToken(token.Token)
	require.NoError(t, err, "delete token error")
	require.True(t, deleted, "invalid deleted value")

	tokenResult, err = b.GetToken(token.Token)
	require.NoError(t, err, "get token error")
	require.Nil(t, tokenResult, "non nil token")
}

func TestBackend_CountUserTokens(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	user := common.NewUser(common.ProviderLocal, "user")
	for i := 0; i < 10; i++ {
		user.NewToken()
	}
	createUser(t, b, user)

	count, err := b.CountUserTokens(user.ID)
	require.NoError(t, err, "get tokens error")
	require.Equal(t, 10, count, "invalid token count")
}

func TestBackend_ForEachToken(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	user := common.NewUser(common.ProviderLocal, "user")
	token := user.NewToken()
	token.Comment = "foo bar"
	createUser(t, b, user)

	count := 0
	f := func(token *common.Token) error {
		count++
		require.Equal(t, "foo bar", token.Comment, "invalid token comment")
		return nil
	}
	err := b.ForEachToken(f)
	require.NoError(t, err, "for each token error : %s", err)
	require.Equal(t, 1, count, "invalid token count")

	f = func(token *common.Token) error {
		return fmt.Errorf("expected")
	}
	err = b.ForEachToken(f)
	require.Errorf(t, err, "expected")
}
