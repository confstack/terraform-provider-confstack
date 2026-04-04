package bdd_test

import (
	"os"

	"github.com/confstack/terraform-provider-confstack/internal/usecase"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Merge Priority Engine", func() {
	var (
		tmpDir      string
		environment string
		tenant      string
		resolver    *usecase.Resolver
		err         error
		result      map[string]interface{}
	)

	BeforeEach(func() {
		tmpDir = setupFixture("merge_priority")
		environment = "dev"
		tenant = "acme"
		resolver = newTestResolver()
	})

	AfterEach(func() {
		os.RemoveAll(tmpDir)
	})

	Context("When evaluating the 8-level priority order (FR-04)", func() {
		BeforeEach(func() {
			result, err = resolveConfig(resolver, tmpDir, environment, tenant)
		})

		It("should successfully merge all levels without error", func() {
			Expect(err).NotTo(HaveOccurred())
		})

		It("should retain the scalar value from the highest priority level (8)", func() {
			Expect(result).To(HaveKeyWithValue("level", 8))
		})

		It("should preserve keys from all levels in the merged map", func() {
			values, ok := result["values"].(map[string]interface{})
			Expect(ok).To(BeTrue())
			Expect(values).To(HaveKeyWithValue("l1", true))
			Expect(values).To(HaveKeyWithValue("l2", true))
			Expect(values).To(HaveKeyWithValue("l3", true))
			Expect(values).To(HaveKeyWithValue("l4", true))
			Expect(values).To(HaveKeyWithValue("l5", true))
			Expect(values).To(HaveKeyWithValue("l6", true))
			Expect(values).To(HaveKeyWithValue("l7", true))
			Expect(values).To(HaveKeyWithValue("l8", true))
		})

		It("should resolve overridden keys to the highest priority level", func() {
			values, ok := result["values"].(map[string]interface{})
			Expect(ok).To(BeTrue())
			Expect(values).To(HaveKeyWithValue("override_me", 8))
		})
	})
})
