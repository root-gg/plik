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

package mongo

import (
	"crypto/tls"
	"net"
	"strconv"
	"time"

	"github.com/root-gg/plik/server/Godeps/_workspace/src/github.com/root-gg/juliet"
	mgo "github.com/root-gg/plik/server/Godeps/_workspace/src/gopkg.in/mgo.v2"
	"github.com/root-gg/plik/server/Godeps/_workspace/src/gopkg.in/mgo.v2/bson"
	"github.com/root-gg/plik/server/common"
)

/*
 * User input is only safe in document field !!!
 * Keys with ( '.', '$', ... ) may be interpreted
 */

// MetadataBackend object
type MetadataBackend struct {
	config  *MetadataBackendConfig
	session *mgo.Session
}

// NewMongoMetadataBackend instantiate a new MongoDB Metadata Backend
// from configuration passed as argument
func NewMongoMetadataBackend(config map[string]interface{}) (mmb *MetadataBackend) {
	mmb = new(MetadataBackend)
	mmb.config = NewMongoMetadataBackendConfig(config)

	// Open connection
	dialInfo := &mgo.DialInfo{}
	dialInfo.Addrs = []string{mmb.config.URL}
	dialInfo.Database = mmb.config.Database
	if mmb.config.Username != "" && mmb.config.Password != "" {
		dialInfo.Username = mmb.config.Username
		dialInfo.Password = mmb.config.Password
	}
	if mmb.config.Ssl {
		dialInfo.DialServer = func(addr *mgo.ServerAddr) (net.Conn, error) {
			return tls.Dial("tcp", addr.String(), &tls.Config{InsecureSkipVerify: true})
		}
	}
	var err error
	mmb.session, err = mgo.DialWithInfo(dialInfo)
	if err != nil {
		common.Logger().Fatalf("Unable to contact mongodb at %s : %s", mmb.config.URL, err.Error())
	}

	// Ensure everything is persisted and replicated
	mmb.session.SetMode(mgo.Strong, false)
	mmb.session.SetSafe(&mgo.Safe{})
	return
}

// Create implementation from MongoDB Metadata Backend
func (mmb *MetadataBackend) Create(ctx *juliet.Context, upload *common.Upload) (err error) {
	log := common.GetLogger(ctx)

	session := mmb.session.Copy()
	defer session.Close()
	collection := session.DB(mmb.config.Database).C(mmb.config.Collection)
	err = collection.Insert(&upload)
	if err != nil {
		err = log.EWarningf("Unable to append metadata to mongodb : %s", err)
	}
	return
}

// Get implementation from MongoDB Metadata Backend
func (mmb *MetadataBackend) Get(ctx *juliet.Context, id string) (u *common.Upload, err error) {
	log := common.GetLogger(ctx)

	session := mmb.session.Copy()
	defer session.Close()
	collection := session.DB(mmb.config.Database).C(mmb.config.Collection)
	u = &common.Upload{}
	err = collection.Find(bson.M{"id": id}).One(u)
	if err != nil {
		err = log.EWarningf("Unable to get metadata from mongodb : %s", err)
	}
	return
}

// AddOrUpdateFile implementation from MongoDB Metadata Backend
func (mmb *MetadataBackend) AddOrUpdateFile(ctx *juliet.Context, upload *common.Upload, file *common.File) (err error) {
	log := common.GetLogger(ctx)

	session := mmb.session.Copy()
	defer session.Close()
	collection := session.DB(mmb.config.Database).C(mmb.config.Collection)
	err = collection.Update(bson.M{"id": upload.ID}, bson.M{"$set": bson.M{"files." + file.ID: file}})
	if err != nil {
		err = log.EWarningf("Unable to get metadata from mongodb : %s", err)
	}
	return
}

// RemoveFile implementation from MongoDB Metadata Backend
func (mmb *MetadataBackend) RemoveFile(ctx *juliet.Context, upload *common.Upload, file *common.File) (err error) {
	log := common.GetLogger(ctx)

	session := mmb.session.Copy()
	defer session.Close()
	collection := session.DB(mmb.config.Database).C(mmb.config.Collection)
	err = collection.Update(bson.M{"id": upload.ID}, bson.M{"$unset": bson.M{"files." + file.Name: ""}})
	if err != nil {
		err = log.EWarningf("Unable to remove file from mongodb : %s", err)
	}
	return
}

// Remove implementation from MongoDB Metadata Backend
func (mmb *MetadataBackend) Remove(ctx *juliet.Context, upload *common.Upload) (err error) {
	log := common.GetLogger(ctx)

	session := mmb.session.Copy()
	defer session.Close()
	collection := session.DB(mmb.config.Database).C(mmb.config.Collection)
	err = collection.Remove(bson.M{"id": upload.ID})
	if err != nil {
		err = log.EWarningf("Unable to remove upload from mongodb : %s", err)
	}
	return
}

// SaveUser implementation from MongoDB Metadata Backend
func (mmb *MetadataBackend) SaveUser(ctx *juliet.Context, user *common.User) (err error) {
	log := common.GetLogger(ctx)

	session := mmb.session.Copy()
	defer session.Close()
	collection := session.DB(mmb.config.Database).C(mmb.config.UserCollection)

	_, err = collection.Upsert(bson.M{"id": user.ID}, &user)
	if err != nil {
		err = log.EWarningf("Unable to save user to mongodb : %s", err)
	}
	return
}

// GetUser implementation from MongoDB Metadata Backend
func (mmb *MetadataBackend) GetUser(ctx *juliet.Context, id string, token string) (user *common.User, err error) {
	log := common.GetLogger(ctx)

	session := mmb.session.Copy()
	defer session.Close()
	collection := session.DB(mmb.config.Database).C(mmb.config.UserCollection)

	user = &common.User{}
	if id != "" {
		err = collection.Find(bson.M{"id": id}).One(user)
		if err == mgo.ErrNotFound {
			return nil, nil
		} else if err != nil {
			err = log.EWarningf("Unable to get user from mongodb : %s", err)
		}
	} else if token != "" {
		err = collection.Find(bson.M{"tokens.token": token}).One(user)
		if err == mgo.ErrNotFound {
			return nil, nil
		} else if err != nil {
			err = log.EWarningf("Unable to get user from mongodb : %s", err)
		}
	} else {
		err = log.EWarning("Unable to get user from mongodb : Missing user id or token")
	}

	return
}

// RemoveUser implementation from MongoDB Metadata Backend
func (mmb *MetadataBackend) RemoveUser(ctx *juliet.Context, user *common.User) (err error) {
	log := common.GetLogger(ctx)

	session := mmb.session.Copy()
	defer session.Close()
	collection := session.DB(mmb.config.Database).C(mmb.config.UserCollection)

	err = collection.Remove(bson.M{"id": user.ID})
	if err != nil {
		err = log.EWarningf("Unable to remove user from mongodb : %s", err)
	}

	return
}

// GetUserUploads implementation from MongoDB Metadata Backend
func (mmb *MetadataBackend) GetUserUploads(ctx *juliet.Context, user *common.User, token *common.Token) (ids []string, err error) {
	log := common.GetLogger(ctx)

	session := mmb.session.Copy()
	defer session.Close()
	collection := session.DB(mmb.config.Database).C(mmb.config.Collection)

	var b bson.M
	if user != nil {
		b = bson.M{"user": user.ID}
	} else if token != nil {
		b = bson.M{"tokens.token": token.Token}
	} else {
		err = log.EWarning("Unable to get user uploads : Missing user id or token")
	}

	var uploads []*common.Upload
	err = collection.Find(b).Select(bson.M{"id": 1}).All(&uploads)
	if err != nil {
		err = log.EWarningf("Unable to get user uploads : %s", err)
		return
	}

	// Get all ids
	for _, upload := range uploads {
		ids = append(ids, upload.ID)
	}

	return
}

// GetUploadsToRemove implementation from MongoDB Metadata Backend
func (mmb *MetadataBackend) GetUploadsToRemove(ctx *juliet.Context) (ids []string, err error) {
	log := common.GetLogger(ctx)

	session := mmb.session.Copy()
	defer session.Close()
	collection := session.DB(mmb.config.Database).C(mmb.config.Collection)

	// Look for expired uploads
	var uploads []*common.Upload
	b := bson.M{"$where": "this.ttl > 0 && " + strconv.Itoa(int(time.Now().Unix())) + " > this.uploadDate + this.ttl"}

	err = collection.Find(b).Select(bson.M{"id": 1}).All(&uploads)
	if err != nil {
		err = log.EWarningf("Unable to get uploads to remove : %s", err)
		return
	}

	// Get all ids
	for _, upload := range uploads {
		ids = append(ids, upload.ID)
	}

	return
}
