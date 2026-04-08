default: build

build:
	go build -o bin/terraform-provider-confstack .

test:
	go test ./internal/... -v -timeout 120s

e2e:
	TF_ACC=1 go test ./tests/e2e/... -v -timeout 300s

bdd:
	go test ./tests/bdd/... -v

cover:
	go test ./internal/... -coverprofile=coverage.raw.out -covermode=atomic
	grep -vE "internal/adapter/driven/logging/" coverage.raw.out > coverage.out
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

.PHONY: build test e2e bdd cover lint fmt install clean generate
