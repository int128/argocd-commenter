CLUSTER_NAME := argocd-commenter-e2e
KUBECONFIG := output/kubeconfig.yaml
export KUBECONFIG

GITHUB_RUN_NUMBER ?= 0
FIXTURE_BRANCH := e2e-test-$(GITHUB_RUN_NUMBER)
export FIXTURE_BRANCH
export GIT_COMMIT_MESSAGE

all:

# cluster
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
	kustomize build applications | sed -e "s/FIXTURE_BRANCH/$(FIXTURE_BRANCH)/g" | kubectl apply -f -
	./wait-for-sync-status.sh app1 $(FIXTURE_BRANCH)/main Synced Succeeded
	./wait-for-sync-status.sh app2 $(FIXTURE_BRANCH)/main Synced Succeeded

deploy-controller: cluster
	cd controller && kustomize edit set image controller="$(CONTROLLER_IMAGE)"
	kustomize build controller | kubectl apply -f -
	bash controller/create-controller-manager-secret.sh
	kubectl -n argocd-commenter-system rollout status deployment argocd-commenter-controller-manager

undeploy:
	kubectl delete namespace/argocd namespace/argocd-commenter-system

# fixture
setup-fixture:
	$(MAKE) -C fixture setup

test:
	$(MAKE) -C fixture test1
	./wait-for-sync-status.sh app1 $(FIXTURE_BRANCH)/main Synced Succeeded
	$(MAKE) -C fixture test2
	./wait-for-sync-status.sh app2 $(FIXTURE_BRANCH)/main OutOfSync Failed

cleanup-fixture:
	$(MAKE) -C fixture cleanup

logs-controller:
	-kubectl -n argocd-commenter-system logs -l control-plane=controller-manager --all-containers --tail=-1
	-kubectl -n argocd logs -l app.kubernetes.io/name=argocd-application-controller --all-containers --tail=-1
