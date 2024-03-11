DOCKER_CMD ?= docker
DOCKER_REGISTRY ?= docker.io
DOCKER_ORG ?= $(USER)
DOCKER_NAME ?= kafui
DOCKER_TAG ?= latest
BUILD_TAG ?= latest

build:
	go build -ldflags "-w -s" ./cmd/kafui
install:
	go install -ldflags "-w -s" ./cmd/kafui
run:
	go run -ldflags "-w -s" ./cmd/kafui
run-mock:
	go run -ldflags "-w -s" ./cmd/kafui --mock
release:
	goreleaser --rm-dist
run-kafka:
	cd example/dockercompose/ && docker compose up -d
docker-build:
	${DOCKER_CMD} build -t ${DOCKER_REGISTRY}/${DOCKER_ORG}/${DOCKER_NAME}:${DOCKER_TAG} .