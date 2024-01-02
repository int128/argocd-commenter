package eventfilters

import (
	argocdv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// ApplicationChangedFunc is a function to compare the application.
// It must return true if the application is changed.
type ApplicationChangedFunc func(appOld, appNew argocdv1alpha1.Application) bool

// ApplicationChanged is an event filter triggering when the application is changed.
type ApplicationChanged ApplicationChangedFunc

var _ predicate.Predicate = ApplicationChanged(func(_, _ argocdv1alpha1.Application) bool { return false })

func (f ApplicationChanged) Update(e event.UpdateEvent) bool {
	appOld, ok := e.ObjectOld.(*argocdv1alpha1.Application)
	if !ok {
		return false
	}
	appNew, ok := e.ObjectNew.(*argocdv1alpha1.Application)
	if !ok {
		return false
	}
	return f(*appOld, *appNew)
}

func (ApplicationChanged) Create(event.CreateEvent) bool {
	return false
}
func (ApplicationChanged) Delete(event.DeleteEvent) bool {
	return false
}
func (ApplicationChanged) Generic(event.GenericEvent) bool {
	return false
}
