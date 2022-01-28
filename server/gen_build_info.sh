#!/usr/bin/env bash

set -e

# arguments
output=$1

FILE="$(dirname "$0")/common/version.go"
if [[ ! -f "$FILE" ]]; then
    echo "$FILE not found"
    exit 1
fi

version=${VERSION:-$(git describe --tags --abbrev=0)}
if [[ -z "$version" ]]; then
    echo "version not found"
    exit 1
fi

if [[ "$output" == "version" ]]; then
  echo "$version"
  exit 0
fi

# some variables
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

if [[ "$output" == "info" ]]; then
  echo "Plik $version built with $goVersion"
  echo "Commit $full_rev mint=$isMint release=$isRelease"
  exit 0
fi

# join strings from array
function join_by { local IFS="$1"; shift; echo "$*"; }

# compute clients code
declare -a clients
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
	client_json="{\"name\": \"$fullName\", \"md5\": \"$md5\", \"path\": \"$client\", \"os\": \"$os\", \"arch\": \"$arch\"}"
	clients+=("$client_json")
done
clients_json="[$(join_by , "${clients[@]}")]"

# get releases
declare -a releases
git config versionsort.prereleaseSuffix -RC
for gitTag in $(git tag --sort version:refname)
do
	if [ -f "changelog/$gitTag" ]; then
		# '%at': author date, UNIX timestamp
		release_date=$(git show -s --pretty="format:%at" "refs/tags/$gitTag")
		release_json="{\"name\": \"$gitTag\", \"date\": $release_date}"
		releases+=("$release_json")
	fi
done
releases_json="[$(join_by , "${releases[@]}")]"

json=$(cat << EOF
{
  "version" : "$version",
  "date" : $date,

  "user" : "$user",
  "host" : "$host",
  "goVersion" : "$goVersion",

  "gitShortRevision" : "$short_rev",
  "gitFullRevision" : "$full_rev",
  "isRelease" : $isRelease,
  "isMint" : $isMint,

  "clients" : $clients_json,
  "releases" : $releases_json
}
EOF
)

if [[ "$output" == "base64" ]]; then
  echo $json | base64 | tr -d '\n'
else
  echo $json
fi