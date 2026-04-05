package bdd_test

import (
	"os"
	"path/filepath"

	"github.com/confstack/terraform-provider-confstack/internal/usecase"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Template Inheritance Engine", func() {
	var (
		tmpDir   string
		layers   []string
		resolver *usecase.Resolver
		err      error
		result   map[string]interface{}
	)

	BeforeEach(func() {
		tmpDir = setupFixture("inheritance")
		resolver = newTestResolver()
		layers = []string{
			filepath.Join(tmpDir, "_global", "defaults.common.yaml"),
			filepath.Join(tmpDir, "dev", "config.common.yaml"),
		}
	})

	AfterEach(func() {
		os.RemoveAll(tmpDir)
	})

	Context("When resolving templates and inheritance (FR-06)", func() {
		BeforeEach(func() {
			result, err = resolveConfig(resolver, layers)
		})

		It("should resolve successfully without error", func() {
			Expect(err).NotTo(HaveOccurred())
		})

		It("should completely strip the _templates blocks from the output", func() {
			sqs := result["sqs_queues"].(map[string]interface{})
			Expect(sqs).NotTo(HaveKey("_templates"))
		})

		It("should completely strip the _inherit blocks from the output", func() {
			sqs := result["sqs_queues"].(map[string]interface{})
			notifications := sqs["notifications"].(map[string]interface{})
			Expect(notifications).NotTo(HaveKey("_inherit"))
			orders := sqs["orders"].(map[string]interface{})
			Expect(orders).NotTo(HaveKey("_inherit"))
		})

		It("should resolve single inheritance and apply overrides", func() {
			sqs := result["sqs_queues"].(map[string]interface{})
			notifications := sqs["notifications"].(map[string]interface{})
			Expect(notifications).To(HaveKeyWithValue("dlq", true))
			Expect(notifications).To(HaveKeyWithValue("visibility_timeout", 30))
			// overridden by entry
			Expect(notifications).To(HaveKeyWithValue("retention", 3600))
		})

		It("should resolve multiple inheritance with exceptions", func() {
			sqs := result["sqs_queues"].(map[string]interface{})
			orders := sqs["orders"].(map[string]interface{})

			// From critical, overridden by entry
			Expect(orders).To(HaveKeyWithValue("visibility_timeout", 120))

			// From critical
			Expect(orders).To(HaveKeyWithValue("retention", 604800))
			Expect(orders).To(HaveKeyWithValue("dlq", true))
			Expect(orders).To(HaveKeyWithValue("dlq_max_retries", 5))
		})
	})
})
