package notification

import (
	"path"
	"slices"
	"strings"

	argocdv1alpha1 "github.com/argoproj/argo-cd/v3/pkg/apis/application/v1alpha1"
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
	absSourcePath := path.Join("/", sourceRevision.Source.Path)
	for _, file := range pull.Files {
		absPullFile := path.Join("/", file)
		if strings.HasPrefix(absPullFile, absSourcePath) {
			return true
		}
		for _, manifestGeneratePath := range manifestGeneratePaths {
			if strings.HasPrefix(absPullFile, manifestGeneratePath) {
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
	var absPaths []string
	manifestGeneratePaths := strings.Split(app.Annotations["argocd.argoproj.io/manifest-generate-paths"], ";")
	for _, manifestGeneratePath := range manifestGeneratePaths {
		if manifestGeneratePath == "" {
			return nil
		}
		if path.IsAbs(manifestGeneratePath) {
			absPaths = append(absPaths, path.Clean(manifestGeneratePath))
			continue
		}

		for _, source := range app.Spec.GetSources() {
			absPaths = append(absPaths, path.Join("/", source.Path, manifestGeneratePath))
		}
	}
	return slices.Compact(absPaths)
}
