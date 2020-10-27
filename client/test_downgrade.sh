#!/bin/bash



set -e
cd $(dirname $0)

###
# Try to downgrade cli client to older target release
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
# Build current client
###

echo "Builinding current plik client"
go build -o plik
CLIENT=$(readlink -f plik)

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
# Downgrade client
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

    # Build server and clients
    echo "Compiling server and clients v$RELEASE :"

    if grep "^deps:" Makefile ; then
        make deps
    fi

    # 1.0.1 fix
    if [ "$RELEASE" == "1.0.1" ] ; then
        ( cd server && go get -v )
    fi

    make clients server

    # Run server
    echo "Start server v$RELEASE :"
    (cd server && ./plikd)&

    # Verify that server is running
    sleep 1
    if ! curl "$URL/version" 2>/dev/null | grep version > /dev/null ; then
        echo "Plik server did not start @ $URL"
        exit 1
    fi

    # Try to downgrade client
    cp $CLIENT ./plik
    for i in $(seq 0 100) ; do echo "y" ; done | ./plik --update

    # Verify updated client
    SERVER_MD5=$(md5sum "clients/$(go env GOOS)-$(go env GOARCH)/plik" | awk '{print $1}')
    CLIENT_MD5=$(md5sum ./plik | awk '{print $1}')
    if [ "$SERVER_MD5" == "$CLIENT_MD5" ];then
        echo -e "\nUpdate to v$RELEASE success\n"
    else
        echo -e "\nUpdate to v$RELEASE fail : md5sum mismatch server($SERVER_MD5) != target($TARGET_MD5)\n"
        exit 1
    fi

    shutdown
done