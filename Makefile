DOCKER_CMD ?= docker
DOCKER_REGISTRY ?= docker.io
DOCKER_ORG ?= emptystate
DOCKER_NAME ?= kafui
DOCKER_TAG ?= latest
BUILD_TAG ?= latest

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
run-kafka:
	cd example/dockercompose/ && docker compose up -d
docker-build:
	${DOCKER_CMD} build -t ${DOCKER_REGISTRY}/${DOCKER_ORG}/${DOCKER_NAME}:${DOCKER_TAG} .