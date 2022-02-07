#! /bin/bash

set -e

ROOT=$(realpath ../..)

function check_docker_connectivity {
    if docker version >/dev/null 2>/dev/null ; then
        true
    else
        echo "Cannot connect to docker daemon."
        if [[ $EUID -ne 0 ]]; then
            echo "Maybe you need to run this as root."
        fi
        false
    fi
}

function pull_docker_image {
    echo -e "\n - Pulling $DOCKER_IMAGE\n"
    docker pull "$DOCKER_IMAGE"
    if docker ps -a -f name="$DOCKER_NAME" | grep "$DOCKER_NAME" > /dev/null ; then
        docker rm -f "$DOCKER_NAME"
    fi
}

function stop {
    if status ; then
        echo -e "\n - Removing $DOCKER_NAME\n"
        docker rm -f "$DOCKER_NAME" >/dev/null
    else
        echo "NOT RUNNING"
    fi
}

function status {
    docker ps -f name="$DOCKER_NAME" | grep "$DOCKER_NAME" > /dev/null
}

function run_tests {
    BACKEND="$1"
    TEST="$2"

    if [[ -z "$BACKEND" ]]; then
        echo "missing backend"
        return 1
    fi

    PLIKD_CONFIG="$ROOT/testing/$BACKEND/plikd.cfg"
    export PLIKD_CONFIG
    export BACKEND

    if [[ -z "$TEST" ]]; then
        ( cd "$ROOT/plik" && GORACE="halt_on_error=1" go test -count=1 -race ./... )
        ( cd "$ROOT/server/server" && GORACE="halt_on_error=1" go test -count=1 -race ./... )

        # Run metadata backend tests
        if [[ "$BACKEND" == "postgres" ]] || [[ "$BACKEND" == "mariadb" ]] || [[ "$BACKEND" == "mysql" ]] || [[ "$BACKEND" == "mssql" ]]; then
            ( cd "$ROOT/server/metadata" && GORACE="halt_on_error=1" go test -count=1 -race ./... )
        fi
    else
        ( cd "$ROOT/plik" && GORACE="halt_on_error=1" go test -count=1 -v -race -run "$TEST" )
        ( cd "$ROOT/server/server" && GORACE="halt_on_error=1" go test -count=1 -race ./... )

        # Run metadata backend test
        if [[ "$BACKEND" == "postgres" ]] || [[ "$BACKEND" == "mariadb" ]] || [[ "$BACKEND" == "mysql" ]] || [[ "$BACKEND" == "mssql" ]]; then
            ( cd "$ROOT/server/metadata" && GORACE="halt_on_error=1" go test -count=1 -v -race -run "$TEST" )
        fi
    fi
}

function run_cmd {
    case "$CMD" in
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
        test)
          stop
          start
          run_tests "$BACKEND" "$TEST"
          ;;
        status)
          if status ; then
              docker ps -f name="$DOCKER_NAME"
          else
              echo "NOT RUNNING"
          fi
          ;;
        *)
        echo "Usage: $0 {start|stop|restart|status|test}"
        exit 1
    esac
}