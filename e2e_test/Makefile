CLUSTER_NAME := argocd-commenter-e2e
KUBECONFIG := output/kubeconfig.yaml
export KUBECONFIG

GITHUB_RUN_NUMBER ?= 0
FIXTURE_BASE_BRANCH := e2e-test/$(GITHUB_RUN_NUMBER)/main
FIXTURE_DIR := argocd-commenter-e2e-test-repository

all:

setup-fixture-branch:
	test -d $(FIXTURE_DIR)
	rsync -av fixture/ $(FIXTURE_DIR)/
	$(MAKE) -C $(FIXTURE_DIR) push-base-branch

cluster: $(KUBECONFIG)
$(KUBECONFIG):
	kind create cluster --name $(CLUSTER_NAME)

delete-cluster:
	kind delete cluster --name $(CLUSTER_NAME)
	-rm $(KUBECONFIG)

deploy-argocd: cluster
	kustomize build argocd | kubectl apply -f -
	kustomize build applications | sed -e "s|FIXTURE_BASE_BRANCH|$(FIXTURE_BASE_BRANCH)|g" | kubectl apply -f -

deploy-controller: cluster
	cd controller && kustomize edit set image controller="$(CONTROLLER_IMAGE)"
	kustomize build controller | kubectl apply -f -
	bash controller/create-controller-manager-secret.sh
	kubectl -n argocd-commenter-system rollout status deployment argocd-commenter-controller-manager

wait-for-sync:
	go run ./waitforapp -revision "`git -C $(FIXTURE_DIR) rev-parse $(FIXTURE_BASE_BRANCH)`" app1 app2 app3

test:
	FIXTURE_DIR=$(FIXTURE_DIR) FIXTURE_BASE_BRANCH=$(FIXTURE_BASE_BRANCH) FIXTURE_BRANCH_PREFIX=$(FIXTURE_BRANCH_PREFIX) go test -v .

logs-controller:
	-kubectl -n argocd-commenter-system logs -l control-plane=controller-manager --all-containers --tail=-1
logs-argocd:
	-kubectl -n argocd logs -l app.kubernetes.io/name=argocd-application-controller --all-containers --tail=-1
