#!/bin/bash

###
# The MIT License (MIT)
#
# Copyright (c) <2015>
# - Mathieu Bodjikian <mathieu@bodjikian.fr>
# - Charles-Antoine Mathieu <skatkatt@root.gg>
#
# Permission is hereby granted, free of charge, to any person obtaining a copy
# of this software and associated documentation files (the "Software"), to deal
# in the Software without restriction, including without limitation the rights
# to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
# copies of the Software, and to permit persons to whom the Software is
# furnished to do so, subject to the following conditions:
#
# The above copyright notice and this permission notice shall be included in
# all copies or substantial portions of the Software.
#
# THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
# IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
# FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
# AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
# LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
# OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
# THE SOFTWARE.
###

set -e
cd "$(dirname "$0")"

# rootgg/swift is a build of https://github.com/ccollicutt/docker-swift-onlyone

DOCKER_IMAGE="rootgg/swift:latest"
DOCKER_NAME="plik.swift"
DOCKER_PORT=2691
SWIFT_DIRECTORY="/tmp/plik.swift.tmpdir"

function start {
    if status ; then
        echo "ALREADY RUNNING"
        exit 0
    else
        echo -e "\n - Pulling $DOCKER_IMAGE\n"
        docker pull "$DOCKER_IMAGE"
        if docker ps -a -f name="$DOCKER_NAME" | grep "$DOCKER_NAME" > /dev/null ; then
            docker rm -f "$DOCKER_NAME"
        fi

        echo -e "\n - Cleaning swift directory $SWIFT_DIRECTORY\n"

        test -d "$SWIFT_DIRECTORY" && rm -rf "$SWIFT_DIRECTORY"
        mkdir -p "$SWIFT_DIRECTORY"

        echo -e "\n - Starting $DOCKER_NAME\n"
        docker run -d -p "$DOCKER_PORT:8080" --name "$DOCKER_NAME" -v "/tmp/plik.swift.tmpdir:/srv" "$DOCKER_IMAGE"

        echo -e "\n - Sleeping a bit ...\n"
        sleep 5

        echo -e "\n - Initializing Swift storage\n"
        bash -c "./initialize.sh"
    fi
}

function stop {
    if status ; then
        echo -e "\n - Removing $DOCKER_NAME\n"
        docker rm -f "$DOCKER_NAME"
    else
        echo "NOT RUNNING"
        exit 0
    fi
}

function status {
    docker ps -f name="$DOCKER_NAME" | grep "$DOCKER_NAME" > /dev/null
}

case "$1" in
  start)
    start
    ;;
  stop)
    stop
    ;;
  restart)
    stop
    start
    ;;
  status)
    if status ; then
        docker ps -f name="$DOCKER_NAME"
    else
        echo "NOT RUNNING"
    fi
    ;;
  *)
	echo "Usage: $0 {start|stop|restart|status}"
	exit 1
esac

exit 0