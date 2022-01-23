#!/bin/bash

set -e

DOCKER_IMAGE="${DOCKER_IMAGE:-rootgg/plik}"

# Plik server release targets
if [[ -z "$TARGETS" ]]; then
  TARGETS="amd64,386,arm,arm64"
  #TARGETS="amd64"
fi

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

RELEASE="false"
if make build-info | grep "release=true" >/dev/null ; then
  RELEASE="true"
else
  echo "!!! Release is not tagged !!!"
fi

rm -rf releases || true
mkdir releases

tag_and_push () {
  source="$1"
  target="$2"

  echo "Tagging $source to $target"
  docker tag "$source" "$target"

  if [[ -n "$PUSH_TO_DOCKER_HUB" ]]; then
    echo "Pushing $target to Docker Hub"
    docker push "$target"
  fi
}

for TARGET in $(echo "$TARGETS" | awk -F, '{for (i=1;i<=NF;i++)print $i}')
do

  echo ""
  echo " Building $TARGET release"
  echo ""

  # Build release
  image="$DOCKER_IMAGE-linux-$TARGET:dev"
  docker build -t "$image" --build-arg "TARGET=$TARGET" .
  tag_and_push "$image" "$image"

  # Extract release archive
  image_id=$(docker images --filter "label=plik-stage=releases" --format "{{.CreatedAt}}\t{{.ID}}" | sort -nr | head -n 1 | cut -f2)
  container_id=$(docker create "$image_id")
  docker cp "$container_id:/go/src/github.com/root-gg/plik/releases/archives/." releases/
  docker rm "$container_id"

  # Tag aliases
  if [[ "$RELEASE" == "true" ]]; then
    tag_and_push "$image" "$DOCKER_IMAGE:$plik_release_version"
    tag_and_push "$image" "$DOCKER_IMAGE:latest"
  fi

  if [[ "$TARGET" == "amd64" ]]; then
      tag_and_push "$image" "$DOCKER_IMAGE:dev"
      if [[ "$RELEASE" == "true" ]]; then
          tag_and_push "$image" "$DOCKER_IMAGE:$plik_release_version"
          tag_and_push "$image" "$DOCKER_IMAGE:latest"
      fi
  fi
done

md5sum releases/* > releases/md5sum.txt

echo ""
echo " Done. Release archives are available in the releases directory"
echo ""
ls -l releases

echo ""
echo " Available Docker images"
echo ""
docker images | grep ^rootgg/plik