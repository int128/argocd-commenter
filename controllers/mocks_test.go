package controllers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
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

func (m *GithubMock) NewHandler() http.Handler {
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

		"GET /api/v3/repos/int128/manifests/commits/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa/pulls": http.NotFound,
		"POST /api/v3/repos/int128/manifests/deployments/999300/statuses":                           m.createDeploymentStatus(999300),
		"POST /api/v3/repos/int128/manifests/deployments/999301/statuses":                           m.createDeploymentStatus(999301),
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer GinkgoRecover()
		methodURI := fmt.Sprintf("%s %s", r.Method, r.RequestURI)
		GinkgoWriter.Println("GITHUB", methodURI)
		handler, ok := handlers[methodURI]
		Expect(ok).Should(BeTrue(), methodURI)
		handler(w, r)
	})
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
		GinkgoWriter.Println("GITHUB", "created comment", strings.TrimSpace(string(b)))
	}
}

func (m *GithubMock) createDeploymentStatus(id int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("content-type", "application/json")
		w.WriteHeader(200)
		m.DeploymentStatuses.call(id)
		b, err := io.ReadAll(r.Body)
		Expect(err).Should(Succeed())
		GinkgoWriter.Println("GITHUB", "created deployment status", strings.TrimSpace(string(b)))
	}
}
