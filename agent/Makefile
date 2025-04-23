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
NPM=npm

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

# Version management targets
.PHONY: version-patch version-minor version-major
version-patch:
	@echo "Bumping patch version..."
	$(NPM) version patch

version-minor:
	@echo "Bumping minor version..."
	$(NPM) version minor

version-major:
	@echo "Bumping major version..."
	$(NPM) version major

# Publish to npm registry
.PHONY: publish
publish: build-all
	@echo "Preparing binaries for npm package..."
	# Copy platform-specific binaries to proper locations
	cp $(OUTPUT_DIR)/$(BINARY_NAME)-linux-amd64 $(OUTPUT_DIR)/$(BINARY_NAME)-linux
	cp $(OUTPUT_DIR)/$(BINARY_NAME)-darwin-amd64 $(OUTPUT_DIR)/$(BINARY_NAME)-darwin
	cp $(OUTPUT_DIR)/$(BINARY_NAME)-windows-amd64.exe $(OUTPUT_DIR)/$(BINARY_NAME)-win.exe
	cp $(OUTPUT_DIR)/$(BINARY_NAME)-linux-amd64 $(OUTPUT_DIR)/$(BINARY_NAME)
	# Make sure binaries are executable
	chmod +x $(OUTPUT_DIR)/$(BINARY_NAME)*
	@echo "Publishing to npm..."
	$(NPM) publish
	@echo "Published successfully!"

.PHONY: help
help:
	@echo "Available targets:"
	@echo "  all:          Build the project (default)"
	@echo "  build:        Build the application"
	@echo "  clean:        Clean build files"
	@echo "  run:          Build and run the application"
	@echo "  build-all:    Build for multiple platforms"
	@echo "  deps:         Get dependencies"
	@echo "  test:         Run tests"
	@echo "  version-patch: Bump patch version (0.0.X)"
	@echo "  version-minor: Bump minor version (0.X.0)"
	@echo "  version-major: Bump major version (X.0.0)"
	@echo "  publish:      Build for all platforms and publish to npm registry"
	@echo "  help:         Show this help message"