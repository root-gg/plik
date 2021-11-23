#!/bin/bash

set -e

plik_release_version=$(server/gen_build_info.sh version)
export plik_release_version

echo ""
echo " Releasing Plik $plik_release_version"
echo ""

if ! make build-info | grep "mint=true" >/dev/null ; then
  echo "!!! Release is not mint !!!"
  echo "Here is the local diff"
  git status
fi

if ! make build-info | grep "release=true" >/dev/null ; then
  echo "!!! Release is not tagged !!!"
fi

docker-compose build --build-arg VERSION=$plik_release_version

echo " Extracting release archives"

rm -rf releases || true
dir="/go/src/github.com/root-gg/plik/releases/archives"
container_id=$(docker create rootgg/plik-release:latest)
docker cp "$container_id":$dir releases
docker rm -v "$container_id"
md5sum releases/* > releases/md5sum.txt

echo ""
echo " Done. Release archives are available in the releases directory"
echo ""
ls -l releases

echo ""
echo " Available Docker images"
echo ""
docker images | grep ^rootgg/plik

echo ""
echo " Push images to Docker Hub"
echo ""
for tag in latest $plik_release_version ; do
  for image in $( docker images ls | grep ^rootgg/plik | grep -v release | grep -v builder | awk {'print $1'} | uniq ) ; do
    echo " - pushing $image:$tag to docker hub"
    docker push $image:$tag
  done
done
