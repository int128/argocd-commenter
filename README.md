# argocd-commenter

This is a Kubernetes Controller to add a comment to the pull request when ArgoCD performs sync operations.

TODO: example screenshot


## Getting Started

### Prerequisite

ArgoCD must be deployed to your cluster.


### Deploy

To deploy the manifest:

```shell
kubectl apply -f https://github.com/int128/argocd-commenter/releases/download/v0.2.0/argocd-commenter.yaml
```

You need to create either Personal Access Token or GitHub App.

- Personal Access Token
    - Belong to a user
    - Share the rate limit in a user
- GitHub App
    - Belong to a user or organization
    - Have each rate limit for an installation

#### Option 1: Personal Access Token

1. Open https://github.com/settings/tokens
1. Generate a new token
1. Create a secret as follows:
    ```shell
    kubectl -n argocd-commenter-system create secret generic controller-manager \
      --from-literal="GITHUB_TOKEN=$YOUR_PERSONAL_ACCESS_TOKEN"
    ```

#### Option 2: GitHub App

1. Create your GitHub App from either link:
    - For a user: https://github.com/settings/apps/new?name=argocd-commenter&url=https://github.com/int128/argocd-commenter&webhook_active=false&pull_requests=write
    - For an organization: https://github.com/:org/settings/apps/new?name=argocd-commenter&url=https://github.com/int128/argocd-commenter&webhook_active=false&pull_requests=write
1. Get the **App ID** from the setting page
1. [Download a private key of the GitHub App](https://docs.github.com/en/developers/apps/authenticating-with-github-apps)
1. [Install your GitHub App on your repository or organization](https://docs.github.com/en/developers/apps/installing-github-apps)
1. Get the **Installation ID** from the URL, like `https://github.com/settings/installations/ID`
1. Create a secret as follows:
    ```shell
    kubectl -n argocd-commenter-system create secret generic controller-manager \
      --from-literal="GITHUB_APP_ID=$YOUR_GITHUB_APP_ID" \
      --from-literal="GITHUB_APP_INSTALLATION_ID=$YOUR_GITHUB_APP_INSTALLATION_ID" \
      --from-file="GITHUB_APP_PRIVATE_KEY=/path/to/private-key.pem"
    ```


## Contribution

This is an open source software. Feel free to contribute to it.

