package notification

import (
	"path"
	"slices"
	"strings"

	argocdv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/int128/argocd-commenter/internal/argocd"
	"github.com/int128/argocd-commenter/internal/github"
)

func filterPullRequestsRelatedToEvent(pulls []github.PullRequest, sourceRevision argocd.SourceRevision, app argocdv1alpha1.Application) []github.PullRequest {
	manifestGeneratePaths := getManifestGeneratePaths(app)

	var relatedPulls []github.PullRequest
	for _, pull := range pulls {
		if isPullRequestRelatedToEvent(pull, sourceRevision, manifestGeneratePaths) {
			relatedPulls = append(relatedPulls, pull)
		}
	}
	return relatedPulls
}

func isPullRequestRelatedToEvent(pull github.PullRequest, sourceRevision argocd.SourceRevision, manifestGeneratePaths []string) bool {
	for _, file := range pull.Files {
		if strings.HasPrefix(file, sourceRevision.Source.Path) {
			return true
		}
		for _, manifestGeneratePath := range manifestGeneratePaths {
			if strings.HasPrefix(file, manifestGeneratePath) {
				return true
			}
		}
	}
	return false
}

// getManifestGeneratePaths returns canonical paths of "argocd.argoproj.io/manifest-generate-paths" annotation.
// It returns nil if the field is nil or empty.
// https://argo-cd.readthedocs.io/en/stable/operator-manual/high_availability/#webhook-and-manifest-paths-annotation
// https://github.com/int128/argocd-commenter/pull/656
func getManifestGeneratePaths(app argocdv1alpha1.Application) []string {
	if app.Annotations == nil {
		return nil
	}
	var canonicalPaths []string
	manifestGeneratePaths := strings.Split(app.Annotations["argocd.argoproj.io/manifest-generate-paths"], ";")
	for _, manifestGeneratePath := range manifestGeneratePaths {
		if manifestGeneratePath == "" {
			return nil
		}

		if path.IsAbs(manifestGeneratePath) {
			// remove leading slash
			canonicalPath := manifestGeneratePath[1:]
			canonicalPaths = append(canonicalPaths, canonicalPath)
			continue
		}

		for _, source := range app.Spec.GetSources() {
			canonicalPath := path.Join(source.Path, manifestGeneratePath)
			if path.IsAbs(canonicalPath) {
				// remove leading slash
				canonicalPath = canonicalPath[1:]
			}
			canonicalPaths = append(canonicalPaths, canonicalPath)
		}
	}
	return slices.Compact(canonicalPaths)
}
