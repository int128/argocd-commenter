#!/usr/bin/env bash
set -o pipefail
set -e

kubectl -n argocd-commenter-system delete secret controller-manager || true

# for personal access token
if [ "$GITHUB_TOKEN" ]; then
  echo 'using GITHUB_TOKEN'
  kubectl -n argocd-commenter-system create secret generic controller-manager \
    --from-literal="GITHUB_TOKEN=$GITHUB_TOKEN"
  exit 0
fi

# for installation access token
if [ "$GITHUB_APP_ID" ]; then
  echo 'using GITHUB_APP_ID'
  github_app_private_key_file="$(mktemp)"
  echo "$GITHUB_APP_PRIVATE_KEY" > "$github_app_private_key_file"
  kubectl -n argocd-commenter-system create secret generic controller-manager \
    --from-literal="GITHUB_APP_ID=$GITHUB_APP_ID" \
    --from-literal="GITHUB_APP_INSTALLATION_ID=$GITHUB_APP_INSTALLATION_ID" \
    --from-file="GITHUB_APP_PRIVATE_KEY=$github_app_private_key_file"
  rm -v "$github_app_private_key_file"
  exit 0
fi

echo 'you need to set either GITHUB_TOKEN or GITHUB_APP_ID' >&2
exit 1
