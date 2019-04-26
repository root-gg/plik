/**

    Plik upload server

The MIT License (MIT)

Copyright (c) <2015> Copyright holders list can be found in AUTHORS file
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
	"encoding/json"
	"errors"
	"github.com/root-gg/juliet"
	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/metadata"
	"sync"
)

// Ensure Testing Metadata Backend implements metadata.Backend interface
var _ metadata.Backend = (*MetadataBackend)(nil)

// MetadataBackend backed in-memory for testing purpose
type MetadataBackend struct {
	uploads map[string]*common.Upload
	users   map[string]*common.User

	err error
	mu  sync.Mutex
}

// NewBackend create a new Testing MetadataBackend
func NewBackend() (b *MetadataBackend) {
	b = new(MetadataBackend)
	b.uploads = make(map[string]*common.Upload)
	b.users = make(map[string]*common.User)
	return b
}

// Upsert create or update upload metadata
func (b *MetadataBackend) Upsert(ctx *juliet.Context, upload *common.Upload) (err error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.err != nil {
		return b.err
	}

	u, err := defCopy(upload)
	if err != nil {
		return err
	}

	b.uploads[upload.ID] = u

	return nil
}

// Get upload metadata
func (b *MetadataBackend) Get(ctx *juliet.Context, id string) (upload *common.Upload, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.err != nil {
		return nil, b.err
	}

	return b.get(ctx, id)
}

func (b *MetadataBackend) get(ctx *juliet.Context, id string) (upload *common.Upload, err error) {
	if upload, ok := b.uploads[id]; ok {

		u, err := defCopy(upload)
		if err != nil {
			return nil, err
		}

		return u, nil
	}

	return nil, errors.New("Upload does not exists")
}

// Remove upload metadata
func (b *MetadataBackend) Remove(ctx *juliet.Context, upload *common.Upload) (err error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.err != nil {
		return b.err
	}

	delete(b.uploads, upload.ID)

	return nil
}

// SaveUser create or update user
func (b *MetadataBackend) SaveUser(ctx *juliet.Context, user *common.User) (err error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.err != nil {
		return b.err
	}

	b.users[user.ID] = user

	return nil
}

// GetUser get a user
func (b *MetadataBackend) GetUser(ctx *juliet.Context, id string, token string) (user *common.User, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.err != nil {
		return nil, b.err
	}

	return b.getUser(ctx, id, token)
}

func (b *MetadataBackend) getUser(ctx *juliet.Context, id string, token string) (user *common.User, err error) {
	if id != "" {
		if user, ok := b.users[id]; ok {
			return user, nil
		}
	} else if token != "" {
		for _, u := range b.users {
			for _, t := range u.Tokens {
				if t.Token == token {
					user = u
					return u, nil
				}
			}
		}
	}

	return nil, nil
}

// RemoveUser remove a user
func (b *MetadataBackend) RemoveUser(ctx *juliet.Context, user *common.User) (err error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.err != nil {
		return b.err
	}

	_, err = b.getUser(ctx, user.ID, "")
	if err != nil {
		return err
	}

	delete(b.users, user.ID)

	return nil
}

// GetUserUploads return a user uploads
func (b *MetadataBackend) GetUserUploads(ctx *juliet.Context, user *common.User, token *common.Token) (ids []string, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.err != nil {
		return nil, b.err
	}

	return b.getUserUploads(ctx, user, token)
}

func (b *MetadataBackend) getUserUploads(ctx *juliet.Context, user *common.User, token *common.Token) (ids []string, err error) {
	if user == nil {
		return nil, errors.New("Missing user")
	}
	for _, upload := range b.uploads {
		if upload.User != user.ID {
			continue
		}
		if token != nil && upload.Token != token.Token {
			continue
		}

		ids = append(ids, upload.ID)
	}

	return ids, nil
}

// GetUserStatistics return a user statistics
func (b *MetadataBackend) GetUserStatistics(ctx *juliet.Context, user *common.User, token *common.Token) (stats *common.UserStats, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.err != nil {
		return nil, b.err
	}

	stats = &common.UserStats{}

	ids, err := b.getUserUploads(ctx, user, token)
	if err != nil {
		return nil, err
	}

	for _, id := range ids {
		upload, err := b.get(ctx, id)
		if err != nil {
			continue
		}

		stats.Uploads++

		for _, file := range upload.Files {
			stats.Files++
			stats.TotalSize += file.CurrentSize
		}
	}

	return stats, nil
}

// GetUsers return all user ids
func (b *MetadataBackend) GetUsers(ctx *juliet.Context) (ids []string, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.err != nil {
		return nil, b.err
	}

	for id := range b.users {
		ids = append(ids, id)
	}

	return ids, nil
}

// GetServerStatistics return server statistics
func (b *MetadataBackend) GetServerStatistics(ctx *juliet.Context) (stats *common.ServerStats, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.err != nil {
		return nil, b.err
	}

	stats = new(common.ServerStats)

	byTypeAggregator := common.NewByTypeAggregator()

	for _, upload := range b.uploads {
		stats.AddUpload(upload)

		for _, file := range upload.Files {
			byTypeAggregator.AddFile(file)
		}
	}

	stats.FileTypeByCount = byTypeAggregator.GetFileTypeByCount(10)
	stats.FileTypeBySize = byTypeAggregator.GetFileTypeBySize(10)

	stats.Users = len(b.users)

	return
}

// GetUploadsToRemove return expired upload ids
func (b *MetadataBackend) GetUploadsToRemove(ctx *juliet.Context) (ids []string, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.err != nil {
		return nil, b.err
	}

	for id, upload := range b.uploads {
		if upload.IsExpired() {
			ids = append(ids, id)
		}
	}

	return ids, nil
}

// SetError sets the error any subsequent method other call will return
func (b *MetadataBackend) SetError(err error) {
	b.err = err
}

func defCopy(upload *common.Upload) (u *common.Upload, err error) {
	u = &common.Upload{}
	j, err := json.Marshal(upload)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(j, u)
	if err != nil {
		return nil, err
	}
	return u, err
}
