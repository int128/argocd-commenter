#!/usr/bin/env bash
set -o pipefail
set -eux

application_name="$1"
want_revision="$(git rev-parse HEAD)"

for (( i = 0; i < 15; i++ )); do
  status="$(kubectl -n argocd get application "$application_name" '-ojsonpath={.status.sync.status}/{.status.sync.revision}')"
  if [[ $status == Synced/$want_revision ]]; then
    exit 0
  fi
  sleep "$i"
done
exit 1
