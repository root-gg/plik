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
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/root-gg/plik/server/Godeps/_workspace/src/github.com/boltdb/bolt"
	"github.com/root-gg/plik/server/common"
)

// MetadataBackend object
type MetadataBackend struct {
	Config *MetadataBackendConfig

	db *bolt.DB
}

// NewBoltMetadataBackend instantiate a new Bolt Metadata Backend
// from configuration passed as argument
func NewBoltMetadataBackend(config map[string]interface{}) (bmb *MetadataBackend) {
	bmb = new(MetadataBackend)
	bmb.Config = NewBoltMetadataBackendConfig(config)

	// Open the Bolt database
	var err error
	bmb.db, err = bolt.Open(bmb.Config.Path, 0600, &bolt.Options{Timeout: 10 * time.Second})
	if err != nil {
		log.Fatalf("Unable to open Bolt database %s : %s", bmb.Config.Path, err)
	}

	// Create Bolt buckets if needed
	err = bmb.db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("metadata"))
		if err != nil {
			return fmt.Errorf("Create bucket: %s", err)
		}

		_, err = tx.CreateBucketIfNotExists([]byte("expired"))
		if err != nil {
			return fmt.Errorf("Create bucket: %s", err)
		}
		return nil
	})
	if err != nil {
		log.Fatalf("Unable to create Bolt buckets : %s", err)
	}

	return
}

// Create implementation for Bolt Metadata Backend
func (bmb *MetadataBackend) Create(ctx *common.PlikContext, upload *common.Upload) (err error) {
	defer ctx.Finalize(err)

	// Serialize metadata to json
	j, err := json.Marshal(upload)
	if err != nil {
		err = ctx.EWarningf("Unable to serialize metadata to json : %s", err)
		return
	}

	// Save json metadata to Bolt database
	err = bmb.db.Update(func(tx *bolt.Tx) error {
		bucketMetadata := tx.Bucket([]byte("metadata"))
		if bucketMetadata == nil {
			return fmt.Errorf("Unable to get metadata Bolt bucket")
		}

		err := bucketMetadata.Put([]byte(upload.ID), j)
		if err != nil {
			return fmt.Errorf("Unable save metadata : %s", err)
		}

		// Index expire date in the expired bucket
		if upload.TTL > 0 {
			bucketExpired := tx.Bucket([]byte("expired"))
			if bucketExpired == nil {
				return fmt.Errorf("Unable to get expired Bolt bucket")
			}

			// Key is the expire timestamp ( 8 bytes )
			// concatenated with the upload id ( 16 bytes )
			// Upload id is stored in the key to ensure uniqueness
			ba := make([]byte, 8)
			expiredTs := upload.Creation + int64(upload.TTL)
			binary.BigEndian.PutUint64(ba, uint64(expiredTs))
			ba = append(ba, []byte(upload.ID)...)

			err := bucketExpired.Put(ba, []byte{})
			if err != nil {
				return fmt.Errorf("Unable to save expire index : %s", err)
			}
		}

		return nil
	})
	if err != nil {
		err = ctx.EWarningf("Unable to save upload metadata : %s", err)
		return
	}

	ctx.Infof("Metadata file successfully saved")
	return
}

// Get implementation for Bolt Metadata Backend
func (bmb *MetadataBackend) Get(ctx *common.PlikContext, id string) (upload *common.Upload, err error) {
	defer ctx.Finalize(err)

	// Get json metadata from Bolt database
	err = bmb.db.View(func(tx *bolt.Tx) error {
		bucketMetadata := tx.Bucket([]byte("metadata"))
		if bucketMetadata == nil {
			return fmt.Errorf("Unable to get metadata Bolt bucket")
		}

		b := bucketMetadata.Get([]byte(id))
		if b == nil || len(b) == 0 {
			return fmt.Errorf("Unable to get upload metadata from Bolt bucket")
		}

		// Unserialize metadata from json
		upload = new(common.Upload)
		if err = json.Unmarshal(b, upload); err != nil {
			return ctx.EWarningf("Unable to unserialize metadata from json \"%s\" : %s", string(b), err)
		}

		return nil
	})
	if err != nil {
		err = ctx.EWarningf("Unable to save upload metadata : %s", err)
		return
	}

	return
}

// AddOrUpdateFile implementation for Bolt Metadata Backend
func (bmb *MetadataBackend) AddOrUpdateFile(ctx *common.PlikContext, upload *common.Upload, file *common.File) (err error) {
	defer ctx.Finalize(err)

	// Update json metadata to Bolt database
	err = bmb.db.Update(func(tx *bolt.Tx) error {
		bucketMetadata := tx.Bucket([]byte("metadata"))
		if bucketMetadata == nil {
			return fmt.Errorf("Unable to get metadata Bolt bucket")
		}

		// Get json
		b := bucketMetadata.Get([]byte(upload.ID))
		if b == nil || len(b) == 0 {
			return fmt.Errorf("Unable to get upload metadata from Bolt bucket")
		}

		// Unserialize metadata from json
		upload := new(common.Upload)
		if err = json.Unmarshal(b, upload); err != nil {
			return ctx.EWarningf("Unable to unserialize metadata from json \"%s\" : %s", string(b), err)
		}

		// Add file to upload
		upload.Files[file.ID] = file

		// Serialize metadata to json
		j, err := json.Marshal(upload)
		if err != nil {
			return ctx.EWarningf("Unable to serialize metadata to json : %s", err)
		}

		// Update Bolt database
		return bucketMetadata.Put([]byte(upload.ID), j)
	})
	if err != nil {
		err = ctx.EWarningf("Unable to save upload metadata : %s", err)
		return
	}

	ctx.Infof("Metadata file successfully updated")
	return
}

// RemoveFile implementation for Bolt Metadata Backend
func (bmb *MetadataBackend) RemoveFile(ctx *common.PlikContext, upload *common.Upload, file *common.File) (err error) {
	defer ctx.Finalize(err)

	// Update json metadata to Bolt database
	err = bmb.db.Update(func(tx *bolt.Tx) error {
		bucketMetadata := tx.Bucket([]byte("metadata"))
		if bucketMetadata == nil {
			return fmt.Errorf("Unable to get metadata Bolt bucket")
		}

		b := bucketMetadata.Get([]byte(upload.ID))
		if b == nil {
			return fmt.Errorf("Unable to get upload metadata from Bolt bucket")
		}

		// Unserialize metadata from json
		var j []byte
		upload = new(common.Upload)
		if err = json.Unmarshal(b, upload); err != nil {
			return ctx.EWarningf("Unable to unserialize metadata from json \"%s\" : %s", string(j), err)
		}

		// Remove file from upload
		_, ok := upload.Files[file.ID]
		if ok {
			delete(upload.Files, file.ID)

			// Serialize metadata to json
			j, err := json.Marshal(upload)
			if err != nil {
				return ctx.EWarningf("Unable to serialize metadata to json : %s", err)
			}

			// Update bolt database
			err = bucketMetadata.Put([]byte(upload.ID), j)
			return err
		}

		return err
	})
	if err != nil {
		err = ctx.EWarningf("Unable to save upload metadata : %s", err)
		return
	}

	ctx.Infof("Metadata successfully updated")
	return nil
}

// Remove implementation for Bolt Metadata Backend
func (bmb *MetadataBackend) Remove(ctx *common.PlikContext, upload *common.Upload) (err error) {

	// Remove upload from bolt database
	err = bmb.db.Update(func(tx *bolt.Tx) error {
		bucketMetadata := tx.Bucket([]byte("metadata"))
		err := bucketMetadata.Delete([]byte(upload.ID))
		if err != nil {
			return err
		}

		// Clean the expired index bucket
		if upload.TTL > 0 {
			bucketExpired := tx.Bucket([]byte("expired"))
			if bucketExpired == nil {
				return fmt.Errorf("Unable to get expired Bolt bucket")
			}

			// Key is the expire timestamps ( 8 bytes )
			// concatenated with the upload id ( 16 bytes )
			// Upload id is stored in the key to ensure uniqueness
			ba := make([]byte, 8)
			expiredTs := upload.Creation + int64(upload.TTL)
			binary.BigEndian.PutUint64(ba, uint64(expiredTs))
			ba = append(ba, []byte(upload.ID)...)

			err := bucketExpired.Delete(ba)
			if err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		err = ctx.EWarningf("Unable to remove upload metadata : %s", err)
		return
	}

	ctx.Infof("Metadata successfully removed")
	return
}

// GetUploadsToRemove implementation for Bolt Metadata Backend
func (bmb *MetadataBackend) GetUploadsToRemove(ctx *common.PlikContext) (ids []string, err error) {

	err = bmb.db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket([]byte("expired")).Cursor()

		// Create seek key at current timestamp + 1
		ba := make([]byte, 8)
		binary.BigEndian.PutUint64(ba, uint64(time.Now().Unix()+1))

		// Seek just after the seek key
		// All uploads above the cursor are expired
		c.Seek(ba)
		for {
			// Scan the bucket upwards
			k, _ := c.Prev()
			if k == nil {
				break
			}

			// Extract upload id from key ( 16 last bytes )
			ids = append(ids, string(k[8:]))
		}

		return nil
	})

	return
}
