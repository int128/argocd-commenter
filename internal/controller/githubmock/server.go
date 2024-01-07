package githubmock

import (
	"fmt"
	"maps"
	"net/http"
	"sync"

	. "github.com/onsi/ginkgo/v2"
)

type Server struct {
	mu       sync.Mutex
	handlers map[string]http.Handler
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
	return sv.handlers[methodURI]
}

func (sv *Server) AddHandlers(handlers map[string]http.Handler) {
	sv.mu.Lock()
	defer sv.mu.Unlock()
	if sv.handlers == nil {
		sv.handlers = make(map[string]http.Handler)
	}
	maps.Copy(sv.handlers, handlers)
}
