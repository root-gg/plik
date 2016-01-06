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

	"bytes"
	"github.com/root-gg/plik/server/Godeps/_workspace/src/github.com/boltdb/bolt"
	"github.com/root-gg/plik/server/Godeps/_workspace/src/github.com/root-gg/juliet"
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
			return fmt.Errorf("Unable to create metadata bucket : %s", err)
		}

		_, err = tx.CreateBucketIfNotExists([]byte("expired"))
		if err != nil {
			return fmt.Errorf("Unable to create expired bucket : %s", err)
		}

		if common.Config.TokenAuthentication {
			_, err := tx.CreateBucketIfNotExists([]byte("tokens"))
			if err != nil {
				return fmt.Errorf("Unable to create token bucket : %s", err)
			}
			_, err = tx.CreateBucketIfNotExists([]byte("tokens_index"))
			if err != nil {
				return fmt.Errorf("Unable to create token bucket : %s", err)
			}
		}

		return nil
	})
	if err != nil {
		log.Fatalf("Unable to create Bolt buckets : %s", err)
	}

	return
}

// Create implementation for Bolt Metadata Backend
func (bmb *MetadataBackend) Create(ctx *juliet.Context, upload *common.Upload) (err error) {
	log := common.GetLogger(ctx)

	// Serialize metadata to json
	j, err := json.Marshal(upload)
	if err != nil {
		err = log.EWarningf("Unable to serialize metadata to json : %s", err)
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
			key := make([]byte, 8)
			expiredTs := upload.Creation + int64(upload.TTL)
			binary.BigEndian.PutUint64(key, uint64(expiredTs))
			key = append(key, []byte(upload.ID)...)

			err := bucketExpired.Put(key, []byte{})
			if err != nil {
				return fmt.Errorf("Unable to save expire index : %s", err)
			}
		}

		// Index auth token in the token index bucket
		if upload.AuthToken != "" {
			bucketToken := tx.Bucket([]byte("tokens_index"))
			if bucketToken == nil {
				return fmt.Errorf("Unable to get token index Bolt bucket")
			}

			// Key is the auth token ( 36 bytes )
			// concatenated with the upload date reversed ( 8 bytes )
			// concatenated with the upload id ( 16 bytes )
			key := make([]byte, 60)
			copy(key[:36], []byte(upload.AuthToken))
			binary.BigEndian.PutUint64(key[36:44], ^uint64(0)-uint64(upload.Creation))
			copy(key[44:], []byte(upload.ID))

			err := bucketToken.Put(key, []byte{})
			if err != nil {
				return fmt.Errorf("Unable to save expire index : %s", err)
			}
		}

		return nil
	})
	if err != nil {
		return
	}

	log.Infof("Metadata file successfully saved")
	return
}

// Get implementation for Bolt Metadata Backend
func (bmb *MetadataBackend) Get(ctx *juliet.Context, id string) (upload *common.Upload, err error) {
	log := common.GetLogger(ctx)
	var b []byte

	// Get json metadata from Bolt database
	err = bmb.db.View(func(tx *bolt.Tx) error {
		bucketMetadata := tx.Bucket([]byte("metadata"))
		if bucketMetadata == nil {
			return fmt.Errorf("Unable to get metadata Bolt bucket")
		}

		b = bucketMetadata.Get([]byte(id))
		if b == nil || len(b) == 0 {
			return fmt.Errorf("Unable to get upload metadata from Bolt bucket")
		}

		return nil
	})
	if err != nil {
		return
	}

	// Unserialize metadata from json
	upload = new(common.Upload)
	if err = json.Unmarshal(b, upload); err != nil {
		err = log.EWarningf("Unable to unserialize metadata from json \"%s\" : %s", string(b), err)
		return
	}

	return
}

// AddOrUpdateFile implementation for Bolt Metadata Backend
func (bmb *MetadataBackend) AddOrUpdateFile(ctx *juliet.Context, upload *common.Upload, file *common.File) (err error) {
	log := common.GetLogger(ctx)

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
			return log.EWarningf("Unable to unserialize metadata from json \"%s\" : %s", string(b), err)
		}

		// Add file to upload
		upload.Files[file.ID] = file

		// Serialize metadata to json
		j, err := json.Marshal(upload)
		if err != nil {
			return log.EWarningf("Unable to serialize metadata to json : %s", err)
		}

		// Update Bolt database
		return bucketMetadata.Put([]byte(upload.ID), j)
	})
	if err != nil {
		return
	}

	log.Infof("Metadata file successfully updated")
	return
}

// RemoveFile implementation for Bolt Metadata Backend
func (bmb *MetadataBackend) RemoveFile(ctx *juliet.Context, upload *common.Upload, file *common.File) (err error) {
	log := common.GetLogger(ctx)

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
			return log.EWarningf("Unable to unserialize metadata from json \"%s\" : %s", string(j), err)
		}

		// Remove file from upload
		_, ok := upload.Files[file.ID]
		if ok {
			delete(upload.Files, file.ID)

			// Serialize metadata to json
			j, err := json.Marshal(upload)
			if err != nil {
				return log.EWarningf("Unable to serialize metadata to json : %s", err)
			}

			// Update bolt database
			err = bucketMetadata.Put([]byte(upload.ID), j)
			return err
		}

		return err
	})
	if err != nil {
		return
	}

	log.Infof("Metadata successfully updated")
	return nil
}

// Remove implementation for Bolt Metadata Backend
func (bmb *MetadataBackend) Remove(ctx *juliet.Context, upload *common.Upload) (err error) {
	log := common.GetLogger(ctx)

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
			key := make([]byte, 8)
			expiredTs := upload.Creation + int64(upload.TTL)
			binary.BigEndian.PutUint64(key, uint64(expiredTs))
			key = append(key, []byte(upload.ID)...)

			err := bucketExpired.Delete(key)
			if err != nil {
				return err
			}
		}

		// Clean the auth token index bucket
		if upload.AuthToken != "" {
			bucketToken := tx.Bucket([]byte("tokens_index"))
			if bucketToken == nil {
				return fmt.Errorf("Unable to get token index Bolt bucket")
			}

			// Key is the auth token ( 36 bytes )
			// concatenated with the upload date ( 8 bytes )
			// concatenated with the upload id ( 8 bytes )
			key := make([]byte, 60)
			copy(key[:36], []byte(upload.AuthToken))
			binary.BigEndian.PutUint64(key[36:44], ^uint64(0)-uint64(upload.Creation))
			copy(key[44:], []byte(upload.ID))

			err := bucketToken.Delete(key)
			if err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return
	}

	log.Infof("Metadata successfully removed")
	return
}

// GetUploadsToRemove implementation for Bolt Metadata Backend
func (bmb *MetadataBackend) GetUploadsToRemove(ctx *juliet.Context) (ids []string, err error) {
	//log := common.GetLogger(ctx)

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
	if err != nil {
		return
	}

	return
}

// GetUploadsWithToken implementation for Bolt Metadata Backend
func (bmb *MetadataBackend) GetUploadsWithToken(ctx *juliet.Context, token string) (ids []string, err error) {
	log := common.GetLogger(ctx)

	err = bmb.db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket([]byte("tokens_index")).Cursor()

		tokenBytes := []byte(token)
		k, _ := c.Seek(tokenBytes)
		for k != nil && bytes.HasPrefix(k, tokenBytes) {
			// Extract upload id from key ( 16 last bytes )
			ids = append(ids, string(k[44:]))

			// Scan the bucket forward
			k, _ = c.Next()
		}

		return nil
	})
	if err != nil {
		log.Warningf("daffouk : %v", err)
		return
	}

	return
}

// SaveToken implementation for Bolt Metadata Backend
func (bmb *MetadataBackend) SaveToken(ctx *juliet.Context, token *common.Token) (err error) {
	log := common.GetLogger(ctx)

	// Serialize token to json
	j, err := json.Marshal(token)
	if err != nil {
		err = log.EWarningf("Unable to serialize token to json : %s", err)
		return
	}

	// Save json metadata to Bolt database
	err = bmb.db.Update(func(tx *bolt.Tx) error {
		bucketMetadata := tx.Bucket([]byte("tokens"))
		if bucketMetadata == nil {
			return fmt.Errorf("Unable to get tokens Bolt bucket")
		}

		err := bucketMetadata.Put([]byte(token.Token), j)
		if err != nil {
			return fmt.Errorf("Unable save token : %s", err)
		}

		return nil
	})
	if err != nil {
		return
	}
	return
}

// GetToken implementation for Bolt Metadata Backend
func (bmb *MetadataBackend) GetToken(ctx *juliet.Context, token string) (t *common.Token, err error) {
	log := common.GetLogger(ctx)
	var b []byte

	// Get json token from Bolt database
	err = bmb.db.View(func(tx *bolt.Tx) error {
		bucketMetadata := tx.Bucket([]byte("tokens"))
		if bucketMetadata == nil {
			return fmt.Errorf("Unable to get tokens Bolt bucket")
		}

		b = bucketMetadata.Get([]byte(token))
		if b == nil || len(b) == 0 {
			return fmt.Errorf("Unable to get token from Bolt bucket")
		}

		return nil
	})
	if err != nil {
		err = log.EWarningf("Unable to save token : %s", err)
		return
	}

	// Unserialize token from json
	t = common.NewToken()
	if err = json.Unmarshal(b, t); err != nil {
		return
	}

	return
}

// ValidateToken implementation for Bolt Metadata Backend
func (bmb *MetadataBackend) ValidateToken(ctx *juliet.Context, token string) (ok bool, err error) {
	// Get json token from Bolt database
	err = bmb.db.View(func(tx *bolt.Tx) error {
		bucketMetadata := tx.Bucket([]byte("tokens"))
		if bucketMetadata == nil {
			return fmt.Errorf("Unable to get tokens Bolt bucket")
		}

		b := bucketMetadata.Get([]byte(token))
		if b == nil || len(b) == 0 {
			return nil
		}

		ok = true
		return nil
	})
	if err != nil {
		return
	}

	return
}

// RevokeToken implementation for Bolt Metadata Backend
func (bmb *MetadataBackend) RevokeToken(ctx *juliet.Context, token string) (err error) {
	// Remove upload from bolt database
	err = bmb.db.Update(func(tx *bolt.Tx) error {
		bucketMetadata := tx.Bucket([]byte("tokens"))
		err := bucketMetadata.Delete([]byte(token))
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return
	}

	return
}
