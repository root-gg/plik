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
	for range 10 {
		user.NewToken()
	}
	createUser(t, b, user)

	tokens, cursor, err := b.GetTokens(user.ID, "", common.NewPagingQuery().WithLimit(5))
	require.NoError(t, err, "get tokens error")
	require.Len(t, tokens, 5, "invalid token count")
	require.NotNil(t, cursor, "invalid nil cursor")
}

func TestBackend_GetTokens_StatsAndSortByUsageSize(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	user := common.NewUser(common.ProviderLocal, "user")
	currentToken := user.NewToken()
	currentToken.Comment = "current"
	lifetimeToken := user.NewToken()
	lifetimeToken.Comment = "ever"
	createUser(t, b, user)

	currentUpload := common.NewUpload()
	currentUpload.User = user.ID
	currentUpload.Token = currentToken.Token
	currentFile := currentUpload.NewFile()
	currentFile.Status = common.FileUploaded
	currentFile.Size = 100
	createUpload(t, b, currentUpload)

	lifetimeUpload := common.NewUpload()
	lifetimeUpload.User = user.ID
	lifetimeUpload.Token = lifetimeToken.Token
	lifetimeFile := lifetimeUpload.NewFile()
	lifetimeFile.Status = common.FileUploaded
	lifetimeFile.Size = 500
	createUpload(t, b, lifetimeUpload)
	err := b.RemoveUpload(lifetimeUpload.ID)
	require.NoError(t, err, "remove upload error")

	tokens, _, err := b.GetTokens(user.ID, StatsSortSize, common.NewPagingQuery().WithLimit(10))
	require.NoError(t, err, "get tokens by current size error")
	require.Len(t, tokens, 2, "invalid token count")
	require.Equal(t, currentToken.Token, tokens[0].Token, "invalid current-size sort order")
	require.NotNil(t, tokens[0].Stats, "missing token stats")
	require.NotNil(t, tokens[0].Stats.Usage, "missing token usage")
	require.Equal(t, int64(100), tokens[0].Stats.Usage.Current.TotalSize, "invalid current token size")
	require.NotNil(t, tokens[0].Stats.Usage.LastUploadAt, "missing token last upload date")

	tokens, _, err = b.GetTokens(user.ID, StatsSortLifetimeSize, common.NewPagingQuery().WithLimit(10))
	require.NoError(t, err, "get tokens by lifetime size error")
	require.Len(t, tokens, 2, "invalid token count")
	require.Equal(t, lifetimeToken.Token, tokens[0].Token, "invalid lifetime-size sort order")
	require.NotNil(t, tokens[0].Stats, "missing token stats")
	require.NotNil(t, tokens[0].Stats.Usage, "missing token usage")
	require.Equal(t, int64(500), tokens[0].Stats.Usage.Lifetime.TotalSize, "invalid lifetime token size")
	require.NotNil(t, tokens[0].Stats.Usage.LastUploadAt, "missing token last upload date")
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
	for range 10 {
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
