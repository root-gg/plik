package common

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jinzhu/copier"
)

// This variable content is meant to be passed the output of the gen_build_info.sh script at compile time using -ldflags
var buildInfoString string

var buildInfo *BuildInfo

func init() {
	buildInfo = &BuildInfo{
		Version: "0.0.0",
	}

	if buildInfoString != "" {
		jsonString, err := base64.StdEncoding.DecodeString(buildInfoString)
		if err != nil {
			panic(fmt.Errorf("Unable to parse build info base64 string : %s", err))
		}

		err = json.Unmarshal(jsonString, buildInfo)
		if err != nil {
			panic(fmt.Errorf("Unable to parse build info json string : %s", err))
		}
	}
}

// BuildInfo export build related variables
type BuildInfo struct {
	Version string `json:"version"`
	Date    int64  `json:"date"`

	User string `json:"user,omitempty"`
	Host string `json:"host,omitempty"`

	GitShortRevision string `json:"gitShortRevision,omitempty"`
	GitFullRevision  string `json:"gitFullRevision,omitempty"`

	IsRelease bool `json:"isRelease"`
	IsMint    bool `json:"isMint"`

	GoVersion string `json:"goVersion,omitempty"`

	Clients  []*Client  `json:"clients"`
	Releases []*Release `json:"releases"`
}

// Client export client build related variables
type Client struct {
	Name string `json:"name"`
	Md5  string `json:"md5"`
	Path string `json:"path"`
	OS   string `json:"os"`
	ARCH string `json:"arch"`
}

// Release export releases related variables
type Release struct {
	Name string `json:"name"`
	Date int64  `json:"date"`
}

// GetBuildInfo get build info
func GetBuildInfo() (bi *BuildInfo) {
	bi = &BuildInfo{}

	// Defensive copy
	err := copier.Copy(bi, buildInfo)
	if err != nil {
		panic(fmt.Errorf("Unable to copy build info : %s", err))
	}

	return bi
}

// Sanitize removes sensitive info from BuildInfo
func (bi *BuildInfo) Sanitize() {
	// Version is needed for the client update to work
	bi.Date = 0
	bi.User = ""
	bi.Host = ""
	bi.GitShortRevision = ""
	bi.GitFullRevision = ""
	bi.IsRelease = false
	bi.IsMint = false
	bi.GoVersion = ""
}

func (bi *BuildInfo) String() string {

	v := fmt.Sprintf("v%s", bi.Version)

	if bi.GitShortRevision != "" {
		v += fmt.Sprintf(" built from git rev %s", bi.GitShortRevision)
	}

	// Compute flags
	var flags []string
	if bi.IsMint {
		flags = append(flags, "mint")
	}
	if bi.IsRelease {
		flags = append(flags, "release")
	}

	if len(flags) > 0 {
		v += fmt.Sprintf(" [%s]", strings.Join(flags, ","))
	}

	if bi.Date > 0 && bi.GoVersion != "" {
		v += fmt.Sprintf(" at %s with %s)", time.Unix(bi.Date, 0), bi.GoVersion)
	}

	return v
}
