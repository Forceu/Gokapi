GOPACKAGE=github.com/forceu/gokapi
BUILD_FLAGS=-ldflags="-s -w -X '$(GOPACKAGE)/internal/environment.Builder=Make Script' -X '$(GOPACKAGE)/internal/environment.BuildTime=$(shell date)'"
BUILD_FLAGS_DEBUG=-ldflags="-X '$(GOPACKAGE)/internal/environment.Builder=Make Script' -X '$(GOPACKAGE)/internal/environment.BuildTime=$(shell date)'"
DOCKER_IMAGE_NAME=gokapi
CONTAINER_TOOL ?= docker

# Default target
.PHONY: all
all: build


.PHONY: build
# Build Gokapi binary
build : 
	@echo "Building binary..."
	@echo
	go generate ./...
	CGO_ENABLED=0 go build $(BUILD_FLAGS) -o ./gokapi $(GOPACKAGE)/cmd/gokapi

.PHONY: build-debug
# Build Gokapi binary
build-debug : 
	@echo "Building binary with debug info..."
	@echo
	go generate ./...
	CGO_ENABLED=0 go build $(BUILD_FLAGS_DEBUG) -o ./gokapi $(GOPACKAGE)/cmd/gokapi

.PHONY: coverage
coverage:
	@echo Generating coverage
	@echo
	GOKAPI_AWS_BUCKET="gokapi" GOKAPI_AWS_REGION="eu-central-1" GOKAPI_AWS_KEY="keyid" GOKAPI_AWS_KEY_SECRET="secret" go test ./... -parallel 8 --tags=test,awstest -coverprofile=/tmp/coverage1.out && go tool cover -html=/tmp/coverage1.out

.PHONY: coverage-specific
coverage-specific:
	@echo Generating coverage for "$(TEST_PACKAGE)"
	@echo
	go test  $(GOPACKAGE)/$(TEST_PACKAGE)/... -parallel 8 --tags=test,awsmock -coverprofile=/tmp/coverage2.out && go tool cover -html=/tmp/coverage2.out	


.PHONY: coverage-all
coverage-all:
	@echo Generating coverage
	@echo
	GOKAPI_AWS_BUCKET="gokapi" GOKAPI_AWS_REGION="eu-central-1" GOKAPI_AWS_KEY="keyid" GOKAPI_AWS_KEY_SECRET="secret" go test ./... -parallel 8 --tags=test,awstest -coverprofile=/tmp/coverage1.out && go tool cover -html=/tmp/coverage1.out


.PHONY: test
test:
	@echo Testing with AWS mock 
	@echo
	go test ./... -parallel 8 --tags=test,awsmock


.PHONY: test-specific
test-specific:
	@echo Testing package "$(TEST_PACKAGE)"
	@echo
	go test $(GOPACKAGE)/$(TEST_PACKAGE)/... -parallel 8 -count=1 --tags=test,awsmock


.PHONY: test-all
test-all:
	@echo Testing all tags 
	@echo
	go test ./... -parallel 8 --tags=test,noaws
	go test ./... -parallel 8 --tags=test,awsmock
	GOKAPI_AWS_BUCKET="gokapi" GOKAPI_AWS_REGION="eu-central-1" GOKAPI_AWS_KEY="keyid" GOKAPI_AWS_KEY_SECRET="secret" go test ./... -parallel 8 --tags=test,awstest

.PHONY: clean
# Deletes binary
clean:
	@echo "Cleaning up..."
	rm -f $(OUTPUT_BIN)

.PHONY: docker-build
# Create a Docker image
# Use make docker-build CONTAINER_TOOL=podman for podman instead of Docker
docker-build: build
	@echo "Building container image..."
	$(CONTAINER_TOOL) build . -t $(DOCKER_IMAGE_NAME)
