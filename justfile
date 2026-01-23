# xlq - jq for Excel

# Detect current platform
os := `uname -s | tr '[:upper:]' '[:lower:]'`
arch := `uname -m | sed 's/x86_64/amd64/' | sed 's/aarch64/arm64/'`
gobin := `if [ -n "$(go env GOBIN)" ]; then go env GOBIN; else echo "$(go env GOPATH)/bin"; fi`

# Build all platforms (cross-compile with goreleaser)
build:
    goreleaser build --snapshot --clean

# Install to GOBIN (like go install)
install: build
    #!/usr/bin/env sh
    set -e
    if [ "{{os}}" = "darwin" ] && [ "{{arch}}" = "arm64" ]; then
        src="dist/xlq_darwin_arm64_v8.0/xlq"
    elif [ "{{os}}" = "darwin" ] && [ "{{arch}}" = "amd64" ]; then
        src="dist/xlq_darwin_amd64_v1/xlq"
    elif [ "{{os}}" = "linux" ] && [ "{{arch}}" = "arm64" ]; then
        src="dist/xlq_linux_arm64_v8.0/xlq"
    else
        src="dist/xlq_linux_amd64_v1/xlq"
    fi
    mkdir -p "{{gobin}}"
    cp "$src" "{{gobin}}/xlq"
    echo "Installed xlq to {{gobin}}/xlq"

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
