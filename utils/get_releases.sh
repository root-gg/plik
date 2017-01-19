#!/bin/bash
curl -s 'https://api.github.com/repos/root-gg/plik/releases' > /tmp/releases
for i in $( seq 0 $(cat /tmp/releases | jq 'length - 1')) ; do NAME=$(cat /tmp/releases | jq -r ".[$i] | .tag_name") ; cat /tmp/releases | jq -r ".[$i] | .body" | sed 's/\r$//' | sed 's/&nbsp;/ /g'> "$NAME" ; done