# Define variables
IMAGE_NAME=gokapi-builder
CONTAINER_WORK_DIR=/src/gokapi

# Check for docker, then podman, otherwise use what the user provided
ifeq ($(origin CONTAINER_TOOL), undefined)
    CONTAINER_TOOL := $(shell command -v docker 2> /dev/null || command -v podman 2> /dev/null)
endif

# Fallback if neither is found and nothing was provided
CONTAINER_TOOL ?= docker

# Default target
all: compile

# Compile target
compile:
	@if [ -z "$$($(CONTAINER_TOOL) images -q $(IMAGE_NAME) 2>/dev/null)" ]; then \
		echo "Image $(IMAGE_NAME) not found. Building..."; \
		$(CONTAINER_TOOL) build . --tag $(IMAGE_NAME); \
	else \
		echo "Image $(IMAGE_NAME) already exists. Skipping build."; \
	fi
	@echo "Running build container to generate binaries"
	$(CONTAINER_TOOL) run --rm -it -v ../:$(CONTAINER_WORK_DIR) -w $(CONTAINER_WORK_DIR) $(IMAGE_NAME)
	
# Deletes binaries
clean:
	@echo "Deleting binaries..."
	rm -f ./*.zip

# Deletes binaries and docker image
clean-all:
	@echo "Deleting binaries and docker image..."
	rm -f ./*.zip
	$(CONTAINER_TOOL) image rm $(IMAGE_NAME)
	

# PHONY targets to avoid conflicts with files of the same name
.PHONY: all compile clean clean-all
