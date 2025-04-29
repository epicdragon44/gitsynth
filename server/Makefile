# Compiler settings
GO := go
GOFMT := gofmt
BINARY_NAME := gitsynth-server
BINARY_DIR := bin
BINARY_PATH := $(BINARY_DIR)/$(BINARY_NAME)

# Build flags
BUILD_FLAGS := -v

# Docker settings
DOCKER_IMAGE := gitsynth-server
DOCKER_TAG := latest

# Source files
SRC_FILES := $(wildcard *.go)

# Build target
.PHONY: all build clean fmt help deploy check-env

all: clean fmt check-env build

build:
	@echo "Building server binary..."
	@mkdir -p $(BINARY_DIR)
	$(GO) build $(BUILD_FLAGS) -o $(BINARY_PATH) .
	@echo "Binary built at $(BINARY_PATH)"

clean:
	@echo "Cleaning build artifacts..."
	@rm -f $(BINARY_PATH)
	@echo "Clean complete"

fmt:
	@echo "Formatting code..."
	$(GOFMT) -w $(SRC_FILES)
	@echo "Format complete"

deploy: build
	@echo "Building Docker image $(DOCKER_IMAGE):$(DOCKER_TAG)..."
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) -f Dockerfile .
	@echo "Docker image built successfully"
	@echo "To run the container use: docker run -d -p 8080:8080 --privileged $(DOCKER_IMAGE):$(DOCKER_TAG)"

# Check for .env file and create from example if it doesn't exist
check-env:
	@if [ ! -f .env ]; then \
		echo "Creating .env file from .env.example..."; \
		cp .env.example .env; \
		echo "Please edit .env file with your actual API keys."; \
	else \
		echo ".env file already exists."; \
	fi

help:
	@echo "Available commands:"
	@echo "  make build     - Build the server binary"
	@echo "  make clean     - Remove the server binary"
	@echo "  make fmt       - Format the Go source code"
	@echo "  make check-env - Check for .env file and create from example if missing"
	@echo "  make deploy    - Build the server binary and create a Docker image"
	@echo "  make all       - Clean, format, check env, and build (default)"
