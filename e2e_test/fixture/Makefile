all:

setup:
	git checkout --orphan "$(FIXTURE_BRANCH)/main"
	git commit -a -m "Initial commit"
	git push origin "$(FIXTURE_BRANCH)/main"

test1:
	git checkout "$(FIXTURE_BRANCH)/main"
	git checkout -b "$(FIXTURE_BRANCH)/test1"
	sed -i -e 's/name: echoserver/name: echoserver-test1/g' app1/deployment/echoserver.yaml
	git commit -a -m 'e2e-test: app1' -m "$(GIT_COMMIT_MESSAGE)"
	git push origin "$(FIXTURE_BRANCH)/test1"
	gh pr create --base "$(FIXTURE_BRANCH)/main" --fill --label e2e-test
	gh pr merge --squash
	git checkout "$(FIXTURE_BRANCH)/main"
	git pull origin "$(FIXTURE_BRANCH)/main" --ff-only

test2:
	git checkout "$(FIXTURE_BRANCH)/main"
	git checkout -b "$(FIXTURE_BRANCH)/test2"
	sed -i -e 's/app: echoserver/app: echoserver-test2/g' app2/deployment/echoserver.yaml
	git commit -a -m 'e2e-test: app2' -m "$(GIT_COMMIT_MESSAGE)"
	git push origin "$(FIXTURE_BRANCH)/test2"
	gh pr create --base "$(FIXTURE_BRANCH)/main" --fill --label e2e-test
	gh pr merge --squash
	git checkout "$(FIXTURE_BRANCH)/main"
	git pull origin "$(FIXTURE_BRANCH)/main" --ff-only

cleanup:
	-git push origin --delete "$(FIXTURE_BRANCH)/main"
	-git push origin --delete "$(FIXTURE_BRANCH)/test1"
	-git push origin --delete "$(FIXTURE_BRANCH)/test2"
