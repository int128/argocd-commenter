GITHUB_RUN_NUMBER ?= 0
FIXTURE_BRANCH := e2e-test/$(GITHUB_RUN_NUMBER)/main

all:

setup-fixture-branch:
	git config user.name 'github-actions[bot]'
	git config user.email '41898282+github-actions[bot]@users.noreply.github.com'
	git checkout -B "$(FIXTURE_BRANCH)"
	git add .
	git commit -m "Initial commit"
	git push origin -f "$(FIXTURE_BRANCH)"

# Test#1
# It updates an image tag of Deployment.
# It will cause the rolling update, that is, Progressing state.
deploy-app1:
	git checkout "$(FIXTURE_BRANCH)"
	sed -i -e 's/echoserver:1.8/echoserver:1.9/g' app1/deployment/echoserver.yaml
	bash deploy.sh app1

# Test#2
# It updates the label to invalid value.
# It will cause this error:
# one or more objects failed to apply, reason: Deployment.apps "echoserver" is invalid: spec.selector: Invalid value: v1.LabelSelector
deploy-app2:
	git checkout "$(FIXTURE_BRANCH)"
	sed -i -e 's/app: echoserver/app: echoserver-test2/g' app2/deployment/echoserver.yaml
	bash deploy.sh app2

# Test#3
# It updates an image tag of CronJob template.
# Application will not transit to Progressing state.
deploy-app3:
	git checkout "$(FIXTURE_BRANCH)"
	sed -i -e 's/busybox:1.28/busybox:1.30/g' app3/cronjob/echoserver.yaml
	bash deploy.sh app3
