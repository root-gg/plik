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
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/root-gg/plik/server/context"
	"strings"

	"github.com/boltdb/bolt"
	"github.com/root-gg/juliet"
	"github.com/root-gg/plik/server/common"
)

// SaveUser implementation for Bolt Metadata Backend
func (b *Backend) SaveUser(ctx *juliet.Context, user *common.User) (err error) {
	log := context.GetLogger(ctx)

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
	err = b.db.Update(func(tx *bolt.Tx) error {
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
func (b *Backend) GetUser(ctx *juliet.Context, id string, token string) (user *common.User, err error) {
	log := context.GetLogger(ctx)

	if id == "" && token == "" {
		err = log.EWarning("Unable to get user : Missing user id or token")
		return
	}

	// Get json user from Bolt database
	err = b.db.View(func(tx *bolt.Tx) error {
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

		b := bucket.Get([]byte(id))

		// User not found but no error
		if b == nil || len(b) == 0 {
			return nil
		}

		// Unserialize user from json
		user = common.NewUser()
		err = json.Unmarshal(b, user)
		if err != nil {
			return log.EWarningf("Unable to unserialize user from json \"%s\" : %s", string(b), err)
		}

		return nil
	})
	if err != nil {
		err = log.EWarningf("Unable to get user : %s", err)
		return
	}

	return
}

// RemoveUser implementation for Bolt Metadata Backend
func (b *Backend) RemoveUser(ctx *juliet.Context, user *common.User) (err error) {
	log := context.GetLogger(ctx)

	if user == nil {
		err = log.EWarning("Unable to remove user : Missing user")
		return
	}

	// Remove user from bolt database
	err = b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("users"))
		if bucket == nil {
			return fmt.Errorf("Unable to get users Bolt bucket")
		}

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
func (b *Backend) GetUserUploads(ctx *juliet.Context, user *common.User, token *common.Token) (ids []string, err error) {
	log := context.GetLogger(ctx)

	if user == nil {
		err = log.EWarning("Unable to get user uploads : Missing user")
		return
	}

	err = b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("uploads"))
		if bucket == nil {
			return fmt.Errorf("Unable to get uploads Bolt bucket")
		}
		cursor := bucket.Cursor()

		// User index key is build as follow :
		//  - User index prefix 2 byte ( "_u" )
		//  - The user id
		//  - The upload date reversed ( 8 bytes )
		//  - The upload id ( 16 bytes )
		// Upload id is stored in the key to ensure uniqueness
		// AuthToken is stored in the value to permit byToken filtering
		startKey := append([]byte{'_', 'u'}, []byte(user.ID)...)

		key, t := cursor.Seek(startKey)
		for key != nil && bytes.HasPrefix(key, startKey) {

			// byToken filter
			if token == nil || string(t) == token.Token {
				// Extract upload id from key ( 16 last bytes )
				ids = append(ids, string(key[len(key)-16:]))
			}

			// Scan the bucket forward
			key, t = cursor.Next()
		}

		return nil
	})
	if err != nil {
		return
	}

	return
}

// GetUsers implementation for Bolt Metadata Backend
func (b *Backend) GetUsers(ctx *juliet.Context) (ids []string, err error) {
	log := context.GetLogger(ctx)

	// Get users from Bolt database
	err = b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("users"))
		if bucket == nil {
			return fmt.Errorf("Unable to get users Bolt bucket")
		}

		cursor := bucket.Cursor()

		for id, _ := cursor.First(); id != nil; id, _ = cursor.Next() {
			strid := string(id)

			// Discard tokens from the token index
			// TODO add an _ in front of the tokens
			if !(strings.HasPrefix(strid, "ovh") || strings.HasPrefix(strid, "google")) {
				continue
			}

			ids = append(ids, strid)
		}

		return nil
	})
	if err != nil {
		err = log.EWarningf("Unable to get users : %s", err)
		return
	}

	return
}

// GetUserStatistics implementation for Bolt Metadata Backend
func (b *Backend) GetUserStatistics(ctx *juliet.Context, user *common.User, token *common.Token) (stats *common.UserStats, err error) {
	//log := context.GetLogger(ctx)

	stats = new(common.UserStats)

	ids, err := b.GetUserUploads(ctx, user, token)
	if err != nil {
		return nil, err
	}

	for _, id := range ids {
		upload, err := b.Get(ctx, id)
		if err != nil {
			continue
		}

		stats.Uploads++
		stats.Files += len(upload.Files)

		for _, file := range upload.Files {
			stats.TotalSize += file.CurrentSize
		}
	}

	return
}
