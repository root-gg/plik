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

	"github.com/root-gg/juliet"
	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/utils"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
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
	log := common.Logger()

	mmb = new(MetadataBackend)
	mmb.config = NewMongoMetadataBackendConfig(config)
	utils.Dump(config)
	utils.Dump(mmb.config)

	// Open connection
	dialInfo := &mgo.DialInfo{}
	dialInfo.Addrs = []string{mmb.config.URL}
	dialInfo.Database = mmb.config.Database
	dialInfo.Timeout = 5 * time.Second
	if mmb.config.Username != "" && mmb.config.Password != "" {
		dialInfo.Username = mmb.config.Username
		dialInfo.Password = mmb.config.Password
	}
	if mmb.config.Ssl {
		dialInfo.DialServer = func(addr *mgo.ServerAddr) (net.Conn, error) {
			return tls.Dial("tcp", addr.String(), &tls.Config{InsecureSkipVerify: true})
		}
	}

	log.Infof("Connecting to mongodb @ %s/%s", mmb.config.URL, mmb.config.Database)

	var err error
	mmb.session, err = mgo.DialWithInfo(dialInfo)
	if err != nil {
		log.Fatalf("Unable to contact mongodb at %s : %s", mmb.config.URL, err.Error())
	}

	log.Infof("Connected to mongodb @ %s/%s", mmb.config.URL, mmb.config.Database)

	// Ensure everything is persisted and replicated
	mmb.session.SetMode(mgo.Strong, false)
	mmb.session.SetSafe(&mgo.Safe{})
	return
}

// Create implementation from MongoDB Metadata Backend
func (mmb *MetadataBackend) Create(ctx *juliet.Context, upload *common.Upload) (err error) {
	log := common.GetLogger(ctx)

	if upload == nil {
		err = log.EWarning("Unable to save upload : Missing upload")
		return
	}

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

	if id == "" {
		err = log.EWarning("Unable to get upload : Missing upload id")
		return
	}

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

	if upload == nil {
		err = log.EWarning("Unable to add file : Missing upload")
		return
	}

	if file == nil {
		err = log.EWarning("Unable to add file : Missing file")
		return
	}

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

	if upload == nil {
		err = log.EWarning("Unable to remove file : Missing upload")
		return
	}

	if file == nil {
		err = log.EWarning("Unable to remove file : Missing file")
		return
	}

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

	if upload == nil {
		err = log.EWarning("Unable to remove upload : Missing upload")
		return
	}

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

	if user == nil {
		err = log.EWarning("Unable to save user : Missing user")
		return
	}

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

	if id == "" && token == "" {
		err = log.EWarning("Unable to get user : Missing user id or token")
		return
	}

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

	user.IsAdmin()

	return
}

// RemoveUser implementation from MongoDB Metadata Backend
func (mmb *MetadataBackend) RemoveUser(ctx *juliet.Context, user *common.User) (err error) {
	log := common.GetLogger(ctx)

	if user == nil {
		err = log.EWarning("Unable to remove user : Missing user")
		return
	}

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

	if user == nil {
		err = log.EWarning("Unable to get user uploads : Missing user")
		return
	}

	session := mmb.session.Copy()
	defer session.Close()
	collection := session.DB(mmb.config.Database).C(mmb.config.Collection)

	b := bson.M{"user": user.ID}
	if token != nil {
		b["token"] = token.Token
	}

	var uploads []*common.Upload
	err = collection.Find(b).Select(bson.M{"id": 1}).Sort("-uploadDate").All(&uploads)
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

// GetUserStatistics implementation for MongoDB Metadata Backend
func (mmb *MetadataBackend) GetUserStatistics(ctx *juliet.Context, user *common.User, token *common.Token) (stats *common.UserStats, err error) {
	log := common.GetLogger(ctx)

	if user == nil {
		err = log.EWarning("Unable to get user uploads : Missing user")
		return
	}

	session := mmb.session.Copy()
	defer session.Close()
	collection := session.DB(mmb.config.Database).C(mmb.config.Collection)

	match := bson.M{"user" : user.ID}
	if token != nil {
		match["token"] = token.Token
	}

	// db.plik_meta.aggregate([{$match: {user:"xxx", token:"xxx"}}, {$project: {"files": {$objectToArray: "$files"}}}, {$unwind: "$files"}, {$group: { _id: null, count: {$sum: 1}, total: {$sum: "$files.v.fileSize"}, uploads: {$addToSet: "$_id"}}}, {$project: {count: "$count", total: "$total", size: { $size: "$uploads"}}}]).pretty()
	pipeline := []bson.M{
		{"$match": match},
		{"$project": bson.M{"file_count": bson.M{"$size": bson.M{"$objectToArray": "$files"}}}},
		{"$unwind": "$files"},
		{"$group": bson.M{"_id": nil, "files": bson.M{"$sum": 1}, "totalSize": bson.M{"$sum": "$files.v.fileSize"}, "uploads": bson.M{"$addToSet": "$_id"}}},
		{"$project": bson.M{"Files": "$files", "TotalSize" : "$totalSize", "Uploads" : bson.M{"$size": "uploads"}}},
	}

	stats = new(common.UserStats)
	err = collection.Pipe(pipeline).One(&stats)
	if err != nil {
		err = log.EWarningf("Unable to get file count from mongodb : %s", err)
	}

	return
}

// GetUsers implementation for MongoDB Metadata Backend
func (mmb *MetadataBackend) GetUsers(ctx *juliet.Context) (ids []string, err error) {
	log := common.GetLogger(ctx)

	session := mmb.session.Copy()
	defer session.Close()
	collection := session.DB(mmb.config.Database).C(mmb.config.UserCollection)

	var results []struct {
		ID string `bson:"id"`
	}
	err = collection.Find(nil).Select(bson.M{"id": 1}).Sort("id").All(&results)
	if err != nil {
		err = log.EWarningf("Unable to get users from mongodb : %s", err)
	}

	for _, result := range results {
		ids = append(ids, result.ID)
	}

	return
}

// GetServerStatistics implementation for MongoDB Metadata Backend
func (mmb *MetadataBackend) GetServerStatistics(ctx *juliet.Context) (stats *common.ServerStats, err error) {
	log := common.GetLogger(ctx)
	stats = new(common.ServerStats)
	session := mmb.session.Copy()
	defer session.Close()

	uploadCollection := session.DB(mmb.config.Database).C(mmb.config.Collection)

	// NUMBER OF UPLOADS
	uploadCount, err := uploadCollection.Find(nil).Count()
	if err != nil {
		err = log.EWarningf("Unable to get upload count from mongodb : %s", err)
	}
	stats.Uploads = uploadCount

	// NUMBER OF ANONYMOUS UPLOADS
	anonymousUploadCount, err := uploadCollection.Find(bson.M{"user": ""}).Count()
	if err != nil {
		err = log.EWarningf("Unable to get anonymous upload count from mongodb : %s", err)
	}
	stats.AnonymousUploads = anonymousUploadCount

	// NUMBER OF FILES
	//db.plik_meta.aggregate([{$project: {"file_count": { $size: { $objectToArray : "$files" } }}}, { $group: { _id : null, total : { $sum : "$file_count" }}}]).pretty()

	pipeline1 := []bson.M{
		{"$project": bson.M{"file_count": bson.M{"$size": bson.M{"$objectToArray": "$files"}}}},
		{"$group": bson.M{"_id": nil, "total": bson.M{"$sum": "$file_count"}}},
	}

	var result1 struct {
		Total int `bson:"total"`
	}

	err = uploadCollection.Pipe(pipeline1).One(&result1)
	if err != nil {
		err = log.EWarningf("Unable to get file count from mongodb : %s", err)
	}

	stats.Files = result1.Total

	// TOTAL SIZE OF ALL FILES
	// db.plik_meta.aggregate([{$project: {"files": { $objectToArray : "$files" } }}, {$unwind: "$files"}, {$group : { _id: null, total : { $sum : "$files.v.fileSize"} }}]).pretty()

	pipeline2 := []bson.M{
		{"$project": bson.M{"files": bson.M{"$objectToArray": "$files"}}},
		{"$unwind": "$files"},
		{"$group": bson.M{"_id": nil, "total": bson.M{"$sum": "$files.v.fileSize"}}},
	}

	var result2 struct {
		Total int64 `bson:"total"`
	}

	err = uploadCollection.Pipe(pipeline2).One(&result2)
	if err != nil {
		err = log.EWarningf("Unable to get total file size from mongodb : %s", err)
	}

	stats.TotalSize = result2.Total

	if !common.Config.NoAnonymousUploads {

		// TOTAL SIZE OF ALL ANONYMOUS UPLOAD FILES
		// db.plik_meta.aggregate([{$match: {user:""}},{$project: {"files": { $objectToArray : "$files" } }}, {$unwind: "$files"}, {$group : { _id: null, total : { $sum : "$files.v.fileSize"} }}]).pretty()

		pipeline3 := []bson.M{
			{"$match": bson.M{"user": ""}},
			{"$project": bson.M{"files": bson.M{"$objectToArray": "$files"}}},
			{"$unwind": "$files"},
			{"$group": bson.M{"_id": nil, "total": bson.M{"$sum": "$files.v.fileSize"}}},
		}

		var result3 struct {
			Total int64 `bson:"total"`
		}

		err = uploadCollection.Pipe(pipeline3).One(&result3)
		if err != nil {
			err = log.EWarningf("Unable to get total file size from mongodb : %s", err)
		}

		stats.AnonymousSize = result3.Total
	}

	// TOTAL FILE SIZE BY FILE TYPE
	// db.plik_meta.aggregate([{$project: {"files": { $objectToArray : "$files" } }}, {$unwind: "$files"}, {$group : { _id: "$files.v.fileType", total : { $sum : "$files.v.fileSize"} }},{ $sort : { total : -1 }},{ $limit : 5 }]).pretty()

	pipeline4 := []bson.M{
		{"$project": bson.M{"files": bson.M{"$objectToArray": "$files"}}},
		{"$unwind": "$files"},
		{"$group": bson.M{"_id": "$files.v.fileType", "total": bson.M{"$sum": 1}}},
		{"$sort": bson.M{"total": -1}},
		{"$limit": 10},
	}

	var result4 []common.FileTypeByCount

	err = uploadCollection.Pipe(pipeline4).All(&result4)
	if err != nil {
		err = log.EWarningf("Unable to get total file size from mongodb : %s", err)
	}

	stats.FileTypeByCount = result4

	// TOTAL FILE SIZE BY FILE TYPE
	// db.plik_meta.aggregate([{$project: {"files": { $objectToArray : "$files" } }}, {$unwind: "$files"}, {$group : { _id: "$files.v.fileType", total : { $sum : 1} }},{ $sort : { total : -1 }},{ $limit : 5 }]).pretty()

	pipeline5 := []bson.M{
		{"$project": bson.M{"files": bson.M{"$objectToArray": "$files"}}},
		{"$unwind": "$files"},
		{"$group": bson.M{"_id": "$files.v.fileType", "total": bson.M{"$sum": "$files.v.fileSize"}}},
		{"$sort": bson.M{"total": -1}},
		{"$limit": 10},
	}

	var result5 []common.FileTypeBySize

	err = uploadCollection.Pipe(pipeline5).All(&result5)
	if err != nil {
		err = log.EWarningf("Unable to get total file size from mongodb : %s", err)
	}

	stats.FileTypeBySize = result5

	userCollection := session.DB(mmb.config.Database).C(mmb.config.UserCollection)

	// NUMBER OF USERS
	userCount, err := userCollection.Find(nil).Count()
	if err != nil {
		err = log.EWarningf("Unable to get user count from mongodb : %s", err)
	}
	stats.Users = userCount

	return
}
