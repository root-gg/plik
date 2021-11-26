#!/bin/bash

set -e
cd "$(dirname "$0")"

BACKEND="minio"
CMD=$1
TEST=$2

source ../utils.sh
check_docker_connectivity

DOCKER_VERSION=${DOCKER_VERSION-latest}
DOCKER_IMAGE="minio/minio:$DOCKER_VERSION"
DOCKER_NAME="plik.minio"
DOCKER_PORT=2604

function start {
    if status ; then
        echo "ALREADY RUNNING"
    else
        pull_docker_image

        echo -e "\n - Starting $DOCKER_NAME\n"

        docker run -d -p "$DOCKER_PORT:9000" \
            -e MINIO_ACCESS_KEY="access_key" \
            -e MINIO_SECRET_KEY="access_key_secret" \
            --name "$DOCKER_NAME" "$DOCKER_IMAGE" \
            server /data

        echo "waiting for minio to start ..."
        sleep 10
        if ! status ; then
            echo "IMAGE IS NOT RUNNING"
            exit 1
        fi
    fi
}

run_cmd