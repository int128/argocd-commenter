package notification

import (
	"context"
	"fmt"
	"strings"

	argocdv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/argoproj/gitops-engine/pkg/sync/common"
	"github.com/go-logr/logr"
	"github.com/int128/argocd-commenter/pkg/github"
)

func (c client) NotifyPhase(ctx context.Context, a argocdv1alpha1.Application) error {
	logger := logr.FromContextOrDiscard(ctx)

	repository := github.ParseRepositoryURL(a.Spec.Source.RepoURL)
	if repository == nil {
		return nil
	}

	logger.Info("creating a comment")
	comment := github.Comment{
		Repository: *repository,
		CommitSHA:  a.Status.Sync.Revision,
		Body:       phaseCommentFor(a),
	}
	if err := c.ghc.AddComment(ctx, comment); err != nil {
		return fmt.Errorf("unable to add a comment: %w", err)
	}

	deploymentURL := a.Annotations["argocd-commenter.int128.github.io/deployment-url"]
	deployment := github.ParseDeploymentURL(deploymentURL)
	if deployment == nil {
		return nil
	}
	logger.Info("creating a deployment status", "deployment", deploymentURL)
	deploymentStatus := github.DeploymentStatus{
		Deployment:  *deployment,
		State:       "failure",
		Description: string(a.Status.OperationState.Phase),
	}
	if err := c.ghc.CreateDeploymentStatus(ctx, deploymentStatus); err != nil {
		return fmt.Errorf("unable to create a deployment status: %w", err)
	}
	return nil
}

func phaseCommentFor(a argocdv1alpha1.Application) string {
	var resources strings.Builder
	if a.Status.OperationState.SyncResult != nil {
		for _, r := range a.Status.OperationState.SyncResult.Resources {
			namespacedName := r.Namespace + "/" + r.Name
			switch r.Status {
			case common.ResultCodeSyncFailed, common.ResultCodePruneSkipped:
				_, _ = fmt.Fprintf(&resources, "- %s `%s`: %s\n", r.Status, namespacedName, r.Message)
			}
		}
	}

	return fmt.Sprintf("## :x: Sync %s: %s\nError while syncing to %s\n%s",
		a.Status.OperationState.Phase,
		a.Name,
		a.Status.Sync.Revision,
		resources.String(),
	)
}
