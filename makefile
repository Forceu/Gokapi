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

coverage:
	@echo Generating coverage
	@echo
	go test ./... -parallel 8 --tags=test,awsmock -coverprofile=/tmp/coverage1.out && go tool cover -html=/tmp/coverage1.out

coverage-specific:
	@echo Generating coverage for "$(TEST_PACKAGE)"
	@echo
	go test  $(GOPACKAGE)/$(TEST_PACKAGE)/... -parallel 8 --tags=test,awsmock -coverprofile=/tmp/coverage2.out && go tool cover -html=/tmp/coverage2.out	


test:
	@echo Testing with AWS mock 
	@echo
	go test ./... -parallel 8 --tags=test,awsmock	

test-specific:
	@echo Testing package "$(TEST_PACKAGE)"
	@echo
	go test  $(GOPACKAGE)/$(TEST_PACKAGE)/... -parallel 8 --tags=test,awsmock


test-all:
	@echo Testing all tags 
	@echo
	go test ./... -parallel 8 --tags=test,noaws
	go test ./... -parallel 8 --tags=test,awsmock
	GOKAPI_AWS_BUCKET="gokapi" GOKAPI_AWS_REGION="eu-central-1" GOKAPI_AWS_KEY="keyid" GOKAPI_AWS_KEY_SECRET="secret" go test ./... -parallel 8 --tags=test,awstest

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
.PHONY: all build clean coverage coverage-specific docker-build test test-all test-specific
