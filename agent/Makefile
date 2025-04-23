# Makefile for GitSynth

# Variables
BINARY_NAME=gitsynth
GO=go
GOBUILD=$(GO) build
GOCLEAN=$(GO) clean
GOTEST=$(GO) test
GOGET=$(GO) get
AGENT_DIR=.
OUTPUT_DIR=./bin

# Make sure binary output directory exists
$(shell mkdir -p $(OUTPUT_DIR))

# Default target
.PHONY: all
all: build

# Build the project
.PHONY: build
build:
	@echo "Building GitSynth..."
	$(GOBUILD) -o $(OUTPUT_DIR)/$(BINARY_NAME) -v

# Clean build files
.PHONY: clean
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -f $(OUTPUT_DIR)/$(BINARY_NAME)

# Run the application
.PHONY: run
run: build
	@echo "Running GitSynth..."
	./$(OUTPUT_DIR)/$(BINARY_NAME)

# Build for multiple platforms (for distribution)
.PHONY: build-all
build-all:
	@echo "Building for multiple platforms..."
	GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(OUTPUT_DIR)/$(BINARY_NAME)-linux-amd64
	GOOS=darwin GOARCH=amd64 $(GOBUILD) -o $(OUTPUT_DIR)/$(BINARY_NAME)-darwin-amd64
	GOOS=windows GOARCH=amd64 $(GOBUILD) -o $(OUTPUT_DIR)/$(BINARY_NAME)-windows-amd64.exe

# Get dependencies
.PHONY: deps
deps:
	@echo "Getting dependencies..."
	$(GOGET) -v ./...

# Test the project
.PHONY: test
test:
	@echo "Testing..."
	$(GOTEST) -v ./...

.PHONY: help
help:
	@echo "Available targets:"
	@echo "  all:       Build the project (default)"
	@echo "  build:     Build the application"
	@echo "  clean:     Clean build files"
	@echo "  run:       Build and run the application"
	@echo "  build-all: Build for multiple platforms"
	@echo "  deps:      Get dependencies"
	@echo "  test:      Run tests"
	@echo "  help:      Show this help message"