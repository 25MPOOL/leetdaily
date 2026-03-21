GO ?= go
BINARY := bin/leetdaily
BOOTSTRAP_TERRAFORM_DIR := infra/bootstrap
APP_TERRAFORM_DIR := infra/terraform

.PHONY: all actionlint build ci clean fmt fmtcheck hooks-install pinact terraform-check terraform-fmtcheck terraform-validate test vet verify workflow-lint

all: build

clean:
	rm -rf bin

build:
	@mkdir -p bin
	$(GO) build -o $(BINARY) ./cmd/leetdaily

ci: verify build workflow-lint terraform-check

hooks-install:
	lefthook install

fmt:
	gofmt -w .

fmtcheck:
	@files="$$(gofmt -l .)"; \
	if [ -n "$$files" ]; then \
		printf '%s\n' "$$files"; \
		echo "run 'make fmt' to fix formatting"; \
		exit 1; \
	fi

test:
	$(GO) test ./...

vet:
	$(GO) vet ./...

verify: fmtcheck vet test

actionlint:
	actionlint

pinact:
	pinact run --check

workflow-lint: actionlint pinact

terraform-fmtcheck:
	terraform -chdir=$(BOOTSTRAP_TERRAFORM_DIR) fmt -check -recursive
	terraform -chdir=$(APP_TERRAFORM_DIR) fmt -check -recursive

terraform-validate:
	terraform -chdir=$(BOOTSTRAP_TERRAFORM_DIR) init -backend=false
	terraform -chdir=$(BOOTSTRAP_TERRAFORM_DIR) validate
	terraform -chdir=$(APP_TERRAFORM_DIR) init -backend=false
	terraform -chdir=$(APP_TERRAFORM_DIR) validate

terraform-check: terraform-fmtcheck terraform-validate
