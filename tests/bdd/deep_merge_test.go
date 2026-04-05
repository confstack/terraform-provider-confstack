package bdd_test

import (
	"os"
	"path/filepath"

	"github.com/confstack/terraform-provider-confstack/internal/usecase"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Deep Merge Engine", func() {
	var (
		tmpDir   string
		layers   []string
		resolver *usecase.Resolver
		err      error
		result   map[string]interface{}
	)

	BeforeEach(func() {
		tmpDir = setupFixture("deep_merge")
		resolver = newTestResolver()
		layers = []string{
			filepath.Join(tmpDir, "_global", "defaults.common.yaml"),
			filepath.Join(tmpDir, "dev", "config.common.yaml"),
		}
	})

	AfterEach(func() {
		os.RemoveAll(tmpDir)
	})

	Context("When performing a deep merge on maps (FR-05)", func() {
		BeforeEach(func() {
			result, err = resolveConfig(resolver, layers)
		})

		It("should successfully merge without error", func() {
			Expect(err).NotTo(HaveOccurred())
		})

		It("should retain keys that were not overridden", func() {
			nested := result["nested"].(map[string]interface{})
			l1 := nested["level1"].(map[string]interface{})
			l2 := l1["level2"].(map[string]interface{})
			Expect(l2).To(HaveKeyWithValue("base_key", true))
		})

		It("should add new keys from the higher priority source", func() {
			nested := result["nested"].(map[string]interface{})
			l1 := nested["level1"].(map[string]interface{})
			l2 := l1["level2"].(map[string]interface{})
			Expect(l2).To(HaveKeyWithValue("new_key", true))
		})

		It("should override scalar values at the leaf level", func() {
			nested := result["nested"].(map[string]interface{})
			l1 := nested["level1"].(map[string]interface{})
			l2 := l1["level2"].(map[string]interface{})
			Expect(l2).To(HaveKeyWithValue("override_me", "overridden"))
		})
	})

	Context("When replacing lists (FR-05)", func() {
		BeforeEach(func() {
			result, err = resolveConfig(resolver, layers)
		})

		It("should successfully resolve without error", func() {
			Expect(err).NotTo(HaveOccurred())
		})

		It("should replace the entire list rather than concatenate", func() {
			list, ok := result["my_list"].([]interface{})
			Expect(ok).To(BeTrue())
			Expect(list).To(HaveLen(1))
			Expect(list[0]).To(Equal("three"))
		})
	})

	Context("When deleting keys with null (FR-05)", func() {
		BeforeEach(func() {
			result, err = resolveConfig(resolver, layers)
		})

		It("should successfully resolve without error", func() {
			Expect(err).NotTo(HaveOccurred())
		})

		It("should explicitly remove keys that are set to null in higher priority", func() {
			Expect(result).NotTo(HaveKey("to_be_deleted"))
			nested := result["nested_delete"].(map[string]interface{})
			Expect(nested).NotTo(HaveKey("to_be_deleted"))
			Expect(nested).To(HaveKeyWithValue("kept", true))
		})
	})
})
