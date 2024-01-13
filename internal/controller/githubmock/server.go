package githubmock

import (
	"fmt"
	"maps"
	"net/http"
	"sync"

	. "github.com/onsi/ginkgo/v2"
)

type Server struct {
	mu     sync.Mutex
	routes map[string]http.Handler
}

func (sv *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer GinkgoRecover()
	methodURI := fmt.Sprintf("%s %s", r.Method, r.RequestURI)
	GinkgoWriter.Println("GITHUB", methodURI)
	handler := sv.getHandler(methodURI)
	if handler == nil {
		http.NotFound(w, r)
		return
	}
	handler.ServeHTTP(w, r)
}

func (sv *Server) getHandler(methodURI string) http.Handler {
	sv.mu.Lock()
	defer sv.mu.Unlock()
	return sv.routes[methodURI]
}

func (sv *Server) Route(routes map[string]http.Handler) {
	sv.mu.Lock()
	defer sv.mu.Unlock()
	if sv.routes == nil {
		sv.routes = make(map[string]http.Handler)
	}
	maps.Copy(sv.routes, routes)
}
