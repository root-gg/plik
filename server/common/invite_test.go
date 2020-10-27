package common

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewInvite(t *testing.T) {
	issuer := &User{ID: "user"}
	invite, err := NewInvite(issuer, 30*24*time.Hour)
	require.NoError(t, err)
	require.NotEmpty(t, invite.ID)
	require.Equal(t, issuer.ID, *invite.Issuer)
	require.False(t, invite.HasExpired())
}

func TestNewInviteNoIssuer(t *testing.T) {
	invite, err := NewInvite(nil, 30*24*time.Hour)
	require.NoError(t, err)
	require.NotEmpty(t, invite.ID)
	require.Nil(t, invite.Issuer)
	require.False(t, invite.HasExpired())
}

func TestNewInviteNoTTL(t *testing.T) {
	issuer := &User{ID: "user"}
	invite, err := NewInvite(issuer, -1)
	require.NoError(t, err)
	require.NotEmpty(t, invite.ID)
	require.Equal(t, issuer.ID, *invite.Issuer)
	require.Nil(t, invite.ExpireAt)
	require.False(t, invite.HasExpired())
}

func TestInvite_HasExpired(t *testing.T) {
	issuer := &User{ID: "user"}
	invite, err := NewInvite(issuer, -1)
	require.NoError(t, err)
	require.False(t, invite.HasExpired())

	invite, err = NewInvite(issuer, 0)
	require.NoError(t, err)
	require.False(t, invite.HasExpired())

	invite, err = NewInvite(issuer, time.Hour)
	require.NoError(t, err)
	require.False(t, invite.HasExpired())

	deadline := time.Now().Add(-time.Hour)
	invite.ExpireAt = &deadline
	require.True(t, invite.HasExpired())
}

func TestInvite_PrepareInsert(t *testing.T) {
	config := NewConfiguration()
	invite, err := NewInvite(NewUser(ProviderLocal, "user"), 0)
	require.NoError(t, err)
	require.NoError(t, invite.PrepareInsert(config))

	invite.Email = "plik@root.gg"
	require.NoError(t, invite.PrepareInsert(config))

	invite.Email = "foo bar"
	RequireError(t, invite.PrepareInsert(config), "invalid email")
}

func TestInvite_String(t *testing.T) {
	issuer := NewUser(ProviderLocal, "user")
	invite, err := NewInvite(issuer, time.Hour)
	require.NoError(t, err)

	require.Contains(t, invite.String(), invite.ID)
	require.Contains(t, invite.String(), issuer.ID)
	require.Contains(t, invite.String(), "expire in")

	newDeadline := invite.ExpireAt.Add(-10 * time.Hour)
	invite.ExpireAt = &newDeadline
	require.Contains(t, invite.String(), "is expired")
}

func TestInvite_GetURL(t *testing.T) {
	issuer := NewUser(ProviderLocal, "user")
	invite, err := NewInvite(issuer, time.Hour)
	invite.Email = "plik@root.gg"
	require.NoError(t, err)

	config := NewConfiguration()
	require.Contains(t, invite.GetURL(config), "/#/register")
	require.Contains(t, invite.GetURL(config), "invite="+invite.ID)
	require.Contains(t, invite.GetURL(config), "invite="+invite.ID)
	require.Contains(t, invite.GetURL(config), "email="+invite.Email)
}
