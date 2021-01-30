CLUSTER_NAME := argocd-commenter-e2e
KUBECONFIG := output/kubeconfig.yaml
export KUBECONFIG

.PHONY: all
all: deploy

.PHONY: cluster
cluster: $(KUBECONFIG)
$(KUBECONFIG):
	kind create cluster --name $(CLUSTER_NAME)

.PHONY: delete-cluster
delete-cluster:
	kind delete cluster --name $(CLUSTER_NAME)
	-rm $(KUBECONFIG)

.PHONY: deploy
deploy: cluster
	kustomize build argocd | kubectl apply -f -
	kubectl -n argocd rollout status deployment argocd-server
	kubectl apply -f applications

.PHONY: undeploy
undeploy:
	kustomize build argocd | kubectl delete -f -
