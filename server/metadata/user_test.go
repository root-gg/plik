package metadata

import (
	"fmt"
	"testing"

	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/stretchr/testify/require"

	"github.com/root-gg/plik/server/common"
)

func createUser(t *testing.T, b *Backend, user *common.User) {
	err := b.CreateUser(user)
	require.NoError(t, err, "create user error", err)
}

func TestBackend_CreateUser(t *testing.T) {
	b := newTestMetadataBackend()

	user := &common.User{ID: "user"}
	createUser(t, b, user)
	require.NotZero(t, user.ID, "missing user id")
	require.NotZero(t, user.CreatedAt, "missing creation date")
}

func TestBackend_CreateUser_Exist(t *testing.T) {
	b := newTestMetadataBackend()

	user := &common.User{ID: "user"}
	createUser(t, b, user)

	err := b.CreateUser(user)
	require.Error(t, err, "create user error")
}

func TestBackend_CreateUserWithInvite(t *testing.T) {
	b := newTestMetadataBackend()

	err := b.CreateUserWithInvite(&common.User{ID: "user1"}, nil)
	require.NoError(t, err, "create user error", err)
	u1, err := b.GetUser("user1")
	require.NoError(t, err)
	require.NotNil(t, u1)

	invite, err := common.NewInvite(nil, 0)
	require.NoError(t, err)
	err = b.CreateInvite(invite)
	require.NoError(t, err)

	err = b.CreateUserWithInvite(&common.User{ID: "user2"}, invite)
	require.NoError(t, err, "create user error", err)
	u2, err := b.GetUser("user2")
	require.NoError(t, err)
	require.NotNil(t, u2)

	invite.ID = "foo"
	err = b.CreateUserWithInvite(&common.User{ID: "user3"}, invite)
	require.Error(t, err, "create user error", err)
	u3, err := b.GetUser("user3")
	require.NoError(t, err)
	require.Nil(t, u3)
}

func TestBackend_UpdateUser(t *testing.T) {
	b := newTestMetadataBackend()

	user := &common.User{ID: "user", Name: "foo"}
	createUser(t, b, user)
	require.NotZero(t, user.ID, "missing user id")
	require.NotZero(t, user.CreatedAt, "missing creation date")

	user.Name = "bar"
	err := b.UpdateUser(user)
	require.NoError(t, err, "update user error")

	result, err := b.GetUser(user.ID)
	require.NoError(t, err, "get user error")
	require.Equal(t, user.ID, result.ID, "invalid user id")
	require.Equal(t, user.Name, result.Name, "invalid user name")
}

func TestBackend_UpdateUser_NotFound(t *testing.T) {
	b := newTestMetadataBackend()

	user := &common.User{ID: "user", Name: "foo"}
	err := b.UpdateUser(user)
	require.NoError(t, err, "update user error")

	result, err := b.GetUser(user.ID)
	require.NoError(t, err, "get user error")
	require.Equal(t, user.ID, result.ID, "invalid user id")
	require.Equal(t, user.Name, result.Name, "invalid user name")
}

func TestBackend_GetUser(t *testing.T) {
	b := newTestMetadataBackend()

	user := &common.User{ID: "user"}
	createUser(t, b, user)

	result, err := b.GetUser(user.ID)
	require.NoError(t, err, "get user error")
	require.Equal(t, user.ID, result.ID, "invalid user id")
}

func TestBackend_GetUser_NotFound(t *testing.T) {
	b := newTestMetadataBackend()

	user, err := b.GetUser("not found")
	require.NoError(t, err, "get user error")
	require.Nil(t, user, "user not nil")
}

func TestBackend_GetUsers(t *testing.T) {
	b := newTestMetadataBackend()

	for i := 0; i < 5; i++ {
		user := common.NewUser(common.ProviderLocal, fmt.Sprintf("user_%d", i))
		createUser(t, b, user)
	}

	for i := 0; i < 5; i++ {
		user := common.NewUser(common.ProviderGoogle, fmt.Sprintf("user_%d", i))
		createUser(t, b, user)
	}

	users, cursor, err := b.GetUsers("", false, common.NewPagingQuery().WithLimit(100))
	require.NoError(t, err, "get user error")
	require.NotNil(t, cursor, "invalid nil cursor")
	require.Len(t, users, 10, "invalid user lenght")

	users, cursor, err = b.GetUsers(common.ProviderGoogle, false, common.NewPagingQuery().WithLimit(100))
	require.NoError(t, err, "get user error")
	require.NotNil(t, cursor, "invalid nil cursor")
	require.Len(t, users, 5, "invalid user lenght")

	users, cursor, err = b.GetUsers("", false, nil)
	require.Error(t, err, "get user error expected")
}

func TestBackend_DeleteUser(t *testing.T) {
	b := newTestMetadataBackend()

	user := &common.User{ID: "user"}

	deleted, err := b.DeleteUser(user.ID)
	require.NoError(t, err, "delete user error")
	require.False(t, deleted, "invalid deleted value")

	createUser(t, b, user)

	deleted, err = b.DeleteUser(user.ID)
	require.NoError(t, err, "delete user error")
	require.True(t, deleted, "invalid deleted value")

	user, err = b.GetUser(user.ID)
	require.NoError(t, err, "get user error")
	require.Nil(t, user, "user not nil")
}

func TestBackend_ForEachUserUploads(t *testing.T) {
	b := newTestMetadataBackend()

	user := common.NewUser(common.ProviderLocal, "user")
	token := user.NewToken()
	createUser(t, b, user)

	for i := 0; i < 2; i++ {
		upload := &common.Upload{}
		upload.User = user.ID
		createUpload(t, b, upload)
	}

	for i := 0; i < 5; i++ {
		upload := &common.Upload{}
		upload.User = user.ID
		upload.Token = token.Token
		createUpload(t, b, upload)
	}

	for i := 0; i < 10; i++ {
		upload := &common.Upload{}
		upload.User = "blah"
		createUpload(t, b, upload)
	}

	count := 0
	f := func(upload *common.Upload) error {
		require.Equal(t, user.ID, upload.User, "invalid upload user")
		count++
		return nil
	}
	err := b.ForEachUserUploads(user.ID, "", f)
	require.NoError(t, err, "for each user upload error")
	require.Equal(t, 7, count, "invalid upload count")

	count = 0
	f = func(upload *common.Upload) error {
		require.Equal(t, user.ID, upload.User, "invalid upload user")
		require.Equal(t, token.Token, upload.Token, "invalid upload token")
		count++
		return nil
	}
	err = b.ForEachUserUploads(user.ID, token.Token, f)
	require.NoError(t, err, "for each user upload error")
	require.Equal(t, 5, count, "invalid upload count")

	f = func(upload *common.Upload) error {
		return fmt.Errorf("expected")
	}
	err = b.ForEachUserUploads(user.ID, "", f)
	require.Error(t, err, "for each user upload error expected")
}

func TestBackend_DeleteUserUploads(t *testing.T) {
	b := newTestMetadataBackend()

	user := common.NewUser(common.ProviderLocal, "user")
	token := user.NewToken()
	createUser(t, b, user)

	for i := 0; i < 2; i++ {
		upload := &common.Upload{}
		upload.User = user.ID
		createUpload(t, b, upload)
	}

	for i := 0; i < 5; i++ {
		upload := &common.Upload{}
		upload.User = user.ID
		upload.Token = token.Token
		createUpload(t, b, upload)
	}

	for i := 0; i < 10; i++ {
		upload := &common.Upload{}
		upload.User = "blah"
		createUpload(t, b, upload)
	}

	deleted, err := b.DeleteUserUploads(user.ID, token.Token)
	require.NoError(t, err, "for each user upload error")
	require.Equal(t, 5, deleted, "invalid upload count")

	deleted, err = b.DeleteUserUploads(user.ID, "")
	require.NoError(t, err, "for each user upload error")
	require.Equal(t, 2, deleted, "invalid upload count")
}

func TestBackend_CountUsers(t *testing.T) {
	b := newTestMetadataBackend()

	user := common.NewUser(common.ProviderLocal, "user")
	createUser(t, b, user)

	count, err := b.CountUsers()
	require.NoError(t, err, "count users error")
	require.Equal(t, 1, count, "invalid user count")
}
