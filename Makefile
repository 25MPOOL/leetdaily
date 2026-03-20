GO ?= go
BINARY := bin/leetdaily

.PHONY: build ci fmt fmtcheck test vet verify

build:
	@mkdir -p bin
	$(GO) build -o $(BINARY) ./cmd/leetdaily

ci: verify build

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
