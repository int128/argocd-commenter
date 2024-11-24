# E2E test for argocd-commenter

## Test environment

Here is a diagram of the test environment.

```mermaid
graph LR
  subgraph Local Cluster
    argo[Argo CD]
    set[ApplicationSet]
    set -. owner .-> app1[Application app1]
    set -. owner .-> app2[Application app2]
    set -. owner .-> app3[Application app3]
  end
  subgraph GitHub Repository
    subgraph Branch
      dir1[Directory app1]
      dir2[Directory app2]
      dir3[Directory app3]
    end
  end
  app1 -. source .-> dir1
  app2 -. source .-> dir2
  app3 -. source .-> dir3
  kubectl -- create --> set
  kubectl -- create --> argo
```

## Local development

### Prerequisites

- docker
- kind
- kustomize
- kubectl
- make
- git
- gh

### How to run

Set up a branch to deploy.

```sh
gh repo clone int128/argocd-commenter-e2e-test argocd-commenter-e2e-test-repository
make setup-fixture-branch
```

Set up a cluster and Argo CD.

```sh
make cluster
make deploy-argocd
make wait-for-apps
```

You can access the cluster.

```console
% export KUBECONFIG=output/kubeconfig.yaml
% k -n argocd get apps
NAME   SYNC STATUS   HEALTH STATUS
app1   Synced        Progressing
app2   Synced        Progressing
app3   Synced        Healthy
```

You can run the controller locally.

```sh
make -C .. run
```

### Clean up

```sh
make delete-cluster
```
