GOPACKAGE=github.com/forceu/gokapi
OUTPUT_BIN=./gokapi
BUILD_FLAGS=-ldflags="-s -w -X '$(GOPACKAGE)/internal/environment.Builder=Make Script' -X '$(GOPACKAGE)/internal/environment.BuildTime=$(shell date)'"
DOCKER_IMAGE_NAME=gokapi
CONTAINER_TOOL ?= docker

# Default target
all: build

# Build Gokapi binary
build:
	@echo "Generating code..."
	@echo
	go generate ./...
	@echo "Building binary..."
	@echo
	CGO_ENABLED=0 go build $(BUILD_FLAGS) -o $(OUTPUT_BIN) $(GOPACKAGE)/cmd/gokapi

# Deletes binary
clean:
	@echo "Cleaning up..."
	rm -f $(OUTPUT_BIN)

# Create a Docker image
# Use make docker-build CONTAINER_TOOL=podman for podman instead of Docker
docker-build: build
	@echo "Building container image..."
	$(CONTAINER_TOOL) build . -t $(DOCKER_IMAGE_NAME)

# PHONY targets to avoid conflicts with files of the same name
.PHONY: all build clean docker-build

