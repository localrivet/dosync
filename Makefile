# Go build optimizations: smaller binaries, reproducible builds
# -ldflags="-s -w": strip debug info and symbol tables
# -trimpath: remove file system paths for reproducibility
# CGO_ENABLED=0: pure-Go build (no C dependencies)
# Optionally, run 'upx --best --lzma <binary>' after build for further compression

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build $(LDFLAGS)
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GO111MODULE=on

# Determine GOOS and GOARCH
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

# Binary and versioning
BINARY_NAME=dosync
VERSION ?= $(shell git describe --tags --always || git rev-parse --short HEAD)
LDFLAGS=-ldflags "-X main.Version=${VERSION} -s -w" -trimpath
BUILD_DIR=release

# Docker Hub parameters
DOCKER=docker
IMAGE_NAME=localrivet/dosync
TAG ?= latest

export CGO_ENABLED=0

.PHONY: help
help:
	@echo "\nAvailable make targets:"
	@echo "  help           Show this help message"
	@echo "  build          Build for current platform (default Go env)"
	@echo "  build-linux    Build for Linux (amd64)"
	@echo "  build-darwin-arm64  Build for macOS (arm64)"
	@echo "  build-darwin-amd64  Build for macOS (amd64)"
	@echo "  build-all      Build for all major platforms (Linux/macOS)"
	@echo "  install        Install the binary to /usr/local/bin"
	@echo "  clean          Remove build artifacts"
	@echo "  run-dev        Build and run in development mode"
	@echo "  test           Run Go tests"
	@echo "  fmt            Run go fmt on source files"
	@echo "  run            Build and run the binary for the current platform"
	@echo "  docker-build   Build Docker image for Docker Hub (single platform)"
	@echo "  docker-tag     Tag Docker image with current version"
	@echo "  docker-push    Push Docker image (latest and version) to Docker Hub"
	@echo "  docker-buildx  Build and push multi-platform Docker image to Docker Hub (recommended)"
	@echo "  release        Commit, tag, push, and build/push multi-platform Docker images for a new version"
	@echo "  release-assets Build all binaries and create a GitHub release with attached assets (requires gh CLI)"
	@echo "  build-linux-arm64   Build for Linux (arm64)"
	@echo "  build-linux-armv7   Build for Linux (arm/v7)"
	@echo ""

.PHONY: all clean build build-linux build-darwin build-all

all: clean build-all

build:
	@echo "Building for current platform..."
	@mkdir -p $(BUILD_DIR)/$(GOOS)/$(GOARCH)
	$(GOBUILD) -o $(BUILD_DIR)/$(GOOS)/$(GOARCH)/$(BINARY_NAME) .

build-linux:
	@echo "Building for Linux (amd64)..."
	@mkdir -p $(BUILD_DIR)/linux/amd64
	GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BUILD_DIR)/linux/amd64/$(BINARY_NAME) .

build-darwin-arm64:
	@echo "Building for macOS (arm64)..."
	@mkdir -p $(BUILD_DIR)/darwin/arm64
	GOOS=darwin GOARCH=arm64 $(GOBUILD) -o $(BUILD_DIR)/darwin/arm64/$(BINARY_NAME) .

build-darwin-amd64:
	@echo "Building for macOS (amd64)..."
	@mkdir -p $(BUILD_DIR)/darwin/amd64
	GOOS=darwin GOARCH=amd64 $(GOBUILD) -o $(BUILD_DIR)/darwin/amd64/$(BINARY_NAME) .

build-linux-arm64:
	@echo "Building for Linux (arm64)..."
	@mkdir -p $(BUILD_DIR)/linux/arm64
	GOOS=linux GOARCH=arm64 $(GOBUILD) -o $(BUILD_DIR)/linux/arm64/$(BINARY_NAME) .

build-linux-armv7:
	@echo "Building for Linux (arm/v7)..."
	@mkdir -p $(BUILD_DIR)/linux/armv7
	GOOS=linux GOARCH=arm GOARM=7 $(GOBUILD) -o $(BUILD_DIR)/linux/armv7/$(BINARY_NAME) .

build-all: build-linux build-linux-arm64 build-linux-armv7 build-darwin-arm64 build-darwin-amd64

install: build
	@echo "Installing to /usr/local/bin/$(BINARY_NAME)..."
	@sudo cp $(BUILD_DIR)/$(GOOS)/$(GOARCH)/$(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)
	@sudo chmod +x /usr/local/bin/$(BINARY_NAME)
	@echo "Installation complete."

clean:
	@echo "Cleaning build directory..."
	@rm -rf $(BUILD_DIR)

# Helper for development testing
run-dev: build
	@echo "Running in development mode..."
	@./$(BUILD_DIR)/$(GOOS)/$(GOARCH)/$(BINARY_NAME) sync -f docker-compose.yml -i 30s --verbose

.PHONY: test
test:
	$(GOTEST) ./

.PHONY: fmt
fmt:
	$(GOCMD) fmt ./cmd/...
	$(GOCMD) fmt ./main.go

.PHONY: run
run:
	rm -f ./release/${GOOS}/$(GOARCH)/$(BINARY_NAME)
	make build
	./release/${GOOS}/$(GOARCH)/$(BINARY_NAME)

.PHONY: docker-build
# Build the Docker image for Docker Hub
docker-build:
	$(DOCKER) build -t $(IMAGE_NAME):$(TAG) .

.PHONY: docker-tag
# Tag the image with the current version (from git)
docker-tag:
	$(DOCKER) tag $(IMAGE_NAME):$(TAG) $(IMAGE_NAME):$(VERSION)

.PHONY: docker-push
# Push both :latest and :<version> tags to Docker Hub
docker-push:
	$(DOCKER) push $(IMAGE_NAME):$(TAG)
	$(DOCKER) push $(IMAGE_NAME):$(VERSION)

.PHONY: docker-buildx
# Build and push multi-platform images to Docker Hub (Alpine-based)
docker-buildx:
	$(DOCKER) buildx build --platform linux/amd64,linux/arm64,linux/arm/v7 \
		-t $(IMAGE_NAME):$(TAG) \
		-t $(IMAGE_NAME):$(VERSION) \
		--push .

.PHONY: release
release:
	@read -p "Enter release version (e.g., v1.0.0): " VERSION; \
	git add .; \
	git commit -m "Release $${VERSION}"; \
	git tag $${VERSION}; \
	git push origin $${VERSION}; \
	git push; \
	$(MAKE) docker-buildx VERSION=$${VERSION}; \
	$(MAKE) release-assets VERSION=$${VERSION}

.PHONY: release-assets
release-assets: build-all
	@read -p "Enter release version (e.g., v1.0.0): " VERSION; \
	gh release create $${VERSION} \
	  release/linux/amd64/dosync#dosync-linux-amd64 \
	  release/linux/arm64/dosync#dosync-linux-arm64 \
	  release/linux/armv7/dosync#dosync-linux-armv7 \
	  release/darwin/amd64/dosync#dosync-darwin-amd64 \
	  release/darwin/arm64/dosync#dosync-darwin-arm64 \
	  --title "$${VERSION}" --notes "Release $${VERSION}"
