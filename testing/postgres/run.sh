#!/bin/bash

set -e
cd "$(dirname "$0")"

BACKEND="postgres"
CMD=$1
TEST=$2

source ../utils.sh
check_docker_connectivity

DOCKER_IMAGE="postgres:latest"
DOCKER_NAME="plik.postgres"
DOCKER_PORT=2602
PASSWORD="password"

function start {
    if status ; then
        echo "ALREADY RUNNING"
    else
        pull_docker_image

        echo -e "\n - Starting $DOCKER_NAME\n"
        docker run -d -p "$DOCKER_PORT:5432" -e POSTGRES_PASSWORD="$PASSWORD" --name "$DOCKER_NAME" "$DOCKER_IMAGE"

        sleep 1
        if ! status ; then
            echo "IMAGE IS NOT RUNNING"
            exit 1
        fi
    fi
}

run_cmd