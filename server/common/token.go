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

package common

import (
	"github.com/root-gg/plik/server/Godeps/_workspace/src/github.com/nu7hatch/gouuid"
	"time"
)

// Token provide a very basic authentication mechanism
type Token struct {
	Token        string `json:"token" bson:"token"`
	CreationDate int64  `json:"creationDate" bson:"creationDate"`
	Comment      string `json:"comment" bson:"comment"`
	SourceIP     string `json:"sourceIp" bson:"sourceIp"`
}

// NewToken create a new Token instance
func NewToken() (t *Token) {
	t = new(Token)
	return
}

// Create initialize a new Token
func (t *Token) Create() (err error) {
	t.CreationDate = time.Now().Unix()
	uuid, err := uuid.NewV4()
	if err != nil {
		return
	}
	t.Token = uuid.String()
	return
}
