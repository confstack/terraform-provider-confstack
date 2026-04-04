package bdd_test

import (
	"os"

	"github.com/confstack/terraform-provider-confstack/internal/usecase"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Advanced Resolution Logic", func() {
	var (
		tmpDir      string
		environment string
		tenant      string
		resolver    *usecase.Resolver
		err         error
		result      map[string]interface{}
	)

	BeforeEach(func() {
		environment = "dev"
		tenant = ""
		resolver = newTestResolver()
	})

	AfterEach(func() {
		if tmpDir != "" {
			os.RemoveAll(tmpDir)
		}
	})

	Context("When handling multiple YAML documents in a single file (FR-03)", func() {
		BeforeEach(func() {
			tmpDir = setupFixture("multiple_docs")
			result, err = resolveConfig(resolver, tmpDir, environment, tenant)
		})

		It("should successfully merge all documents sequentially", func() {
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(HaveKeyWithValue("foo", "baz"))
			Expect(result).To(HaveKeyWithValue("new_key", true))
		})
	})

	Context("When detecting case-insensitivity collisions in filenames (FR-01)", func() {
		BeforeEach(func() {
			tmpDir = setupFixture("case_collision")
			result, err = resolveConfig(resolver, tmpDir, environment, tenant)
		})

		It("should return a plan-time error to ensure deterministic behavior", func() {
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("case collision"))
		})
	})

	Context("When resolving templates via bubble-up (FR-06)", func() {
		BeforeEach(func() {
			tmpDir = setupFixture("bubble_up")
			result, err = resolveConfig(resolver, tmpDir, environment, tenant)
		})

		It("should find and resolve templates from parent maps", func() {
			Expect(err).NotTo(HaveOccurred())
			services := result["services"].(map[string]interface{})
			app1 := services["app1"].(map[string]interface{})
			Expect(app1).To(HaveKeyWithValue("retention", 3600))
			Expect(app1).To(HaveKeyWithValue("dlq", true))
			Expect(app1).To(HaveKeyWithValue("name", "service-one"))
		})
	})

	Context("When using templating context and Sprig functions (FR-02)", func() {
		BeforeEach(func() {
			tmpDir = setupFixture("templating_context")
			result, err = resolveConfig(resolver, tmpDir, "production", "")
		})

		It("should correctly resolve .Environment, .Tenant, and Sprig helpers", func() {
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(HaveKeyWithValue("env_name", "PRODUCTION"))
			Expect(result).To(HaveKeyWithValue("tenant_name", "no-tenant"))
			Expect(result).To(HaveKeyWithValue("sprig_test", "HELLO WORLD"))
		})
	})
})
