#!/bin/bash

set -e
cd "$(dirname "$0")"

BACKEND="mssql"
CMD=$1
TEST=$2

source ../utils.sh
check_docker_connectivity

DOCKER_VERSION=${DOCKER_VERSION-2019-latest}
DOCKER_IMAGE="mcr.microsoft.com/mssql/server:$DOCKER_VERSION"
DOCKER_NAME="plik.mssql"
DOCKER_PORT=2605
PASSWORD="P@ssw0rd"

function start {
    if status ; then
        echo "ALREADY RUNNING"
    else
        pull_docker_image

        echo -e "\n - Starting $DOCKER_NAME\n"
        docker run -d -p "$DOCKER_PORT:1433" \
            -e "ACCEPT_EULA=Y" \
            -e "SA_PASSWORD=$PASSWORD" \
            --name "$DOCKER_NAME" "$DOCKER_IMAGE"

        echo "waiting for mssql to start ..."
        sleep 10
        if ! status ; then
            echo "IMAGE IS NOT RUNNING"
            exit 1
        fi
    fi
}

run_cmd