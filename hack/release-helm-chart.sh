#!/bin/bash
set -eux -o pipefail

: "$IMG"

chart_name=argocd-commenter
chart_registry="oci://ghcr.io/$GITHUB_REPOSITORY/charts"

if [[ $GITHUB_REF_TYPE = tag ]]; then
  chart_version="${GITHUB_REF_NAME#v}"
else
  chart_version=0.0.0
fi

# Set the default image
yq -i '.manager.image.repository = env(IMG)' charts/chart/values.yaml
helm package charts/chart --app-version "$chart_version" --version "$chart_version"

if [[ $GITHUB_REF_TYPE = tag ]]; then
  helm push "${chart_name}-${chart_version}.tgz" "$chart_registry"
fi
