package bdd_test

import (
	"os"
	"path/filepath"

	"github.com/confstack/terraform-provider-confstack/internal/usecase"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Advanced Resolution Logic", func() {
	var (
		tmpDir   string
		resolver *usecase.Resolver
		err      error
		result   map[string]interface{}
	)

	BeforeEach(func() {
		resolver = newTestResolver()
	})

	AfterEach(func() {
		if tmpDir != "" {
			_ = os.RemoveAll(tmpDir)
		}
	})

	Context("When handling multiple YAML documents in a single file (FR-03)", func() {
		BeforeEach(func() {
			tmpDir = setupFixture("multiple_docs")
			layers := []string{
				filepath.Join(tmpDir, "_global", "defaults.common.yaml"),
			}
			result, err = resolveConfig(resolver, layers)
		})

		It("should successfully merge all documents sequentially", func() {
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(HaveKeyWithValue("foo", "baz"))
			Expect(result).To(HaveKeyWithValue("new_key", true))
		})
	})

	Context("When resolving templates via bubble-up (FR-06)", func() {
		BeforeEach(func() {
			tmpDir = setupFixture("bubble_up")
			layers := []string{
				filepath.Join(tmpDir, "_global", "defaults.common.yaml"),
			}
			result, err = resolveConfig(resolver, layers)
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

	Context("When using Sprig functions in templates (FR-02)", func() {
		BeforeEach(func() {
			tmpDir = setupFixture("templating_context")
			layers := []string{
				filepath.Join(tmpDir, "_global", "defaults.common.yaml"),
			}
			result, err = resolveConfig(resolver, layers)
		})

		It("should correctly resolve Sprig helpers", func() {
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(HaveKeyWithValue("sprig_test", "HELLO WORLD"))
		})
	})
})
