package notification

import (
	"fmt"
	"strings"

	argocdv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/argoproj/gitops-engine/pkg/health"
	"github.com/int128/argocd-commenter/internal/argocd"
	"github.com/int128/argocd-commenter/internal/github"
)

func NewDeploymentStatusOnHealthChanged(app argocdv1alpha1.Application, argocdURL string) *DeploymentStatus {
	deploymentURL := argocd.GetDeploymentURL(app)
	deployment := github.ParseDeploymentURL(deploymentURL)
	if deployment == nil {
		return nil
	}
	ds := generateHealthDeploymentStatus(app, argocdURL)
	if ds == nil {
		return nil
	}
	return &DeploymentStatus{
		GitHubDeployment:       *deployment,
		GitHubDeploymentStatus: *ds,
	}
}

func generateHealthDeploymentStatus(app argocdv1alpha1.Application, argocdURL string) *github.DeploymentStatus {
	ds := github.DeploymentStatus{
		LogURL: fmt.Sprintf("%s/applications/%s", argocdURL, app.Name),
	}
	if len(app.Status.Summary.ExternalURLs) > 0 {
		ds.EnvironmentURL = app.Status.Summary.ExternalURLs[0]
	}
	ds.Description = trimDescription(generateHealthDeploymentStatusDescription(app))
	switch app.Status.Health.Status {
	case health.HealthStatusHealthy:
		ds.State = "success"
		return &ds
	case health.HealthStatusDegraded:
		ds.State = "failure"
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

func NewDeploymentStatusOnDeletion(app argocdv1alpha1.Application, argocdURL string) *DeploymentStatus {
	deploymentURL := argocd.GetDeploymentURL(app)
	deployment := github.ParseDeploymentURL(deploymentURL)
	if deployment == nil {
		return nil
	}
	ds := github.DeploymentStatus{
		LogURL: fmt.Sprintf("%s/applications/%s", argocdURL, app.Name),
		State:  "inactive",
	}
	return &DeploymentStatus{
		GitHubDeployment:       *deployment,
		GitHubDeploymentStatus: ds,
	}
}

func trimDescription(s string) string {
	// The maximum description length is 140 characters.
	// https://docs.github.com/en/rest/deployments/statuses?apiVersion=2022-11-28#create-a-deployment-status
	if len(s) < 140 {
		return s
	}
	return s[0:139]
}
