DOCKER_CMD ?= docker
DOCKER_REGISTRY ?= docker.io
DOCKER_ORG ?= emptystate
DOCKER_NAME ?= kafui
DOCKER_TAG ?= latest
BUILD_TAG ?= latest

GOBIN ?= $$(go env GOPATH)/bin

.PHONY: build install run run-mock release release-snapshot test test-short test-integration test-benchmarks run-kafka stop-kafka docker-build
.PHONY: vhs vhs-install vhs-clean
.PHONY: build-debug run-debug test-debug


install-go-test-coverage:
	go install github.com/vladopajic/go-test-coverage/v2@latest

install-go-cover-treemap:
	go install github.com/nikolaydubina/go-cover-treemap@latest

check-coverage: install-go-test-coverage
	go test ./... -coverprofile=./coverage.out -covermode=atomic -coverpkg=./...
	${GOBIN}/go-test-coverage --config=./.testcoverage.yml

treemap-coverage: install-go-cover-treemap
	${GOBIN}/go-cover-treemap -coverprofile ./coverage.out > coverage.svg

build:
	go build -ldflags "-w -s" .

build-debug:
	go build -tags debug -ldflags "-w -s" -o kafui-debug .

run-debug:
	go run -tags debug -ldflags "-w -s" .

test-debug:
	go test -tags debug -v -cover -coverprofile=coverage.debug.out ./pkg/ui/debug/...
	go tool cover -html=coverage.debug.out -o coverage.debug.html
	@echo "Debug test coverage report generated: coverage.debug.html"

install:
	go install -ldflags "-w -s" .
run:
	go run -ldflags "-w -s" .
run-mock:
	go run -ldflags "-w -s" . --mock
release:
	goreleaser --clean
release-snapshot:
	goreleaser --clean --snapshot
test:
	go test -v -cover -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"
test-short:
	go test -short -v ./...
test-integration:
	./scripts/run_integration_tests.sh
test-benchmarks:
	go test -bench=. -benchmem ./pkg/kafui/ -run "^$$"
run-kafka:
	cd example/dockercompose/ && docker compose up -d
stop-kafka:
	cd example/dockercompose/ && docker compose down
docker-build:
	${DOCKER_CMD} build -t ${DOCKER_REGISTRY}/${DOCKER_ORG}/${DOCKER_NAME}:${DOCKER_TAG} .

# VHS Integration Tests

vhs-install:
	@echo "Installing VHS..."
	go install github.com/charmbracelet/vhs@latest
	@echo "VHS installed. Make sure $$GOPATH/bin is in your PATH"

vhs: vhs-install
	@echo "Running VHS topic navigation test..."
	go test ./test/vhs/... -run TestVHS_TopicNavigation -v

vhs-clean:
	@echo "Cleaning VHS output..."
	rm -rf test/vhs/output/*.gif
	@echo "VHS output cleaned"
