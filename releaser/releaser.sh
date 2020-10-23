#!/bin/bash

set -e

###
# This needs to be called inside a cross-compillation capable environement ( see Dockerfile )
###

make clean

# Default server/release targets
if [[ -z "$TARGETS" ]];then
  TARGETS="amd64,386,arm,arm64"
  #TARGETS="amd64"
fi

# Default client targets
if [[ -z "$CLIENT_TARGETS" ]];then
  CLIENT_TARGETS="darwin-amd64,freebsd-386,freebsd-amd64,linux-386,linux-amd64,linux-arm,openbsd-386,openbsd-amd64,windows-amd64,windows-386"
  #CLIENT_TARGETS="linux-amd64"
fi

RELEASE_VERSION=$(server/gen_build_info.sh version)

echo ""
echo "Building clients"
echo ""

rm -rf clients || true
mkdir -p clients/bash
cp client/plik.sh clients/bash

for TARGET in $(echo "$CLIENT_TARGETS" | awk -F, '{for (i=1;i<=NF;i++)print $i}')
do
  GOOS=$(echo "$TARGET" | cut -d "-" -f 1);
  export GOOS
	GOARCH=$(echo "$TARGET" | cut -d "-" -f 2);
	export GOARCH

  CLIENT_DIR="clients/$TARGET"
  CLIENT_MD5="$CLIENT_DIR/MD5SUM"

  if [[ "$GOOS" == "windows" ]] ; then
    CLIENT_PATH="$CLIENT_DIR/plik.exe"
  else
    CLIENT_PATH="$CLIENT_DIR/plik"
  fi

  echo "################################################"
  echo "Building Plik client for $TARGET to $CLIENT_PATH"
  make client

  mkdir -p "$CLIENT_DIR"
  cp client/plik "$CLIENT_PATH"
  md5sum "$CLIENT_PATH" | awk '{print $1}' > "$CLIENT_MD5"
done

if [[ "$TARGETS" == "skip" ]]; then
  exit 0
fi

# Assert frontend has been built already ( copied from previous docker stage )
if [[ ! -d "webapp/dist" ]]; then
  echo "Missing webapp distribution build"
  exit 1
fi

echo ""
echo "Building servers and release archives"
echo ""

rm -rf releases || true
mkdir -p releases/archives
function build_release {
    echo "#################################"
    echo "Building server for $GOOS $GOARCH"
    make server

    RELEASE="plik-$RELEASE_VERSION-$GOOS-$GOARCH"
    RELEASE_DIR="releases/$RELEASE"
    RELEASE_ARCHIVE="archives/$RELEASE.tar.gz"

    mkdir $RELEASE_DIR
    mkdir $RELEASE_DIR/webapp
    mkdir $RELEASE_DIR/server

    # Copy release artifacts
    cp -r clients $RELEASE_DIR
	  cp -r changelog $RELEASE_DIR
	  cp -r webapp/dist $RELEASE_DIR/webapp/dist
	  cp server/plikd.cfg $RELEASE_DIR/server
	  cp server/plikd $RELEASE_DIR/server/plikd

    echo "Building release archive for $GOOS $GOARCH to $RELEASE_ARCHIVE"
	  ( cd releases && tar czf $RELEASE_ARCHIVE $RELEASE )
}

# We build the server only for linux
export GOOS=linux
export CGO_ENABLED=1

for TARGET in $(echo "$TARGETS" | awk -F, '{for (i=1;i<=NF;i++)print $i}')
do
  export GOARCH=$TARGET

  # set cross compiler
  case "$TARGET" in
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

  build_release
done