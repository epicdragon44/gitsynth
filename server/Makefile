# Makefile for GitSynth Server

# Variables
BINARY_NAME=gitsynth-server
GO=go
GOBUILD=$(GO) build
GOCLEAN=$(GO) clean
GOTEST=$(GO) test
GOGET=$(GO) get
GORUN=$(GO) run
SERVER_DIR=.
OUTPUT_DIR=./bin
CONFIG_FILE=config.yml

# Make sure binary output directory exists
$(shell mkdir -p $(OUTPUT_DIR))

# Default target
.PHONY: all
all: build

# Build the project
.PHONY: build
build:
	@echo "Building GitSynth Server..."
	$(GOBUILD) -o $(OUTPUT_DIR)/$(BINARY_NAME) -v ./...

# Clean build files
.PHONY: clean
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -rf $(OUTPUT_DIR)/$(BINARY_NAME)

# Run tests for the server
.PHONY: test
test:
	@echo "Testing server..."
	$(GOTEST) -v ./...

# Get dependencies
.PHONY: deps
deps:
	@echo "Getting dependencies..."
	$(GOGET) -v ./...

# Run the server directly using go run (for development)
.PHONY: run
run:
	@echo "Running server with go run..."
	$(GORUN) *.go

# Serve the server using the compiled binary
.PHONY: serve
serve: build
	@echo "Starting GitSynth Server..."
	./$(OUTPUT_DIR)/$(BINARY_NAME)

# Build for multiple platforms (for distribution)
.PHONY: build-all
build-all:
	@echo "Building for multiple platforms..."
	GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(OUTPUT_DIR)/$(BINARY_NAME)-linux-amd64 ./...
	GOOS=darwin GOARCH=amd64 $(GOBUILD) -o $(OUTPUT_DIR)/$(BINARY_NAME)-darwin-amd64 ./...
	GOOS=windows GOARCH=amd64 $(GOBUILD) -o $(OUTPUT_DIR)/$(BINARY_NAME)-windows-amd64.exe ./...

.PHONY: help
help:
	@echo "Available targets:"
	@echo "  all:          Build the project (default)"
	@echo "  build:        Build the server application"
	@echo "  clean:        Clean build files"
	@echo "  test:         Run tests"
	@echo "  deps:         Get dependencies"
	@echo "  run:          Run server directly using go run (for development)"
	@echo "  serve:        Build and serve the server using the compiled binary"
	@echo "  build-all:    Build for multiple platforms"
	@echo "  help:         Show this help message"