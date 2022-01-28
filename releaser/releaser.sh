#!/usr/bin/env bash

set -e

unamestr=$(uname)
if [ "$unamestr" = 'FreeBSD' ]; then
  MAKE="gmake"
  TAR="gtar"
else
  MAKE="make"
  TAR="tar"
fi

$MAKE clean

# Assert frontend has been built already ( copied from previous docker stage )
if [[ ! -d "webapp/dist" ]]; then
  echo "Missing webapp distribution build"
  exit 1
fi

RELEASE_VERSION=$(server/gen_build_info.sh version)

# Default client targets
if [[ -z "$CLIENT_TARGETS" ]];then
  CLIENT_TARGETS="darwin/amd64,freebsd/386,freebsd/amd64,linux/386,linux/amd64,linux/arm,linux/arm64,openbsd/386,openbsd/amd64,windows/amd64,windows/386"
  #CLIENT_TARGETS="linux/amd64"
fi

echo ""
echo "Building clients for version $RELEASE_VERSION"
echo ""

rm -rf clients || true
mkdir -p clients/bash
cp client/plik.sh clients/bash

for TARGET in $(echo "$CLIENT_TARGETS" | awk -F, '{for (i=1;i<=NF;i++)print $i}')
do
  GOOS=$(echo "$TARGET" | cut -d "/" -f 1);
  export GOOS
	GOARCH=$(echo "$TARGET" | cut -d "/" -f 2);
	export GOARCH

  CLIENT_DIR="clients/${TARGET//\//-}"
  CLIENT_MD5="$CLIENT_DIR/MD5SUM"

  if [[ "$GOOS" == "windows" ]] ; then
    CLIENT_PATH="$CLIENT_DIR/plik.exe"
  else
    CLIENT_PATH="$CLIENT_DIR/plik"
  fi

  echo "################################################"
  echo "Building Plik client for $TARGET to $CLIENT_PATH"
  $MAKE --no-print-directory client

  mkdir -p "$CLIENT_DIR"
  cp client/plik "$CLIENT_PATH"
  md5sum "$CLIENT_PATH" | awk '{print $1}' > "$CLIENT_MD5"
done

echo ""
echo "Building Plik server v$RELEASE_VERSION $TARGETOS/$TARGETARCH$TARGETVARIANT"
echo ""

export GOOS=$TARGETOS
export GOARCH=$TARGETARCH
export GOARM=${TARGETVARIANT//v/}
export CGO_ENABLED=1

# set cross compiler
if [[ -z "$CC" ]]; then
  case "$TARGETARCH" in
    "amd64")
      unset CC
      ;;
    "386")
      export CC=i686-linux-gnu-gcc
      ;;
    "arm")
      export CC=arm-linux-gnueabi-gcc
      ;;
    "arm64")
      export CC=aarch64-linux-gnu-gcc
      ;;
  esac
fi

$MAKE --no-print-directory server

echo ""
echo "Building Plik release v$RELEASE_VERSION $TARGETOS/$TARGETARCH$TARGETVARIANT"
echo ""

RELEASE_DIR="release"

mkdir $RELEASE_DIR
mkdir $RELEASE_DIR/webapp
mkdir $RELEASE_DIR/server

# Copy release artifacts
cp -r clients $RELEASE_DIR
cp -r changelog $RELEASE_DIR
cp -r webapp/dist $RELEASE_DIR/webapp/dist
cp server/plikd.cfg $RELEASE_DIR/server
cp server/plikd $RELEASE_DIR/server/plikd

RELEASE="plik-$RELEASE_VERSION-$GOOS-$GOARCH"
RELEASE_ARCHIVE="$RELEASE.tar.gz"

echo ""
echo "Building Plik release archive $RELEASE_ARCHIVE"
echo ""

$TAR czf $RELEASE_ARCHIVE --transform "s,^$RELEASE_DIR,$RELEASE," $RELEASE_DIR
$TAR tf $RELEASE_ARCHIVE