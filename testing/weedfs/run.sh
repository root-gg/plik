#!/bin/bash

set -e
cd "$(dirname "$0")"

DOCKER_COMPOSE_FILE="docker-compose.yml"
DOCKER_NAMES=( "plik.weedfs.master" "plik.weedfs.volume" )

function start {
    docker-compose -f "$DOCKER_COMPOSE_FILE" up -d
}

function stop {
    docker-compose -f "$DOCKER_COMPOSE_FILE" down
}

function status {
    for name in "${DOCKER_NAMES[@]}"
    do
        if docker ps -f name="$name" | grep "$name" > /dev/null ; then
            echo "$name is RUNNING"
        else
            echo "$name is NOT RUNNING"
        fi
    done
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
    status
    ;;
  *)
	echo "Usage: $0 {start|stop|restart|status}"
	exit 1
esac

exit 0