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
	"github.com/root-gg/plik/server/metadata/exporter"
	"log"
	"time"

	"bytes"
	"github.com/boltdb/bolt"
	"github.com/root-gg/juliet"
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
		_, err := tx.CreateBucketIfNotExists([]byte("uploads"))
		if err != nil {
			return fmt.Errorf("Unable to create metadata bucket : %s", err)
		}

		_, err = tx.CreateBucketIfNotExists([]byte("users"))
		if err != nil {
			return fmt.Errorf("Unable to create user bucket : %s", err)
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

	if upload == nil {
		err = log.EWarning("Unable to save upload : Missing upload")
		return
	}

	// Serialize metadata to json
	j, err := json.Marshal(upload)
	if err != nil {
		err = log.EWarningf("Unable to serialize metadata to json : %s", err)
		return
	}

	// Save json metadata to Bolt database
	err = bmb.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("uploads"))
		if bucket == nil {
			return fmt.Errorf("Unable to get metadata Bolt bucket")
		}

		err := bucket.Put([]byte(upload.ID), j)
		if err != nil {
			return fmt.Errorf("Unable save metadata : %s", err)
		}

		// User index
		if upload.User != "" {
			// User index key is build as follow :
			//  - User index prefix 2 byte ( "_u" )
			//  - The user id
			//  - The upload date reversed ( 8 bytes )
			//  - The upload id ( 16 bytes )
			// Upload id is stored in the key to ensure uniqueness
			// AuthToken is stored in the value to permit byToken filtering
			timestamp := make([]byte, 8)
			binary.BigEndian.PutUint64(timestamp, ^uint64(0)-uint64(upload.Creation))

			key := append([]byte{'_', 'u'}, []byte(upload.User)...)
			key = append(key, timestamp...)
			key = append(key, []byte(upload.ID)...)

			err := bucket.Put(key, []byte(upload.Token))
			if err != nil {
				return fmt.Errorf("Unable to save user index : %s", err)
			}
		}

		// Expire date index
		if upload.TTL > 0 {
			// Expire index is build as follow :
			//  - Expire index pplik.dbrefix 2 byte ( "_e" )
			//  - The expire timestamp ( 8 bytes )
			//  - The upload id ( 16 bytes )
			// Upload id is stored in the key to ensure uniqueness
			timestamp := make([]byte, 8)
			expiredTs := upload.Creation + int64(upload.TTL)
			binary.BigEndian.PutUint64(timestamp, uint64(expiredTs))

			key := append([]byte{'_', 'e'}, timestamp...)
			key = append(key, []byte(upload.ID)...)

			err := bucket.Put(key, []byte{})
			if err != nil {
				return fmt.Errorf("Unable to save expire index : %s", err)
			}
		}

		return nil
	})
	if err != nil {
		return
	}

	log.Infof("Upload metadata successfully saved")
	return
}

// Get implementation for Bolt Metadata Backend
func (bmb *MetadataBackend) Get(ctx *juliet.Context, id string) (upload *common.Upload, err error) {
	log := common.GetLogger(ctx)
	var b []byte

	if id == "" {
		err = log.EWarning("Unable to get upload : Missing upload id")
		return
	}

	// Get json metadata from Bolt database
	err = bmb.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("uploads"))
		if bucket == nil {
			return fmt.Errorf("Unable to get metadata Bolt bucket")
		}

		b = bucket.Get([]byte(id))
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

	if upload == nil {
		err = log.EWarning("Unable to add file : Missing upload")
		return
	}

	if file == nil {
		err = log.EWarning("Unable to add file : Missing file")
		return
	}

	// Update json metadata to Bolt database
	err = bmb.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("uploads"))
		if bucket == nil {
			return fmt.Errorf("Unable to get metadata Bolt bucket")
		}

		// Get json
		b := bucket.Get([]byte(upload.ID))
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
		return bucket.Put([]byte(upload.ID), j)
	})
	if err != nil {
		return
	}

	log.Infof("Upload metadata successfully updated")
	return
}

// RemoveFile implementation for Bolt Metadata Backend
func (bmb *MetadataBackend) RemoveFile(ctx *juliet.Context, upload *common.Upload, file *common.File) (err error) {
	log := common.GetLogger(ctx)

	if upload == nil {
		err = log.EWarning("Unable to remove file : Missing upload")
		return
	}

	if file == nil {
		err = log.EWarning("Unable to remove file : Missing file")
		return
	}

	// Update json metadata to Bolt database
	err = bmb.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("uploads"))
		if bucket == nil {
			return fmt.Errorf("Unable to get metadata Bolt bucket")
		}

		b := bucket.Get([]byte(upload.ID))
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
			err = bucket.Put([]byte(upload.ID), j)
			return err
		}

		return err
	})
	if err != nil {
		return
	}

	log.Infof("Upload metadata successfully updated")
	return nil
}

// Remove implementation for Bolt Metadata Backend
func (bmb *MetadataBackend) Remove(ctx *juliet.Context, upload *common.Upload) (err error) {
	log := common.GetLogger(ctx)

	if upload == nil {
		err = log.EWarning("Unable to remove upload : Missing upload")
		return
	}

	// Remove upload from bolt database
	err = bmb.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("uploads"))
		err := bucket.Delete([]byte(upload.ID))
		if err != nil {
			return err
		}

		// Remove upload user index
		if upload.User != "" {
			// User index key is build as follow :
			//  - User index prefix 2 byte ( "_u" )
			//  - The user id
			//  - The upload date reversed ( 8 bytes )
			//  - The upload id ( 16 bytes )
			// Upload id is stored in the key to ensure uniqueness
			// AuthToken is stored in the value to permit byToken filtering
			timestamp := make([]byte, 8)
			binary.BigEndian.PutUint64(timestamp, ^uint64(0)-uint64(upload.Creation))

			key := append([]byte{'_', 'u'}, []byte(upload.User)...)
			key = append(key, timestamp...)
			key = append(key, []byte(upload.ID)...)

			err := bucket.Delete(key)
			if err != nil {
				return fmt.Errorf("Unable to delete user index : %s", err)
			}
		}

		// Remove upload expire date index
		if upload.TTL > 0 {
			// Expire index is build as follow :
			//  - Expire index prefix 2 byte ( "_e" )
			//  - The expire timestamp ( 8 bytes )
			//  - The upload id ( 16 bytes )
			// Upload id is stored in the key to ensure uniqueness
			timestamp := make([]byte, 8)
			expiredTs := upload.Creation + int64(upload.TTL)
			binary.BigEndian.PutUint64(timestamp, uint64(expiredTs))
			key := append([]byte{'_', 'e'}, timestamp...)
			key = append(key, []byte(upload.ID)...)

			err := bucket.Delete(key)
			if err != nil {
				return fmt.Errorf("Unable to delete expire index : %s", err)
			}
		}

		return nil
	})
	if err != nil {
		return
	}

	log.Infof("Upload metadata successfully removed")
	return
}

// SaveUser implementation for Bolt Metadata Backend
func (bmb *MetadataBackend) SaveUser(ctx *juliet.Context, user *common.User) (err error) {
	log := common.GetLogger(ctx)

	if user == nil {
		err = log.EWarning("Unable to save user : Missing user")
		return
	}

	// Serialize user to json
	j, err := json.Marshal(user)
	if err != nil {
		err = log.EWarningf("Unable to serialize user to json : %s", err)
		return
	}

	// Save json user to Bolt database
	err = bmb.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("users"))
		if bucket == nil {
			return fmt.Errorf("Unable to get users Bolt bucket")
		}

		// Get current tokens
		tokens := make(map[string]*common.Token)
		b := bucket.Get([]byte(user.ID))
		if b != nil && len(b) != 0 {
			// Unserialize user from json
			u := common.NewUser()
			if err = json.Unmarshal(b, u); err != nil {
				return fmt.Errorf("Unable unserialize json user : %s", err)
			}

			for _, token := range u.Tokens {
				tokens[token.Token] = token
			}
		}

		// Save user
		err := bucket.Put([]byte(user.ID), j)
		if err != nil {
			return fmt.Errorf("Unable save user : %s", err)
		}

		// Update token index
		for _, token := range user.Tokens {
			if _, ok := tokens[token.Token]; !ok {
				// New token
				err := bucket.Put([]byte(token.Token), []byte(user.ID))
				if err != nil {
					return fmt.Errorf("Unable save new token index : %s", err)
				}
			}
			delete(tokens, token.Token)
		}

		for _, token := range tokens {
			// Deleted token
			err := bucket.Delete([]byte(token.Token))
			if err != nil {
				return fmt.Errorf("Unable delete token index : %s", err)
			}
		}

		return nil
	})
	if err != nil {
		return
	}

	log.Infof("User successfully saved")

	return
}

// GetUser implementation for Bolt Metadata Backend
func (bmb *MetadataBackend) GetUser(ctx *juliet.Context, id string, token string) (u *common.User, err error) {
	log := common.GetLogger(ctx)
	var b []byte

	if id == "" && token == "" {
		err = log.EWarning("Unable to get user : Missing user id or token")
		return
	}

	// Get json user from Bolt database
	err = bmb.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("users"))
		if bucket == nil {
			return fmt.Errorf("Unable to get users Bolt bucket")
		}

		if id == "" && token != "" {
			// token index lookup
			idBytes := bucket.Get([]byte(token))
			if idBytes == nil || len(idBytes) == 0 {
				return nil
			}
			id = string(idBytes)
		}

		b = bucket.Get([]byte(id))
		return nil
	})
	if err != nil {
		err = log.EWarningf("Unable to get user : %s", err)
		return
	}

	// User not found but no error
	if b == nil || len(b) == 0 {
		return
	}

	// Unserialize user from json
	u = common.NewUser()
	if err = json.Unmarshal(b, u); err != nil {
		return
	}

	return
}

// RemoveUser implementation for Bolt Metadata Backend
func (bmb *MetadataBackend) RemoveUser(ctx *juliet.Context, user *common.User) (err error) {
	log := common.GetLogger(ctx)

	if user == nil {
		err = log.EWarning("Unable to remove user : Missing user")
		return
	}

	// Remove user from bolt database
	err = bmb.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("users"))
		err := bucket.Delete([]byte(user.ID))
		if err != nil {
			return err
		}

		// Update token index
		for _, token := range user.Tokens {
			err := bucket.Delete([]byte(token.Token))
			if err != nil {
				return fmt.Errorf("Unable delete token index : %s", err)
			}
		}

		return nil
	})
	if err != nil {
		return
	}

	log.Infof("User successfully removed")

	return
}

// GetUserUploads implementation for Bolt Metadata Backend
func (bmb *MetadataBackend) GetUserUploads(ctx *juliet.Context, user *common.User, token *common.Token) (ids []string, err error) {
	log := common.GetLogger(ctx)

	if user == nil {
		err = log.EWarning("Unable to get user uploads : Missing user")
		return
	}

	err = bmb.db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket([]byte("uploads")).Cursor()

		// User index key is build as follow :
		//  - User index prefix 2 byte ( "_u" )
		//  - The user id
		//  - The upload date reversed ( 8 bytes )
		//  - The upload id ( 16 bytes )
		// Upload id is stored in the key to ensure uniqueness
		// AuthToken is stored in the value to permit byToken filtering
		startKey := append([]byte{'_', 'u'}, []byte(user.ID)...)

		k, t := c.Seek(startKey)
		for k != nil && bytes.HasPrefix(k, startKey) {

			// byToken filter
			if token == nil || string(t) == token.Token {
				// Extract upload id from key ( 16 last bytes )
				ids = append(ids, string(k[len(k)-16:]))
			}

			// Scan the bucket forward
			k, t = c.Next()
		}

		return nil
	})
	if err != nil {
		return
	}

	return
}

// GetUploadsToRemove implementation for Bolt Metadata Backend
func (bmb *MetadataBackend) GetUploadsToRemove(ctx *juliet.Context) (ids []string, err error) {
	err = bmb.db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket([]byte("uploads")).Cursor()

		// Expire index is build as follow :
		//  - Expire index prefix 2 byte ( "_e" )
		//  - The expire timestamp ( 8 bytes )
		//  - The upload id ( 16 bytes )
		// Upload id is stored in the key to ensure uniqueness

		// Create seek key at current timestamp + 1
		timestamp := make([]byte, 8)
		binary.BigEndian.PutUint64(timestamp, uint64(time.Now().Unix()+1))
		startKey := append([]byte{'_', 'e'}, timestamp...)

		// Seek just after the seek key
		// All uploads above the cursor are expired
		c.Seek(startKey)
		for {
			// Scan the bucket upwards
			k, _ := c.Prev()
			if k == nil || !bytes.HasPrefix(k, []byte("_e")) {
				break
			}

			// Extract upload id from key ( 16 last bytes )
			ids = append(ids, string(k[10:]))
		}

		return nil
	})
	if err != nil {
		return
	}

	return
}

// Export implementation for bolt metadata backend
func (bmb *MetadataBackend) Export(ctx *juliet.Context, path string) (err error) {
	fmt.Printf("Exporting metadata from %s to %s\n", bmb.Config.Path, path)

	e, err := exporter.NewExporter(path)
	if err != nil {
		return err
	}
	defer e.Close()

	err = bmb.db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket([]byte("uploads")).Cursor()
		c.Seek([]byte{})
		for {
			// Scan the bucket
			k, v := c.Next()
			if k == nil {
				break
			}
			if bytes.HasPrefix(k, []byte("_")) {
				// Skip index entries
				continue
			}

			// Unserialize metadata from json
			upload := new(common.Upload)
			if err = json.Unmarshal(v, upload); err != nil {
				return fmt.Errorf("Unable to unserialize upload metadata from json \"%s\" : %s", string(v), err)
			}

			err = e.AddUpload(upload)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	err = bmb.db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket([]byte("users")).Cursor()
		c.Seek([]byte{})
		for {
			// Scan the bucket
			k, v := c.Next()
			if k == nil {
				break
			}
			if !(bytes.HasPrefix(k, []byte("google")) || bytes.HasPrefix(k, []byte("ovh"))) {
				// Skip index entries
				continue
			}

			// Unserialize metadata from json
			user := new(common.User)
			if err = json.Unmarshal(v, user); err != nil {
				return fmt.Errorf("Unable to unserialize user metadata from json \"%s\" : %s", string(v), err)
			}

			err = e.AddUser(user)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}
