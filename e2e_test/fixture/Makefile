GITHUB_RUN_NUMBER ?= 0
FIXTURE_BASE_BRANCH := e2e-test/$(GITHUB_RUN_NUMBER)/main
FIXTURE_BRANCH_PREFIX := e2e-test/$(GITHUB_RUN_NUMBER)

all:

push-base-branch:
	git config user.name 'github-actions[bot]'
	git config user.email '41898282+github-actions[bot]@users.noreply.github.com'
	git checkout -B "$(FIXTURE_BASE_BRANCH)"
	git add .
	git commit -a -m "Initial commit"
	git push origin -f "$(FIXTURE_BASE_BRANCH)"

update-manifest-app1:
	git checkout "$(FIXTURE_BASE_BRANCH)"
	git checkout -b "$(FIXTURE_BRANCH_PREFIX)/update-manifest-app1"
	sed -i -e 's/name: echoserver/name: echoserver-test1/g' app1/deployment/echoserver.yaml
	git commit -a -m 'e2e-test: update-manifest-app1'
	git push origin -f "$(FIXTURE_BRANCH_PREFIX)/update-manifest-app1"
	gh pr create --base "$(FIXTURE_BASE_BRANCH)" --fill --body "$(PULL_REQUEST_BODY)" --label e2e-test
	gh pr merge --squash
	git checkout "$(FIXTURE_BASE_BRANCH)"
	git pull origin "$(FIXTURE_BASE_BRANCH)" --ff-only

update-manifest-app2:
	git checkout "$(FIXTURE_BASE_BRANCH)"
	git checkout -b "$(FIXTURE_BRANCH_PREFIX)/update-manifest-app2"
	sed -i -e 's/app: echoserver/app: echoserver-test2/g' app2/deployment/echoserver.yaml
	git commit -a -m 'e2e-test: update-manifest-app2'
	git push origin -f "$(FIXTURE_BRANCH_PREFIX)/update-manifest-app2"
	gh pr create --base "$(FIXTURE_BASE_BRANCH)" --fill --body "$(PULL_REQUEST_BODY)" --label e2e-test
	gh pr merge --squash
	git checkout "$(FIXTURE_BASE_BRANCH)"
	git pull origin "$(FIXTURE_BASE_BRANCH)" --ff-only

update-manifest-app3:
	git checkout "$(FIXTURE_BASE_BRANCH)"
	git checkout -b "$(FIXTURE_BRANCH_PREFIX)/update-manifest-app3"
	sed -i -e 's/app: echoserver/app: echoserver-test3/g' app3/cronjob/echoserver.yaml
	git commit -a -m 'e2e-test: update-manifest-app3'
	git push origin -f "$(FIXTURE_BRANCH_PREFIX)/update-manifest-app3"
	gh pr create --base "$(FIXTURE_BASE_BRANCH)" --fill --body "$(PULL_REQUEST_BODY)" --label e2e-test
	gh pr merge --squash
	git checkout "$(FIXTURE_BASE_BRANCH)"
	git pull origin "$(FIXTURE_BASE_BRANCH)" --ff-only
