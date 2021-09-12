all:

setup:
	git checkout --orphan "$(FIXTURE_BRANCH)"
	git commit -a -m "Initial commit"
	git push origin "$(FIXTURE_BRANCH)"

test: test-app1 test-app2

test-app1:
	git checkout "$(FIXTURE_BRANCH)"
	git checkout -b "$(FIXTURE_BRANCH)-app1-topic"
	sed -i -e 's/name: echoserver/name: echoserver-test1/g' app1/deployment/echoserver.yaml
	git commit -a -m 'e2e-test: app1' -m "$(GIT_COMMIT_MESSAGE)"
	git push origin "$(FIXTURE_BRANCH)-app1-topic"
	gh pr create --base "$(FIXTURE_BRANCH)" --fill --label e2e-test
	gh pr merge --squash

test-app2:
	git checkout "$(FIXTURE_BRANCH)"
	git checkout -b "$(FIXTURE_BRANCH)-app2-topic"
	sed -i -e 's/app: echoserver/app: echoserver-test2/g' app2/deployment/echoserver.yaml
	git commit -a -m 'e2e-test: app2' -m "$(GIT_COMMIT_MESSAGE)"
	git push origin "$(FIXTURE_BRANCH)-app2-topic"
	gh pr create --base "$(FIXTURE_BRANCH)" --fill --label e2e-test
	gh pr merge --squash

cleanup:
	-git push origin --delete "$(FIXTURE_BRANCH)"
	-git push origin --delete "$(FIXTURE_BRANCH)-app1-topic"
	-git push origin --delete "$(FIXTURE_BRANCH)-app2-topic"
