#!/usr/bin/env bash
set -o pipefail
set -eu

for (( i = 0; i < 30; i++ )); do
  echo "[wait-for-docker-build] pulling $CONTROLLER_IMAGE"
  if docker pull "$CONTROLLER_IMAGE"; then
    revision="$(docker image inspect -f '{{index .Config.Labels "org.opencontainers.image.revision"}}' "$CONTROLLER_IMAGE")"
    echo "[wait-for-docker-build] got $revision want $GITHUB_SHA"
    if [[ $revision == $GITHUB_SHA ]]; then
      exit 0
    fi
  fi
  echo "[wait-for-docker-build] retry #$i"
  sleep 10
done
exit 1
