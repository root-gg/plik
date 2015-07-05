#!/bin/bash

set -e
. go-crosscompilation.sh

VERSION=$(cat VERSION)

if [ "$1" == "env" ]; then
    PLATFORMS_COUNT=$(echo $PLATFORMS | sed -e "s/\ /\n/g" | wc -l)
    PLATFORMS_OK_COUNT=$(ls -1 $(go env GOROOT)/pkg | grep -v -E "obj|tool|_race" | wc -l)

    if [ "$PLATFORMS_COUNT" != "$PLATFORMS_OK_COUNT" ]; then
        go-crosscompile-build-all
    else
        echo "The environement seems to be already ok :)"
    fi

elif [ "$1" == "servers" ]; then

    cd server
    echo "Compiling server..."

    sed -i -e "s/##VERSION##/$VERSION/g" common/config.go

    for PLATFORM in $PLATFORMS; do
        GOOS=${PLATFORM%/*}
        GOARCH=${PLATFORM#*/}
        DIR=${GOOS}-${GOARCH}
        EXECUTABLE="plikd"

        if [ "$GOOS" == "windows" ]; then
            EXECUTABLE="plikd.exe"
        fi

        mkdir -p ../servers/$DIR
        echo " - go-${GOOS}-${GOARCH} build -o ../servers/$DIR/$EXECUTABLE"
        go-${GOOS}-${GOARCH} build -o ../servers/$DIR/$EXECUTABLE
        md5sum ../servers/$DIR/$EXECUTABLE | awk '{print $1}' > ../servers/$DIR/MD5SUM
    done

    git checkout common/config.go

elif [ "$1" == "debs" ]; then

    if ! `hash dpkg-deb 2> /dev/null`; then
        echo "Please install dpkg-deb to build debian packages."
        exit 1
    fi

    echo "Making server debian packages..."

    SERVERS_DIR=servers
    DEBS_DST_DIR=debs/server
    DEB_CFG_DIR=server/build/deb/DEBIAN

    # Building packages
    for ARCH in amd64 i386 armhf ; do
        DEBROOT=$DEBS_DST_DIR/$ARCH
        mkdir -p $DEBROOT/usr/local/bin
        mkdir -p $DEBROOT/etc/init.d

        cp -R $DEB_CFG_DIR $DEBROOT
        cp -R server/plikd.cfg $DEBROOT/etc/plikd.cfg
        cp -R server/plikd.init $DEBROOT/etc/init.d/plikd
        chmod +x $DEBROOT/etc/init.d/plikd

        sed -i -e "s/##ARCH##/$ARCH/g" $DEBROOT/DEBIAN/control
        sed -i -e "s/##VERSION##/$VERSION/g" $DEBROOT/DEBIAN/control

        if [ $ARCH == 'i386' ]; then
            cp servers/linux-386/plikd $DEBROOT/usr/local/bin
        elif [ $ARCH == 'armhf' ]; then
            cp servers/linux-arm/plikd $DEBROOT/usr/local/bin
        else
            cp servers/linux-$ARCH/plikd $DEBROOT/usr/local/bin
        fi

        echo " - Building $ARCH package in $DEBS_DST_DIR/plikd-$ARCH.deb"
        dpkg-deb --build $DEBROOT $DEBS_DST_DIR/plikd-$ARCH.deb > /dev/null
    done
fi
