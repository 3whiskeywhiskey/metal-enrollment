.PHONY: build build-server build-builder build-ipxe-server run clean test docker-build deploy help

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Binary names
BINARY_SERVER=bin/server
BINARY_BUILDER=bin/builder
BINARY_IPXE=bin/ipxe-server

# Docker image names
IMAGE_PREFIX=metal-enrollment
SERVER_IMAGE=$(IMAGE_PREFIX)/server:latest
BUILDER_IMAGE=$(IMAGE_PREFIX)/builder:latest
IPXE_IMAGE=$(IMAGE_PREFIX)/ipxe-server:latest

all: build

help:
	@echo "Metal Enrollment - Makefile targets:"
	@echo "  build              - Build all binaries"
	@echo "  build-server       - Build server binary"
	@echo "  build-builder      - Build builder binary"
	@echo "  build-ipxe-server  - Build iPXE server binary"
	@echo "  run                - Run server locally"
	@echo "  test               - Run tests"
	@echo "  clean              - Clean build artifacts"
	@echo "  docker-build       - Build all Docker images"
	@echo "  deploy             - Deploy to Kubernetes"
	@echo "  build-registration - Build registration NixOS image"

build: build-server build-builder build-ipxe-server

build-server:
	@mkdir -p bin
	$(GOBUILD) -o $(BINARY_SERVER) ./cmd/server

build-builder:
	@mkdir -p bin
	$(GOBUILD) -o $(BINARY_BUILDER) ./cmd/builder

build-ipxe-server:
	@mkdir -p bin
	$(GOBUILD) -o $(BINARY_IPXE) ./cmd/ipxe-server

run: build-server
	$(BINARY_SERVER)

test:
	$(GOTEST) -v ./...

clean:
	$(GOCLEAN)
	rm -rf bin/
	rm -rf nixos/registration/result
	rm -f *.db

docker-build:
	docker build -t $(SERVER_IMAGE) -f deployments/docker/Dockerfile.server .
	docker build -t $(BUILDER_IMAGE) -f deployments/docker/Dockerfile.builder .
	docker build -t $(IPXE_IMAGE) -f deployments/docker/Dockerfile.ipxe-server .

deploy:
	kubectl apply -f deployments/kubernetes/namespace.yaml
	kubectl apply -f deployments/kubernetes/configmap.yaml
	kubectl apply -f deployments/kubernetes/pvc.yaml
	kubectl apply -f deployments/kubernetes/deployment-server.yaml
	kubectl apply -f deployments/kubernetes/deployment-builder.yaml
	kubectl apply -f deployments/kubernetes/deployment-ipxe-server.yaml
	kubectl apply -f deployments/kubernetes/ingress.yaml

build-registration:
	cd nixos/registration && ./build.sh

deps:
	$(GOMOD) download
	$(GOMOD) tidy

.DEFAULT_GOAL := help
