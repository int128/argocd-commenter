# E2E test for argocd-commenter

## Test environment

Here is a diagram of the test environment.

```mermaid
graph LR
  subgraph Local Cluster
    app1[Application app1]
    app2[Application app2]
    appN[Application appN]
    argo[Argo CD]
  end
  subgraph GitHub Repository<br>int128/argocd-commenter-e2e-test
    ref[Branch e2e-test/GITHUB_RUN_NUMBER/main]
  end
  app1 -.source.-> ref
  app2 -.source.-> ref
  appN -.source.-> ref
  kubectl --create--> argo
  kubectl --create--> app1
  kubectl --create--> app2
  kubectl --create--> appN
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
