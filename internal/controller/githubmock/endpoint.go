package githubmock

import (
	"encoding/json"
	"net/http"
	"sync/atomic"

	"github.com/google/go-github/v72/github"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func ListPullRequestsWithCommit(number int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("content-type", "application/json")
		w.WriteHeader(200)
		Expect(json.NewEncoder(w).Encode([]*github.PullRequest{{Number: github.Ptr(number)}})).Should(Succeed())
	}
}

func ListPullRequestFiles() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("content-type", "application/json")
		w.WriteHeader(200)
		Expect(json.NewEncoder(w).Encode([]*github.CommitFile{{Filename: github.Ptr("test/deployment.yaml")}})).Should(Succeed())
	}
}

type recorder struct {
	counter atomic.Int32
}

func (e *recorder) Count() int {
	return int(e.counter.Load())
}

type CreateComment struct {
	recorder
}

func (e *CreateComment) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	e.counter.Add(1)
	var req github.IssueComment
	Expect(json.NewDecoder(r.Body).Decode(&req)).Should(Succeed())
	GinkgoWriter.Println("GITHUB", "created comment", req)
	w.Header().Add("content-type", "application/json")
	w.WriteHeader(200)
}

type ListDeploymentStatus struct {
	Response []*github.DeploymentStatus
}

func (e *ListDeploymentStatus) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	w.Header().Add("content-type", "application/json")
	w.WriteHeader(200)
	Expect(json.NewEncoder(w).Encode(e.Response)).Should(Succeed())
}

type CreateDeploymentStatus struct {
	recorder
}

func (e *CreateDeploymentStatus) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	e.counter.Add(1)
	var req github.DeploymentStatusRequest
	Expect(json.NewDecoder(r.Body).Decode(&req)).Should(Succeed())
	GinkgoWriter.Println("GITHUB", "created deployment status", req)
	w.Header().Add("content-type", "application/json")
	w.WriteHeader(200)
}
