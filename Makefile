# Check if the environment file exists
ENVFILE := ./env/postgres.env
ifneq ("$(wildcard $(ENVFILE))","")
	include $(ENVFILE)
	export $(shell sed 's/=.*//' $(ENVFILE))
endif

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GO111MODULE=on

# Determine GOOS and GOARCH
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

# Docker parameters
EXECUTABLE=dosync
NAMESPACE=localrivet
DOCKER=docker
DOCKER_BUILD=$(DOCKER) build
REGISTRY=registry.digitalocean.com
REGISTRY_REPO=${NAMESPACE}/${EXECUTABLE}
TAG=latest
REGISTRY_URL=$(REGISTRY)/$(REGISTRY_REPO)
$(eval COMMIT_HASH := $(shell git rev-parse --short HEAD))
TIMESTAMP ?= $(shell date +"%Y%m%d%H%M%S")
	VERSION ?= $(shell git describe --tags --always || git rev-parse --short HEAD)
LDFLAGS ?= -X 'main.Version=$(VERSION)'

BINARY_NAME=dosync
VERSION?=0.1.0
LDFLAGS=-ldflags "-X main.Version=${VERSION}"
BUILD_DIR=release

.PHONY: all clean build build-linux build-darwin build-all

all: clean build-all

build:
	@echo "Building for current platform..."
	@mkdir -p $(BUILD_DIR)/$(shell go env GOOS)/$(shell go env GOARCH)
	@go build $(LDFLAGS) -o $(BUILD_DIR)/$(shell go env GOOS)/$(shell go env GOARCH)/$(BINARY_NAME) .

build-linux:
	@echo "Building for Linux (amd64)..."
	@mkdir -p $(BUILD_DIR)/linux/amd64
	@GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/linux/amd64/$(BINARY_NAME) .

build-darwin-arm64:
	@echo "Building for macOS (arm64)..."
	@mkdir -p $(BUILD_DIR)/darwin/arm64
	@GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/darwin/arm64/$(BINARY_NAME) .

build-darwin-amd64:
	@echo "Building for macOS (amd64)..."
	@mkdir -p $(BUILD_DIR)/darwin/amd64
	@GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/darwin/amd64/$(BINARY_NAME) .

build-all: build-linux build-darwin-arm64 build-darwin-amd64

install: build
	@echo "Installing to /usr/local/bin/$(BINARY_NAME)..."
	@sudo cp $(BUILD_DIR)/$(shell go env GOOS)/$(shell go env GOARCH)/$(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)
	@sudo chmod +x /usr/local/bin/$(BINARY_NAME)
	@echo "Installation complete."

clean:
	@echo "Cleaning build directory..."
	@rm -rf $(BUILD_DIR)

# Helper for development testing
run-dev: build
	@echo "Running in development mode..."
	@./$(BUILD_DIR)/$(shell go env GOOS)/$(shell go env GOARCH)/$(BINARY_NAME) sync -f docker-compose.yml -i 30s --verbose

.PHONY: test
test:
	$(GOTEST) ./

.PHONY: fmt
fmt:
	$(GOCMD) fmt ./cmd/...
	$(GOCMD) fmt ./main.go

.PHONY: run
run:
	rm -f ./release/${GOOS}/$(GOARCH)/$(EXECUTABLE)
	make build
	./release/${GOOS}/$(GOARCH)/$(EXECUTABLE)

.PHONY: docker-build
docker-build:
	$(DOCKER_BUILD) -t $(REGISTRY_URL):latest -t $(REGISTRY_URL):$(TAG)-$(TIMESTAMP)-$(COMMIT_HASH) .

.PHONY: docker-push
docker-push:
	@echo "To push to DigitalOcean Container Registry, first login with:"
	@echo "doctl registry login"
	@echo "Then run the following commands:"
	@echo "docker push $(REGISTRY_URL):latest"
	@echo "docker push $(REGISTRY_URL):$(TAG)-$(TIMESTAMP)-$(COMMIT_HASH)"
