package githubmock

import (
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"strings"
	"sync"

	"github.com/google/go-github/v57/github"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type Server struct {
	mu       sync.Mutex
	handlers map[string]http.HandlerFunc

	Comments           Endpoint[int, any]
	DeploymentStatuses Endpoint[int, []*github.DeploymentStatus]
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
	handler(w, r)
}

func (sv *Server) getHandler(methodURI string) http.HandlerFunc {
	sv.mu.Lock()
	defer sv.mu.Unlock()
	return sv.handlers[methodURI]
}

func (sv *Server) AddHandlers(handlers map[string]http.HandlerFunc) {
	sv.mu.Lock()
	defer sv.mu.Unlock()
	if sv.handlers == nil {
		sv.handlers = make(map[string]http.HandlerFunc)
	}
	maps.Copy(sv.handlers, handlers)
}

func ListPullRequestsWithCommit(number int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("content-type", "application/json")
		w.WriteHeader(200)
		Expect(json.NewEncoder(w).Encode([]*github.PullRequest{{Number: github.Int(number)}})).Should(Succeed())
	}
}

func ListFiles() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("content-type", "application/json")
		w.WriteHeader(200)
		Expect(json.NewEncoder(w).Encode([]*github.CommitFile{{Filename: github.String("test/deployment.yaml")}})).Should(Succeed())
	}
}

func (sv *Server) CreateComment(number int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("content-type", "application/json")
		w.WriteHeader(200)
		sv.Comments.call(number)
		b, err := io.ReadAll(r.Body)
		Expect(err).Should(Succeed())
		GinkgoWriter.Println("GITHUB", "created comment", strings.TrimSpace(string(b)))
	}
}

func (sv *Server) CreateDeploymentStatus(id int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("content-type", "application/json")
		w.WriteHeader(200)
		sv.DeploymentStatuses.call(id)
		var ds github.DeploymentStatusRequest
		Expect(json.NewDecoder(r.Body).Decode(&ds)).Should(Succeed())
		GinkgoWriter.Println("GITHUB", "created deployment status", ds)
		sv.DeploymentStatuses.SetResponse(id, []*github.DeploymentStatus{{State: ds.State}})
	}
}

func (sv *Server) ListDeploymentStatus(id int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ds := sv.DeploymentStatuses.getResponse(id)
		if ds == nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Add("content-type", "application/json")
		w.WriteHeader(200)
		Expect(json.NewEncoder(w).Encode(ds)).Should(Succeed())
	}
}

type Endpoint[K int | string, V interface{}] struct {
	mu       sync.Mutex
	counter  map[K]int
	response map[K]V
}

func (r *Endpoint[K, V]) CountBy(key K) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.counter[key]
}

func (r *Endpoint[K, V]) call(key K) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.counter == nil {
		r.counter = make(map[K]int)
	}
	r.counter[key]++
	return r.counter[key]
}

func (r *Endpoint[K, V]) getResponse(k K) V {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.response[k]
}

func (r *Endpoint[K, V]) SetResponse(k K, v V) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.response == nil {
		r.response = make(map[K]V)
	}
	r.response[k] = v
}
