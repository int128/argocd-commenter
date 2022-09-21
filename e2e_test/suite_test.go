package e2e_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	fixtureDir          = WorkingDir(os.Getenv("FIXTURE_DIR"))
	fixtureBaseBranch   = os.Getenv("FIXTURE_BASE_BRANCH")
	fixtureBranchPrefix = os.Getenv("FIXTURE_BRANCH_PREFIX")
)

func TestE2E(t *testing.T) {
	if fixtureDir == "" {
		t.Skipf("FIXTURE_DIR is not set")
	}

	RegisterFailHandler(Fail)
	RunSpecs(t, "E2E Suite")
}

type WorkingDir string

func (wd WorkingDir) Run(ctx context.Context, name string, args ...string) {
	By(fmt.Sprintf("Run %s %s", name, strings.Join(args, " ")), func() {
		c := exec.CommandContext(ctx, name, args...)
		c.Stdout = GinkgoWriter
		c.Stderr = GinkgoWriter
		c.Dir = string(wd)
		err := c.Run()
		Expect(err).NotTo(HaveOccurred())
	})
}
