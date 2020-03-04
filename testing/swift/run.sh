#!/bin/bash

set -e
cd "$(dirname "$0")"

BACKEND="swift"
CMD=$1
TEST=$2

source ../utils.sh
check_docker_connectivity

# rootgg/swift is a build of https://github.com/ccollicutt/docker-swift-onlyone

DOCKER_IMAGE="rootgg/swift:latest"
DOCKER_NAME="plik.swift"
DOCKER_PORT=2603
#SWIFT_DIRECTORY="/tmp/plik.swift.tmpdir"

function start {
    if status ; then
        echo "ALREADY RUNNING"
    else
        pull_docker_image

        echo -e "\n - Starting $DOCKER_NAME\n"
        docker run -d -p "$DOCKER_PORT:8080" --name "$DOCKER_NAME" "$DOCKER_IMAGE"

        for i in $(seq 0 30)
        do
            echo "Waiting for everything to start"
            sleep 1

            DOCKER_ID=$(docker ps -q -f "name=$DOCKER_NAME")
            if [ -z "$DOCKER_ID" ]; then
                echo "Unable to get CONTAINER ID for $DOCKER_NAME"
                exit 1
            fi

            READY="0"
            if curl -s --max-time 1 "http://127.0.0.1:$DOCKER_PORT/info" >/dev/null 2>/dev/null ; then
                READY="1"
                break
            fi
        done

        if [ "$READY" == "1" ]; then
            echo -e "\n - Initializing Swift\n"
            ./initialize.sh
        else
            echo -e "\n - Unable to connect to Swift\n"
            exit 1
        fi
    fi
}

run_cmd