CLUSTER_NAME := argocd-commenter-e2e
KUBECONFIG := output/kubeconfig.yaml
export KUBECONFIG

GITHUB_RUN_NUMBER ?= 0
FIXTURE_BRANCH := e2e-test/$(GITHUB_RUN_NUMBER)
export FIXTURE_BRANCH

all:

setup-fixture-branch:
	test -d argocd-commenter-e2e-test-repository
	rsync -av fixture/ argocd-commenter-e2e-test-repository/
	$(MAKE) -C argocd-commenter-e2e-test-repository push-base-branch

# cluster
cluster: $(KUBECONFIG)
$(KUBECONFIG):
	kind create cluster --name $(CLUSTER_NAME)

delete-cluster:
	kind delete cluster --name $(CLUSTER_NAME)
	-rm $(KUBECONFIG)

deploy-argocd: cluster
	kustomize build argocd | kubectl apply -f -
	kustomize build applications | sed -e "s|FIXTURE_BRANCH|$(FIXTURE_BRANCH)|g" | kubectl apply -f -
	kubectl -n argocd rollout status statefulsets argocd-application-controller

wait-for-sync:
	./wait-for-sync-status.sh app1 $(FIXTURE_BRANCH)/main Synced Succeeded Healthy
	./wait-for-sync-status.sh app2 $(FIXTURE_BRANCH)/main Synced Succeeded Healthy
	./wait-for-sync-status.sh app3 $(FIXTURE_BRANCH)/main Synced Succeeded Healthy

deploy-controller: cluster
	cd controller && kustomize edit set image controller="$(CONTROLLER_IMAGE)"
	kustomize build controller | kubectl apply -f -
	bash controller/create-controller-manager-secret.sh
	kubectl -n argocd-commenter-system rollout status deployment argocd-commenter-controller-manager

undeploy:
	kubectl delete namespace/argocd namespace/argocd-commenter-system

# fixture
deploy-app1:
	kubectl -n argocd annotate application app1 'argocd-commenter.int128.github.io/deployment-url=$(DEPLOYMENT_URL)'
	$(MAKE) -C argocd-commenter-e2e-test-repository update-manifest-app1
	./wait-for-sync-status.sh app1 $(FIXTURE_BRANCH)/main Synced Succeeded Healthy

restart-app1:
	kubectl -n app1 delete pod -l app=echoserver --force
	./wait-for-sync-status.sh app1 $(FIXTURE_BRANCH)/main Synced Succeeded Progressing
	./wait-for-sync-status.sh app1 $(FIXTURE_BRANCH)/main Synced Succeeded Healthy

deploy-app2:
	kubectl -n argocd annotate application app2 'argocd-commenter.int128.github.io/deployment-url=$(DEPLOYMENT_URL)'
	$(MAKE) -C argocd-commenter-e2e-test-repository update-manifest-app2
	./wait-for-sync-status.sh app2 $(FIXTURE_BRANCH)/main OutOfSync Failed Healthy

deploy-app3:
	kubectl -n argocd annotate application app3 'argocd-commenter.int128.github.io/deployment-url=$(DEPLOYMENT_URL)'
	$(MAKE) -C argocd-commenter-e2e-test-repository update-manifest-app3
	./wait-for-sync-status.sh app3 $(FIXTURE_BRANCH)/main Synced Succeeded Healthy

logs-controller:
	-kubectl -n argocd-commenter-system logs -l control-plane=controller-manager --all-containers --tail=-1
logs-argocd:
	-kubectl -n argocd logs -l app.kubernetes.io/name=argocd-application-controller --all-containers --tail=-1
