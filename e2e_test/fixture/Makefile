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

# Test#1
# It updates an image tag of Deployment.
# It will cause the rolling update, that is, Progressing state.
update-manifest-app1:
	git checkout "$(FIXTURE_BASE_BRANCH)"
	git checkout -b "$(FIXTURE_BRANCH_PREFIX)/update-manifest-app1"
	sed -i -e 's/DEPLOYMENT_URL/$(DEPLOYMENT_URL)/g' app1/githubdeployment/app1.yaml
	sed -i -e 's/echoserver:1.8/echoserver:1.9/g' app1/deployment/echoserver.yaml
	git commit -a -m 'e2e-test: update-manifest-app1'
	git push origin -f "$(FIXTURE_BRANCH_PREFIX)/update-manifest-app1"
	gh pr create --base "$(FIXTURE_BASE_BRANCH)" --fill --body "$(PULL_REQUEST_BODY)" --label e2e-test
	gh pr merge --squash
	git checkout "$(FIXTURE_BASE_BRANCH)"
	git pull origin "$(FIXTURE_BASE_BRANCH)" --ff-only

# Test#2
# It updates the label to invalid value.
# It will cause this error:
# one or more objects failed to apply, reason: Deployment.apps "echoserver" is invalid: spec.selector: Invalid value: v1.LabelSelector
update-manifest-app2:
	git checkout "$(FIXTURE_BASE_BRANCH)"
	git checkout -b "$(FIXTURE_BRANCH_PREFIX)/update-manifest-app2"
	sed -i -e 's/DEPLOYMENT_URL/$(DEPLOYMENT_URL)/g' app2/githubdeployment/app2.yaml
	sed -i -e 's/app: echoserver/app: echoserver-test2/g' app2/deployment/echoserver.yaml
	git commit -a -m 'e2e-test: update-manifest-app2'
	git push origin -f "$(FIXTURE_BRANCH_PREFIX)/update-manifest-app2"
	gh pr create --base "$(FIXTURE_BASE_BRANCH)" --fill --body "$(PULL_REQUEST_BODY)" --label e2e-test
	gh pr merge --squash
	git checkout "$(FIXTURE_BASE_BRANCH)"
	git pull origin "$(FIXTURE_BASE_BRANCH)" --ff-only

# Test#3
# It updates an image tag of CronJob template.
# Application will not transit to Progressing state.
update-manifest-app3:
	git checkout "$(FIXTURE_BASE_BRANCH)"
	git checkout -b "$(FIXTURE_BRANCH_PREFIX)/update-manifest-app3"
	sed -i -e 's/DEPLOYMENT_URL/$(DEPLOYMENT_URL)/g' app3/githubdeployment/app3.yaml
	sed -i -e 's/busybox:1.28/busybox:1.30/g' app3/cronjob/echoserver.yaml
	git commit -a -m 'e2e-test: update-manifest-app3'
	git push origin -f "$(FIXTURE_BRANCH_PREFIX)/update-manifest-app3"
	gh pr create --base "$(FIXTURE_BASE_BRANCH)" --fill --body "$(PULL_REQUEST_BODY)" --label e2e-test
	gh pr merge --squash
	git checkout "$(FIXTURE_BASE_BRANCH)"
	git pull origin "$(FIXTURE_BASE_BRANCH)" --ff-only
