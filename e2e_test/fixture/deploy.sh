#!/bin/bash
set -eux -o pipefail

: "$GITHUB_REPOSITORY"
: "$GITHUB_REF_NAME"
: "$GITHUB_RUN_NUMBER"

app="$1"
head_branch="e2e-test/$GITHUB_RUN_NUMBER/deploy-$app"
base_branch="e2e-test/$GITHUB_RUN_NUMBER/main"

cat > "$app/metadata.json" <<EOF
{"deploymentURL": "${DEPLOYMENT_URL}"}
EOF

# Commit the changes
git checkout -b "$head_branch"
git add .
git commit -m "e2e-test: Update $app"

# Push and merge it
git push origin -f "$head_branch"
gh pr create --base "$base_branch" --fill --body "$GITHUB_REPOSITORY#${GITHUB_REF_NAME%%/*}" --label e2e-test
gh pr merge --squash

# Reset the local state
git checkout "$base_branch"
git pull origin "$base_branch" --ff-only
