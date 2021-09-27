all:

.PHONY: setup
setup:
	git config user.name 'github-actions[bot]'
	git config user.email '41898282+github-actions[bot]@users.noreply.github.com'
	git checkout -B base
	git add .
	git commit -a -m "Initial commit"
	git branch "$(FIXTURE_BRANCH)/test1/main" base
	git push origin -f "$(FIXTURE_BRANCH)/test1/main"
	git branch "$(FIXTURE_BRANCH)/test2/main" base
	git push origin -f "$(FIXTURE_BRANCH)/test2/main"
	git branch "$(FIXTURE_BRANCH)/test3/main" base
	git push origin -f "$(FIXTURE_BRANCH)/test3/main"

.PHONY: test1
test1:
	git checkout -B "$(FIXTURE_BRANCH)/test1/topic" "$(FIXTURE_BRANCH)/test1/main"
	sed -i -e 's/name: echoserver/name: echoserver-test1/g' app/deployment.yaml
	git commit -a -m 'test1'
	git push origin -f "$(FIXTURE_BRANCH)/test1/topic"
	gh pr create --base "$(FIXTURE_BRANCH)/test1/main" --fill --body "$(PULL_REQUEST_BODY)" --label e2e-test
	gh pr merge --squash
	git checkout "$(FIXTURE_BRANCH)/test1/main"
	git pull origin "$(FIXTURE_BRANCH)/test1/main" --ff-only

.PHONY: test2
test2:
	git checkout -B "$(FIXTURE_BRANCH)/test2/topic" "$(FIXTURE_BRANCH)/test2/main"
	# this will cause sync failure
	sed -i -e 's/app: echoserver/app: echoserver-test2/g' app/deployment.yaml
	git commit -a -m 'test2'
	git push origin -f "$(FIXTURE_BRANCH)/test2/topic"
	gh pr create --base "$(FIXTURE_BRANCH)/test2/main" --fill --body "$(PULL_REQUEST_BODY)" --label e2e-test
	gh pr merge --squash
	git checkout "$(FIXTURE_BRANCH)/test2/main"
	git pull origin "$(FIXTURE_BRANCH)/test2/main" --ff-only

.PHONY: test3
test3:
	git checkout -B "$(FIXTURE_BRANCH)/test3/topic" "$(FIXTURE_BRANCH)/test3/main"
	# this will cause crash loop
	cp -v test3/deployment.yaml app/deployment.yaml
	git commit -a -m 'test3'
	git push origin -f "$(FIXTURE_BRANCH)/test3/topic"
	gh pr create --base "$(FIXTURE_BRANCH)/test3/main" --fill --body "$(PULL_REQUEST_BODY)" --label e2e-test
	gh pr merge --squash
	git checkout "$(FIXTURE_BRANCH)/test3/main"
	git pull origin "$(FIXTURE_BRANCH)/test3/main" --ff-only
