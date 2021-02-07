#!/usr/bin/env bash
set -o pipefail
set -eu

application_name="$1"
want_branch="$2"
want_status="$3"
want_revision="$(git rev-parse "$want_branch")"

for (( i = 0; i < 10; i++ )); do
  status="$(kubectl -n argocd get application "$application_name" '-ojsonpath={.status.sync.status}/{.status.sync.revision}')"
  echo "[wait-for-sync-status] got:  $status"
  echo "[wait-for-sync-status] want: $want_status/$want_revision"
  if [[ $status == $want_status/$want_revision ]]; then
    exit 0
  fi
  echo "[wait-for-sync-status] retry after ${i}s"
  sleep "$i"
done
exit 1
