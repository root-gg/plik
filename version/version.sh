#!/bin/bash



set -e

FILE=$(dirname $0)/version.go
if [[ ! -f "$FILE" ]]; then
    echo "$FILE not found"
    exit 1
fi

cat $FILE | sed -n 's/.*version = "\(.*\)"/\1/p'