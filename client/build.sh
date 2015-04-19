#!/bin/bash

set -e
. client/go-crosscompilation.sh

if [ "$1" == "env" ]; then
    PLATFORMS_COUNT=$(echo $PLATFORMS | sed -e "s/\ /\n/g" | wc -l)
    PLATFORMS_OK_COUNT=$(ls -1 $(go env GOROOT)/pkg | grep -v -E "obj|tool|_race" | wc -l)

    if [ "$PLATFORMS_COUNT" != "$PLATFORMS_OK_COUNT" ]; then
        go-crosscompile-build-all
    else
        echo "The environement seems to be already ok :)"
    fi

elif [ "$1" == "clients" ]; then
    cd client
    echo "Compiling client..."

    for PLATFORM in $PLATFORMS; do
        GOOS=${PLATFORM%/*}
        GOARCH=${PLATFORM#*/}
        DIR=${GOOS}-${GOARCH}
        EXECUTABLE="plik"

        if [ "$GOOS" == "windows" ]; then
            EXECUTABLE="plik.exe"
        fi

        mkdir -p ../clients/$DIR
        echo " - go-${GOOS}-${GOARCH} build -o ../clients/$DIR/$EXECUTABLE"
        go-${GOOS}-${GOARCH} build -o ../clients/$DIR/$EXECUTABLE
    done
fi
