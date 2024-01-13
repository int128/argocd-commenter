package githubmock

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync/atomic"

	"github.com/google/go-github/v58/github"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func ListPullRequestsWithCommit(number int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("content-type", "application/json")
		w.WriteHeader(200)
		Expect(json.NewEncoder(w).Encode([]*github.PullRequest{{Number: github.Int(number)}})).Should(Succeed())
	}
}

func ListPullRequestFiles() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("content-type", "application/json")
		w.WriteHeader(200)
		Expect(json.NewEncoder(w).Encode([]*github.CommitFile{{Filename: github.String("test/deployment.yaml")}})).Should(Succeed())
	}
}

type Comment struct {
	createCounter atomic.Int32
}

func (e *Comment) CreateCount() int {
	return int(e.createCounter.Load())
}

func (e *Comment) Create() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("content-type", "application/json")
		w.WriteHeader(200)
		e.createCounter.Add(1)
		b, err := io.ReadAll(r.Body)
		Expect(err).Should(Succeed())
		GinkgoWriter.Println("GITHUB", "created comment", strings.TrimSpace(string(b)))
	}
}

type DeploymentStatus struct {
	createCounter atomic.Int32
	resp          []*github.DeploymentStatus
	NotFound      bool
}

func (e *DeploymentStatus) CreateCount() int {
	return int(e.createCounter.Load())
}

func (e *DeploymentStatus) Create() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("content-type", "application/json")
		w.WriteHeader(200)
		e.createCounter.Add(1)
		var req github.DeploymentStatusRequest
		Expect(json.NewDecoder(r.Body).Decode(&req)).Should(Succeed())
		GinkgoWriter.Println("GITHUB", "created deployment status", req)
		e.resp = []*github.DeploymentStatus{{State: req.State}}
	}
}

func (e *DeploymentStatus) List() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if e.NotFound {
			http.NotFound(w, r)
			return
		}
		w.Header().Add("content-type", "application/json")
		w.WriteHeader(200)
		if e.resp == nil {
			Expect(json.NewEncoder(w).Encode([]*github.DeploymentStatus{})).Should(Succeed())
			return
		}
		Expect(json.NewEncoder(w).Encode(e.resp)).Should(Succeed())
	}
}
