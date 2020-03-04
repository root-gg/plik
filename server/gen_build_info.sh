#!/bin/bash

set -e

# some variables
version=$1
user=$(whoami)
host=$(hostname)
repo=$(pwd)
date=$(date "+%s")
goVersion=$(go version | sed -e "s/go version //")
isRelease=false
isMint=false

# get git current revision
sh=`git rev-list --pretty=format:%h HEAD --max-count=1 | sed '1s/commit /full_rev=/;2s/^/short_rev=/'`
eval "$sh"  # Sets the full_rev & short_rev variables.

# get git version tag
tag=$(git show-ref --tags | egrep "refs/tags/$version$" | cut -d " " -f1)
if [[ $tag = $full_rev ]]; then
    isRelease=true
fi

# get repository status
is_mint_repo() {
  git rev-parse --verify HEAD >/dev/null &&
  git update-index --refresh >/dev/null &&
  git diff-files --quiet &&
  git diff-index --cached --quiet HEAD
}
if is_mint_repo; then
    isMint=true
fi

echo "Plik $version with go $goVersion"
echo "Commit $full_rev mint=$isMint release=$isRelease"

# compute clients code
clients=""
clientList=$(find clients -name "plik*" 2> /dev/null | sort -n)
for client in $clientList ; do
	folder=$(echo $client | cut -d "/" -f2)
	binary=$(echo $client | cut -d "/" -f3)
	os=$(echo $folder | cut -d "-" -f1)
	arch=$(echo $folder | cut -d "-" -f2)
	md5=$(md5sum $client | cut -d " " -f1)

	prettyOs=""
	prettyArch=""

	case "$os" in
		"darwin") 	prettyOs="MacOS" ;;
		"linux") 	prettyOs="Linux" ;;
		"windows") 	prettyOs="Windows" ;;
		"openbsd")	prettyOs="OpenBSD" ;;
		"freebsd")	prettyOs="FreeBSD" ;;
		"bash")		prettyOs="Bash (curl)" ;;
	esac

	case "$arch" in
		"386")		prettyArch="32bit" ;;
		"amd64")	prettyArch="64bit" ;;
		"arm")		prettyArch="ARM" ;;
	esac

	fullName="$prettyOs $prettyArch"
	clientCode="&Client{Name: \"$fullName\", Md5: \"$md5\", Path: \"$client\", OS: \"$os\", ARCH: \"$arch\"}"
	clients+=$'\t\t'"buildInfo.Clients = append(buildInfo.Clients, $clientCode)"$'\n'
done

# get releases
releases=""
git config versionsort.prereleaseSuffix -RC
for gitTag in $(git tag --sort version:refname)
do
	if [ -f "changelog/$gitTag" ]; then
		# '%at': author date, UNIX timestamp
		releaseDate=$(git show -s --pretty="format:%at" "refs/tags/$gitTag")
		releaseCode="&Release{Name: \"$gitTag\", Date: $releaseDate}"
		releases+=$'\t\t'"buildInfo.Releases = append(buildInfo.Releases, $releaseCode)"$'\n'
	fi
done

cat > "server/common/version.go" <<EOF 
package common

//
// This file is generated automatically by gen_build_info.sh
//

import (
	"fmt"
	"strings"
	"time"
)

var buildInfo *BuildInfo

// BuildInfo export build related variables
type BuildInfo struct {
	Version string \`json:"version"\`
	Date    int64  \`json:"date"\`

	User string \`json:"user"\`
	Host string \`json:"host"\`

	GitShortRevision string \`json:"gitShortRevision"\`
	GitFullRevision  string \`json:"gitFullRevision"\`

	IsRelease bool \`json:"isRelease"\`
	IsMint    bool \`json:"isMint"\`

	GoVersion string \`json:"goVersion"\`

	Clients  []*Client  \`json:"clients"\`
	Releases []*Release \`json:"releases"\`
}

// Client export client build related variables
type Client struct {
	Name string \`json:"name"\`
	Md5  string \`json:"md5"\`
	Path string \`json:"path"\`
	OS   string \`json:"os"\`
	ARCH string \`json:"arch"\`
}

// Release export releases related variables
type Release struct {
	Name string \`json:"name"\`
	Date int64  \`json:"date"\`
}

// GetBuildInfo get or instanciate BuildInfo structure
func GetBuildInfo() *BuildInfo {
	if buildInfo == nil {
		buildInfo = new(BuildInfo)
		buildInfo.Clients = make([]*Client, 0)

		buildInfo.Version = "$version"
		buildInfo.Date = $date

		buildInfo.User = "$user"
		buildInfo.Host = "$host"
		buildInfo.GoVersion = "$goVersion"

		buildInfo.GitShortRevision = "$short_rev"
		buildInfo.GitFullRevision = "$full_rev"

		buildInfo.IsRelease = $isRelease
		buildInfo.IsMint = $isMint

		// Clients
$clients
		// Releases
$releases
	}

	return buildInfo
}

func (bi *BuildInfo) String() string {

	v := fmt.Sprintf("v%s (built from git rev %s", bi.Version, bi.GitShortRevision)

	// Compute flags
	var flags []string
	if buildInfo.IsMint {
		flags = append(flags, "mint")
	}
	if buildInfo.IsRelease {
		flags = append(flags, "release")
	}

	if len(flags) > 0 {
		v += fmt.Sprintf(" [%s]", strings.Join(flags, ","))
	}

	v += fmt.Sprintf(" at %s with %s)", time.Unix(bi.Date, 0), bi.GoVersion)

	return v
}
EOF
