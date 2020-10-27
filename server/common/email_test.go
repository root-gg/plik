package common

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsValidEmail(t *testing.T) {
	require.True(t, IsValidEmail("plik@root.gg"))
	require.False(t, IsValidEmail("plik"))
	require.False(t, IsValidEmail("@root.gg"))
	require.False(t, IsValidEmail(""))
}

func TestConfiguration_CheckEmail(t *testing.T) {
	config := NewConfiguration()

	require.NoError(t, config.CheckEmail("plik@root.gg"))
	RequireError(t, config.CheckEmail(""), "invalid email")
	RequireError(t, config.CheckEmail("plik"), "invalid email")

	config.EmailValidDomains = []string{"root.gg"}
	require.NoError(t, config.CheckEmail("plik@root.gg"))
	RequireError(t, config.CheckEmail("plik@gmail.com"), "invalid email domain")

	config.GoogleValidDomains = []string{"gmail.com"}
	require.NoError(t, config.CheckEmail("plik@root.gg"))
	require.NoError(t, config.CheckEmail("plik@gmail.com"))
	RequireError(t, config.CheckEmail("plik@plik.com"), "invalid email domain")
}
