#!/bin/bash

set -e
. go-crosscompilation.sh

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
        md5sum ../clients/$DIR/$EXECUTABLE | awk '{print $1}' > ../clients/$DIR/MD5SUM
    done

elif [ "$1" == "debs" ]; then

    if ! `hash dpkg-deb 2> /dev/null`; then
        echo "Please install dpkg-deb to build debian packages."
        exit 1
    fi

    echo "Making client debian packages..."

    VERSION=$(cat VERSION)
    CLIENTS_DIR=clients
    DEBS_DST_DIR=debs/client 
    DEB_CFG_DIR=client/build/deb/DEBIAN

    # Building packages
    for ARCH in amd64 i386 armhf ; do
        DEBROOT=$DEBS_DST_DIR/$ARCH
        mkdir -p $DEBROOT/usr/local/bin
        cp -R $DEB_CFG_DIR $DEBROOT
        sed -i -e "s/##ARCH##/$ARCH/g" $DEBROOT/DEBIAN/control 
        sed -i -e "s/##VERSION##/$VERSION/g" $DEBROOT/DEBIAN/control

        if [ $ARCH == 'i386' ]; then
            cp clients/linux-386/plik $DEBROOT/usr/local/bin
        elif [ $ARCH == 'armhf' ]; then
            cp clients/linux-arm/plik $DEBROOT/usr/local/bin
        else
            cp clients/linux-$ARCH/plik $DEBROOT/usr/local/bin
        fi

        echo " - Building $ARCH package in $DEBS_DST_DIR/plik-$ARCH.deb"
        dpkg-deb --build $DEBROOT $DEBS_DST_DIR/plik-$ARCH.deb > /dev/null
    done
fi
