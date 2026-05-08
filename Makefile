.PHONY: build test lint cover clean

# Build the CLI binary
build:
	go build -o mmgo ./cmd/mmgo

# Run all tests with race detection and coverage
test:
	go test ./... -race -cover

# Run golangci-lint
lint:
	golangci-lint run ./...

# Generate HTML coverage report
cover:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Remove build artifacts
clean:
	rm -f mmgo coverage.out coverage.html
