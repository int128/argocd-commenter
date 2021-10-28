# argocd-commenter [![docker](https://github.com/int128/argocd-commenter/actions/workflows/docker.yaml/badge.svg)](https://github.com/int128/argocd-commenter/actions/workflows/docker.yaml)

This is a Kubernetes Controller to notify a change of Argo CD Application status.


## Example: Pull Request

In the [GitOps](https://www.weave.works/technologies/gitops/) way, you merge a pull request to deploy a change to Kubernetes cluster.
argocd-commenter allows you to receive a notification after merge.

When an Application is syncing, synced or healthy, argocd-commenter creates a comment.
argocd-commenter determines a pull request from revision of Application.

![image](https://user-images.githubusercontent.com/321266/139166345-8edd77cb-319a-43df-b09a-40c18de74716.png)

When the sync was failed, argocd-commenter creates a comment.

![image](https://user-images.githubusercontent.com/321266/139166379-78b431b0-4439-4c86-9280-566424501ac4.png)

You can see the examples in [e2e-test](https://github.com/int128/argocd-commenter-e2e-test/pulls?q=is%3Apr+is%3Aclosed).


## Example: Deployment



![image](https://user-images.githubusercontent.com/321266/139166278-e74f6d1b-c722-430f-850c-2f7135e251d6.png)


## Getting Started

### Prerequisite

Argo CD is running in your Kubernetes cluster.


### Setup

To deploy the manifest:

```shell
kubectl apply -f https://github.com/int128/argocd-commenter/releases/download/v0.3.0/argocd-commenter.yaml
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
    - For a user: https://github.com/settings/apps/new?name=argocd-commenter&url=https://github.com/int128/argocd-commenter&webhook_active=false&contents=read&pull_requests=write
    - For an organization: https://github.com/organizations/:org/settings/apps/new?name=argocd-commenter&url=https://github.com/int128/argocd-commenter&webhook_active=false&contents=read&pull_requests=write (replace `:org` with your organization)
1. Get the **App ID** from the setting page
1. [Download a private key of the GitHub App](https://docs.github.com/en/developers/apps/authenticating-with-github-apps)
1. [Set a custom badge for the GitHub App](https://docs.github.com/en/developers/apps/creating-a-custom-badge-for-your-github-app)
    - Logo of Argo CD is available in [CNCF Branding](https://cncf-branding.netlify.app/projects/argo/)
1. [Install your GitHub App on your repository or organization](https://docs.github.com/en/developers/apps/installing-github-apps)
1. Get the **Installation ID** from the URL, like `https://github.com/settings/installations/ID`
1. Create a secret as follows:
    ```shell
    kubectl -n argocd-commenter-system create secret generic controller-manager \
      --from-literal="GITHUB_APP_ID=$YOUR_GITHUB_APP_ID" \
      --from-literal="GITHUB_APP_INSTALLATION_ID=$YOUR_GITHUB_APP_INSTALLATION_ID" \
      --from-file="GITHUB_APP_PRIVATE_KEY=/path/to/private-key.pem"
    ```


### Verify setup

Make sure the controller is running.

```shell
kubectl -n argocd-commenter-system rollout status deployment argocd-commenter-controller-manager
```


## Contribution

This is an open source software. Feel free to contribute to it.

