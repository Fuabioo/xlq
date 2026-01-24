# xlq

**jq for Excel** - A streaming xlsx CLI tool and MCP server.

xlq provides efficient, memory-bounded operations on Excel files. It can handle million-row spreadsheets without loading them entirely into memory.

## Features

- **Streaming Architecture**: Process files of any size with <100MB memory
- **Dual Mode**: CLI for humans, MCP server for AI agents
- **Multiple Formats**: JSON (default), CSV, TSV output
- **Unix Philosophy**: Simple, composable commands

## Installation

### Homebrew

```bash
brew tap Fuabioo/tap
brew install xlq
```

### Built from Source

```bash
git clone https://github.com/fuabioo/xlq.git
cd xlq
just build
just install  # Installs to $GOPATH/bin
```

### Pre-built Binaries

```bash
# Linux amd64
curl -L https://github.com/fuabioo/xlq/releases/latest/download/xlq-linux-amd64 -o xlq
chmod +x xlq

# macOS arm64 (Apple Silicon)
curl -L https://github.com/fuabioo/xlq/releases/latest/download/xlq-darwin-arm64 -o xlq
chmod +x xlq
```

## CLI Usage

```bash
# List all sheets in workbook
xlq sheets data.xlsx

# Get sheet metadata (rows, columns, headers)
xlq info data.xlsx
xlq info data.xlsx "Sheet Name"

# Read first/last N rows
xlq head data.xlsx -n 20
xlq tail data.xlsx -n 20

# Read specific range
xlq read data.xlsx A1:D100
xlq read data.xlsx Sheet2 B5:E50

# Get single cell
xlq cell data.xlsx A1
xlq cell data.xlsx Sheet2 C5

# Search for pattern
xlq search data.xlsx "error"
xlq search data.xlsx -i "ERROR"        # case-insensitive
xlq search data.xlsx -r "ERR-[0-9]+"   # regex
xlq search data.xlsx -s Sheet1 "value" # search single sheet
```

### Output Formats

```bash
# Default: JSON (compact, token-efficient)
xlq head data.xlsx -n 5

# CSV format
xlq head data.xlsx -n 5 --format csv

# TSV format
xlq head data.xlsx -n 5 --format tsv
```

## MCP Server Mode

xlq can run as an MCP (Model Context Protocol) server for AI agent integration:

```bash
xlq mcp
```

### Claude Desktop Configuration

```bash
claude mcp add --scope user --transport stdio excel xlq mcp
```

```bash
claude mcp remove excel
```


### Available MCP Tools

| Tool | Description |
|------|-------------|
| `sheets` | List all sheets in workbook |
| `info` | Get sheet metadata |
| `read` | Read cell range |
| `head` | Get first N rows |
| `tail` | Get last N rows |
| `search` | Search for pattern |
| `cell` | Get single cell value |

## Examples

### Pipe to jq for processing

```bash
# Get all product names
xlq read products.xlsx A:A | jq -r '.[][]'

# Count rows
xlq info data.xlsx | jq '.rows'

# Filter search results
xlq search data.xlsx "error" | jq '.results[] | select(.row > 100)'
```

### Export to CSV

```bash
xlq read data.xlsx --format csv > export.csv
```

### Quick data inspection

```bash
# What sheets exist?
xlq sheets report.xlsx

# What's in the first sheet?
xlq head report.xlsx -n 5

# What does row 1000 look like?
xlq read report.xlsx A1000:Z1000
```

## Performance

xlq uses a streaming architecture that never loads entire files into memory:

| File Size | Rows | Memory Usage | Time |
|-----------|------|--------------|------|
| 10 MB | 50K | ~30 MB | <1s |
| 100 MB | 500K | ~50 MB | ~3s |
| 1 GB | 5M | ~80 MB | ~30s |

Memory usage stays constant regardless of file size.

## Building

Requires [just](https://github.com/casey/just) command runner.

```bash
just build          # Build binary
just test           # Run tests
just coverage       # Run tests with coverage
just lint           # Run linter
just dist           # Build for all platforms
just clean          # Remove artifacts
just help           # List all recipes
```

## License

MIT License - see LICENSE file for details.
