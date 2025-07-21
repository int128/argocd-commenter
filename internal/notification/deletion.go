package notification

import (
	"context"
	"fmt"

	argocdv1alpha1 "github.com/argoproj/argo-cd/v3/pkg/apis/application/v1alpha1"
	"github.com/int128/argocd-commenter/internal/argocd"
	"github.com/int128/argocd-commenter/internal/github"
)

func (c client) CreateDeploymentStatusOnDeletion(ctx context.Context, app argocdv1alpha1.Application, argocdURL string) error {
	deploymentURL := argocd.GetDeploymentURL(app)
	deployment := github.ParseDeploymentURL(deploymentURL)
	if deployment == nil {
		return nil
	}
	ds := &DeploymentStatus{
		GitHubDeployment: *deployment,
		GitHubDeploymentStatus: github.DeploymentStatus{
			LogURL: fmt.Sprintf("%s/applications/%s", argocdURL, app.Name),
			State:  "inactive",
		},
	}

	if err := c.createDeploymentStatus(ctx, *ds); err != nil {
		return fmt.Errorf("unable to create a deployment status: %w", err)
	}
	return nil
}
