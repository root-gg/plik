#!/bin/bash

set -e
cd "$(dirname "$0")"

source ./utils.sh
check_docker_connectivity

BACKENDS=(
    mariadb
    mysql
    postgres
    #mssql // There is currently an issue with reserved keyword user not being escaped from queries
    minio
    swift
)

if [[ -n "$1" ]]; then
    BACKENDS=( "$1" )
fi

TEST="$2"

for BACKEND in "${BACKENDS[@]}"
do
    if [[ ! -d $BACKEND ]];then
        echo -e "\n invalid backend $BACKEND\n"
        exit 1
    fi

    echo -e "\n - Tesing $BACKEND :\n"

    "$BACKEND/run.sh" test "$TEST"
done