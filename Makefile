.PHONY: all test build lint lintmax docker-lint gosec govulncheck goreleaser tag-major tag-minor tag-patch release bump-glazed install codeql-local

all: test build

VERSION=v0.1.14

docker-lint:
	docker run --rm -v $(shell pwd):/app -w /app golangci/golangci-lint:v2.0.2 golangci-lint run -v

lint:
	golangci-lint run -v

lintmax:
	golangci-lint run -v --max-same-issues=100

test:
	go test ./...

build:
	go generate ./...
	go build ./...

#goreleaser:
# .goreleaser release --skip=sign --snapshot --clean

tag-major:
	git tag $(shell svu major)

tag-minor:
	git tag $(shell svu minor)

tag-patch:
	git tag $(shell svu patch)

release:
	git push origin --tags
	GOPROXY=proxy.golang.org go list -m github.com/go-go-golems/geppetto@$(shell svu current)

bump-glazed:
	go get github.com/go-go-golems/glazed@latest
	go get github.com/go-go-golems/clay@latest
	go mod tidy

gosec:
	go install github.com/securego/gosec/v2/cmd/gosec@latest
	gosec -exclude=G101,G304,G301,G306,G204 -exclude-dir=.history ./...

govulncheck:
	go install golang.org/x/vuln/cmd/govulncheck@latest
	govulncheck ./...

install:
	go build -o ./dist/geppetto ./cmd/geppetto && \
		cp ./dist/geppetto $(shell which geppetto)

# Path to CodeQL CLI - adjust based on installation location
CODEQL_PATH ?= $(shell which codeql)
# Path to CodeQL queries - adjust based on where you cloned the repository
CODEQL_QUERIES ?= $(HOME)/codeql-go/ql/src/go

# Create CodeQL database and run analysis
codeql-local:
	@if [ -z "$(CODEQL_PATH)" ]; then echo "CodeQL CLI not found. Install from https://github.com/github/codeql-cli-binaries/releases"; exit 1; fi
	@if [ ! -d "$(CODEQL_QUERIES)" ]; then echo "CodeQL queries not found. Clone from https://github.com/github/codeql-go"; exit 1; fi
	$(CODEQL_PATH) database create --language=go --source-root=. ./codeql-db
	$(CODEQL_PATH) database analyze ./codeql-db $(CODEQL_QUERIES)/Security --format=sarif-latest --output=codeql-results.sarif
	@echo "Results saved to codeql-results.sarif"
