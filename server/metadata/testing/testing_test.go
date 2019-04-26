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

package testing

import (
	"errors"
	"testing"
	"time"

	"github.com/root-gg/juliet"
	"github.com/root-gg/logger"
	"github.com/root-gg/plik/server/common"
	"github.com/stretchr/testify/require"
)

func newTestingContext(config *common.Configuration) (ctx *juliet.Context) {
	ctx = juliet.NewContext()
	ctx.Set("config", config)
	ctx.Set("logger", logger.NewLogger())
	return ctx
}

func TestNewBoltMetadataBackendInvalidPath(t *testing.T) {
	NewBackend()
}

func TestUpsert(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	backend := NewBackend()

	upload := common.NewUpload()
	upload.Create()

	err := backend.Upsert(ctx, upload)
	require.NoError(t, err, "upsert error")
}

func TestUpsertError(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	backend := NewBackend()
	backend.SetError(errors.New("error"))

	upload := common.NewUpload()
	upload.Create()

	err := backend.Upsert(ctx, upload)
	require.Error(t, err, "missing error")
	require.Equal(t, "error", err.Error(), "invalid error")
}

func TestGet(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	backend := NewBackend()

	upload := common.NewUpload()
	upload.Create()

	err := backend.Upsert(ctx, upload)
	require.NoError(t, err, "upsert error")

	u, err := backend.Get(ctx, upload.ID)
	require.NoError(t, err, "get error")
	require.NotNil(t, u, "invalid upload")
}

func TestGetNotFound(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	backend := NewBackend()

	upload := common.NewUpload()
	upload.Create()

	_, err := backend.Get(ctx, upload.ID)
	require.Error(t, err, "get error")
	require.Contains(t, err.Error(), "Upload does not exists", "invalid not nil upload")
}

func TestGetError(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	backend := NewBackend()
	backend.SetError(errors.New("error"))

	upload := common.NewUpload()
	upload.Create()

	_, err := backend.Get(ctx, upload.ID)
	require.Error(t, err, "missing error")
	require.Equal(t, "error", err.Error(), "invalid error")
}

func TestRemove(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	backend := NewBackend()

	upload := common.NewUpload()
	upload.Create()

	err := backend.Upsert(ctx, upload)
	require.NoError(t, err, "upsert error")

	err = backend.Remove(ctx, upload)
	require.NoError(t, err, "remove error")

	_, err = backend.Get(ctx, upload.ID)
	require.Error(t, err, "get error")
	require.Contains(t, err.Error(), "Upload does not exists", "invalid not nil upload")
}

func TestRemoveError(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	backend := NewBackend()
	backend.SetError(errors.New("error"))

	upload := common.NewUpload()
	upload.Create()

	err := backend.Remove(ctx, upload)
	require.Error(t, err, "missing error")
	require.Equal(t, "error", err.Error(), "invalid error")
}

func TestSaveUser(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	backend := NewBackend()

	user := common.NewUser()
	user.ID = "user"

	err := backend.SaveUser(ctx, user)
	require.NoError(t, err, "save user error")
}

func TestSaveUserError(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	backend := NewBackend()
	backend.SetError(errors.New("error"))

	user := common.NewUser()
	user.ID = "user"

	err := backend.SaveUser(ctx, user)
	require.Error(t, err, "missing error")
	require.Equal(t, "error", err.Error(), "invalid error")
}

func TestGetUser(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	backend := NewBackend()

	user := common.NewUser()
	user.ID = "user"

	err := backend.SaveUser(ctx, user)
	require.NoError(t, err, "save user error")

	u, err := backend.GetUser(ctx, user.ID, "")
	require.NoError(t, err, "save user error")
	require.NotNil(t, u, "invalid nil user")
	require.Equal(t, user, u, "invalid user")

}

func TestGetUserToken(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	backend := NewBackend()

	user := common.NewUser()
	user.ID = "user"
	token := user.NewToken()

	err := backend.SaveUser(ctx, user)
	require.NoError(t, err, "save user error")

	user2 := common.NewUser()
	user.ID = "user2"

	err = backend.SaveUser(ctx, user2)
	require.NoError(t, err, "save user error")

	u, err := backend.GetUser(ctx, "", token.Token)
	require.NoError(t, err, "save user error")
	require.NotNil(t, u, "invalid nil user")
	require.Equal(t, user, u, "invalid user")
}

func TestGetUserError(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	backend := NewBackend()
	backend.SetError(errors.New("error"))

	user := common.NewUser()
	user.ID = "user"

	_, err := backend.GetUser(ctx, user.ID, "")
	require.Error(t, err, "missing error")
	require.Equal(t, "error", err.Error(), "invalid error")
}

func TestGetUserUploads(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	backend := NewBackend()

	user := common.NewUser()
	user.ID = "user"

	upload := common.NewUpload()
	upload.Create()
	upload.User = user.ID

	err := backend.Upsert(ctx, upload)
	require.NoError(t, err, "upsert error")

	upload2 := common.NewUpload()
	upload2.Create()

	err = backend.Upsert(ctx, upload2)
	require.NoError(t, err, "upsert error")

	uploads, err := backend.GetUserUploads(ctx, user, nil)
	require.NoError(t, err, "save user error")
	require.NotNil(t, uploads, "invalid nil uploads")
	require.Len(t, uploads, 1, "invalid upload count")
}

func TestGetUserUploadsToken(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	backend := NewBackend()

	user := common.NewUser()
	user.ID = "user"
	token := user.NewToken()

	upload := common.NewUpload()
	upload.Create()
	upload.User = user.ID
	upload.Token = token.Token

	err := backend.Upsert(ctx, upload)
	require.NoError(t, err, "upsert error")

	upload2 := common.NewUpload()
	upload2.Create()
	upload2.User = user.ID

	err = backend.Upsert(ctx, upload2)
	require.NoError(t, err, "upsert error")

	uploads, err := backend.GetUserUploads(ctx, user, token)
	require.NoError(t, err, "get user upload error")
	require.NotNil(t, uploads, "invalid nil uploads")
	require.Len(t, uploads, 1, "invalid upload count")
}

func TestGetUserUploadsNoUser(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	backend := NewBackend()

	user := common.NewUser()
	user.ID = "user"

	err := backend.SaveUser(ctx, user)
	require.NoError(t, err, "save user error")

	_, err = backend.GetUserUploads(ctx, nil, nil)
	require.Error(t, err, "get user uploads error")
}

func TestGetUserUploadsError(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	backend := NewBackend()
	backend.SetError(errors.New("error"))

	user := common.NewUser()
	user.ID = "user"

	_, err := backend.GetUserUploads(ctx, user, nil)
	require.Error(t, err, "missing error")
	require.Equal(t, "error", err.Error(), "invalid error")
}

func TestGetUserStatistics(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	backend := NewBackend()

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
	ctx := newTestingContext(common.NewConfiguration())
	backend := NewBackend()

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

func TestGetUserStatisticsNoUser(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	backend := NewBackend()

	user := common.NewUser()
	user.ID = "user"

	err := backend.SaveUser(ctx, user)
	require.NoError(t, err, "save user error")

	_, err = backend.GetUserStatistics(ctx, nil, nil)
	require.Error(t, err, "get user statistics error")
}

func TestGetUserStatisticsError(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	backend := NewBackend()
	backend.SetError(errors.New("error"))

	user := common.NewUser()
	user.ID = "user"

	_, err := backend.GetUserStatistics(ctx, user, nil)
	require.Error(t, err, "missing error")
	require.Equal(t, "error", err.Error(), "invalid error")
}

func TestGetUploadsToRemove(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	backend := NewBackend()

	upload := common.NewUpload()
	upload.Create()
	upload.TTL = 1
	upload.Creation = time.Now().Add(-10 * time.Minute).Unix()
	require.True(t, upload.IsExpired(), "upload should have expired")

	err := backend.Upsert(ctx, upload)
	require.NoError(t, err, "create error")

	upload2 := common.NewUpload()
	upload2.Create()
	upload2.TTL = 0
	upload2.Creation = time.Now().Add(-10 * time.Minute).Unix()
	require.False(t, upload2.IsExpired(), "upload should not have expired")

	err = backend.Upsert(ctx, upload2)
	require.NoError(t, err, "create error")

	upload3 := common.NewUpload()
	upload3.Create()
	upload3.TTL = 86400
	upload3.Creation = time.Now().Add(-10 * time.Minute).Unix()
	require.False(t, upload3.IsExpired(), "upload should not have expired")

	err = backend.Upsert(ctx, upload3)
	require.NoError(t, err, "create error")

	ids, err := backend.GetUploadsToRemove(ctx)
	require.NoError(t, err, "get upload to remove error")
	require.Len(t, ids, 1, "invalid uploads to remove count")
	require.Equal(t, upload.ID, ids[0], "invalid uploads to remove id")
}

func TestGetUploadsToRemoveError(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	backend := NewBackend()
	backend.SetError(errors.New("error"))

	_, err := backend.GetUploadsToRemove(ctx)
	require.Error(t, err, "missing error")
	require.Equal(t, "error", err.Error(), "invalid error")
}

func TestGetServerStatistics(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	backend := NewBackend()

	type pair struct {
		typ   string
		size  int64
		count int
	}

	plan := []pair{
		{"type1", 1, 1},
		{"type2", 1000, 5},
		{"type3", 1000 * 1000, 10},
		{"type4", 1000 * 1000 * 1000, 15},
	}

	for _, item := range plan {
		for i := 0; i < item.count; i++ {
			upload := common.NewUpload()
			upload.Create()
			file := upload.NewFile()
			file.Type = item.typ
			file.CurrentSize = item.size

			err := backend.Upsert(ctx, upload)
			require.NoError(t, err, "create error")
		}
	}

	stats, err := backend.GetServerStatistics(ctx)
	require.NoError(t, err, "get server statistics error")
	require.NotNil(t, stats, "invalid server statistics")
	require.Equal(t, 31, stats.Uploads, "invalid upload count")
	require.Equal(t, 31, stats.Files, "invalid files count")
	require.Equal(t, int64(15010005001), stats.TotalSize, "invalid total file size")
	require.Equal(t, 31, stats.AnonymousUploads, "invalid anonymous upload count")
	require.Equal(t, int64(15010005001), stats.AnonymousSize, "invalid anonymous total file size")
	require.Equal(t, 10, len(stats.FileTypeByCount), "invalid file type by count length")
	require.Equal(t, "type4", stats.FileTypeByCount[0].Type, "invalid file type by count type")
	require.Equal(t, 10, len(stats.FileTypeBySize), "invalid file type by size length")
	require.Equal(t, "type4", stats.FileTypeBySize[0].Type, "invalid file type by size type")
}

func TestGetServerStatisticsError(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	backend := NewBackend()
	backend.SetError(errors.New("error"))

	_, err := backend.GetServerStatistics(ctx)
	require.Error(t, err, "missing error")
	require.Equal(t, "error", err.Error(), "invalid error")
}

func TestGetUsers(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	backend := NewBackend()

	user := common.NewUser()
	user.ID = "user"

	err := backend.SaveUser(ctx, user)
	require.NoError(t, err, "save user error")

	user2 := common.NewUser()
	user2.ID = "user2"

	err = backend.SaveUser(ctx, user2)
	require.NoError(t, err, "save user error")

	ids, err := backend.GetUsers(ctx)
	require.NoError(t, err, "get server statistics error")
	require.NotNil(t, ids, "invalid nil user ids")
	require.Len(t, ids, 2, "invalid user count")
}

func TestGetUsersError(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	backend := NewBackend()
	backend.SetError(errors.New("error"))

	_, err := backend.GetUsers(ctx)
	require.Error(t, err, "missing error")
	require.Equal(t, "error", err.Error(), "invalid error")
}
