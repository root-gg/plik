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
	"testing"
	"time"

	"github.com/boltdb/bolt"
	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
	"github.com/stretchr/testify/require"
)

func TestGetUploadsToRemoveNoUploadBucket(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	backend, cleanup := newBackend(t)
	defer cleanup()

	err := backend.db.Update(func(tx *bolt.Tx) error {
		return tx.DeleteBucket([]byte("uploads"))
	})
	require.NoError(t, err, "unable to remove uploads bucket")

	_, err = backend.GetUploadsToRemove(ctx)
	require.Error(t, err, "missing error")
}

func TestGetUploadsToRemove(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	backend, cleanup := newBackend(t)
	defer cleanup()

	upload := common.NewUpload()
	upload.Create()
	upload.TTL = 1
	upload.Creation = time.Now().Add(-10 * time.Minute).Unix()

	err := backend.Upsert(ctx, upload)
	require.NoError(t, err, "create error")

	upload2 := common.NewUpload()
	upload2.Create()
	upload.TTL = 0
	upload.Creation = time.Now().Add(-10 * time.Minute).Unix()

	err = backend.Upsert(ctx, upload2)
	require.NoError(t, err, "create error")

	upload3 := common.NewUpload()
	upload3.Create()
	upload.TTL = 86400
	upload.Creation = time.Now().Add(-10 * time.Minute).Unix()

	err = backend.Upsert(ctx, upload3)
	require.NoError(t, err, "create error")

	ids, err := backend.GetUploadsToRemove(ctx)
	require.NoError(t, err, "get upload to remove error")
	require.Len(t, ids, 1, "invalid uploads to remove count")
}

func TestGetServerStatisticsNoUploadBucke(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	backend, cleanup := newBackend(t)
	defer cleanup()

	err := backend.db.Update(func(tx *bolt.Tx) error {
		return tx.DeleteBucket([]byte("uploads"))
	})
	require.NoError(t, err, "unable to remove uploads bucket")

	_, err = backend.GetServerStatistics(ctx)
	require.Error(t, err, "missing error")
}

func TestGetServerStatistics(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	backend, cleanup := newBackend(t)
	defer cleanup()

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
