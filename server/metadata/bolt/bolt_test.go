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
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/boltdb/bolt"
	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
	"github.com/stretchr/testify/require"
)

func newBackend(t *testing.T) (backend *Backend, cleanup func()) {
	dir, err := ioutil.TempDir("", "pliktest")
	require.NoError(t, err, "unable to create temp directory")

	backend, err = NewBackend(&Config{Path: dir + "/plik.db"})
	require.NoError(t, err, "unable to create bolt metadata backend")
	cleanup = func() {
		err := os.RemoveAll(dir)
		if err != nil {
			fmt.Println(err)
		}
	}

	return backend, cleanup
}

func TestNewBoltMetadataBackendInvalidPath(t *testing.T) {
	_, err := NewBackend(&Config{Path: string([]byte{0})})
	require.Error(t, err, "able to create bolt metadata backend")
}

func TestNewBoltMetadataBackend(t *testing.T) {
	backend, cleanup := newBackend(t)
	defer cleanup()
	require.NotNil(t, backend, "invalid nil backend")
}

func TestUpsertNoUpload(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	backend, cleanup := newBackend(t)
	defer cleanup()

	err := backend.Upsert(ctx, nil)
	require.Error(t, err, "missing error")
	require.Contains(t, err.Error(), "Missing upload", "invalid error")
}

func TestUpsertNoUploadBucket(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	backend, cleanup := newBackend(t)
	defer cleanup()

	err := backend.db.Update(func(tx *bolt.Tx) error {
		return tx.DeleteBucket([]byte("uploads"))
	})
	require.NoError(t, err, "unable to remove uploads bucket")

	upload := common.NewUpload()
	upload.Create()

	err = backend.Upsert(ctx, upload)
	require.Error(t, err, "missing error")
}

func TestUpsert(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	backend, cleanup := newBackend(t)
	defer cleanup()

	upload := common.NewUpload()
	upload.Create()

	err := backend.Upsert(ctx, upload)
	require.NoError(t, err, "create error")
}

func TestUpsertUser(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	backend, cleanup := newBackend(t)
	defer cleanup()

	upload := common.NewUpload()
	upload.User = "user"
	upload.Create()

	err := backend.Upsert(ctx, upload)
	require.NoError(t, err, "create error")
}

func TestUpsertTTL(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	backend, cleanup := newBackend(t)
	defer cleanup()

	upload := common.NewUpload()
	upload.TTL = 86400
	upload.Create()

	err := backend.Upsert(ctx, upload)
	require.NoError(t, err, "create error")
}

func TestGetNoUpload(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	backend, cleanup := newBackend(t)
	defer cleanup()

	_, err := backend.Get(ctx, "")
	require.Error(t, err, "missing error")
	require.Contains(t, err.Error(), "Missing upload", "invalid error")
}

func TestGetNoUploadBucket(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	backend, cleanup := newBackend(t)
	defer cleanup()

	err := backend.db.Update(func(tx *bolt.Tx) error {
		return tx.DeleteBucket([]byte("uploads"))
	})
	require.NoError(t, err, "unable to remove uploads bucket")

	_, err = backend.Get(ctx, "id")
	require.Error(t, err, "missing error")
}

func TestGetNotFound(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	backend, cleanup := newBackend(t)
	defer cleanup()

	_, err := backend.Get(ctx, "id")
	require.Error(t, err, "missing error")
	require.Contains(t, err.Error(), "Unable to get upload metadata from Bolt bucket", "invalid error")
}

func TestGetInvalidJSON(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	backend, cleanup := newBackend(t)
	defer cleanup()

	upload := common.NewUpload()
	upload.Create()

	err := backend.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("uploads"))
		if bucket == nil {
			return errors.New("unable to get upload bucket")
		}

		err := bucket.Put([]byte(upload.ID), []byte("invalid_json_value"))
		if err != nil {
			return errors.New("unable to put value")
		}

		return nil
	})
	require.NoError(t, err, "bolt error")

	_, err = backend.Get(ctx, upload.ID)
	require.Error(t, err, "missing error")
	require.Contains(t, err.Error(), "Unable to unserialize metadata from json", "invalid error")
}

func TestGet(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	backend, cleanup := newBackend(t)
	defer cleanup()

	upload := common.NewUpload()
	upload.Create()

	err := backend.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("uploads"))
		if bucket == nil {
			return errors.New("unable to get upload bucket")
		}

		jsonValue, err := json.Marshal(upload)
		if err != nil {
			return err
		}

		err = bucket.Put([]byte(upload.ID), jsonValue)
		if err != nil {
			return errors.New("unable to put value")
		}

		return nil
	})
	require.NoError(t, err, "bolt error")

	_, err = backend.Get(ctx, upload.ID)
	require.NoError(t, err, "unable to get upload")
}

func TestRemoveNoUpload(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	backend, cleanup := newBackend(t)
	defer cleanup()

	err := backend.Remove(ctx, nil)
	require.Error(t, err, "missing error")
	require.Contains(t, err.Error(), "Missing upload", "invalid error")
}

func TestRemoveNoUploadBucket(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	backend, cleanup := newBackend(t)
	defer cleanup()

	err := backend.db.Update(func(tx *bolt.Tx) error {
		return tx.DeleteBucket([]byte("uploads"))
	})
	require.NoError(t, err, "unable to remove uploads bucket")

	upload := common.NewUpload()
	upload.Create()

	err = backend.Remove(ctx, upload)
	require.Error(t, err, "missing error")
}

func TestRemoveNotFound(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	backend, cleanup := newBackend(t)
	defer cleanup()

	upload := common.NewUpload()
	upload.Create()

	err := backend.Remove(ctx, upload)
	require.NoError(t, err, "remove error")
}

func TestRemove(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	backend, cleanup := newBackend(t)
	defer cleanup()

	upload := common.NewUpload()
	upload.Create()

	err := backend.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("uploads"))
		if bucket == nil {
			return errors.New("unable to get upload bucket")
		}

		jsonValue, err := json.Marshal(upload)
		if err != nil {
			return err
		}

		err = bucket.Put([]byte(upload.ID), jsonValue)
		if err != nil {
			return errors.New("unable to put value")
		}

		return nil
	})
	require.NoError(t, err, "bolt error")

	err = backend.Remove(ctx, upload)
	require.NoError(t, err, "remove error")
}

func TestRemoveUploadWithUser(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	backend, cleanup := newBackend(t)
	defer cleanup()

	upload := common.NewUpload()
	upload.User = "user"
	upload.Create()

	err := backend.Remove(ctx, upload)
	require.NoError(t, err, "remove error")
}

func TestRemoveUploadWithTTL(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	backend, cleanup := newBackend(t)
	defer cleanup()

	upload := common.NewUpload()
	upload.TTL = 86400
	upload.Create()

	err := backend.Remove(ctx, upload)
	require.NoError(t, err, "remove error")
}
