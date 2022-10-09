package controllers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/google/go-github/v47/github"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type Recorder[K int | string] struct {
	m       sync.Mutex
	counter map[K]int
}

func (r *Recorder[K]) CountBy(key K) int {
	r.m.Lock()
	defer r.m.Unlock()
	return r.counter[key]
}

func (r *Recorder[K]) call(key K) int {
	r.m.Lock()
	defer r.m.Unlock()

	if r.counter == nil {
		r.counter = make(map[K]int)
	}
	r.counter[key]++
	return r.counter[key]
}

type GithubMock struct {
	Comments           Recorder[int]
	DeploymentStatuses Recorder[int]
}

func (m *GithubMock) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	methodURI := fmt.Sprintf("%s %s", r.Method, r.RequestURI)
	GinkgoWriter.Printf("GITHUB %s\n", methodURI)

	handlers := map[string]http.HandlerFunc{
		"GET /api/v3/repos/int128/manifests/commits/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa100/pulls": m.listPullRequestsWithCommit(100),
		"GET /api/v3/repos/int128/manifests/pulls/100/files":                                        m.listFiles(),
		"POST /api/v3/repos/int128/manifests/issues/100/comments":                                   m.createComment(100),
		"POST /api/v3/repos/int128/manifests/deployments/999100/statuses":                           m.createDeploymentStatus(999100),

		"GET /api/v3/repos/int128/manifests/commits/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa101/pulls": m.listPullRequestsWithCommit(101),
		"GET /api/v3/repos/int128/manifests/pulls/101/files":                                        m.listFiles(),
		"POST /api/v3/repos/int128/manifests/issues/101/comments":                                   m.createComment(101),
		"POST /api/v3/repos/int128/manifests/deployments/999101/statuses":                           m.createDeploymentStatus(999101),

		"GET /api/v3/repos/int128/manifests/commits/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa200/pulls": m.listPullRequestsWithCommit(200),
		"GET /api/v3/repos/int128/manifests/pulls/200/files":                                        m.listFiles(),
		"POST /api/v3/repos/int128/manifests/issues/200/comments":                                   m.createComment(200),

		"GET /api/v3/repos/int128/manifests/commits/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa201/pulls": m.listPullRequestsWithCommit(201),
		"GET /api/v3/repos/int128/manifests/pulls/201/files":                                        m.listFiles(),
		"POST /api/v3/repos/int128/manifests/issues/201/comments":                                   m.createComment(201),

		"POST /api/v3/repos/int128/manifests/deployments/999202/statuses": m.createDeploymentStatus(999202),
		"POST /api/v3/repos/int128/manifests/deployments/999203/statuses": m.createDeploymentStatus(999203),
	}

	handler, ok := handlers[methodURI]
	if !ok {
		http.NotFound(w, r)
		return
	}
	handler(w, r)
}

func (m *GithubMock) listPullRequestsWithCommit(number int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("content-type", "application/json")
		w.WriteHeader(200)
		Expect(json.NewEncoder(w).Encode([]*github.PullRequest{{Number: github.Int(number)}})).Should(Succeed())
	}
}

func (m *GithubMock) listFiles() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("content-type", "application/json")
		w.WriteHeader(200)
		Expect(json.NewEncoder(w).Encode([]*github.CommitFile{{Filename: github.String("test/deployment.yaml")}})).Should(Succeed())
	}
}

func (m *GithubMock) createComment(number int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("content-type", "application/json")
		w.WriteHeader(200)
		m.Comments.call(number)
		b, err := io.ReadAll(r.Body)
		Expect(err).Should(Succeed())
		GinkgoWriter.Printf("GITHUB comment %s\n", string(b))
	}
}

func (m *GithubMock) createDeploymentStatus(id int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("content-type", "application/json")
		w.WriteHeader(200)
		m.DeploymentStatuses.call(id)
		b, err := io.ReadAll(r.Body)
		Expect(err).Should(Succeed())
		GinkgoWriter.Printf("GITHUB deployment %s\n", string(b))
	}
}
