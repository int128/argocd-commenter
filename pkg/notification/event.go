package notification

import (
	"path/filepath"
	"strings"

	argocdv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
)

type Event struct {
	PhaseIsChanged        bool
	HealthIsChanged       bool
	ApplicationIsDeleting bool
	Application           argocdv1alpha1.Application
	ArgoCDURL             string
}

// GetManifestGeneratePaths returns canonical paths of "argocd.argoproj.io/manifest-generate-paths" annotation.
// It returns nil if the field is nil or empty.
// https://argo-cd.readthedocs.io/en/stable/operator-manual/high_availability/#webhook-and-manifest-paths-annotation
func (e Event) GetManifestGeneratePaths() []string {
	if e.Application.Annotations == nil {
		return nil
	}
	var canonicalPaths []string
	annotatedPaths := strings.Split(e.Application.Annotations["argocd.argoproj.io/manifest-generate-paths"], ";")
	for _, path := range annotatedPaths {
		if path == "" {
			return nil
		}
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
		canonicalPaths = append(canonicalPaths, absolutePath)
	}
	return canonicalPaths
}
