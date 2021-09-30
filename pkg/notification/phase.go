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

func (c client) NotifyPhase(ctx context.Context, a argocdv1alpha1.Application, argoCDURL string) error {
	logger := logr.FromContextOrDiscard(ctx)
	argocdApplicationURL := fmt.Sprintf("%s/applications/%s", argoCDURL, a.Name)
	if err := c.notifyPhaseComment(ctx, logger, a, argocdApplicationURL); err != nil {
		logger.Error(err, "unable to notify a phase comment")
	}
	if err := c.notifyPhaseDeployment(ctx, logger, a, argocdApplicationURL); err != nil {
		logger.Error(err, "unable to notify a phase deployment")
	}
	return nil
}

func (c client) notifyPhaseComment(ctx context.Context, logger logr.Logger, a argocdv1alpha1.Application, argocdApplicationURL string) error {
	repository := github.ParseRepositoryURL(a.Spec.Source.RepoURL)
	if repository == nil {
		return nil
	}
	if a.Status.OperationState.Operation.Sync == nil {
		return fmt.Errorf("status.operationState.operation.sync == nil")
	}
	revision := a.Status.OperationState.Operation.Sync.Revision

	body := phaseCommentFor(a, argocdApplicationURL)

	logger.Info("creating a comment", "repository", repository, "revision", revision)
	if err := c.ghc.CreateComment(ctx, *repository, revision, body); err != nil {
		return fmt.Errorf("unable to create a comment: %w", err)
	}
	return nil
}

func phaseCommentFor(a argocdv1alpha1.Application, argocdApplicationURL string) string {
	revision := a.Status.OperationState.Operation.Sync.Revision
	if a.Status.OperationState.Phase == synccommon.OperationRunning {
		return fmt.Sprintf(":warning: Syncing [%s](%s) to %s", a.Name, argocdApplicationURL, revision)
	}
	if a.Status.OperationState.Phase == synccommon.OperationSucceeded {
		return fmt.Sprintf(":white_check_mark: Synced [%s](%s) to %s", a.Name, argocdApplicationURL, revision)
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("## :x: Sync %s: [%s](%s)\nError while syncing to %s:\n",
		a.Status.OperationState.Phase,
		a.Name,
		argocdApplicationURL,
		revision,
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

func (c client) notifyPhaseDeployment(ctx context.Context, logger logr.Logger, a argocdv1alpha1.Application, argocdApplicationURL string) error {
	deploymentURL := a.Annotations["argocd-commenter.int128.github.io/deployment-url"]
	deployment := github.ParseDeploymentURL(deploymentURL)
	if deployment == nil {
		return nil
	}

	deploymentStatus := github.DeploymentStatus{
		Description: fmt.Sprintf("Argo CD operation was %s", a.Status.OperationState.Phase),
		LogURL:      argocdApplicationURL,
	}
	if len(a.Status.Summary.ExternalURLs) > 0 {
		deploymentStatus.EnvironmentURL = a.Status.Summary.ExternalURLs[0]
	}
	switch a.Status.OperationState.Phase {
	case synccommon.OperationRunning:
		deploymentStatus.State = "queued"
	case synccommon.OperationSucceeded:
		deploymentStatus.State = "in_progress"
	default:
		deploymentStatus.State = "failure"
	}

	logger.Info("creating a deployment status", "deployment", deploymentURL)
	if err := c.ghc.CreateDeploymentStatus(ctx, *deployment, deploymentStatus); err != nil {
		return fmt.Errorf("unable to create a deployment status: %w", err)
	}
	return nil
}
