CLUSTER_NAME := argocd-commenter-e2e
KUBECONFIG := output/kubeconfig.yaml
export KUBECONFIG

GITHUB_RUN_NUMBER ?= 0
FIXTURE_BRANCH := e2e-test/$(GITHUB_RUN_NUMBER)/main
FIXTURE_DIR := argocd-commenter-e2e-test-repository

all:

setup-fixture-branch:
	test -d $(FIXTURE_DIR)
	rsync -av fixture/ $(FIXTURE_DIR)/
	$(MAKE) -C $(FIXTURE_DIR) setup-fixture-branch

# cluster
cluster: $(KUBECONFIG)
$(KUBECONFIG):
	kind create cluster --name $(CLUSTER_NAME)

delete-cluster:
	kind delete cluster --name $(CLUSTER_NAME)
	-rm $(KUBECONFIG)

deploy-argocd: cluster
	# To avoid the error "ensure CRDs are installed first"
	-kustomize build argocd | kubectl apply -f - --server-side
	kustomize build argocd | kubectl apply -f - --server-side
	kustomize build applications | sed -e "s|FIXTURE_BRANCH|$(FIXTURE_BRANCH)|g" | kubectl apply -f -

deploy-controller: cluster
	cd controller && kustomize edit set image controller="$(CONTROLLER_IMAGE)"
	kustomize build controller | kubectl apply -f -
	bash controller/create-controller-manager-secret.sh
	kubectl -n argocd-commenter-system rollout status deployment argocd-commenter-controller-manager

wait-for-apps:
	kubectl -n argocd rollout status deployment -l app.kubernetes.io/part-of=argocd
	# applicationset-controller must start up after repo-server
	kubectl -n argocd rollout restart deployment -l app.kubernetes.io/name=argocd-applicationset-controller
	go run ./waitforapp -revision "`git -C $(FIXTURE_DIR) rev-parse $(FIXTURE_BRANCH)`" app1 app2 app3

# fixture
deploy-app1:
	$(MAKE) -C $(FIXTURE_DIR) deploy-app1
	go run ./waitforapp -revision "`git -C $(FIXTURE_DIR) rev-parse $(FIXTURE_BRANCH)`" app1

restart-app1:
	kubectl -n app1 delete pod -l app=echoserver --force
	go run ./waitforapp -revision "`git -C $(FIXTURE_DIR) rev-parse $(FIXTURE_BRANCH)`" app1

deploy-app2:
	$(MAKE) -C $(FIXTURE_DIR) deploy-app2
	go run ./waitforapp -revision "`git -C $(FIXTURE_DIR) rev-parse $(FIXTURE_BRANCH)`" -sync OutOfSync -operation Failed app2

deploy-app3:
	$(MAKE) -C $(FIXTURE_DIR) deploy-app3
	go run ./waitforapp -revision "`git -C $(FIXTURE_DIR) rev-parse $(FIXTURE_BRANCH)`" app3

logs-controller:
	-kubectl -n argocd-commenter-system logs -l control-plane=controller-manager --all-containers --tail=-1
logs-argocd-application-controller:
	-kubectl -n argocd logs -l app.kubernetes.io/name=argocd-application-controller --all-containers --tail=-1
logs-argocd-applicationset-controller:
	-kubectl -n argocd logs -l app.kubernetes.io/name=argocd-applicationset-controller --all-containers --tail=-1
logs-argocd-repo-server:
	-kubectl -n argocd logs -l app.kubernetes.io/name=argocd-repo-server --all-containers --tail=-1
