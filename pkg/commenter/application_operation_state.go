package commenter

import (
	"context"
	"fmt"
	"strings"

	argocdv1alpha1 "github.com/argoproj/argo-cd/pkg/apis/application/v1alpha1"
	"github.com/argoproj/gitops-engine/pkg/sync/common"
	"github.com/go-logr/logr"
	"github.com/int128/argocd-commenter/pkg/github"
)

type ApplicationOperationState struct {
	Log logr.Logger
}

func (cmt *ApplicationOperationState) Do(ctx context.Context, application argocdv1alpha1.Application) error {
	if application.Status.OperationState == nil {
		cmt.Log.Info("skip nil operationState (never reach here)", "status", application.Status)
		return nil
	}
	repository, err := github.ParseRepositoryURL(application.Spec.Source.RepoURL)
	if err != nil {
		cmt.Log.Info("skip non-GitHub URL", "error", err)
		return nil
	}

	comment := github.CommitComment{
		Repository: *repository,
		CommitSHA:  application.Status.Sync.Revision,
		Body:       cmt.commentBody(application),
	}
	cmt.Log.Info("adding a comment", "comment", comment)
	if err := github.CreateCommitComment(ctx, comment); err != nil {
		return fmt.Errorf("could not add a comment: %w", err)
	}
	return nil
}

func (cmt *ApplicationOperationState) commentBody(application argocdv1alpha1.Application) string {
	var phase string
	switch application.Status.OperationState.Phase {
	case common.OperationRunning:
		phase = ":gear: Syncing"
	case common.OperationSucceeded:
		phase = ":white_check_mark: Sync OK"
	case common.OperationFailed:
		phase = ":x: Sync Failed"
	default:
		phase = fmt.Sprintf(":warning: Sync %s", application.Status.OperationState.Phase)
	}

	var resources strings.Builder
	if application.Status.OperationState.SyncResult != nil {
		for _, r := range application.Status.OperationState.SyncResult.Resources {
			namespacedName := r.Namespace + "/" + r.Name
			switch r.Status {
			case common.ResultCodeSynced:
				_, _ = fmt.Fprintf(&resources, "- :white_check_mark: `%s` %s\n", namespacedName, r.Message)
			case common.ResultCodeSyncFailed:
				_, _ = fmt.Fprintf(&resources, "- :x: `%s` %s\n", namespacedName, r.Message)
			default:
				_, _ = fmt.Fprintf(&resources, "- :warning: `%s` %s %s\n", namespacedName, r.Status, r.Message)
			}
		}
	}

	return fmt.Sprintf("## %s: %s -> %s\n%s",
		phase,
		application.Name,
		application.Status.Sync.Revision,
		resources.String(),
	)
}
