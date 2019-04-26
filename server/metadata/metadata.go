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

package metadata

import (
	"github.com/root-gg/juliet"
	"github.com/root-gg/plik/server/common"
)

// Backend interface describes methods that metadata backends
// must implements to be compatible with plik.
type Backend interface {
	// Upload
	Upsert(ctx *juliet.Context, upload *common.Upload) (err error)
	Get(ctx *juliet.Context, id string) (upload *common.Upload, err error)
	Remove(ctx *juliet.Context, upload *common.Upload) (err error)

	// User
	SaveUser(ctx *juliet.Context, user *common.User) (err error)
	GetUser(ctx *juliet.Context, id string, token string) (user *common.User, err error)
	RemoveUser(ctx *juliet.Context, user *common.User) (err error)
	GetUserUploads(ctx *juliet.Context, user *common.User, token *common.Token) (ids []string, err error)
	GetUserStatistics(ctx *juliet.Context, user *common.User, token *common.Token) (stats *common.UserStats, err error)

	// Server
	GetUsers(ctx *juliet.Context) (ids []string, err error)
	GetServerStatistics(ctx *juliet.Context) (stats *common.ServerStats, err error)
	GetUploadsToRemove(ctx *juliet.Context) (ids []string, err error)
}
