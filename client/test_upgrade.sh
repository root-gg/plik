#!/bin/bash



set -e
cd $(dirname $0)
cd ..

###
# Try to upgrade cli client from older target releases
###

RELEASES=(
    #1.0
    #1.0.1
    #1.1-RC1
    #1.1-RC2
    #1.1
    #1.1.1
    #1.2-RC1
    #1.2-RC2
    #1.2-RC3
    1.2
    1.2.1
    1.2.2
    1.2.3
    1.2.4
    1.3-RC1
    1.3
)

###
# Check that no plikd is running
###

URL="http://127.0.0.1:8080"
if curl "$URL/version" 2>/dev/null | grep version > /dev/null ; then
    echo "A plik instance is running @ $URL"
    exit 1
fi

###
# Build server and clients
###

echo "Build current server and clients"
make clean clients server

###
# Run server
###

echo "Start Plik server :"
(cd server && ./plikd)&

# Verify that server is running
sleep 1
if ! curl "$URL/version" 2>/dev/null | grep version > /dev/null ; then
    echo "Plik server did not start @ $URL"
    exit 1
fi

# Defer server shutdown
function shutdown {
    echo "Shutting down plik server"
    PID=$(ps a | grep plikd | grep -v grep | awk '{print $1}')
    if [ "x$PID" != "x" ];then
        kill $PID
        sleep 1
        PID=$(ps a | grep plikd | grep -v grep | awk '{print $1}')
        if [ "x$PID" != "x" ];then
            kill -9 $PID
        fi
    fi
}
trap shutdown EXIT

###
# Get current client info
###

CLIENT_DIR="clients/$(go env GOOS)-$(go env GOARCH)"

CLIENT_BIN="$CLIENT_DIR/plik"
if [ ! -f "$CLIENT_BIN" ];then
    echo "Missing $CLIENT_BIN"
    exit 1
fi

MD5SUM_FILE="$CLIENT_DIR/MD5SUM"
if [ ! -f "$MD5SUM_FILE" ];then
    echo "Missing $MD5SUM_FILE"
    exit 1
fi

CLIENT_MD5=$(md5sum $CLIENT_BIN | awk '{print $1}')
SERVER_MD5=$(cat $MD5SUM_FILE)

if [ "$CLIENT_MD5" != "$SERVER_MD5" ];then
    echo "md5sum mismatch real($CLIENT_MD5) != server($SERVER_MD5)"
    exit 1
fi

###
# Setup temporary build environment
###

PLIK_PACKAGE="github.com/root-gg/plik"
TMPDIR=$(mktemp -d)
export GOPATH="$TMPDIR"
BUILD_PATH="$GOPATH/src/$PLIK_PACKAGE"

###
# Create .plikrc file
###

export PLIKRC="$TMPDIR/.plikrc"
echo "URL = \"$URL\"" > $PLIKRC

###
# Upgrade clients
###

for RELEASE in ${RELEASES[@]}
do
    # Clean
    cd $TMPDIR
    rm -rf $TMPDIR/*
    mkdir -p $BUILD_PATH

    # Git clone
    echo "Cloning git repository at tag $RELEASE :"
    git clone -b $RELEASE --depth 1 https://$PLIK_PACKAGE $BUILD_PATH
    cd $BUILD_PATH

    # Build client
    echo "Compiling client v$RELEASE :"
    make client

    # Update client
    echo "Update client from v$RELEASE :"
    for i in $(seq 0 100) ; do echo "y" ; done | client/plik --update

    # Verify updated client
    TARGET_MD5=$(md5sum "client/plik" | awk '{print $1}')
    if [ "$SERVER_MD5" == "$TARGET_MD5" ];then
        echo -e "\nUpdate from v$RELEASE success\n"
    else
        echo -e "\nUpdate from v$RELEASE fail : md5sum mismatch server($SERVER_MD5) != target($TARGET_MD5)\n"
        exit 1
    fi
done