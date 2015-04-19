package utils

import (
	"bytes"
	"encoding/json"
	"github.com/BurntSushi/toml"
	"log"
	"reflect"
)

/*
 * Assign a map[string]interface{} to a struct mapping the map pairs to
 * the structure members by name using reflexion.
 */
func Assign(config interface{}, values map[string]interface{}) {
	s := reflect.ValueOf(config).Elem()
	t := reflect.TypeOf(config)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	for key, val := range values {
		if typ, ok := t.FieldByName(key); ok {
			s.FieldByName(key).Set(reflect.ValueOf(val).Convert(typ.Type))
		}
	}
}

type Configuration struct {
	Debug         bool
	ListenAddress string
	ListenPort    int
	MaxFileSize   int

	DefaultTtl int
	MaxTtl     int

	SslCert string
	SslKey  string

	UploadIpRestriction bool
	UploadIpSubnets     []string
	UploadIpGetMethod   string

	UploadIdLength int
	FileIdLength   int

	MetadataBackend       string
	MetadataBackendConfig map[string]interface{}

	DataBackend       string
	DataBackendConfig map[string]interface{}

	ShortenBackend       string
	ShortenBackendConfig map[string]interface{}
}

// Global var to store conf
var Config *Configuration

func NewConfiguration() (this *Configuration) {
	this = new(Configuration)
	this.ListenAddress = "0.0.0.0"
	this.ListenPort = 8080
	this.UploadIpRestriction = false
	this.UploadIpGetMethod = "request"
	this.MetadataBackend = "file"
	this.MaxFileSize = 1048576 // 1MB
	this.DefaultTtl = 2592000  // 30 days
	this.MaxTtl = 0
	this.SslCert = ""
	this.SslKey = ""
	return
}

func LoadConfiguration(file string) {
	Config = NewConfiguration()
	if _, err := toml.DecodeFile(file, Config); err != nil {
		log.Println(err)
	}
	Config.Dump()
}

/*
 * Display configuration
 */
func (this *Configuration) Dump() {
	Sdump(this)
}

func Debug(message string) {
	if Config.Debug {
		log.Println(message)
	}
}

func Dump(data interface{}) {
	log.Println(Sdump(data))
}

func Sdump(data interface{}) string {
	buf := new(bytes.Buffer)
	if json, err := json.Marshal(data); err != nil {
		log.Printf("Unable to dump data %v : %s", data, err)
	} else {
		buf.Write(json)
		buf.WriteString("\n")
	}
	return string(buf.Bytes())
}
