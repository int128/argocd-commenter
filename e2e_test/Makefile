CLUSTER_NAME := argocd-commenter-e2e
KUBECONFIG := output/kubeconfig.yaml
export KUBECONFIG

GITHUB_RUN_ID ?= 0
export GITHUB_RUN_ID

all:

test:
	./run-test.sh

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
	kubectl -n argocd rollout status statefulsets argocd-application-controller

deploy-controller: cluster
	DOCKER_BUILDKIT=1 docker build .. --tag controller:latest --cache-from ghcr.io/int128/argocd-commenter:latest --build-arg BUILDKIT_INLINE_CACHE=1
	kind load docker-image controller:latest --name $(CLUSTER_NAME)
	kustomize build controller | kubectl apply -f -
	-kubectl -n argocd-commenter-system create secret generic argocd-commenter-secret --from-literal="GITHUB_TOKEN=$(GITHUB_TOKEN)"
	kubectl -n argocd-commenter-system rollout status deployment argocd-commenter-controller-manager

undeploy:
	kubectl delete namespace/argocd namespace/argocd-commenter-system

# CI specific tasks
ci-setup:
	git config user.name ci
	git config user.email ci@example.com
	-git push origin --delete e2e-test-$(GITHUB_RUN_ID)-main
	-git push origin --delete e2e-test-$(GITHUB_RUN_ID)-fixture1

ci-cleanup:
	-git push origin --delete e2e-test-$(GITHUB_RUN_ID)-main
	-git push origin --delete e2e-test-$(GITHUB_RUN_ID)-fixture1
	-kubectl -n argocd-commenter-system logs -l control-plane=controller-manager --all-containers
