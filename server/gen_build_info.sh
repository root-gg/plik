#!/bin/bash
set -e

# some variables
version=$1
user=$(whoami)
host=$(hostname)
repo=$(pwd)
date=$(date "+%s%N")
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


cat > "server/common/version.go" <<EOF 
package common

var buildInfo *BuildInfo

type BuildInfo struct {
	Version string
	Date    int64

	User string
	Host string

	GitShortRevision string
	GitFullRevision  string

	IsRelease bool
	IsMint    bool
}

func GetBuildInfo() *BuildInfo {
	if buildInfo == nil {
		buildInfo = new(BuildInfo)

		buildInfo.Version = "$version"
		buildInfo.Date = $date

		buildInfo.User = "$user"
		buildInfo.Host = "$host"

		buildInfo.GitShortRevision = "$short_rev"
		buildInfo.GitFullRevision = "$full_rev"

		buildInfo.IsRelease = $isRelease
		buildInfo.IsMint = $isMint
	}

	return buildInfo
}
EOF
