CLUSTER_NAME := argocd-commenter-e2e
KUBECONFIG := output/kubeconfig.yaml
export KUBECONFIG

GITHUB_RUN_ID ?= 0
MAIN_BRANCH_NAME := e2e-test-$(GITHUB_RUN_ID)-main

GITHUB_PR_NUMBER ?= 0

all:

test:
	# create a pull request
	git checkout -b $(MAIN_BRANCH_NAME)-fixture1
	sed -i -e 's/name: echoserver/name: echoserver-fixture1/g' helloworld/deployment/echoserver.yaml
	git commit -a -m "fixture1"
	git push origin $(MAIN_BRANCH_NAME)-fixture1
	gh pr create --base $(MAIN_BRANCH_NAME) --title "e2e-test: fixture1" --body "This is created by #$(GITHUB_PR_NUMBER)"
	gh pr merge --squash
	# wait for sync
	git checkout $(MAIN_BRANCH_NAME)
	git pull origin --ff-only $(MAIN_BRANCH_NAME)
	./wait-for-synced.sh helloworld
	kubectl -n helloworld rollout status deployment echoserver-fixture1
	sleep 10

# environment
cluster: $(KUBECONFIG)
$(KUBECONFIG):
	kind create cluster --name $(CLUSTER_NAME)

delete-cluster:
	kind delete cluster --name $(CLUSTER_NAME)
	-rm $(KUBECONFIG)

deploy: deploy-argocd deploy-controller

deploy-argocd: cluster
	kustomize build argocd | kubectl apply -f -
	git branch $(MAIN_BRANCH_NAME)
	git push origin $(MAIN_BRANCH_NAME)
	kustomize build applications | sed -e "s/MAIN_BRANCH_NAME/$(MAIN_BRANCH_NAME)/g" | kubectl apply -f -
	kubectl -n argocd rollout status statefulsets argocd-application-controller
	./wait-for-synced.sh helloworld

deploy-controller: cluster build-controller
	kind load docker-image controller:latest --name $(CLUSTER_NAME)
	kustomize build controller | kubectl apply -f -
	-kubectl -n argocd-commenter-system create secret generic argocd-commenter-secret --from-literal="GITHUB_TOKEN=$(GITHUB_TOKEN)"
	kubectl -n argocd-commenter-system rollout status deployment argocd-commenter-controller-manager

build-controller:
	DOCKER_BUILDKIT=1 docker build .. --tag controller:latest --cache-from ghcr.io/int128/argocd-commenter:latest --build-arg BUILDKIT_INLINE_CACHE=1

undeploy:
	kubectl delete namespace/argocd namespace/argocd-commenter-system

# CI specific tasks
ci-setup:
	git config user.name ci
	git config user.email ci@example.com

ci-cleanup:
	-git push origin --delete $(MAIN_BRANCH_NAME)
	-git push origin --delete $(MAIN_BRANCH_NAME)-fixture1
	-kubectl -n argocd-commenter-system logs -l control-plane=controller-manager --all-containers
