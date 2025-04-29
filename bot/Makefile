# Node.js and npm settings
NPM := npm

# Docker settings
DOCKER_IMAGE := bot
DOCKER_TAG := latest

# App settings (override these with environment variables when running make deploy)
APP_ID ?= <app-id>
PRIVATE_KEY ?= <pem-value>

.PHONY: all build run clean deploy help

all: build

build:
	@echo "Installing dependencies..."
	$(NPM) install

run: build
	@echo "Running the bot..."
	$(NPM) start

clean:
	@echo "Cleaning node_modules..."
	@rm -rf node_modules
	deploy:
	@echo "Building Docker image $(DOCKER_IMAGE):$(DOCKER_TAG)..."
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .
	@echo "Docker image built successfully"
	@echo "Starting container..."
	docker run -e APP_ID=$(APP_ID) -e PRIVATE_KEY=$(PRIVATE_KEY) $(DOCKER_IMAGE):$(DOCKER_TAG)

help:
	@echo "Available commands:"
	@echo "  make build   - Install dependencies"
	@echo "  make run     - Install dependencies and run the bot"
	@echo "  make clean   - Remove node_modules directory"
	@echo "  make deploy  - Build Docker image and run container"
	@echo "  make all     - Same as make build (default)"