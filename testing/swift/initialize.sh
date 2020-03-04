#!/bin/bash

set -e
cd "$(dirname "$0")"

URL="http://127.0.0.1:2603"
CNAME="plik"
USER="test:tester"
PASSWORD="testing"

echo " - Check Swift API"

if ! curl -s $URL/info > /dev/null ; then
    echo "Can't reach swift api @ $URL"
    exit 1
fi

echo " - Get auth token"

TOKEN=$(curl -v -H "X-Storage-User: $USER" -H "X-Storage-Pass: $PASSWORD" "$URL/auth/v1.0" 2>&1 | sed -n 's/^.*X-Auth-Token: \(.*\)$/\1/p')

if [ -z "$TOKEN" ]; then
    echo "Unable to get auth token"
    exit 1
fi

URL="$URL/v1/AUTH_test"

echo " - Create container $CNAME"

curl -s -X PUT -H "X-Auth-Token: $TOKEN" "$URL/$CNAME" > /dev/null

echo " - List containers"

curl -s -X GET -H "X-Auth-Token: $TOKEN" "$URL" | grep $CNAME > /dev/null

echo " - Put testobject"

curl -s -X PUT -H "X-Auth-Token: $TOKEN" -T $0 "$URL/$CNAME/testobject" > /dev/null

echo " - Check container content"

curl -s -X GET -H "X-Auth-Token: $TOKEN" "$URL/$CNAME" | grep testobject > /dev/null

echo " - Retrieve testobject"

curl -s -X GET -H "X-Auth-Token: $TOKEN" -o /dev/null "$URL/$CNAME/testobject" > /dev/null

echo " - Delete testobject"

curl -s -X DELETE -H "X-Auth-Token: $TOKEN" "$URL/$CNAME/testobject" > /dev/null

echo "Everything looks good. Enjoy ;)"