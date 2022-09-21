package e2e_test

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("app1", func() {
	Context("When update the image tag", func() {
		It("Should be progressing state", func() {
			ctx := context.TODO()

			fixtureDir.Run(ctx, "git", "checkout", fixtureBaseBranch)
			fixtureDir.Run(ctx, "git", "checkout", "-b", fixtureBranchPrefix)

			Expect(false).Should(BeTrue())
		})
	})
})
