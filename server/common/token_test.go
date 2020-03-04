package common

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewToken(t *testing.T) {
	token := NewToken()
	require.NotNil(t, token, "invalid token")
	require.NotZero(t, token.Token, "missing token")
}
