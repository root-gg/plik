package metadata

import (
	"fmt"
	"testing"
	"time"

	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/stretchr/testify/require"

	"github.com/root-gg/plik/server/common"
)

func TestBackend_CreateInvite(t *testing.T) {
	b := newTestMetadataBackend()

	invite, err := common.NewInvite(nil, 30*24*time.Hour)
	require.NoError(t, err)
	invite.Admin = true
	err = b.CreateInvite(invite)
	require.NoError(t, err, "create invite error")
	require.True(t, invite.Admin)

	err = b.CreateInvite(invite)
	require.Error(t, err, "create invite error expected")
}

func TestBackend_CreateInviteWithIssuer(t *testing.T) {
	b := newTestMetadataBackend()

	user := common.NewUser(common.ProviderLocal, "admin")
	createUser(t, b, user)
	require.NotZero(t, user.ID, "missing user id")
	require.NotZero(t, user.CreatedAt, "missing creation date")

	invite, err := user.NewInvite(30 * 24 * time.Hour)
	err = b.CreateInvite(invite)
	require.NoError(t, err, "create invite error")

	err = b.CreateInvite(invite)
	require.Error(t, err, "create invite error expected")
}

func TestBackend_GetInvite(t *testing.T) {
	b := newTestMetadataBackend()

	user := common.NewUser(common.ProviderLocal, "admin")
	createUser(t, b, user)
	require.NotZero(t, user.ID, "missing user id")
	require.NotZero(t, user.CreatedAt, "missing creation date")

	invite, err := user.NewInvite(30 * 24 * time.Hour)
	err = b.CreateInvite(invite)
	require.NoError(t, err, "create invite error")

	inviteResult, err := b.GetInvite(invite.ID)
	require.NoError(t, err, "get invite error")
	require.NotNil(t, inviteResult, "nil invite")
	require.Equal(t, invite.ID, inviteResult.ID, "invalid invite invite")
	require.Equal(t, invite.Issuer, inviteResult.Issuer, "invalid invite user id")
	require.Equal(t, invite.ExpireAt.Unix(), inviteResult.ExpireAt.Unix(), "invalid invite user id")
}

func TestBackend_GetUserInvites(t *testing.T) {
	b := newTestMetadataBackend()

	user := common.NewUser(common.ProviderLocal, "user")
	createUser(t, b, user)

	for i := 0; i < 10; i++ {
		invite, err := user.NewInvite(time.Hour)
		require.NoError(t, err, "new invite error")

		err = b.CreateInvite(invite)
		require.NoError(t, err, "create invite error")
	}

	user2 := common.NewUser(common.ProviderLocal, "user2")
	createUser(t, b, user2)

	for i := 0; i < 5; i++ {
		invite, err := user2.NewInvite(time.Hour)
		require.NoError(t, err, "new invite error")

		err = b.CreateInvite(invite)
		require.NoError(t, err, "create invite error")
	}

	for i := 0; i < 2; i++ {
		invite, err := common.NewInvite(nil, time.Hour)
		require.NoError(t, err, "new invite error")

		err = b.CreateInvite(invite)
		require.NoError(t, err, "create invite error")
	}

	invites, cursor, err := b.GetUserInvites(user.ID, common.NewPagingQuery().WithLimit(100))
	require.NoError(t, err, "get invites error")
	require.Len(t, invites, 10, "invalid invite count")
	require.NotNil(t, cursor, "invalid nil cursor")
	for _, invite := range invites {
		require.Equal(t, user.ID, *invite.Issuer, "invalid invite issuer")
	}

	invites, cursor, err = b.GetUserInvites(user2.ID, common.NewPagingQuery().WithLimit(100))
	require.NoError(t, err, "get invites error")
	require.Len(t, invites, 5, "invalid invite count")
	require.NotNil(t, cursor, "invalid nil cursor")
	for _, invite := range invites {
		require.Equal(t, user2.ID, *invite.Issuer, "invalid invite issuer")
	}

	invites, cursor, err = b.GetUserInvites("", common.NewPagingQuery().WithLimit(100))
	require.NoError(t, err, "get invites error")
	require.Len(t, invites, 2, "invalid invite count")
	require.NotNil(t, cursor, "invalid nil cursor")
	for _, invite := range invites {
		require.Nil(t, invite.Issuer, "invalid invite issuer")
	}

	invites, cursor, err = b.GetUserInvites("*", common.NewPagingQuery().WithLimit(100))
	require.NoError(t, err, "get invites error")
	require.Len(t, invites, 17, "invalid invite count")
	require.NotNil(t, cursor, "invalid nil cursor")
}

func TestBackend_DeleteInvite(t *testing.T) {
	b := newTestMetadataBackend()

	deleted, err := b.DeleteInvite("invite")
	require.NoError(t, err, "get invite error")
	require.False(t, deleted, "invalid deleted value")

	user := common.NewUser(common.ProviderLocal, "user")
	createUser(t, b, user)

	invite, err := user.NewInvite(time.Hour)
	require.NoError(t, err)

	err = b.CreateInvite(invite)
	require.NoError(t, err, "create invite error")

	inviteResult, err := b.GetInvite(invite.ID)
	require.NoError(t, err, "get invite error")
	require.NotNil(t, inviteResult, "nil invite")

	deleted, err = b.DeleteInvite(invite.ID)
	require.NoError(t, err, "delete invite error")
	require.True(t, deleted, "invalid deleted value")

	inviteResult, err = b.GetInvite(invite.ID)
	require.NoError(t, err, "get invite error")
	require.Nil(t, inviteResult, "non nil invite")
}

func TestBackend_CountUserInvites(t *testing.T) {
	b := newTestMetadataBackend()

	user := common.NewUser(common.ProviderLocal, "user")
	createUser(t, b, user)

	for i := 0; i < 10; i++ {
		invite, err := user.NewInvite(time.Hour)
		require.NoError(t, err, "new invite error")

		err = b.CreateInvite(invite)
		require.NoError(t, err, "create invite error")
	}

	user2 := common.NewUser(common.ProviderLocal, "user2")
	createUser(t, b, user2)

	for i := 0; i < 5; i++ {
		invite, err := user2.NewInvite(time.Hour)
		require.NoError(t, err, "new invite error")

		err = b.CreateInvite(invite)
		require.NoError(t, err, "create invite error")
	}

	for i := 0; i < 2; i++ {
		invite, err := common.NewInvite(nil, time.Hour)
		require.NoError(t, err, "new invite error")

		err = b.CreateInvite(invite)
		require.NoError(t, err, "create invite error")
	}

	count, err := b.CountUserInvites(user.ID)
	require.NoError(t, err, "get invites error")
	require.Equal(t, 10, count, "invalid invite count")

	count, err = b.CountUserInvites(user2.ID)
	require.NoError(t, err, "get invites error")
	require.Equal(t, 5, count, "invalid invite count")

	count, err = b.CountUserInvites("")
	require.NoError(t, err, "get invites error")
	require.Equal(t, 2, count, "invalid invite count")

	count, err = b.CountUserInvites("*")
	require.NoError(t, err, "get invites error")
	require.Equal(t, 17, count, "invalid invite count")
}

func TestBackend_ForEachInvite(t *testing.T) {
	b := newTestMetadataBackend()

	user := common.NewUser(common.ProviderLocal, "user")
	createUser(t, b, user)

	invite, err := user.NewInvite(time.Hour)
	require.NoError(t, err)

	err = b.CreateInvite(invite)
	require.NoError(t, err, "create invite error")

	count := 0
	f := func(invite *common.Invite) error {
		count++
		require.False(t, invite.HasExpired(), "invalid invite expired status")
		return nil
	}
	err = b.ForEachInvites(f)
	require.NoError(t, err, "for each invite error : %s", err)
	require.Equal(t, 1, count, "invalid invite count")

	f = func(invite *common.Invite) error {
		return fmt.Errorf("expected")
	}
	err = b.ForEachInvites(f)
	require.Errorf(t, err, "expected")
}

func TestBackend_DeleteExpiredInvites(t *testing.T) {
	b := newTestMetadataBackend()

	invite1, err := common.NewInvite(nil, 0)
	require.NoError(t, err)
	err = b.CreateInvite(invite1)
	require.NoError(t, err)

	invite2, err := common.NewInvite(nil, time.Microsecond)
	require.NoError(t, err)
	err = b.CreateInvite(invite2)
	require.NoError(t, err)

	invite3, err := common.NewInvite(nil, time.Hour)
	require.NoError(t, err)
	err = b.CreateInvite(invite3)
	require.NoError(t, err)

	removed, err := b.DeleteExpiredInvites()
	require.Nil(t, err, "delete expired invite error")
	require.Equal(t, 1, removed, "removed expired invite count mismatch")
}
