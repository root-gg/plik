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
    sed -e "s/,/\n/g" | sed -e "s/[\"{}]//g" | grep $KEY | cut -d ":" -f2-
}

function qecho(){ 
    if [ "$QUIET" == false ]; then echo "$@"; fi
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
PLIK_TOKEN=${PLIK_TOKEN-""}
QUIET=false
SECURE=false
PASSPHRASE=""
ARCHIVE=false
ONESHOT=false
REMOVABLE=false
TTL=0

#
## Read ~/.plikrc file
#

PLIKRC=${PLIKRC-"$HOME/.plikrc"}
if [ ! -f "$PLIKRC" ]; then
    PLIKRC="/etc/plik/plikrc"
fi

if [ -f "$PLIKRC" ]; then
    # Evironment variable takes precedence over plikrc file
    if [ "$PLIK_URL" == "" ]; then
      URL=$(grep URL $PLIKRC | grep -Po '(http[^\"]*)')
      if [ "$URL" != "" ]; then
          PLIK_URL=$URL
      fi
    fi
    TOKEN=$(grep Token $PLIKRC | sed -n 's/^.*"\(.*\)".*$/\1/p' )
    if [ "$TOKEN" != "" ]; then
        PLIK_TOKEN=$TOKEN
    fi
fi

# Default URL to local instance
PLIK_URL=${PLIK_URL-"http://127.0.0.1:8080"}

#
## Parse arguments
#

declare -a files
while [ $# -gt 0 ] ; do
    case "$1" in
        -u)                   shift ; PLIK_URL="$1"   ; shift ;;
        -T)                   shift ; PLIK_TOKEN="$1" ; shift ;;
        -o) ONESHOT=true    ; shift ;;
        -r) REMOVABLE=true  ; shift ;;
        -t)                   shift ; setTtl $1       ; shift ;;
        -a) ARCHIVE=true    ; shift ;;
        -s) SECURE=true     ; shift ;;
        -p) SECURE=true     ; shift ; PASSPHRASE="$1" ; shift ;;
        -q) QUIET=true      ; shift ;;
        --) shift ;;
        -*) qecho "bad option '$1'" ; exit 1 ;;
        *) files=("${files[@]}" "$1") ; shift ;;
    esac
done

if [ "${#files[@]}" == 0 ]; then
    qecho "No files specified !"
    exit 1
fi

#
## Create new upload
#

if [ "$PLIK_TOKEN" != "" ]; then
    AUTH_TOKEN_HEADER="-H \"X-PlikToken: $PLIK_TOKEN\""
fi

OPTIONS="{ \"OneShot\" : $ONESHOT, \"Removable\" : $REMOVABLE, \"Ttl\" : $TTL }"
qecho -e "Create new upload on $PLIK_URL...\n"

CREATE_UPLOAD_CMD="curl -s -X POST $AUTH_TOKEN_HEADER -d '$OPTIONS' ${PLIK_URL}/upload"
NEW_UPLOAD_RESP=$(eval $CREATE_UPLOAD_CMD)
UPLOAD_ID=$(echo $NEW_UPLOAD_RESP | jsonValue id)

DOWNLOAD_DOMAIN=$(echo $NEW_UPLOAD_RESP | jsonValue downloadDomain)
if [ "$DOWNLOAD_DOMAIN" == "" ]; then
  DOWNLOAD_DOMAIN=$PLIK_URL
fi

# Handle error
if [ "$UPLOAD_ID" == "" ]; then
    ERROR_MSG=$(echo $NEW_UPLOAD_RESP | jsonValue message)
    if [ "$ERROR_MSG" != "" ]; then
        echo $ERROR_MSG
    elif [ "$NEW_UPLOAD_RESP" != "" ]; then
        echo $NEW_UPLOAD_RESP
    fi
    exit 1
fi

UPLOAD_TOKEN=$(echo $NEW_UPLOAD_RESP | jsonValue uploadToken)
UPLOAD_TOKEN_HEADER="-H \"X-UploadToken: $UPLOAD_TOKEN\""

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
        UPLOAD_COMMAND+="curl -s -X POST $AUTH_TOKEN_HEADER $UPLOAD_TOKEN_HEADER -F \"file=@-;filename=$FILENAME\" $PLIK_URL/file/$UPLOAD_ID"
    else
        UPLOAD_COMMAND+="curl -s -X POST $AUTH_TOKEN_HEADER $UPLOAD_TOKEN_HEADER -F \"file=@$FILE;filename=$FILENAME\" $PLIK_URL/file/$UPLOAD_ID"
    fi

    FILE_RESP=$(eval $UPLOAD_COMMAND)
    FILE_ID=$(echo $FILE_RESP | jsonValue id)

    # Handle error
    if [ "$FILE_ID" == "" ]; then
        ERROR_MSG=$(echo $FILE_RESP | jsonValue message)
        if [ "$ERROR_MSG" != "" ]; then
            echo $ERROR_MSG
        elif [ "$FILE_RESP" != "" ]; then
            echo $FILE_RESP
        fi
        exit 1
    fi

    FILE_MD5=$(echo $FILE_RESP | jsonValue fileMd5)
    FILE_NAME=$(echo $FILE_RESP | jsonValue fileName)
    FILE_STATUS=$(echo $FILE_RESP | jsonValue status)
    FILE_URL="$DOWNLOAD_DOMAIN/file/$UPLOAD_ID/$FILE_ID/$FILE_NAME"

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

