package bdd_test

import (
	"os"
	"path/filepath"

	"github.com/confstack/terraform-provider-confstack/internal/usecase"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Merge Priority Engine", func() {
	var (
		tmpDir   string
		resolver *usecase.Resolver
		err      error
		result   map[string]interface{}
	)

	BeforeEach(func() {
		tmpDir = setupFixture("merge_priority")
		resolver = newTestResolver()
	})

	AfterEach(func() {
		_ = os.RemoveAll(tmpDir)
	})

	Context("When merging 8 layers in priority order (last wins)", func() {
		BeforeEach(func() {
			layers := []string{
				filepath.Join(tmpDir, "layer1.yaml"),
				filepath.Join(tmpDir, "layer2.yaml"),
				filepath.Join(tmpDir, "layer3.yaml"),
				filepath.Join(tmpDir, "layer4.yaml"),
				filepath.Join(tmpDir, "layer5.yaml"),
				filepath.Join(tmpDir, "layer6.yaml"),
				filepath.Join(tmpDir, "layer7.yaml"),
				filepath.Join(tmpDir, "layer8.yaml"),
			}
			result, err = resolveConfig(resolver, layers)
		})

		It("should successfully merge all layers without error", func() {
			Expect(err).NotTo(HaveOccurred())
		})

		It("should retain the scalar value from the highest priority layer (8)", func() {
			Expect(result).To(HaveKeyWithValue("level", 8))
		})

		It("should preserve unique keys contributed by each layer", func() {
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

		It("should resolve overridden keys to the highest priority layer", func() {
			values, ok := result["values"].(map[string]interface{})
			Expect(ok).To(BeTrue())
			Expect(values).To(HaveKeyWithValue("override_me", 8))
		})
	})
})
