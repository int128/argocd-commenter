#!/usr/bin/env bash
set -o pipefail
set -eu

application_name="$1"
want_branch="$2"
want_revision="$(git rev-parse "$want_branch")"
want_sync_status="$3"
want_phase="$4"

for (( i = 0; i < 10; i++ )); do
  got="$(kubectl -n argocd get application "$application_name" '-ojsonpath={.status.sync.status}/{.status.operationState.phase}/{.status.sync.revision}')"
  want="$want_sync_status/$want_phase/$want_revision"
  echo "[wait-for-sync-status] got:  $got"
  echo "[wait-for-sync-status] want: $want"
  if [[ $got == $want ]]; then
    exit 0
  fi
  echo "[wait-for-sync-status] retry after ${i}s"
  sleep "$i"
done
exit 1
