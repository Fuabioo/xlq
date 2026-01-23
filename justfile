# xlq - jq for Excel

binary := "xlq"
version := "1.0.0"
ldflags := "-s -w -X main.version=" + version

# Default recipe: build
default: build

# Build the binary
build:
    go build -ldflags "{{ldflags}}" -o {{binary}} ./cmd/xlq

# Build a fully static binary (Linux)
build-static:
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "{{ldflags}}" -o {{binary}} ./cmd/xlq

# Run all tests
test:
    go test -v ./...

# Run tests with coverage report
coverage:
    go test -coverprofile=coverage.out ./...
    go tool cover -func=coverage.out
    @echo "\nTo view HTML report: go tool cover -html=coverage.out"

# Run linter
lint:
    golangci-lint run ./...

# Install binary to GOPATH/bin
install: build
    cp {{binary}} $GOPATH/bin/{{binary}}

# Install to /usr/local/bin (requires sudo)
install-local: build
    sudo cp {{binary}} /usr/local/bin/{{binary}}

# Remove build artifacts
clean:
    rm -f {{binary}}
    rm -f coverage.out
    rm -rf dist

# Build for multiple platforms
dist: clean
    mkdir -p dist
    GOOS=linux GOARCH=amd64 go build -ldflags "{{ldflags}}" -o dist/{{binary}}-linux-amd64 ./cmd/xlq
    GOOS=linux GOARCH=arm64 go build -ldflags "{{ldflags}}" -o dist/{{binary}}-linux-arm64 ./cmd/xlq
    GOOS=darwin GOARCH=amd64 go build -ldflags "{{ldflags}}" -o dist/{{binary}}-darwin-amd64 ./cmd/xlq
    GOOS=darwin GOARCH=arm64 go build -ldflags "{{ldflags}}" -o dist/{{binary}}-darwin-arm64 ./cmd/xlq
    GOOS=windows GOARCH=amd64 go build -ldflags "{{ldflags}}" -o dist/{{binary}}-windows-amd64.exe ./cmd/xlq

# Run the binary
run *args:
    go run ./cmd/xlq {{args}}

# Run MCP server mode
mcp:
    go run ./cmd/xlq --mcp

# Show available recipes
help:
    @just --list
