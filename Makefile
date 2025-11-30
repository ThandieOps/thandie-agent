.PHONY: fmt run build execute clean

# Binary name
BINARY_NAME=thandie
BUILD_DIR=./bin
CMD_PATH=./cmd/thandie
WORKSPACE=~/Workspace

# Default target
.DEFAULT_GOAL := help

# Format Go code
fmt:
	@echo "# Formatting Go code..."
	go fmt ./...

# Run the application
run:
	@echo "# Running application..."
	go run $(CMD_PATH) scan --workspace $(WORKSPACE)

# Build the executable
build: fmt
	@echo "# Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_PATH)
	@echo "# Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# Execute the built binary
execute: build
	@echo "# Executing $(BINARY_NAME)..."
	$(BUILD_DIR)/$(BINARY_NAME) scan --workspace $(WORKSPACE)

# Clean build artifacts
clean:
	@echo "# Cleaning build artifacts..."
	rm -rvf $(BUILD_DIR)
	@echo "# Clean complete"

# Help target
help:
	@echo "Available targets:"
	@echo "  make fmt      - Format Go code"
	@echo "  make run      - Run the application with go run"
	@echo "  make build    - Build the executable"
	@echo "  make execute  - Build and execute the binary"
	@echo "  make clean    - Remove build artifacts"
