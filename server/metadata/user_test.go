package metadata

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/root-gg/plik/server/common"
)

func createUser(t *testing.T, b *Backend, user *common.User) {
	err := b.CreateUser(user)
	require.NoError(t, err, "create user error", err)
}

func TestBackend_CreateUser(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	user := &common.User{ID: "user"}
	createUser(t, b, user)
	require.NotZero(t, user.ID, "missing user id")
	require.NotZero(t, user.CreatedAt, "missing creation date")
}

func TestBackend_CreateUser_Exist(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	user := &common.User{ID: "user"}
	createUser(t, b, user)

	err := b.CreateUser(user)
	require.Error(t, err, "create user error")
}

func TestBackend_UpdateUser(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

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
	defer shutdownTestMetadataBackend(b)

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
	defer shutdownTestMetadataBackend(b)

	user := &common.User{ID: "user"}
	createUser(t, b, user)

	result, err := b.GetUser(user.ID)
	require.NoError(t, err, "get user error")
	require.Equal(t, user.ID, result.ID, "invalid user id")
}

func TestBackend_GetUser_NotFound(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	user, err := b.GetUser("not found")
	require.NoError(t, err, "get user error")
	require.Nil(t, user, "user not nil")
}

func TestBackend_GetUsers(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	for i := range 5 {
		user := common.NewUser(common.ProviderLocal, fmt.Sprintf("user_%d", i))
		createUser(t, b, user)
	}

	for i := range 5 {
		user := common.NewUser(common.ProviderGoogle, fmt.Sprintf("user_%d", i))
		createUser(t, b, user)
	}

	users, cursor, err := b.GetUsers("", nil, false, "", common.NewPagingQuery().WithLimit(100))
	require.NoError(t, err, "get user error")
	require.NotNil(t, cursor, "invalid nil cursor")
	require.Len(t, users, 10, "invalid user length")

	users, cursor, err = b.GetUsers(common.ProviderGoogle, nil, false, "", common.NewPagingQuery().WithLimit(100))
	require.NoError(t, err, "get user error")
	require.NotNil(t, cursor, "invalid nil cursor")
	require.Len(t, users, 5, "invalid user length")

	users, cursor, err = b.GetUsers("", nil, false, "", nil)
	require.Error(t, err, "get user error expected")
}

func TestBackend_GetUsers_AdminFilter(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	for i := range 3 {
		user := common.NewUser(common.ProviderLocal, fmt.Sprintf("admin_%d", i))
		user.IsAdmin = true
		createUser(t, b, user)
	}

	for i := range 5 {
		user := common.NewUser(common.ProviderLocal, fmt.Sprintf("user_%d", i))
		createUser(t, b, user)
	}

	// Filter admins only
	adminTrue := true
	users, _, err := b.GetUsers("", &adminTrue, false, "", common.NewPagingQuery().WithLimit(100))
	require.NoError(t, err, "get admin users error")
	require.Len(t, users, 3, "invalid admin user count")
	for _, u := range users {
		require.True(t, u.IsAdmin, "expected admin user")
	}

	// Filter non-admins only
	adminFalse := false
	users, _, err = b.GetUsers("", &adminFalse, false, "", common.NewPagingQuery().WithLimit(100))
	require.NoError(t, err, "get non-admin users error")
	require.Len(t, users, 5, "invalid non-admin user count")
	for _, u := range users {
		require.False(t, u.IsAdmin, "expected non-admin user")
	}

	// No filter (nil) returns all
	users, _, err = b.GetUsers("", nil, false, "", common.NewPagingQuery().WithLimit(100))
	require.NoError(t, err, "get all users error")
	require.Len(t, users, 8, "invalid total user count")
}

func TestBackend_GetUsers_SortByUsageSize(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	currentUser := common.NewUser(common.ProviderLocal, "current_size")
	createUser(t, b, currentUser)
	lifetimeUser := common.NewUser(common.ProviderLocal, "lifetime_size")
	createUser(t, b, lifetimeUser)

	currentUpload := common.NewUpload()
	currentUpload.User = currentUser.ID
	currentFile := currentUpload.NewFile()
	currentFile.Status = common.FileUploaded
	currentFile.Size = 100
	createUpload(t, b, currentUpload)

	lifetimeUpload := common.NewUpload()
	lifetimeUpload.User = lifetimeUser.ID
	lifetimeFile := lifetimeUpload.NewFile()
	lifetimeFile.Status = common.FileUploaded
	lifetimeFile.Size = 500
	createUpload(t, b, lifetimeUpload)
	err := b.RemoveUpload(lifetimeUpload.ID)
	require.NoError(t, err, "remove upload error")

	users, _, err := b.GetUsers("", nil, false, StatsSortSize, common.NewPagingQuery().WithLimit(10))
	require.NoError(t, err, "get users by current size error")
	require.Len(t, users, 2, "invalid user count")
	require.Equal(t, currentUser.ID, users[0].ID, "invalid current-size sort order")
	require.NotNil(t, users[0].Stats, "missing user stats")
	require.Equal(t, int64(100), users[0].Stats.TotalSize, "invalid user current size")

	users, _, err = b.GetUsers("", nil, false, StatsSortLifetimeSize, common.NewPagingQuery().WithLimit(10))
	require.NoError(t, err, "get users by lifetime size error")
	require.Len(t, users, 2, "invalid user count")
	require.Equal(t, lifetimeUser.ID, users[0].ID, "invalid lifetime-size sort order")
	require.NotNil(t, users[0].Stats, "missing user stats")
	require.Equal(t, int64(500), users[0].Stats.Usage.Lifetime.TotalSize, "invalid user lifetime size")
}

// TestBackend_GetUsers_SortByDownloadedBytes mirrors
// TestBackend_GetUsers_SortByUsageSize for sort=downloadedBytes: it must order
// users by their usage_stats.downloaded_bytes (via the same join pattern as
// current/lifetime size), populated here through a real RecordFileDownload so
// the join reads the same column the live download path writes.
func TestBackend_GetUsers_SortByDownloadedBytes(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	heavyUser := common.NewUser(common.ProviderLocal, "heavy_downloader")
	createUser(t, b, heavyUser)
	lightUser := common.NewUser(common.ProviderLocal, "light_downloader")
	createUser(t, b, lightUser)
	idleUser := common.NewUser(common.ProviderLocal, "idle_downloader")
	createUser(t, b, idleUser)

	heavyUpload := &common.Upload{User: heavyUser.ID}
	heavyFile := heavyUpload.NewFile()
	heavyFile.Status = common.FileUploaded
	heavyFile.Size = 5000
	createUpload(t, b, heavyUpload)
	require.NoError(t, b.RecordFileDownload(heavyUpload, heavyFile, 5000, true))

	lightUpload := &common.Upload{User: lightUser.ID}
	lightFile := lightUpload.NewFile()
	lightFile.Status = common.FileUploaded
	lightFile.Size = 100
	createUpload(t, b, lightUpload)
	require.NoError(t, b.RecordFileDownload(lightUpload, lightFile, 100, true))

	users, _, err := b.GetUsers("", nil, false, StatsSortDownloadedBytes, common.NewPagingQuery().WithLimit(10))
	require.NoError(t, err, "get users by downloaded bytes error")
	require.Len(t, users, 3, "invalid user count")
	require.Equal(t, heavyUser.ID, users[0].ID, "invalid downloaded-bytes sort order")
	require.NotNil(t, users[0].Stats, "missing user stats")
	require.Equal(t, int64(5000), users[0].Stats.Usage.Downloads.Bytes, "invalid user downloaded bytes")
	require.Equal(t, lightUser.ID, users[1].ID, "invalid downloaded-bytes sort order")

	// idleUser never downloaded anything. CreateUser seeds every user with its
	// own usage_stats row (so there is no missing-row edge here); downloaded_bytes
	// on that row simply stays at its default 0.
	require.Equal(t, idleUser.ID, users[2].ID, "idle user with zero downloaded bytes must sort last (desc)")
}

func TestBackend_SearchUsers(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	// Create test users with varied logins / names / emails
	alice := common.NewUser(common.ProviderLocal, "alice")
	alice.Login = "alice"
	alice.Name = "Alice Wonderland"
	alice.Email = "alice@example.com"
	createUser(t, b, alice)

	bob := common.NewUser(common.ProviderLocal, "bob")
	bob.Login = "bob"
	bob.Name = "Bob Builder"
	bob.Email = "bob@example.com"
	createUser(t, b, bob)

	charlie := common.NewUser(common.ProviderGoogle, "charlie")
	charlie.Login = "charlie"
	charlie.Name = "Charlie Chaplin"
	charlie.Email = "charlie@example.com"
	charlie.IsAdmin = true
	createUser(t, b, charlie)

	// Search by login prefix
	users, err := b.SearchUsers("ali", "", nil, 5)
	require.NoError(t, err)
	require.Len(t, users, 1)
	require.Equal(t, "alice", users[0].Login)

	// Search matches name
	users, err = b.SearchUsers("Builder", "", nil, 5)
	require.NoError(t, err)
	require.Len(t, users, 1)
	require.Equal(t, "bob", users[0].Login)

	// Search matches email
	users, err = b.SearchUsers("charlie@", "", nil, 5)
	require.NoError(t, err)
	require.Len(t, users, 1)
	require.Equal(t, "charlie", users[0].Login)

	// Search matches multiple — results sorted by login
	users, err = b.SearchUsers("example.com", "", nil, 10)
	require.NoError(t, err)
	require.Len(t, users, 3)
	require.Equal(t, "alice", users[0].Login)
	require.Equal(t, "bob", users[1].Login)
	require.Equal(t, "charlie", users[2].Login)

	// No results
	users, err = b.SearchUsers("zzz_no_match", "", nil, 5)
	require.NoError(t, err)
	require.Len(t, users, 0)

	// Empty query returns error
	_, err = b.SearchUsers("", "", nil, 5)
	require.Error(t, err)
}

func TestBackend_SearchUsers_WithFilters(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	alice := common.NewUser(common.ProviderLocal, "alice")
	alice.Login = "alice"
	createUser(t, b, alice)

	bob := common.NewUser(common.ProviderGoogle, "bob")
	bob.Login = "bob"
	bob.IsAdmin = true
	createUser(t, b, bob)

	// Both have "l" or "b" in their ID — search for common substring "local:" or just use broad search
	// Search all, filter by provider
	users, err := b.SearchUsers("alice", "local", nil, 5)
	require.NoError(t, err)
	require.Len(t, users, 1)
	require.Equal(t, "alice", users[0].Login)

	users, err = b.SearchUsers("alice", "google", nil, 5)
	require.NoError(t, err)
	require.Len(t, users, 0)

	// Search all, filter by admin
	adminTrue := true
	users, err = b.SearchUsers("bob", "", &adminTrue, 5)
	require.NoError(t, err)
	require.Len(t, users, 1)
	require.Equal(t, "bob", users[0].Login)

	adminFalse := false
	users, err = b.SearchUsers("bob", "", &adminFalse, 5)
	require.NoError(t, err)
	require.Len(t, users, 0)
}

func TestBackend_SearchUsers_Limit(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	for i := range 10 {
		user := common.NewUser(common.ProviderLocal, fmt.Sprintf("user_%02d", i))
		user.Login = fmt.Sprintf("user_%02d", i)
		createUser(t, b, user)
	}

	// Limit 3
	users, err := b.SearchUsers("user_", "", nil, 3)
	require.NoError(t, err)
	require.Len(t, users, 3)

	// Limit > 20 gets capped to 20
	users, err = b.SearchUsers("user_", "", nil, 50)
	require.NoError(t, err)
	require.Len(t, users, 10) // only 10 exist
}

func TestBackend_DeleteUser(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

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
	defer shutdownTestMetadataBackend(b)

	user := common.NewUser(common.ProviderLocal, "user")
	token := user.NewToken()
	createUser(t, b, user)

	for range 2 {
		upload := &common.Upload{}
		upload.User = user.ID
		createUpload(t, b, upload)
	}

	for range 5 {
		upload := &common.Upload{}
		upload.User = user.ID
		upload.Token = token.Token
		createUpload(t, b, upload)
	}

	for range 10 {
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
	defer shutdownTestMetadataBackend(b)

	user := common.NewUser(common.ProviderLocal, "user")
	token := user.NewToken()
	createUser(t, b, user)

	// Create uploads with files in various states
	var tokenUploadIDs []string
	for range 5 {
		upload := &common.Upload{}
		upload.User = user.ID
		upload.Token = token.Token
		createUpload(t, b, upload)
		tokenUploadIDs = append(tokenUploadIDs, upload.ID)
		// Add a file with "uploaded" status (has data on disk)
		file := upload.NewFile()
		file.Status = common.FileUploaded
		err := b.CreateFile(file)
		require.NoError(t, err)
		// Add a file with "missing" status (no data on disk)
		file2 := upload.NewFile()
		file2.Status = common.FileMissing
		err = b.CreateFile(file2)
		require.NoError(t, err)
	}

	var otherUploadIDs []string
	for range 2 {
		upload := &common.Upload{}
		upload.User = user.ID
		createUpload(t, b, upload)
		otherUploadIDs = append(otherUploadIDs, upload.ID)
	}

	for range 10 {
		upload := &common.Upload{}
		upload.User = "blah"
		createUpload(t, b, upload)
	}

	// Delete only token uploads
	deleted, err := b.RemoveUserUploads(user.ID, token.Token)
	require.NoError(t, err, "remove user uploads error")
	require.Equal(t, 5, deleted, "invalid upload count")

	// Verify file statuses were batch-updated
	for _, id := range tokenUploadIDs {
		upload, err := b.GetUpload(id)
		require.NoError(t, err)
		require.Nil(t, upload, "upload should be soft-deleted")

		// Check file statuses via unscoped query
		var files []*common.File
		err = b.db.Where(&common.File{UploadID: id}).Find(&files).Error
		require.NoError(t, err)
		require.Len(t, files, 2, "expected 2 files per upload")
		for _, f := range files {
			if f.Status == common.FileRemoved || f.Status == common.FileDeleted {
				continue
			}
			t.Errorf("unexpected file status %q for file %s", f.Status, f.ID)
		}
	}

	// Other user uploads should be unaffected
	deleted, err = b.RemoveUserUploads(user.ID, "")
	require.NoError(t, err, "remove user uploads error")
	require.Equal(t, 2, deleted, "invalid upload count")

	// No-op when nothing left
	deleted, err = b.RemoveUserUploads(user.ID, "")
	require.NoError(t, err, "remove user uploads error")
	require.Equal(t, 0, deleted, "should be zero")
}

func TestBackend_CountUsers(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	user := common.NewUser(common.ProviderLocal, "user")
	createUser(t, b, user)

	count, err := b.CountUsers("", nil)
	require.NoError(t, err, "count users error")
	require.Equal(t, int64(1), count, "invalid user count")
}
