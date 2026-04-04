default: build

build:
	go build -o bin/terraform-provider-confstack .

test:
	go test ./internal/... -v -timeout 120s

testacc:
	TF_ACC=1 go test ./internal/adapter/driving/terraform/... -v -timeout 300s

cover:
	go test ./internal/... -coverprofile=coverage.out -covermode=atomic
	go tool cover -html=coverage.out

lint:
	golangci-lint run ./...

fmt:
	gofmt -s -w .
	goimports -w .

install: build
	mkdir -p ~/.terraform.d/plugins/registry.terraform.io/confstack/confstack/0.1.0/linux_amd64/
	cp bin/terraform-provider-confstack ~/.terraform.d/plugins/registry.terraform.io/confstack/confstack/0.1.0/linux_amd64/

clean:
	rm -rf bin/

generate:
	go generate ./...

.PHONY: build test testacc cover lint fmt install clean generate
