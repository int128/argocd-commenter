CLUSTER_NAME := argocd-commenter-e2e
KUBECONFIG := output/kubeconfig.yaml
export KUBECONFIG

GITHUB_RUN_ID ?= 0
FIXTURE_BRANCH := e2e-test-$(GITHUB_RUN_ID)

GITHUB_PR_NUMBER ?= 0

all:

test:
	# wait for sync
	./wait-for-sync-status.sh test1-fixture $(FIXTURE_BRANCH)-test1-main Synced Succeeded
	./wait-for-sync-status.sh test2-fixture $(FIXTURE_BRANCH)-test2-main Synced Succeeded
	# test
	FIXTURE_BRANCH=$(FIXTURE_BRANCH) ./test-suite.sh
	# wait for sync
	./wait-for-sync-status.sh test1-fixture $(FIXTURE_BRANCH)-test1-main Synced Succeeded
	./wait-for-sync-status.sh test2-fixture $(FIXTURE_BRANCH)-test2-main OutOfSync Failed

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
	kustomize build applications | sed -e "s/FIXTURE_BRANCH/$(FIXTURE_BRANCH)/g" | kubectl apply -f -
	kubectl -n argocd rollout status statefulsets argocd-application-controller

deploy-controller: cluster build-controller
	kind load docker-image controller:latest --name $(CLUSTER_NAME)
	kustomize build controller | kubectl apply -f -
	./create-controller-manager-secret.sh
	kubectl -n argocd-commenter-system rollout status deployment argocd-commenter-controller-manager

build-controller:
	DOCKER_BUILDKIT=1 docker build .. --tag controller:latest --cache-from ghcr.io/int128/argocd-commenter:latest --build-arg BUILDKIT_INLINE_CACHE=1

undeploy:
	kubectl delete namespace/argocd namespace/argocd-commenter-system

setup:
	git branch -f $(FIXTURE_BRANCH)-test1-main
	git branch -f $(FIXTURE_BRANCH)-test2-main
	git push origin -f $(FIXTURE_BRANCH)-test1-main $(FIXTURE_BRANCH)-test2-main

cleanup:
	-git branch -D $(FIXTURE_BRANCH)-test1-main
	-git branch -D $(FIXTURE_BRANCH)-test1-topic
	-git branch -D $(FIXTURE_BRANCH)-test2-main
	-git branch -D $(FIXTURE_BRANCH)-test2-topic
	-git push origin --delete $(FIXTURE_BRANCH)-test1-main
	-git push origin --delete $(FIXTURE_BRANCH)-test1-topic
	-git push origin --delete $(FIXTURE_BRANCH)-test2-main
	-git push origin --delete $(FIXTURE_BRANCH)-test2-topic
	-kubectl -n argocd-commenter-system logs -l control-plane=controller-manager --all-containers
