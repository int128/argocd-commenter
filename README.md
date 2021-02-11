# argocd-commenter

This is a Kubernetes Controller to add a comment to the pull request when ArgoCD performs sync operations.

TODO: example screenshot


## Getting Started

### Prerequisite

- ArgoCD is running on your cluster
- You need to generate a personal access token from https://github.com/settings/tokens

### Deploy

To deploy the manifest:

```shell
kubectl apply -f https://github.com/int128/argocd-commenter/releases/download/v0.2.0/argocd-commenter.yaml
kubectl -n argocd-commenter-system create secret generic controller-manager --from-literal="GITHUB_TOKEN=YOUR_PERSONAL_ACCESS_TOKEN"
```
