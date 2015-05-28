#!/bin/bash

#
## Plik - Simple bash uploading script
#

set -e

#
## Funcs
#
green='\e[0;32m'
endColor='\e[0m'
function jsonValue() {
    KEY=$1
    num=$2
    awk -F"[,:}]" '{for(i=1;i<=NF;i++){if($i~/\042'$KEY'\042/){print $(i+1)}}}' | tr -d '"' | sed -n ${num}p
}
function qecho(){ 
    if [ "$QUIET" == false ]; then echo $@; fi
}
function generatePassphrase() {
    < /dev/urandom tr -dc A-Za-z0-9 | head -c${1:-32};echo;
}
function setTtl() {
    unit="${1: -1}"
    value="${1:: -1}"
    case "$unit" in
        "m") TTL=$(expr $value \* 60);;
        "h") TTL=$(expr $value \* 3600);;
        "d") TTL=$(expr $value \* 86400);;
        *)   TTL=$1;;
    esac
    return
}

#
## Vars
#
PLIK_URL=${PLIK_URL-"http://127.0.0.1:8080"}
PASSPHRASE=""
QUIET=false
SECURE=false
ARCHIVE=false
ONESHOT=false
REMOVABLE=false
TTL=0

#
## Parse arguments
#
declare -a files

while [ $# -gt 0 ] ; do
    case "$1" in
        -q) QUIET=true      ; shift ;;
        -s) SECURE=true     ; shift ;;
        -a) ARCHIVE=true    ; shift ;;
        -o) ONESHOT=true    ; shift ;;
        -r) REMOVABLE=true  ; shift ;;
        -t)                   shift ; setTtl $1       ; shift ;;
        -p) SECURE=true     ; shift ; PASSPHRASE="$1" ; shift ;;
        --) shift ;;
        -*) qecho "bad option '$1'" ; exit 1 ;;
        *) files=("${files[@]}" "$1") ; shift ;;
    esac
done

if [ "${#files[@]}" == 0 ]; then
    qecho "No files specified !"
    exit 1
fi
if [ -e "$HOME/.plikrc" ]; then
    URL=$(grep Url ~/.plikrc | grep -Po '(http[^\"]*)')

    if [ "$URL" != "" ]; then
        PLIK_URL=$URL
    fi
fi


#
## Create new upload
#

qecho -e "Creating upload on $PLIK_URL...\n"
OPTIONS="{ \"OneShot\" : $ONESHOT, \"Removable\" : $REMOVABLE, \"Ttl\" : $TTL }"
NEW_UPLOAD_RESP=$(curl -s -X POST -d "$OPTIONS" ${PLIK_URL}/upload)
UPLOAD_ID=$(echo $NEW_UPLOAD_RESP | jsonValue id)
UPLOAD_TOKEN=$(echo $NEW_UPLOAD_RESP | jsonValue uploadToken)
qecho -e " --> ${green}$PLIK_URL/#/?id=$UPLOAD_ID${endColor}\n"


#
## Test if we have to archive
#
for file in "${files[@]}"
do
    if [ -d "$file" ]; then
        ARCHIVE=true
        break
    fi
done

if [ "$ARCHIVE" == true ]; then

    ARCHIVE_NAME="archive.tar.gz"
    if [ "${#files[@]}" == 1 ]; then
        ARCHIVE_NAME="$(basename ${files[0]}).tar.gz"
    fi

    ARCHIVE_CMD="tar --create --gzip ${files[@]}"

    unset files
    declare -a files
    files[0]=$ARCHIVE_DST
fi


#
## Upload files
#

qecho -e "Uploading files...\n"

for FILE in "${files[@]}"
do
    STDIN=false

    FILENAME=$FILE
    if [[ "$FILE" == *\/* ]]; then
        FILENAME=$(basename $FILE)
    fi

    UPLOAD_COMMAND=""
    if [ "$ARCHIVE" == true ]; then
        UPLOAD_COMMAND+="$ARCHIVE_CMD | "
        FILENAME=$ARCHIVE_NAME
        STDIN=true
    fi

    if [ "$SECURE" == true ]; then
        if [ "$PASSPHRASE" == "" ]; then
            PASSPHRASE=$(generatePassphrase)
        fi

        UPLOAD_COMMAND+="openssl aes-256-cbc -e -pass pass:$PASSPHRASE "
        if [ "$ARCHIVE" == false ]; then
            UPLOAD_COMMAND+="-in $FILE "
        fi

        UPLOAD_COMMAND+=" | "
        STDIN=true
    fi

    if [ "$STDIN" == true ]; then
        UPLOAD_COMMAND+="curl -s -X POST --header \"X-UploadToken: $UPLOAD_TOKEN\" -F \"file=@-;filename=$FILENAME\" $PLIK_URL/upload/$UPLOAD_ID/file"
    else
        UPLOAD_COMMAND+="curl -s -X POST --header \"X-UploadToken: $UPLOAD_TOKEN\" -F \"file=@$FILE;filename=$FILENAME\" $PLIK_URL/upload/$UPLOAD_ID/file"
    fi

    FILE_RESP=$(eval $UPLOAD_COMMAND)
    FILE_ID=$(echo $FILE_RESP | jsonValue id)
    FILE_MD5=$(echo $FILE_RESP | jsonValue fileMd5)
    FILE_NAME=$(echo $FILE_RESP | jsonValue fileName)
    FILE_STATUS=$(echo $FILE_RESP | jsonValue status)
    FILE_URL="$PLIK_URL/file/$UPLOAD_ID/$FILE_ID/$FILE_NAME"

    # Compute get command
    COMMAND="curl -s $FILE_URL"

    if [ "$SECURE" == true ]; then
        COMMAND+=" | openssl aes-256-cbc -d -pass \"pass:$PASSPHRASE\""
    fi

    if [ "$ARCHIVE" == true ]; then
        COMMAND+=" | tar zxvf -"
    else
        COMMAND+=" > $FILE_NAME"
    fi

    # Output
    if [ "$QUIET" == true ]; then
        echo "$FILE_URL"
    else
        echo "$COMMAND"
    fi
done
qecho

