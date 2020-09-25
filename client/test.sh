#!/bin/bash

set -e
export SHELLOPTS
export ENABLE_RACE_DETECTOR=1

ORIGIN=$(dirname $(readlink -f $0))
cd "$ORIGIN/.."

###
# Check that no plikd is running
###

URL="http://127.0.0.1:8080"
if curl "$URL/version" 2>/dev/null | grep version >/dev/null 2>&1 ; then
    echo "A plik instance is running @ $URL"
    exit 1
fi

###
# Build server and clients
###

echo "Build current server and clients"
make clean client server
CLIENT=$(readlink -f client/plik)

###
# Create temporary environement
###

TMPDIR=$(mktemp -d)
SERVER_LOG="$TMPDIR/server_log"
CLIENT_LOG="$TMPDIR/client_log"

SPECIMEN="$TMPDIR/SPECIMEN"
    cat > $SPECIMEN << EOF
Lorem ipsum dolor sit amet, eu munere invenire est, in vel liber salutatus, id eum atqui iisque. Ut eam semper possim ullamcorper. Quodsi complectitur an mea. Oratio pertinacia ius ea, duo quem dolorum omittam at. Vix eu idque admodum, quem animal eam et.
Cu eum ullum constituto theophrastus, te eam nihil ignota iudicabit. Pri cu minim voluptatum inciderint. Ne nec inani latine. Ei voluptua splendide sit.
At vix clita aliquam docendi. Ex eum utroque dignissim theophrastus, nullam facete vituperatoribus his ne, mei ad delectus constituto. Qui ne euripidis liberavisse, te per labores lucilius, eu ferri convenire mea. Ius dico ceteros feugait eu, cu nisl magna option pro, cu agam veritus aliquando has. At pro mandamus qualisque, eu vis nostro aeterno.
Erat vulputate intellegebat an nam, te reque atomorum molestiae eos. Illud corpora incorrupte est cu, nullam audiam id per, mel et dicta legimus suscipiantur. Ad simul perfecto per, his veri legimus te. Cum aeque dissentiet et, atomorum aliquando effctx := newTestingContext(config)endi ex vix, his ei soleat omnium impetus.
Sed electram dignissim reformidans ut. In vim graeco torquatos pertinacia, duis tamquam duo id. Et viderer debitis vocibus quo, ea vero movet atomorum pri. Atqui delicatissimi an vis, amet deseruisse ius et. Eos rationibus scriptorem ex, vim meis eirmod consequuntur in.
EOF

###
# Run server
###

echo -n "Start Plik server : "

PLIKD_CONFIG=${PLIKD_CONFIG-../server/plikd.cfg}
echo "PLIKD_CONFIG=$PLIKD_CONFIG"

(cd server && ./plikd --config $PLIKD_CONFIG > $SERVER_LOG 2>&1) >/dev/null 2>&1 &

# Verify that server is running
sleep 1
if curl "$URL/version" 2>/dev/null | grep version >/dev/null 2>&1 ; then
    echo "Plik server is running"
else
    echo "Plik server is not running"
    cat $SERVER_LOG
    exit 1
fi

# Defer server shutdown
function shutdown {
    EXITCODE=$?
    if [ $EXITCODE -ne 0 ]; then
        echo -e "FAIL\n"
        if [ -f $SERVER_LOG ]; then
            echo "last server logs :"
            cat $SERVER_LOG
            echo -e "\n"
        fi
        if [ -f $CLIENT_LOG ]; then
            echo "last client logs :"
            echo $UPLOAD_CMD
            cat $CLIENT_LOG
            echo -e "\n"
        fi
    fi
    PID=$(ps a | grep plikd | grep -v grep | awk '{print $1}')
    if [ "x$PID" != "x" ];then
        echo "Shutting down plik server"
        kill $PID
        sleep 1
        PID=$(ps a | grep plikd | grep -v grep | awk '{print $1}')
        if [ "x$PID" != "x" ];then
            kill -9 $PID
        fi
    fi
}
trap shutdown EXIT

###
# Helpers
###

function before
{
    cd $TMPDIR

    rm -rf $TMPDIR/upload
    rm -rf $TMPDIR/download
    mkdir $TMPDIR/upload
    mkdir $TMPDIR/download

    # Reset .plikrc file
    export PLIKRC="$TMPDIR/.plikrc"
    echo "URL = \"$URL\"" > $PLIKRC
    if [ "$SECURE" != "" ]; then
        echo "Secure = true" >> $PLIKRC
    fi

    unset SECURE
    unset UPLOAD_CMD
    unset UPLOAD_ID
    unset UPLOAD_OPTS
    unset UPLOAD_USER
    unset UPLOAD_PWD

    truncate -s 0 $SERVER_LOG
    truncate -s 0 $CLIENT_LOG
}

# Upload files in the upload directory
function upload {
    cd $TMPDIR/upload
    UPLOAD_CMD="$CLIENT $@ *"
    eval "$UPLOAD_CMD" >$CLIENT_LOG 2>&1
}

# Upload files in the upload directory
function uploadStdin {
    file=$1
    shift
    UPLOAD_CMD="cat $file | $CLIENT $@"
    eval "$UPLOAD_CMD" >$CLIENT_LOG 2>&1
}

# Get upload options from server api
function uploadOpts {
    UPLOAD_ID=$( cat $CLIENT_LOG | sed -n 's/^.*http.*\/\?id=\(.*\)$/\1/p' )
    local CURL_CMD="curl -s"
    if [ "$UPLOAD_USER" != "" ] && [ "$UPLOAD_PWD" != "" ]; then
        CURL_CMD="$CURL_CMD -u $UPLOAD_USER:$UPLOAD_PWD"
    fi
    CURL_CMD="$CURL_CMD $URL/upload/$UPLOAD_ID"
    UPLOAD_OPTS=$( eval "$CURL_CMD" 2>/dev/null | python -m json.tool )
}

# Download files by running the output cmds
function download {
    cd $TMPDIR/download
    local COMMANDS=$(cat $CLIENT_LOG | grep curl)
    local IFS='\n'
    for COMMAND in "$COMMANDS"
    do
        eval "$COMMAND" >/dev/null 2>/dev/null
    done
}

# Compare upload and download directories
function check {
    diff --brief -r $TMPDIR/upload $TMPDIR/download
}

###
# Tests
###

echo -n " - help : "

before
$CLIENT --help >$CLIENT_LOG 2>&1
grep 'Usage' $CLIENT_LOG >/dev/null 2>/dev/null

before
$CLIENT -h >$CLIENT_LOG 2>&1
grep 'Usage' $CLIENT_LOG >/dev/null 2>/dev/null

echo "OK"

#---------------------------------------------

echo -n " - version : "

before
$CLIENT --version >$CLIENT_LOG 2>&1
grep "Plik client v" $CLIENT_LOG >/dev/null 2>/dev/null

before
$CLIENT -v >$CLIENT_LOG 2>&1
grep "Plik client v" $CLIENT_LOG >/dev/null 2>/dev/null

echo "OK"

#---------------------------------------------

#---------------------------------------------

echo -n " - info : "

before
$CLIENT --info >$CLIENT_LOG 2>&1
grep "Plik client version :" $CLIENT_LOG >/dev/null 2>/dev/null
grep "Plik server url :" $CLIENT_LOG >/dev/null 2>/dev/null
grep "Plik server version :" $CLIENT_LOG >/dev/null 2>/dev/null
grep "Plik server configuration :" $CLIENT_LOG >/dev/null 2>/dev/null

before
$CLIENT -i >$CLIENT_LOG 2>&1
grep "Plik client version :" $CLIENT_LOG >/dev/null 2>/dev/null
grep "Plik server url :" $CLIENT_LOG >/dev/null 2>/dev/null
grep "Plik server version :" $CLIENT_LOG >/dev/null 2>/dev/null
grep "Plik server configuration :" $CLIENT_LOG >/dev/null 2>/dev/null

echo "OK"

#---------------------------------------------

echo -n " - debug : "

before
$CLIENT -d $SPECIMEN >$CLIENT_LOG 2>&1
grep "Arguments" $CLIENT_LOG >/dev/null 2>/dev/null
grep "Configuration" $CLIENT_LOG >/dev/null 2>/dev/null

before
$CLIENT --debug $SPECIMEN >$CLIENT_LOG 2>&1
grep "Arguments" $CLIENT_LOG >/dev/null 2>/dev/null
grep "Configuration" $CLIENT_LOG >/dev/null 2>/dev/null

echo "OK"

#---------------------------------------------

echo -n " - single file : "
before
cp $SPECIMEN $TMPDIR/upload/FILE1
upload && download && check
echo "OK"

#---------------------------------------------

echo -n " - single file with custom name : "
before
cp $SPECIMEN $TMPDIR/upload/FILE1
upload --name CUSTOM
mv $TMPDIR/upload/FILE1 $TMPDIR/upload/CUSTOM
download && check

before
cp $SPECIMEN $TMPDIR/upload/FILE1
upload -n CUSTOM
mv $TMPDIR/upload/FILE1 $TMPDIR/upload/CUSTOM
download && check

echo "OK"

#---------------------------------------------

echo -n " - multiple files : "
before
cp $SPECIMEN $TMPDIR/upload/FILE1
cp $SPECIMEN $TMPDIR/upload/FILE2
upload && download && check
echo "OK"

#---------------------------------------------

echo -n " - stdin : "
before
cp $SPECIMEN $TMPDIR/upload/FILE1

uploadStdin "$TMPDIR/upload/FILE1" --name "FILE1" && download && check
echo "OK"

#---------------------------------------------

echo -n " - disable stdin : "
before
cp $SPECIMEN $TMPDIR/upload/FILE1

# Use temporary keyring
cat >$PLIKRC << EOF
URL = "$URL"
DisableStdin = true
EOF

uploadStdin "$TMPDIR/upload/FILE1" --name "FILE1" || true
cat $CLIENT_LOG | grep -i "stdin is disabled" >/dev/null 2>&1

uploadStdin "$TMPDIR/upload/FILE1" --stdin --name "FILE1" && download && check
echo "OK"

#---------------------------------------------

###
# Upload options
###

echo -n " - one shot : "

before
cp $SPECIMEN $TMPDIR/upload/FILE1
upload -o && uploadOpts
echo "$UPLOAD_OPTS" | grep '"oneShot": true' >/dev/null 2>/dev/null

before
cp $SPECIMEN $TMPDIR/upload/FILE1
upload --oneshot && uploadOpts
echo "$UPLOAD_OPTS" | grep '"oneShot": true' >/dev/null 2>/dev/null

echo "OK"

#---------------------------------------------

echo -n " - removable : "

before
cp $SPECIMEN $TMPDIR/upload/FILE1
upload -r && uploadOpts
echo "$UPLOAD_OPTS" | grep '"removable": true' >/dev/null 2>/dev/null

before
cp $SPECIMEN $TMPDIR/upload/FILE1
upload --removable && uploadOpts
echo "$UPLOAD_OPTS" | grep '"removable": true' >/dev/null 2>/dev/null

echo "OK"

#---------------------------------------------

echo -n " - streaming : "

before
FILE="FILE_$RANDOM"
cp $SPECIMEN $TMPDIR/upload/$FILE
# Start upload cmd in background to create upload then kill it
(
    set -e
    upload -S &
    child=$(ps x | grep "plik \-S $FILE" | awk '{print $1}')
    sleep 1
    (
        kill -0 $child && kill $child
        sleep 1
        kill -0 $child && kill -9 $child
    ) >/dev/null 2>&1 &
)
sleep 3
uploadOpts
echo "$UPLOAD_OPTS" | grep '"stream": true' >/dev/null 2>/dev/null

before
FILE="FILE_$RANDOM"
cp $SPECIMEN $TMPDIR/upload/$FILE
# Start upload cmd in background to create upload then kill it
(
    set -e
    upload --stream &
    child=$(ps x | grep "plik \--stream $FILE" | awk '{print $1}')
    sleep 1
    (
        kill -0 $child && kill $child
        sleep 1
        kill -0 $child && kill -9 $child
    ) >/dev/null 2>&1 &
)
sleep 3
uploadOpts
echo "$UPLOAD_OPTS" | grep '"stream": true' >/dev/null 2>/dev/null

echo "OK"

#---------------------------------------------

echo -n " - ttl : "

before
cp $SPECIMEN $TMPDIR/upload/FILE1
upload -t 3600 && uploadOpts
echo "$UPLOAD_OPTS" | grep '"ttl": 3600' >/dev/null 2>/dev/null

before
cp $SPECIMEN $TMPDIR/upload/FILE1
upload --ttl 3600 && uploadOpts
echo "$UPLOAD_OPTS" | grep '"ttl": 3600' >/dev/null 2>/dev/null

echo "OK"

#---------------------------------------------

echo -n " - password : "
before
cp $SPECIMEN $TMPDIR/upload/FILE1
UPLOAD_USER="foo"
UPLOAD_PWD="bar"
upload --password "$UPLOAD_USER:$UPLOAD_PWD" && uploadOpts
echo "$UPLOAD_OPTS" | grep '"protectedByPassword": true' >/dev/null 2>/dev/null
echo "OK"

#---------------------------------------------

echo -n " - prompted password : "
before
cp $SPECIMEN $TMPDIR/upload/FILE1
UPLOAD_USER="foo"
UPLOAD_PWD="bar"
echo -e "$UPLOAD_USER\n$UPLOAD_PWD\n" | upload -p && uploadOpts
echo "$UPLOAD_OPTS" | grep '"protectedByPassword": true' >/dev/null 2>/dev/null
echo "OK"

#---------------------------------------------

echo -n " - comments : "
before
cp $SPECIMEN $TMPDIR/upload/FILE1
upload --comments "foobar" && uploadOpts
echo "$UPLOAD_OPTS" | grep '"comments": "foobar"' >/dev/null 2>/dev/null
echo "OK"

#---------------------------------------------

echo -n " - quiet : "

before
cp $SPECIMEN $TMPDIR/upload/FILE1
upload -q
test $(cat $CLIENT_LOG | wc -l) -eq 1
grep "$URL/file/.*/.*/FILE1" $CLIENT_LOG >/dev/null 2>/dev/null

before
cp $SPECIMEN $TMPDIR/upload/FILE1
upload --quiet
test $(cat $CLIENT_LOG | wc -l) -eq 1
grep "$URL/file/.*/.*/FILE1" $CLIENT_LOG >/dev/null 2>/dev/null

echo "OK"

#---------------------------------------------

echo -n " - not secure : "

SECURE="true"
before
cp $SPECIMEN $TMPDIR/upload/FILE1
upload --not-secure && download && check
# should not pipe curl return to a secure option command
grep '^curl ' $CLIENT_LOG | grep -v '|' >/dev/null 2>/dev/null
echo "OK"

###
# Tar archive
###

echo -n " - tar single file : "
before
cp $SPECIMEN $TMPDIR/upload/FILE1
upload -a && download && check
echo "OK"

#---------------------------------------------

echo -n " - tar multiple file : "
before
cp $SPECIMEN $TMPDIR/upload/FILE1
cp $SPECIMEN $TMPDIR/upload/FILE2
upload -a && download && check
echo "OK"

#---------------------------------------------

echo -n " - tar directory : "
before
mkdir $TMPDIR/upload/DIR
cp $SPECIMEN $TMPDIR/upload/DIR/FILE1
cp $SPECIMEN $TMPDIR/upload/DIR/FILE2
upload && download && check
echo "OK"

#---------------------------------------------

echo -n " - tar custom compression codec : "
before
cp $SPECIMEN $TMPDIR/upload/FILE1
upload -a --compress bzip2 && download && check
grep 'FILE1.tar.bz2' $CLIENT_LOG >/dev/null 2>/dev/null
echo "OK"

#---------------------------------------------

echo -n " - tar custom options : "
before
cp $SPECIMEN $TMPDIR/upload/FILE1
cp $SPECIMEN $TMPDIR/upload/EXCLUDE
upload -a --archive-options "'--exclude=EXCLUDE'"
rm $TMPDIR/upload/EXCLUDE
download && check
echo "OK"

#---------------------------------------------

echo -n " - tar file with custom name : "
before
cp $SPECIMEN $TMPDIR/upload/FILE1
upload -a --name foobar.tar.gz && download && check
grep 'foobar.tar.gz' $CLIENT_LOG >/dev/null 2>/dev/null
echo "OK"

#---------------------------------------------

echo -n " - tar directory with custom name : "
before
mkdir $TMPDIR/upload/DIR
cp $SPECIMEN $TMPDIR/upload/DIR/FILE1
cp $SPECIMEN $TMPDIR/upload/DIR/FILE2
upload -a --name foobar.tar.gz && download && check
grep 'foobar.tar.gz' $CLIENT_LOG >/dev/null 2>/dev/null
echo "OK"

###
# Zip archive
###

echo -n " - zip single file : "
before
cp $SPECIMEN $TMPDIR/upload/FILE1
upload --archive zip && download
# Unzip manually
cd $TMPDIR/download
test -f FILE1.zip || exit 1
unzip FILE1.zip >/dev/null 2>/dev/null
rm FILE1.zip
check
echo "OK"

#---------------------------------------------

echo -n " - zip directory : "
before
mkdir $TMPDIR/upload/DIR
cp $SPECIMEN $TMPDIR/upload/DIR/FILE1
cp $SPECIMEN $TMPDIR/upload/DIR/FILE2
upload --archive zip && download
# Unzip manually
cd $TMPDIR/download
test -f DIR.zip || exit 1
unzip DIR.zip >/dev/null 2>/dev/null
rm DIR.zip
check
echo "OK"

#---------------------------------------------

echo -n " - zip custom options : "
before
cp $SPECIMEN $TMPDIR/upload/FILE1
cp $SPECIMEN $TMPDIR/upload/EXCLUDE
upload --archive zip --archive-options "'--exclude EXCLUDE'"
rm $TMPDIR/upload/EXCLUDE
download
# Unzip manually
cd $TMPDIR/download
test -f archive.zip || exit 1
unzip archive.zip >/dev/null 2>/dev/null
rm archive.zip
check
echo "OK"

#---------------------------------------------

echo -n " - zip file with custom name : "
before
cp $SPECIMEN $TMPDIR/upload/FILE1
upload --archive zip --name foobar.zip && download
# Unzip manually
cd $TMPDIR/download
test -f foobar.zip || exit 1
unzip foobar.zip >/dev/null 2>/dev/null
rm foobar.zip
check
echo "OK"

#---------------------------------------------

echo -n " - zip directory with custom name : "
before
mkdir $TMPDIR/upload/DIR
cp $SPECIMEN $TMPDIR/upload/DIR/FILE1
cp $SPECIMEN $TMPDIR/upload/DIR/FILE2
upload --archive zip --name foobar.zip && download
# Unzip manually
cd $TMPDIR/download
test -f foobar.zip || exit 1
unzip foobar.zip >/dev/null 2>/dev/null
rm foobar.zip
check
echo "OK"

###
# Openssl
###

echo -n " - openssl auto passphrase : "
before
cp $SPECIMEN $TMPDIR/upload/FILE1
upload -s && download && check
grep 'Passphrase' $CLIENT_LOG >/dev/null 2>/dev/null
grep 'openssl.*pass' $CLIENT_LOG >/dev/null 2>/dev/null
echo "OK"

#---------------------------------------------

echo -n " - openssl custom passphrase : "
before
cp $SPECIMEN $TMPDIR/upload/FILE1
upload -s --passphrase foobar && download && check
grep 'openssl.*pass.*foobar' $CLIENT_LOG >/dev/null 2>/dev/null
echo "OK"

#---------------------------------------------

echo -n " - openssl prompted passphrase : "
before
cp $SPECIMEN $TMPDIR/upload/FILE1
echo "foobar" | upload -s --passphrase - && download && check
grep 'openssl.*pass.*foobar' $CLIENT_LOG >/dev/null 2>/dev/null
echo "OK"

#---------------------------------------------

echo -n " - openssl custom cipher : "
before
cp $SPECIMEN $TMPDIR/upload/FILE1
echo "foobar" | upload -s --cipher blowfish && download && check
grep 'openssl.*blowfish' $CLIENT_LOG >/dev/null 2>/dev/null
echo "OK"

#---------------------------------------------

echo -n " - openssl custom options : "
before
cp $SPECIMEN $TMPDIR/upload/FILE1
upload -s --secure-options '-a' && download && check
curl $(cat $CLIENT_LOG | grep "curl" | sed -n 's/^.*"\(.*\)".*$/\1/p') >$TMPDIR/download/ARMORED 2>/dev/null
file $TMPDIR/download/ARMORED | grep "ASCII text\|base64" >/dev/null 2>/dev/null
echo "OK"

###
# PGP
###

echo -n " - pgp : "
before

export GNUPGHOME="$TMPDIR"

cat >$TMPDIR/pgp.key << EOF
-----BEGIN PGP PRIVATE KEY BLOCK-----
Version: GnuPG v1

lQHYBFYLA7cBBACgMFkOEqqWop6bQYp4LGq0A79XakKj1vYVYom7Jg+V9utPQsK3
29rKzSBAiq2yWAQLNyJ6dpyFctabSU1pJ/OAsvssuMxio/M6Kf+mvMHmAjyW9s5F
fqeZROKjyOswFwkPKS36ifOKm2CigfJlMavV74h3p4f9JznWYn0MBrCDMQARAQAB
AAP+I5ZKGonEEx4CjWxUllkLxX01o3ZsYpitZ9fR0F1mxgKqiRvERXNW2ooSmbQV
XZMXJuSzSLCUGkOGcM4qn+truXhE3vbxEtyNpKQP1ae/m2zLcUJk3JJWWmnt/BPD
JSRsOtOGovoasKVxK58BQ1UfkV0Mred31NxYVvU3LxAhCAkCAMRt1aWgfWgAA007
j+wDtXz9qeAuBQ8jb5OZ/O0WU9mNfGauFVeby0Mld0m7ofUyAVdIGn26woY/hPCL
zCMcDZ0CANDE7vyYHRWcNclETzqkDuCKB/MG62jPRerF9QLKozDcP8fu8xm//iYd
K8v3UTHbhZ5X2wvcyMIQxm7Iov18oaUB/iJH5jUlhyHZOWIZXJ/xpV/fFZr+DIAi
kL7KP+nknyFrP4czcfzhSLkQKyU67ODkPfxqltf7SVZlnqk5GKpXB4WfBbQrcGxp
ay5yb290LmdnIChwbGlrIHRlc3Qga2V5KSA8cGxpa0Byb290LmdnPoi4BBMBAgAi
BQJWCwO3AhsDBgsJCAcDAgYVCAIJCgsEFgIDAQIeAQIXgAAKCRBqKML4FUjUXW18
A/9Rutp5SWnk+Vi4nbFh2QAyl5rwDdF45mzZBY1DQsBzpkREg8URvBLN0lpWDr4k
4Mm7ONIqmAta23NvPe4yR1f68Q2SGsheUbL27vGcbQ/bY1pkzTRSGZFnWu3Q6Oo2
0gOo8b0HsbJMF4VJvwmAhXk+IiIbUpQ0Zep27BwQWagmDp0B2ARWCwO3AQQAp1GU
ZsAPOUgtm/gLA6fYf9OuUvaUprVL7GBpdhjIA1r5syJrCxRtWocvxH+EMHgF6CCq
Qe03PODI8NhjK1zCCZr1CQRontD1a59CHdSFk+2eTa40CNsJ17f16eiDwE8GvhNT
T/ZGEztS9b5uCp0higrAqtKTvx0NsX/V3juJBtMAEQEAAQAD/1OARJn0voQ9T7m3
U7Pa15KPjz+LHIuIDeBlCyyrWGJITDZIdnhclOhpb/7WDp/rvjLm3mExY/BHVDDS
JMe2roSyraBj86SnejJDuA1JWhCLvBF8bQKrXNMdKH76gAdcT++tEuYMRmlur22z
PcW+FDZspr4lRn33AZPtHn21mrdNAgDCeasfTHlgXrOQ9o/iNVC9tfxFjVEe3JCC
nOaoBNNpDrOe8xpJ2amYbW0I3KkoYx2Q2hZKuIgj88WyoGdiirRnAgDcQId88AcH
vKPKunU4oFfePtqLjX5s5TKffcqmTtQRW0sqcoo8tNECXq8lsk9PUoihs15Ux2X4
r6LexhGrMHa1AfwKZJBWXwoxzUKYVQW3qRh1MokzHS+ZLT25w/7Co/IF+CjLeSiU
IzKv2YBRXHV9YeTpqUxFSGOyIiAgC6kap0HVnbCInwQYAQIACQUCVgsDtwIbDAAK
CRBqKML4FUjUXRfBA/4pLcWcBOJ8suh7kTgmicZA55bAbY+CTnNlHma7pzW1rcqD
TojG/RllyilI8QHfR9+da/iEGoAcY8eTgpAYZfNnd8tCy1bQQM+YkjAgh7lFEUdV
Wslu8jCqJpbcKUL7k2mfTKwJ97h1Go5LMurSR9W2psZrmyHXbCccu0CghK/Y7g==
=/xyA
-----END PGP PRIVATE KEY BLOCK-----
EOF

touch "$TMPDIR/pubring.gpg"
gpg --import $TMPDIR/pgp.key >/dev/null 2>/dev/null

# Use temporary keyring
cat >$PLIKRC << EOF
URL = "$URL"

[SecureOptions]
  Keyring = "$TMPDIR/pubring.gpg"

EOF

cp $SPECIMEN $TMPDIR/upload/FILE1
upload --debug --secure pgp --recipient 'plik.root.gg'
download && check
echo "OK"

###
# UPDATE
###

shutdown
before
rm $SERVER_LOG
rm $CLIENT_LOG
cd $ORIGIN

#echo " - upgrade : ( this might take a long time ... )"
#./test_upgrade.sh
#echo " - downgrade : ( this might take a long time ... )"
#./test_downgrade.sh

exit 0
