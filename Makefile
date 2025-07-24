DOCKER_CMD ?= docker
DOCKER_REGISTRY ?= docker.io
DOCKER_ORG ?= emptystate
DOCKER_NAME ?= kafui
DOCKER_TAG ?= latest
BUILD_TAG ?= latest

.PHONY: build install run run-mock release release-snapshot test test-short test-integration test-benchmarks run-kafka stop-kafka docker-build

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
	go test -v ./pkg/kafui/ -run "TestInit|TestDataSourceSwitching|TestConsumeTopicIntegration|TestConfigurationIntegration|TestUIWorkflowIntegration"
test-benchmarks:
	go test -bench=. -benchmem ./pkg/kafui/ -run "^$$"
run-kafka:
	cd example/dockercompose/ && docker compose up -d
stop-kafka:
	cd example/dockercompose/ && docker compose down
docker-build:
	${DOCKER_CMD} build -t ${DOCKER_REGISTRY}/${DOCKER_ORG}/${DOCKER_NAME}:${DOCKER_TAG} .
