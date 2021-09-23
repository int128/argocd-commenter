package notification

import (
	"context"
	"fmt"
	"strings"

	argocdv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	synccommon "github.com/argoproj/gitops-engine/pkg/sync/common"
	"github.com/go-logr/logr"
	"github.com/int128/argocd-commenter/pkg/github"
)

func (c client) NotifyPhase(ctx context.Context, a argocdv1alpha1.Application) error {
	logger := logr.FromContextOrDiscard(ctx)
	if err := c.notifyPhaseComment(ctx, logger, a); err != nil {
		logger.Error(err, "unable to notify a phase comment")
	}
	if err := c.notifyPhaseDeployment(ctx, logger, a); err != nil {
		logger.Error(err, "unable to notify a phase deployment")
	}
	return nil
}

func (c client) notifyPhaseComment(ctx context.Context, logger logr.Logger, a argocdv1alpha1.Application) error {
	repository := github.ParseRepositoryURL(a.Spec.Source.RepoURL)
	if repository == nil {
		return nil
	}

	logger.Info("creating a comment", "repository", repository)
	body := phaseCommentFor(a)
	if err := c.ghc.CreateComment(ctx, *repository, a.Status.Sync.Revision, body); err != nil {
		return fmt.Errorf("unable to create a comment: %w", err)
	}
	return nil
}

func phaseCommentFor(a argocdv1alpha1.Application) string {
	if a.Status.OperationState.Phase == synccommon.OperationRunning {
		return fmt.Sprintf(":warning: %s: Sync %s", a.Name, a.Status.OperationState.Phase)
	}
	if a.Status.OperationState.Phase == synccommon.OperationSucceeded {
		return fmt.Sprintf(":white_check_mark: %s: Sync %s", a.Name, a.Status.OperationState.Phase)
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("## :x: %s: Sync %s\nError while syncing to %s\n",
		a.Name,
		a.Status.OperationState.Phase,
		a.Status.Sync.Revision,
	))
	if a.Status.OperationState.SyncResult != nil {
		for _, r := range a.Status.OperationState.SyncResult.Resources {
			namespacedName := r.Namespace + "/" + r.Name
			switch r.Status {
			case synccommon.ResultCodeSyncFailed, synccommon.ResultCodePruneSkipped:
				b.WriteString(fmt.Sprintf("- %s `%s`: %s\n", r.Status, namespacedName, r.Message))
			}
		}
	}
	return b.String()
}

func (c client) notifyPhaseDeployment(ctx context.Context, logger logr.Logger, a argocdv1alpha1.Application) error {
	deploymentURL := a.Annotations["argocd-commenter.int128.github.io/deployment-url"]
	deployment := github.ParseDeploymentURL(deploymentURL)
	if deployment == nil {
		return nil
	}

	logger.Info("creating a deployment status", "deployment", deploymentURL)
	deploymentStatus := phaseDeploymentStatusFor(a)
	if err := c.ghc.CreateDeploymentStatus(ctx, *deployment, deploymentStatus); err != nil {
		return fmt.Errorf("unable to create a deployment status: %w", err)
	}
	return nil
}

func phaseDeploymentStatusFor(a argocdv1alpha1.Application) github.DeploymentStatus {
	switch a.Status.OperationState.Phase {
	case synccommon.OperationRunning:
		return github.DeploymentStatus{State: "queued", Description: string(a.Status.OperationState.Phase)}
	case synccommon.OperationSucceeded:
		return github.DeploymentStatus{State: "in_progress", Description: string(a.Status.OperationState.Phase)}
	}
	return github.DeploymentStatus{State: "failure", Description: string(a.Status.OperationState.Phase)}
}
