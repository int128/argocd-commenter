# argocd-commenter

This is a Kubernetes Controller to add a comment to the pull request when ArgoCD performs sync operations.

TODO: example screenshot


## Getting Started

### Prerequisite

ArgoCD must be deployed to your cluster.

You need to create either Personal Access Token or GitHub App.

1. Personal Access Token
    - Belong to a user
    - Share the rate limit in a user
1. GitHub App
    - Belong to a user or organization
    - Have each rate limit for an installation

#### Option 1: Personal Access Token

Open https://github.com/settings/tokens and generate a new token.

#### Option 2: GitHub App

Create your GitHub App:

- For a user: https://github.com/settings/apps/new?name=argocd-commenter&url=https://github.com/int128/argocd-commenter&webhook_active=false&pull_requests=write
- For an organization: https://github.com/:org/settings/apps/new?name=argocd-commenter&url=https://github.com/int128/argocd-commenter&webhook_active=false&pull_requests=write

[Download a private key of the GitHub App](https://docs.github.com/en/developers/apps/authenticating-with-github-apps).
[Install your GitHub App on your repository or organization](https://docs.github.com/en/developers/apps/installing-github-apps).


### Deploy

To deploy the manifest:

```shell
kubectl apply -f https://github.com/int128/argocd-commenter/releases/download/v0.2.0/argocd-commenter.yaml
```

If you use your Personal Access Token, create a secret as follows:

```shell
kubectl -n argocd-commenter-system create secret generic controller-manager --from-literal="GITHUB_TOKEN=$YOUR_PERSONAL_ACCESS_TOKEN"
```

If you use your GitHub App, create a secret as follows:

```shell
kubectl -n argocd-commenter-system create secret generic controller-manager \
  --from-literal="GITHUB_APP_ID=$YOUR_GITHUB_APP_ID" \
  --from-literal="GITHUB_APP_INSTALLATION_ID=$YOUR_GITHUB_APP_INSTALLATION_ID" \
  --from-file="GITHUB_APP_PRIVATE_KEY=/path/to/your_github_app_private_key.pem"
```

