#!/usr/bin/env bash
set -o pipefail
set -eux

: "$GITHUB_REF"
: "$GITHUB_RUN_ID"

pr_number="$(echo "$GITHUB_REF" | cut -f3 -d/)"
main_branch_name="e2e-test-${GITHUB_RUN_ID}-main"
fixture_branch_name="e2e-test-${GITHUB_RUN_ID}-fixture1"

# deploy the main branch
git branch "$main_branch_name"
git push origin "$main_branch_name"
kustomize build applications | sed -e "s/MAIN_BRANCH_NAME/$main_branch_name/g" | kubectl apply -f -
./wait-for-synced.sh helloworld

# apply change to the main branch
git checkout -b "$fixture_branch_name"
sed -i -e 's/name: echoserver/name: echoserver-fixture1/g' helloworld/deployment/echoserver.yaml
git commit -a -m "#${pr_number}"
git push origin "$fixture_branch_name"
gh pr create --fill --base "$main_branch_name"
gh pr merge --squash
git checkout "$main_branch_name"

# wait for the change
git pull origin --ff-only "$main_branch_name"
./wait-for-synced.sh helloworld
kubectl -n helloworld rollout status deployment echoserver-fixture1
sleep 30
