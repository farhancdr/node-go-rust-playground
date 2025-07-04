# Makefile to manage Nodjes, Go, and Rust playgrounds

# Variables
NODEJS_DIR = node
GO_DIR = go
RUST_DIR = rust

# Default target
.PHONY: all
all: setup run

# Setup all playgrounds
.PHONY: setup
setup: setup-node setup-go setup-rust

# Run all playgrounds
.PHONY: run
run: run-node run-go run-rust

# Nodejs targets
.PHONY: setup-node
setup-node:
	@echo "Setting up Nodejs playground..."
	@cd $(NODEJS_DIR) && bun install

.PHONY: run-node
run-node:
	@echo "Running Nodejs playground..."
	@cd $(NODEJS_DIR) && bun start

# Go targets
.PHONY: setup-go
setup-go:
	@echo "Setting up Go playground..."
	@cd $(GO_DIR) && go mod tidy

.PHONY: run-go
run-go:
	@echo "Running Go playground..."
	@cd $(GO_DIR) && go run main.go

# Rust targets
.PHONY: setup-rust
setup-rust:
	@echo "Setting up Rust playground..."
	@cd $(RUST_DIR) && cargo build

.PHONY: run-rust
run-rust:
	@echo "Running Rust playground..."
	@cd $(RUST_DIR) && cargo run

# Clean generated files
.PHONY: clean
clean:
	@echo "Cleaning Nodejs playground..."
	@cd $(NODEJS_DIR) && rm -rf node_modules bun.lock
	@echo "Cleaning Go playground..."
	@cd $(GO_DIR) && rm -rf go.sum
	@echo "Cleaning Rust playground..."
	@cd $(RUST_DIR) && cargo clean

# Help
.PHONY: help
help:
	@echo "Available commands:"
	@echo "  make setup          - Set up all playgrounds"
	@echo "  make run            - Run all playgrounds"
	@echo "  make setup-node 		 - Set up Nodejs playground"
	@echo "  make run-node 			 - Run Nodejs playground"
	@echo "  make setup-go       - Set up Go playground"
	@echo "  make run-go       	 - Run Go playground"
	@echo "  make setup-rust     - Set up Rust playground"
	@echo "  make run-rust     	 - Run Rust playground"
	@echo "  make clean        	 - Clean generated files"
	@echo "  make help         	 - Show this help message"