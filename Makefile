.PHONY: build test install clean lint fmt vet run

# Binary name
BINARY=cimon

# Build flags
LDFLAGS=-s -w

# Default target
all: build

# Build the binary
build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/cimon

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Install to GOPATH/bin
install:
	go install ./cmd/cimon

# Clean build artifacts
clean:
	rm -f $(BINARY)
	rm -f coverage.out coverage.html
	rm -rf dist/

# Run linter (requires golangci-lint)
lint:
	golangci-lint run ./...

# Format code
fmt:
	go fmt ./...

# Vet code
vet:
	go vet ./...

# Run the application
run:
	go run ./cmd/cimon

# Tidy dependencies
tidy:
	go mod tidy

# Download dependencies
deps:
	go mod download

# Build for release (requires goreleaser)
release:
	goreleaser release --clean

# Build snapshot (for testing goreleaser config)
snapshot:
	goreleaser build --snapshot --clean
