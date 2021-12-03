#!/bin/bash

set -e
cd "$(dirname "$0")"

BACKEND="mariadb"
CMD=$1
TEST=$2

source ../utils.sh
check_docker_connectivity

DOCKER_VERSION=${DOCKER_VERSION-latest}
DOCKER_IMAGE="mariadb:$DOCKER_VERSION"
DOCKER_NAME="plik.mariadb"
DOCKER_PORT=2601

function start {
    if status ; then
        echo "ALREADY RUNNING"
    else
        pull_docker_image

        echo -e "\n - Starting $DOCKER_NAME\n"
        docker run -d -p "$DOCKER_PORT:3306" \
            -e MYSQL_ROOT_PASSWORD="password" \
            -e MYSQL_DATABASE="plik" \
            -e MYSQL_USER="plik" \
            -e MYSQL_PASSWORD="password" \
            --name "$DOCKER_NAME" "$DOCKER_IMAGE"

        echo "waiting for mariadb to start ..."
        sleep 10
        if ! status ; then
            echo "IMAGE IS NOT RUNNING"
            exit 1
        fi
    fi
}

run_cmd