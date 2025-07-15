# bcc-exporter Makefile

# Variables
BINARY_NAME=bcc-exporter
GO_FILES=$(shell find . -name "*.go" -type f)
BUILD_DIR=.

# Default target
.PHONY: all
all: build

# Build the binary
.PHONY: build
build: $(BINARY_NAME)

$(BINARY_NAME): $(GO_FILES) go.mod
	go build -o $(BINARY_NAME) .

# Clean build artifacts
.PHONY: clean
clean:
	rm -f $(BINARY_NAME)

# Run the application (requires sudo)
.PHONY: run
run: build
	sudo ./$(BINARY_NAME)

# Run with custom port
.PHONY: run-port
run-port: build
	sudo ./$(BINARY_NAME) -port $(PORT)

# Run with authentication
.PHONY: run-auth
run-auth: build
	sudo ./$(BINARY_NAME) -password $(PASSWORD)

# Format Go code
.PHONY: fmt
fmt:
	go fmt ./...

# Run Go vet
.PHONY: vet
vet:
	go vet ./...

# Run tests
.PHONY: test
test:
	go test ./...

# Install dependencies
.PHONY: deps
deps:
	go mod download
	go mod tidy

# Check if BCC tools are installed
.PHONY: check-deps
check-deps:
	@echo "Checking for required dependencies..."
	@which profile-bpfcc > /dev/null || (echo "ERROR: profile-bpfcc not found. Install with: sudo apt-get install bpfcc-tools" && exit 1)
	@echo "✓ profile-bpfcc found"
	@echo "✓ All dependencies satisfied"

# Development target - format, vet, and build
.PHONY: dev
dev: fmt vet build

# Help target
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build      - Build the bcc-exporter binary"
	@echo "  run        - Build and run the application (requires sudo)"
	@echo "  run-port   - Build and run on custom port (usage: make run-port PORT=9090)"
	@echo "  run-auth   - Build and run with authentication (usage: make run-auth PASSWORD=secret)"
	@echo "  clean      - Remove build artifacts"
	@echo "  fmt        - Format Go code"
	@echo "  vet        - Run go vet"
	@echo "  test       - Run tests"
	@echo "  deps       - Download and tidy dependencies"
	@echo "  check-deps - Check if required system dependencies are installed"
	@echo "  dev        - Format, vet, and build (development workflow)"
	@echo "  help       - Show this help message"
