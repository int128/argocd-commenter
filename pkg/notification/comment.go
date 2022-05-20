package notification

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/argoproj/gitops-engine/pkg/health"
	synccommon "github.com/argoproj/gitops-engine/pkg/sync/common"
	"github.com/go-logr/logr"
	"github.com/int128/argocd-commenter/pkg/github"
)

func (c client) Comment(ctx context.Context, e Event) error {
	logger := logr.FromContextOrDiscard(ctx)

	if e.Application.Status.OperationState == nil {
		return fmt.Errorf("status.operationState == nil")
	}
	if e.Application.Status.OperationState.Operation.Sync == nil {
		return fmt.Errorf("status.operationState.operation.sync == nil")
	}

	repository := github.ParseRepositoryURL(e.Application.Spec.Source.RepoURL)
	if repository == nil {
		return nil
	}

	body := generateComment(e)
	if body == "" {
		logger.Info("nothing to comment", "event", e)
		return nil
	}

	revision := e.Application.Status.OperationState.Operation.Sync.Revision
	pulls, err := c.ghc.ListPullRequests(ctx, *repository, revision)
	if err != nil {
		return fmt.Errorf("unable to list pull requests of revision %s: %w", revision, err)
	}

	relatedPullNumbers := filterPullRequestsRelatedToEvent(pulls, e)
	logger.Info("creating a comment", "repository", repository, "pulls", relatedPullNumbers)
	if err := c.ghc.CreateComment(ctx, *repository, relatedPullNumbers, body); err != nil {
		return fmt.Errorf("unable to create a comment: %w", err)
	}
	return nil
}

func filterPullRequestsRelatedToEvent(pulls []github.PullRequest, e Event) []int {
	var numbers []int
	for _, pull := range pulls {
		if isPullRequestRelatedToEvent(pull, e) {
			numbers = append(numbers, pull.Number)
		}
	}
	return numbers
}

func isPullRequestRelatedToEvent(pull github.PullRequest, e Event) bool {
	// support manifest path annotation
	// see https://argo-cd.readthedocs.io/en/stable/operator-manual/high_availability/#webhook-and-manifest-paths-annotation
	manifestPaths := []string{e.Application.Spec.Source.Path}
	annotatedPaths := strings.Split(e.Application.Annotations["argocd.argoproj.io/manifest-generate-paths"], ";")
	for _, path := range annotatedPaths {
		// convert to absolute path
		absolutePath := path
		if !filepath.IsAbs(path) {
			absolutePath = filepath.Join(e.Application.Spec.Source.Path, path)
		}
		// remove leading slash
		if absolutePath[0:1] == "/" {
			absolutePath = absolutePath[1:]
		}
		// add to list of manifest paths
		manifestPaths = append(manifestPaths, absolutePath)
	}

	for _, file := range pull.Files {
		for _, path := range manifestPaths {
			if strings.HasPrefix(file, path) {
				return true
			}
		}
	}
	return false
}

func generateComment(e Event) string {
	revision := e.Application.Status.OperationState.Operation.Sync.Revision
	argocdApplicationURL := fmt.Sprintf("%s/applications/%s", e.ArgoCDURL, e.Application.Name)

	if e.PhaseIsChanged {
		switch e.Application.Status.OperationState.Phase {
		case synccommon.OperationRunning:
			return fmt.Sprintf(":warning: Syncing [%s](%s) to %s", e.Application.Name, argocdApplicationURL, revision)
		case synccommon.OperationSucceeded:
			return fmt.Sprintf(":white_check_mark: Synced [%s](%s) to %s", e.Application.Name, argocdApplicationURL, revision)
		}

		var b strings.Builder
		b.WriteString(fmt.Sprintf("## :x: Sync %s: [%s](%s)\nError while syncing to %s:\n",
			e.Application.Status.OperationState.Phase,
			e.Application.Name,
			argocdApplicationURL,
			revision,
		))
		if e.Application.Status.OperationState.SyncResult != nil {
			for _, r := range e.Application.Status.OperationState.SyncResult.Resources {
				namespacedName := r.Namespace + "/" + r.Name
				switch r.Status {
				case synccommon.ResultCodeSyncFailed, synccommon.ResultCodePruneSkipped:
					b.WriteString(fmt.Sprintf("- %s `%s`: %s\n", r.Status, namespacedName, r.Message))
				}
			}
		}
		return b.String()
	}

	if e.HealthIsChanged {
		switch e.Application.Status.Health.Status {
		case health.HealthStatusHealthy:
			return fmt.Sprintf("## %s %s: [%s](%s)\nDeployed %s",
				":white_check_mark:",
				e.Application.Status.Health.Status,
				e.Application.Name,
				argocdApplicationURL,
				revision,
			)
		case health.HealthStatusDegraded:
			return fmt.Sprintf("## %s %s: [%s](%s)\nDeployed %s",
				":x:",
				e.Application.Status.Health.Status,
				e.Application.Name,
				argocdApplicationURL,
				revision,
			)
		}
	}

	return ""
}
