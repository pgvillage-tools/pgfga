all: inttest

build:
	go build ./cmd/pgfga

debug:
	~/go/bin/dlv debug --headless --listen=:2345 --api-version=2 --accept-multiclient ./cmd/pgfga

run:
	./pgfga

fmt:
	gofmt -w .

test: sec lint

sec:
	gosec ./...
lint:
	golangci-lint run

inttest:
	./docker-compose-tests.sh

# Same version as the go-test-coverage action in .github/workflows/ci-run-on-pr.yaml: Dependabot bumps that one,
# update this one with it. Since v2.20.0 it needs Go 1.27 to install (GOTOOLCHAIN=auto downloads it).
GO_TEST_COVERAGE_VERSION ?= v2.23.0

.PHONY: install-go-test-coverage
install-go-test-coverage:
	go install github.com/vladopajic/go-test-coverage/v2@$(GO_TEST_COVERAGE_VERSION)

.PHONY: test-coverage
test-coverage: ## Run the unit tests with a coverage profile (cover.out).
	go test $$(go list ./... | grep -v /e2e) -coverprofile=./cover.out -covermode=atomic -coverpkg=./...

.PHONY: check-coverage
check-coverage: install-go-test-coverage test-coverage
	go-test-coverage --config=./.testcoverage.yml

