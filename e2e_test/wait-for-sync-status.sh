#!/usr/bin/env bash
set -o pipefail
set -eu

application_name="$1"
want_branch="$2"
want_revision="$(git -C argocd-commenter-e2e-test rev-parse "$want_branch")"
want_sync_status="$3"
want_phase="$4"
want_health_status="$5"

for (( i = 0; i < 10; i++ )); do
  got="$(kubectl -n argocd get application "$application_name" '-ojsonpath={.status.sync.status}/{.status.operationState.phase}/{.status.health.status}/{.status.sync.revision}')"
  want="$want_sync_status/$want_phase/$want_health_status/$want_revision"
  echo "[wait-for-sync-status] got:  $got"
  echo "[wait-for-sync-status] want: $want"
  if [[ $got == $want ]]; then
    exit 0
  fi
  echo "[wait-for-sync-status] retry #$i"
  sleep 3
done
exit 1
