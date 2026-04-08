package bdd_test

import (
	"os"
	"path/filepath"

	"github.com/confstack/terraform-provider-confstack/internal/domain"
	"github.com/confstack/terraform-provider-confstack/internal/usecase"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Glob Pattern Layer Expansion", func() {
	var (
		tmpDir   string
		resolver *usecase.Resolver
		err      error
		result   map[string]interface{}
	)

	BeforeEach(func() {
		tmpDir = setupFixture("glob_layers")
		resolver = newTestResolver()
	})

	AfterEach(func() {
		_ = os.RemoveAll(tmpDir)
	})

	Context("When layers contains a glob pattern matching multiple files", func() {
		BeforeEach(func() {
			globPattern := filepath.Join(tmpDir, "overrides", "*.yaml")
			result, err = resolveConfig(resolver, []string{globPattern})
		})

		It("should resolve without error", func() {
			Expect(err).NotTo(HaveOccurred())
		})

		It("should merge all matched files in alphabetical order (last wins)", func() {
			Expect(result).To(HaveKeyWithValue("env", "compute"))
		})

		It("should retain keys from earlier matched files", func() {
			Expect(result).To(HaveKeyWithValue("network_key", "network_val"))
			Expect(result).To(HaveKeyWithValue("compute_key", "compute_val"))
		})
	})

	Context("When layers mixes a literal path with a glob pattern", func() {
		BeforeEach(func() {
			basePath := filepath.Join(tmpDir, "base.yaml")
			globPattern := filepath.Join(tmpDir, "overrides", "*.yaml")
			result, err = resolveConfig(resolver, []string{basePath, globPattern})
		})

		It("should resolve without error", func() {
			Expect(err).NotTo(HaveOccurred())
		})

		It("should apply glob results after the literal (glob overrides literal)", func() {
			Expect(result).To(HaveKeyWithValue("env", "compute"))
		})

		It("should retain keys from the literal base layer", func() {
			Expect(result).To(HaveKeyWithValue("base_key", "base_val"))
		})

		It("should report 3 loaded layers total", func() {
			basePath := filepath.Join(tmpDir, "base.yaml")
			globPattern := filepath.Join(tmpDir, "overrides", "*.yaml")
			req, reqErr := domain.NewResolveRequest([]string{basePath, globPattern})
			Expect(reqErr).NotTo(HaveOccurred())
			resp, resolveErr := resolver.Resolve(nil, req) //nolint:staticcheck
			Expect(resolveErr).NotTo(HaveOccurred())
			Expect(resp.LoadedLayers).To(HaveLen(3))
		})
	})

	Context("When a glob pattern matches no files and on_missing_layer=skip", func() {
		It("should skip silently and not return an error", func() {
			basePath := filepath.Join(tmpDir, "base.yaml")
			noMatchGlob := filepath.Join(tmpDir, "nonexistent", "*.yaml")
			result, err = resolveConfig(resolver, []string{basePath, noMatchGlob},
				domain.WithOnMissingLayer("skip"),
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(HaveKeyWithValue("env", "base"))
		})
	})

	Context("When a glob pattern matches a file whose name contains glob metacharacters", func() {
		It("should treat the expanded match as a literal loaded layer", func() {
			bracketPath := filepath.Join(tmpDir, "overrides", "config[prod].yaml")
			err = os.WriteFile(bracketPath, []byte("env: bracketed\nbracket_key: bracket_val\n"), 0o644)
			Expect(err).NotTo(HaveOccurred())

			basePath := filepath.Join(tmpDir, "base.yaml")
			globPattern := filepath.Join(tmpDir, "overrides", "*.yaml")
			req, reqErr := domain.NewResolveRequest([]string{basePath, globPattern})
			Expect(reqErr).NotTo(HaveOccurred())

			resp, resolveErr := resolver.Resolve(nil, req) //nolint:staticcheck
			Expect(resolveErr).NotTo(HaveOccurred())
			Expect(resp.Output).To(HaveKeyWithValue("env", "bracketed"))
			Expect(resp.Output).To(HaveKeyWithValue("bracket_key", "bracket_val"))
			Expect(resp.LoadedLayers).To(ContainElement(bracketPath))
		})
	})
})
