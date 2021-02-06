#!/usr/bin/env bash
set -o pipefail
set -eu

application_name="$1"
want_revision="$(git rev-parse HEAD)"

for (( i = 0; i < 10; i++ )); do
  status="$(kubectl -n argocd get application "$application_name" '-ojsonpath={.status.sync.status}/{.status.sync.revision}')"
  echo "[wait-for-synced] got:  $status"
  echo "[wait-for-synced] want: Synced/$want_revision"
  if [[ $status == Synced/$want_revision ]]; then
    exit 0
  fi
  echo "[wait-for-synced] retry after ${i}s"
  sleep "$i"
done
exit 1
