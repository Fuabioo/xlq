# xlq - jq for Excel

# Build all platforms (cross-compile with goreleaser)
build:
    goreleaser build --snapshot --clean

# Run all tests
test:
    go test -v ./...

# Run tests with coverage
coverage:
    go test -coverprofile=coverage.out ./...
    go tool cover -func=coverage.out

# Run linter
lint:
    golangci-lint run ./...

# Remove build artifacts
clean:
    rm -rf dist coverage.out

# Run directly (development)
run *args:
    go run ./cmd/xlq {{args}}

# Run MCP server mode
mcp:
    go run ./cmd/xlq --mcp
