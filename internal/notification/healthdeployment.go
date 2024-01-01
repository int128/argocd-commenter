package notification

import (
	"context"
	"fmt"
	"strings"

	argocdv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/argoproj/gitops-engine/pkg/health"
	"github.com/int128/argocd-commenter/internal/argocd"
	"github.com/int128/argocd-commenter/internal/github"
)

func (c client) CreateDeploymentStatusOnHealthChanged(ctx context.Context, app argocdv1alpha1.Application, argocdURL string) error {
	ds := generateDeploymentStatusOnHealthChanged(app, argocdURL)
	if ds == nil {
		return nil
	}
	if err := c.createDeploymentStatus(ctx, *ds); err != nil {
		return fmt.Errorf("unable to create a deployment status: %w", err)
	}
	return nil
}

func generateDeploymentStatusOnHealthChanged(app argocdv1alpha1.Application, argocdURL string) *DeploymentStatus {
	deploymentURL := argocd.GetDeploymentURL(app)
	deployment := github.ParseDeploymentURL(deploymentURL)
	if deployment == nil {
		return nil
	}

	ds := DeploymentStatus{
		GitHubDeployment: *deployment,
		GitHubDeploymentStatus: github.DeploymentStatus{
			LogURL:      fmt.Sprintf("%s/applications/%s", argocdURL, app.Name),
			Description: trimDescription(generateHealthDeploymentStatusDescription(app)),
		},
	}
	if len(app.Status.Summary.ExternalURLs) > 0 {
		ds.GitHubDeploymentStatus.EnvironmentURL = app.Status.Summary.ExternalURLs[0]
	}
	switch app.Status.Health.Status {
	case health.HealthStatusHealthy:
		ds.GitHubDeploymentStatus.State = "success"
		return &ds
	case health.HealthStatusDegraded:
		ds.GitHubDeploymentStatus.State = "failure"
		return &ds
	}
	return nil
}

func generateHealthDeploymentStatusDescription(app argocdv1alpha1.Application) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("%s:\n%s\n",
		app.Status.Health.Status,
		app.Status.Health.Message,
	))
	for _, r := range app.Status.Resources {
		if r.Health == nil {
			continue
		}
		namespacedName := r.Namespace + "/" + r.Name
		switch r.Health.Status {
		case health.HealthStatusDegraded, health.HealthStatusMissing:
			b.WriteString(fmt.Sprintf("%s: %s: %s\n", namespacedName, r.Health.Status, r.Health.Message))
		}
	}
	return b.String()
}

func trimDescription(s string) string {
	// The maximum description length is 140 characters.
	// https://docs.github.com/en/rest/deployments/statuses?apiVersion=2022-11-28#create-a-deployment-status
	if len(s) < 140 {
		return s
	}
	return s[0:139]
}
