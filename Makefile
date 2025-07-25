DOCKER_CMD ?= docker
DOCKER_REGISTRY ?= docker.io
DOCKER_ORG ?= emptystate
DOCKER_NAME ?= kafui
DOCKER_TAG ?= latest
BUILD_TAG ?= latest

GOBIN ?= $$(go env GOPATH)/bin

.PHONY: build install run run-mock release release-snapshot test test-short test-integration test-benchmarks run-kafka stop-kafka docker-build


install-go-test-coverage:
	go install github.com/vladopajic/go-test-coverage/v2@latest

check-coverage: install-go-test-coverage
	go test ./... -coverprofile=./cover.out -covermode=atomic -coverpkg=./...
	${GOBIN}/go-test-coverage --config=./.testcoverage.yml

build:
	go build -ldflags "-w -s" .
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
