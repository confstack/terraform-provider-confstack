package bdd_test

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/confstack/terraform-provider-confstack/internal/domain"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("SensitiveFlatOutput", func() {
	var (
		tmpDir   string
		resolver = newTestResolver()
	)

	AfterEach(func() {
		if tmpDir != "" {
			_ = os.RemoveAll(tmpDir)
		}
	})

	Context("When secrets are used in layer files", func() {
		It("includes only secret keys (with plaintext values) in SensitiveFlatOutput", func() {
			tmpDir = setupFixture("secrets")
			layers := []string{
				filepath.Join(tmpDir, "_global", "defaults.common.yaml"),
			}

			result, err := resolveResult(
				resolver,
				layers,
				domain.WithSecrets(map[string]string{"DB_PASSWORD": "hunter2"}),
			)
			Expect(err).NotTo(HaveOccurred())

			// db.password is a standalone sentinel — plaintext value expected.
			Expect(result.SensitiveFlatOutput).To(HaveKey("db.password"))
			Expect(fmt.Sprintf("%v", result.SensitiveFlatOutput["db.password"])).To(Equal("hunter2"))

			// app.dsn is an inline sentinel — plaintext DSN expected.
			Expect(result.SensitiveFlatOutput).To(HaveKey("app.dsn"))
			Expect(fmt.Sprintf("%v", result.SensitiveFlatOutput["app.dsn"])).To(ContainSubstring("hunter2"))

			// Non-secret keys must be absent.
			Expect(result.SensitiveFlatOutput).NotTo(HaveKey("db.host"))
			Expect(result.SensitiveFlatOutput).NotTo(HaveKey("app.name"))
		})
	})

	Context("When no secrets are used", func() {
		It("returns an empty SensitiveFlatOutput", func() {
			tmpDir = setupFixture("templating_context")
			layers := []string{
				filepath.Join(tmpDir, "_global", "defaults.common.yaml"),
			}

			result, err := resolveResult(resolver, layers)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.SensitiveFlatOutput).To(BeEmpty())
		})
	})
})
