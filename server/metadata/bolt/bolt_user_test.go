/**

    Plik upload server

The MIT License (MIT)

Copyright (c) <2015>
	- Mathieu Bodjikian <mathieu@bodjikian.fr>
	- Charles-Antoine Mathieu <skatkatt@root.gg>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
**/

package bolt

import (
	"errors"
	"testing"

	"github.com/boltdb/bolt"
	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
	"github.com/stretchr/testify/require"
)

func TestSaveUserNoUser(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	backend, cleanup := newBackend(t)
	defer cleanup()

	err := backend.SaveUser(ctx, nil)
	require.Error(t, err, "missing error")
	require.Contains(t, err.Error(), "Missing user", "invalid error")
}

func TestSaveUserNoUsersBucket(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	backend, cleanup := newBackend(t)
	defer cleanup()

	err := backend.db.Update(func(tx *bolt.Tx) error {
		return tx.DeleteBucket([]byte("users"))
	})
	require.NoError(t, err, "unable to remove uploads bucket")

	user := common.NewUser()
	err = backend.SaveUser(ctx, user)
	require.Error(t, err, "missing error")
}

func TestSaveUser(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	backend, cleanup := newBackend(t)
	defer cleanup()

	user := common.NewUser()
	user.ID = "user"

	err := backend.SaveUser(ctx, user)
	require.NoError(t, err, "save user error")
}

func TestSaveUserToken(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	backend, cleanup := newBackend(t)
	defer cleanup()

	user := common.NewUser()
	user.ID = "user"
	user.NewToken()

	err := backend.SaveUser(ctx, user)
	require.NoError(t, err, "save user error")
}

func TestSaveUserTokenUpdate(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	backend, cleanup := newBackend(t)
	defer cleanup()

	user := common.NewUser()
	user.ID = "user"
	user.NewToken()

	err := backend.SaveUser(ctx, user)
	require.NoError(t, err, "save user error")

	user.Tokens = nil
	user.NewToken()

	err = backend.SaveUser(ctx, user)
	require.NoError(t, err, "update error")
}

func TestGetUserNoUser(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	backend, cleanup := newBackend(t)
	defer cleanup()

	_, err := backend.GetUser(ctx, "", "")
	require.Error(t, err, "missing error")
	require.Contains(t, err.Error(), "Missing user", "invalid error")
}

func TestGetUserNoUserBucket(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	backend, cleanup := newBackend(t)
	defer cleanup()

	err := backend.db.Update(func(tx *bolt.Tx) error {
		return tx.DeleteBucket([]byte("users"))
	})
	require.NoError(t, err, "unable to remove uploads bucket")

	_, err = backend.GetUser(ctx, "id", "")
	require.Error(t, err, "missing error")
}

func TestGetUserNotFound(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	backend, cleanup := newBackend(t)
	defer cleanup()

	user, err := backend.GetUser(ctx, "id", "")
	require.NoError(t, err, "missing error")
	require.Nil(t, user, "invalid not nil user")
}

func TestGeUserInvalidJSON(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	backend, cleanup := newBackend(t)
	defer cleanup()

	user := common.NewUser()
	user.ID = "user"

	err := backend.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("users"))
		if bucket == nil {
			return errors.New("unable to get upload bucket")
		}

		err := bucket.Put([]byte(user.ID), []byte("invalid_json_value"))
		if err != nil {
			return errors.New("unable to put value")
		}

		return nil
	})
	require.NoError(t, err, "bolt error")

	_, err = backend.GetUser(ctx, user.ID, "")
	require.Error(t, err, "missing error")
}

func TestGetUser(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	backend, cleanup := newBackend(t)
	defer cleanup()

	user := common.NewUser()
	user.ID = "user"

	err := backend.SaveUser(ctx, user)
	require.NoError(t, err, "save user error")

	_, err = backend.GetUser(ctx, user.ID, "")
	require.NoError(t, err, "unable to get user")
}

func TestGetUserByToken(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	backend, cleanup := newBackend(t)
	defer cleanup()

	user := common.NewUser()
	user.ID = "user"
	token := user.NewToken()

	err := backend.SaveUser(ctx, user)
	require.NoError(t, err, "save user error")

	u, err := backend.GetUser(ctx, "", token.Token)
	require.NoError(t, err, "unable to get user")
	require.NotNil(t, u, "invalid nil user")
	require.Equal(t, user.ID, u.ID, "invalid user")
}

func TestRemoveUserNoUpload(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	backend, cleanup := newBackend(t)
	defer cleanup()

	err := backend.RemoveUser(ctx, nil)
	require.Error(t, err, "missing error")
	require.Contains(t, err.Error(), "Missing user", "invalid error")
}

func TestRemoveUserNoUsersBucket(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	backend, cleanup := newBackend(t)
	defer cleanup()

	err := backend.db.Update(func(tx *bolt.Tx) error {
		return tx.DeleteBucket([]byte("users"))
	})
	require.NoError(t, err, "unable to remove uploads bucket")

	user := common.NewUser()
	err = backend.RemoveUser(ctx, user)
	require.Error(t, err, "missing error")
}

func TestRemoveUser(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	backend, cleanup := newBackend(t)
	defer cleanup()

	user := common.NewUser()
	user.ID = "user"
	err := backend.SaveUser(ctx, user)
	require.NoError(t, err, "save user error")

	err = backend.RemoveUser(ctx, user)
	require.NoError(t, err, "remove user error")

	u, err := backend.GetUser(ctx, user.ID, "")
	require.NoError(t, err, "get user error")
	require.Nil(t, u, "non nil removed user")
}

func TestRemoveUserNotFound(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	backend, cleanup := newBackend(t)
	defer cleanup()

	user := common.NewUser()
	user.ID = "user"

	err := backend.RemoveUser(ctx, user)
	require.NoError(t, err, "remove user error")
}

func TestGetUserUploadsNoUpload(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	backend, cleanup := newBackend(t)
	defer cleanup()

	_, err := backend.GetUserUploads(ctx, nil, nil)
	require.Error(t, err, "missing error")
	require.Contains(t, err.Error(), "Missing user", "invalid error")
}

func TestGetUserUploadsNoUploadsBucket(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	backend, cleanup := newBackend(t)
	defer cleanup()

	err := backend.db.Update(func(tx *bolt.Tx) error {
		return tx.DeleteBucket([]byte("uploads"))
	})
	require.NoError(t, err, "unable to remove uploads bucket")

	user := common.NewUser()
	_, err = backend.GetUserUploads(ctx, user, nil)
	require.Error(t, err, "missing error")
}

func TestGetUserUploads(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	backend, cleanup := newBackend(t)
	defer cleanup()

	user := common.NewUser()
	user.ID = "user"

	upload := common.NewUpload()
	upload.User = user.ID
	upload.Create()

	err := backend.Upsert(ctx, upload)
	require.NoError(t, err, "create error")

	upload2 := common.NewUpload()
	upload2.User = "another_user"
	upload2.Create()

	err = backend.Upsert(ctx, upload2)
	require.NoError(t, err, "create error")

	uploads, err := backend.GetUserUploads(ctx, user, nil)
	require.NoError(t, err, "get user error")
	require.Equal(t, 1, len(uploads), "invalid upload count")
	require.Equal(t, upload.ID, uploads[0], "invalid upload id")
}

func TestGetUserUploadsToken(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	backend, cleanup := newBackend(t)
	defer cleanup()

	user := common.NewUser()
	user.ID = "user"
	token := user.NewToken()

	upload := common.NewUpload()
	upload.User = user.ID
	upload.Create()

	err := backend.Upsert(ctx, upload)
	require.NoError(t, err, "create error")

	upload2 := common.NewUpload()
	upload2.User = user.ID
	upload2.Token = token.Token
	upload2.Create()

	err = backend.Upsert(ctx, upload2)
	require.NoError(t, err, "create error")

	upload3 := common.NewUpload()
	upload3.User = "another_user"
	upload3.Create()

	err = backend.Upsert(ctx, upload3)
	require.NoError(t, err, "create error")

	uploads, err := backend.GetUserUploads(ctx, user, token)
	require.NoError(t, err, "get user error")
	require.Equal(t, 1, len(uploads), "invalid upload count")
	require.Equal(t, upload2.ID, uploads[0], "invalid upload id")
}

func TestGetUsersNoUsersBucket(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	backend, cleanup := newBackend(t)
	defer cleanup()

	err := backend.db.Update(func(tx *bolt.Tx) error {
		return tx.DeleteBucket([]byte("users"))
	})
	require.NoError(t, err, "unable to remove uploads bucket")

	_, err = backend.GetUsers(ctx)
	require.Error(t, err, "missing error")
}

func TestGetUsers(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	backend, cleanup := newBackend(t)
	defer cleanup()

	user1 := common.NewUser()
	user1.ID = "ovh:test"
	user1.NewToken()

	err := backend.SaveUser(ctx, user1)
	require.NoError(t, err, "save user error")

	user2 := common.NewUser()
	user2.ID = "google:test"
	user2.NewToken()

	err = backend.SaveUser(ctx, user2)
	require.NoError(t, err, "save user error")

	users, err := backend.GetUsers(ctx)
	require.NoError(t, err, "get users error")
	require.Equal(t, 2, len(users), "invalid users count")
}

func TestGetUserStatistics(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	backend, cleanup := newBackend(t)
	defer cleanup()

	user := common.NewUser()
	user.ID = "user"

	upload := common.NewUpload()
	upload.User = user.ID
	upload.Create()
	file1 := upload.NewFile()
	file1.CurrentSize = 1
	file2 := upload.NewFile()
	file2.CurrentSize = 2

	err := backend.Upsert(ctx, upload)
	require.NoError(t, err, "create error")

	upload2 := common.NewUpload()
	upload2.User = user.ID
	upload2.Create()
	file3 := upload2.NewFile()
	file3.CurrentSize = 3

	err = backend.Upsert(ctx, upload2)
	require.NoError(t, err, "create error")

	upload3 := common.NewUpload()
	upload3.Create()
	file4 := upload3.NewFile()
	file4.CurrentSize = 3000

	err = backend.Upsert(ctx, upload3)
	require.NoError(t, err, "create error")

	stats, err := backend.GetUserStatistics(ctx, user, nil)
	require.NoError(t, err, "get users error")
	require.Equal(t, 2, stats.Uploads, "invalid uploads count")
	require.Equal(t, 3, stats.Files, "invalid files count")
	require.Equal(t, int64(6), stats.TotalSize, "invalid file size")
}

func TestGetUserStatisticsToken(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	backend, cleanup := newBackend(t)
	defer cleanup()

	user := common.NewUser()
	user.ID = "user"
	token := user.NewToken()

	upload := common.NewUpload()
	upload.User = user.ID
	upload.Create()
	file1 := upload.NewFile()
	file1.CurrentSize = 1
	file2 := upload.NewFile()
	file2.CurrentSize = 2

	err := backend.Upsert(ctx, upload)
	require.NoError(t, err, "create error")

	upload2 := common.NewUpload()
	upload2.User = user.ID
	upload2.Token = token.Token
	upload2.Create()
	file3 := upload2.NewFile()
	file3.CurrentSize = 3

	err = backend.Upsert(ctx, upload2)
	require.NoError(t, err, "create error")

	upload3 := common.NewUpload()
	upload3.Create()
	file4 := upload3.NewFile()
	file4.CurrentSize = 3000

	err = backend.Upsert(ctx, upload3)
	require.NoError(t, err, "create error")

	stats, err := backend.GetUserStatistics(ctx, user, token)
	require.NoError(t, err, "get users error")
	require.Equal(t, 1, stats.Uploads, "invalid uploads count")
	require.Equal(t, 1, stats.Files, "invalid files count")
	require.Equal(t, int64(3), stats.TotalSize, "invalid file size")
}
