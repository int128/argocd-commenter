#!/usr/bin/env bash
set -o pipefail
set -eux

pull_request_body="This is created by e2e-test of #$GITHUB_PR_NUMBER"
commit_comment="Fixture for #$GITHUB_PR_NUMBER"

# test1
git checkout "${FIXTURE_BRANCH}-test1-main"
git checkout -b "${FIXTURE_BRANCH}-test1-topic"

sed -i -e 's/name: echoserver/name: echoserver-test1/g' test1-fixture/deployment/echoserver.yaml

git commit -a -m "$commit_comment (test1)"
git push origin "${FIXTURE_BRANCH}-test1-topic"
gh pr create --base "${FIXTURE_BRANCH}-test1-main" --title "Fixture 1: sync success" --body "$pull_request_body" --label e2e-test
gh pr merge --squash

git checkout "${FIXTURE_BRANCH}-test1-main"
git pull origin --ff-only "${FIXTURE_BRANCH}-test1-main"
git branch -D "${FIXTURE_BRANCH}-test1-topic"

# test2
git checkout "${FIXTURE_BRANCH}-test2-main"
git checkout -b "${FIXTURE_BRANCH}-test2-topic"

sed -i -e 's/app: echoserver/app: echoserver-test2/g' test2-fixture/deployment/echoserver.yaml

git commit -a -m "$commit_comment (test2)"
git push origin "${FIXTURE_BRANCH}-test2-topic"
gh pr create --base "${FIXTURE_BRANCH}-test2-main" --title "Fixture 2: sync failure" --body "$pull_request_body" --label e2e-test
gh pr merge --squash

git checkout "${FIXTURE_BRANCH}-test2-main"
git pull origin --ff-only "${FIXTURE_BRANCH}-test2-main"
git branch -D "${FIXTURE_BRANCH}-test2-topic"
