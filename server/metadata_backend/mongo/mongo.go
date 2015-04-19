package mongo

import (
	"crypto/tls"
	"github.com/root-gg/plik/server/utils"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"log"
	"net"
	"strconv"
	"time"
)

/*
 * User input is only safe in document field !!!
 * Keys with ( '.', '$', ... ) may be interpreted
 */

type MongoMetadataBackendConfig struct {
	Url        string
	Database   string
	Collection string
	Username   string
	Password   string
	Ssl        bool
}

func NewMongoMetadataBackendConfig(config map[string]interface{}) (this *MongoMetadataBackendConfig) {
	this = new(MongoMetadataBackendConfig)
	this.Url = "127.0.0.1:27017"
	this.Database = "plik"
	this.Collection = "meta"
	utils.Assign(this, config)
	return
}

type MongoMetadataBackend struct {
	config  *MongoMetadataBackendConfig
	session *mgo.Session
}

func NewMongoMetadataBackend(config map[string]interface{}) (this *MongoMetadataBackend) {
	this = new(MongoMetadataBackend)
	this.config = NewMongoMetadataBackendConfig(config)

	var err error
	dialInfo := &mgo.DialInfo{}
	dialInfo.Addrs = []string{this.config.Url}
	dialInfo.Database = this.config.Database
	if this.config.Username != "" && this.config.Password != "" {
		dialInfo.Username = this.config.Username
		dialInfo.Password = this.config.Password
	}
	if this.config.Ssl {
		dialInfo.DialServer = func(addr *mgo.ServerAddr) (net.Conn, error) {
			return tls.Dial("tcp", addr.String(), &tls.Config{InsecureSkipVerify: true})
		}
	}
	this.session, err = mgo.DialWithInfo(dialInfo)
	if err != nil {
		log.Fatalf(err.Error())
	}
	this.session.SetMode(mgo.Strong, false)
	this.session.SetSafe(&mgo.Safe{})
	return
}

func (this *MongoMetadataBackend) Create(upload *utils.Upload) (err error) {
	session := this.session.Copy()
	defer session.Close()
	collection := session.DB(this.config.Database).C(this.config.Collection)
	collection.Insert(&upload)
	return nil
}

func (this *MongoMetadataBackend) Get(id string) (u *utils.Upload, err error) {
	session := this.session.Copy()
	defer session.Close()
	collection := session.DB(this.config.Database).C(this.config.Collection)
	u = &utils.Upload{}
	err = collection.Find(bson.M{"id": id}).One(u)
	return
}

func (this *MongoMetadataBackend) AddOrUpdateFile(upload *utils.Upload, file *utils.File) (err error) {
	session := this.session.Copy()
	defer session.Close()
	collection := session.DB(this.config.Database).C(this.config.Collection)
	return collection.Update(bson.M{"id": upload.Id}, bson.M{"$set": bson.M{"files." + file.Id: file}})
}

func (this *MongoMetadataBackend) RemoveFile(upload *utils.Upload, file *utils.File) (err error) {
	session := this.session.Copy()
	defer session.Close()
	collection := session.DB(this.config.Database).C(this.config.Collection)
	return collection.Update(bson.M{"id": upload.Id}, bson.M{"$unset": bson.M{"files." + file.Name: ""}})
}

func (this *MongoMetadataBackend) Remove(upload *utils.Upload) (err error) {
	session := this.session.Copy()
	defer session.Close()
	collection := session.DB(this.config.Database).C(this.config.Collection)
	return collection.Remove(bson.M{"id": upload.Id})
}

func (this *MongoMetadataBackend) GetUploadsToRemove() (ids []string, err error) {
	session := this.session.Copy()
	defer session.Close()
	collection := session.DB(this.config.Database).C(this.config.Collection)

	// Make request
	ids = make([]string, 0)
	uploads := make([]*utils.Upload, 0)
	b := bson.M{"$where": strconv.Itoa(int(time.Now().Unix())) + " > this.uploadDate+this.ttl"}

	// Exec it
	err = collection.Find(b).All(&uploads)
	if err != nil {
		return
	}

	// Append all uploads to the toRemove list
	for _, upload := range uploads {
		ids = append(ids, upload.Id)
	}

	return
}
