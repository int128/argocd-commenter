package notification

import (
	"path/filepath"
	"strings"

	argocdv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/int128/argocd-commenter/pkg/github"
)

func filterPullRequestsRelatedToEvent(pulls []github.PullRequest, app argocdv1alpha1.Application) []int {
	var numbers []int
	for _, pull := range pulls {
		if isPullRequestRelatedToEvent(pull, app) {
			numbers = append(numbers, pull.Number)
		}
	}
	return numbers
}

func isPullRequestRelatedToEvent(pull github.PullRequest, app argocdv1alpha1.Application) bool {
	if app.Spec.Source == nil {
		return false
	}

	// support manifest path annotation
	// see https://argo-cd.readthedocs.io/en/stable/operator-manual/high_availability/#webhook-and-manifest-paths-annotation
	// https://github.com/int128/argocd-commenter/pull/656
	manifestGeneratePaths := getManifestGeneratePaths(app)

	for _, file := range pull.Files {
		if strings.HasPrefix(file, app.Spec.Source.Path) {
			return true
		}
		for _, path := range manifestGeneratePaths {
			if strings.HasPrefix(file, path) {
				return true
			}
		}
	}
	return false
}

// getManifestGeneratePaths returns canonical paths of "argocd.argoproj.io/manifest-generate-paths" annotation.
// It returns nil if the field is nil or empty.
// https://argo-cd.readthedocs.io/en/stable/operator-manual/high_availability/#webhook-and-manifest-paths-annotation
func getManifestGeneratePaths(app argocdv1alpha1.Application) []string {
	if app.Annotations == nil {
		return nil
	}
	if app.Spec.Source == nil {
		return nil
	}
	var canonicalPaths []string
	annotatedPaths := strings.Split(app.Annotations["argocd.argoproj.io/manifest-generate-paths"], ";")
	for _, path := range annotatedPaths {
		if path == "" {
			return nil
		}
		// convert to absolute path
		absolutePath := path
		if !filepath.IsAbs(path) {
			absolutePath = filepath.Join(app.Spec.Source.Path, path)
		}
		// remove leading slash
		if absolutePath[0:1] == "/" {
			absolutePath = absolutePath[1:]
		}
		// add to list of manifest paths
		canonicalPaths = append(canonicalPaths, absolutePath)
	}
	return canonicalPaths
}
