package bdd_test

import (
	"context"
	"io"
	"os"
	"path/filepath"

	"github.com/confstack/terraform-provider-confstack/internal/adapter/driven/filesystem"
	"github.com/confstack/terraform-provider-confstack/internal/adapter/driven/logging"
	"github.com/confstack/terraform-provider-confstack/internal/adapter/driven/template"
	"github.com/confstack/terraform-provider-confstack/internal/adapter/driven/yaml"
	"github.com/confstack/terraform-provider-confstack/internal/domain"
	"github.com/confstack/terraform-provider-confstack/internal/usecase"
	. "github.com/onsi/gomega"
)

func newTestResolver() *usecase.Resolver {
	return usecase.NewResolver(
		filesystem.NewReader(),
		yaml.NewParser(),
		template.NewEngine(),
		logging.NewNopLogger(),
	)
}

func setupFixture(name string) string {
	tmpDir, err := os.MkdirTemp("", "confstack-bdd-*")
	Expect(err).NotTo(HaveOccurred())

	srcDir, err := filepath.Abs(filepath.Join("testdata", name))
	Expect(err).NotTo(HaveOccurred())

	err = filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		destPath := filepath.Join(tmpDir, relPath)

		if info.IsDir() {
			return os.MkdirAll(destPath, info.Mode())
		}

		srcFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer func() { _ = srcFile.Close() }()

		destFile, err := os.Create(destPath)
		if err != nil {
			return err
		}
		defer func() { _ = destFile.Close() }()

		_, err = io.Copy(destFile, srcFile)
		return err
	})
	Expect(err).NotTo(HaveOccurred())

	return tmpDir
}

func resolveConfig(resolver *usecase.Resolver, layers []string, opts ...func(*domain.ResolveRequest)) (map[string]interface{}, error) {
	allOpts := []func(*domain.ResolveRequest){
		domain.WithVariables(map[string]string{}),
		domain.WithSecrets(map[string]string{}),
	}
	allOpts = append(allOpts, opts...)

	req, err := domain.NewResolveRequest(layers, allOpts...)
	if err != nil {
		return nil, err
	}
	resp, err := resolver.Resolve(context.Background(), req)
	if err != nil {
		return nil, err
	}
	return resp.Output, nil
}
