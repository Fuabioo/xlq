# xlq - Streaming XLSX CLI/MCP Tool

## Project Overview

**xlq** is a streaming Excel (.xlsx) CLI tool and MCP server that provides "jq for Excel" functionality.

**Key Characteristics:**
- Language: Pure Go (no CGO)
- Memory: <100MB for any file size (streaming-only architecture)
- Dual-mode: CLI and MCP server (stdio)
- Output: JSON-first (token-efficient), with CSV/TSV options

## Architecture Principles

1. **Streaming-First**: Never load entire sheets into memory. Use `excelize.Rows()` streaming API exclusively.
2. **Bounded Memory**: Ring buffers for tail operations. Future enhancement: LRU cache for shared strings.
3. **Go Error Handling**: Always return errors, never panic. Wrap errors with context.
4. **No CGO**: Pure Go for maximum portability and simple builds.
5. **Small Files**: Keep files <500 lines, functions <50 lines.

## Code Conventions

### Error Handling

```go
// GOOD - Go-style error handling
func GetSheets(path string) ([]string, error) {
    f, err := excelize.OpenFile(path)
    if err != nil {
        return nil, fmt.Errorf("failed to open file %s: %w", path, err)
    }
    defer f.Close()

    sheets := f.GetSheetList()
    if len(sheets) == 0 {
        return nil, fmt.Errorf("no sheets found in %s", path)
    }

    return sheets, nil
}

// BAD - Ignoring errors
func GetSheets(path string) []string {
    f, _ := excelize.OpenFile(path) // NEVER DO THIS
    return f.GetSheetList()
}
```

### Streaming Pattern

```go
// GOOD - Channel-based streaming
func StreamRows(f *excelize.File, sheet string) (<-chan Row, error) {
    rows, err := f.Rows(sheet)
    if err != nil {
        return nil, fmt.Errorf("failed to get rows: %w", err)
    }

    ch := make(chan Row)
    go func() {
        defer close(ch)
        defer rows.Close()

        for rows.Next() {
            row, err := rows.Columns()
            if err != nil {
                // Handle error (could send on error channel)
                return
            }
            ch <- Row{Cells: row}
        }
    }()

    return ch, nil
}
```

### Testing

- Unit tests for all packages in `internal/*`
- Test files named `*_test.go`
- Use `testdata/` directory for fixture xlsx files
- Memory profiling tests for large files

```go
// Example test
func TestGetSheets(t *testing.T) {
    sheets, err := GetSheets("testdata/small.xlsx")
    if err != nil {
        t.Fatalf("GetSheets failed: %v", err)
    }

    if len(sheets) != 3 {
        t.Errorf("expected 3 sheets, got %d", len(sheets))
    }
}
```

## Project Type

**Backend CLI Tool** - Go-based command-line tool with MCP server capability.

## Build & Test

```bash
# Build
make build

# Test
make test

# Install
make install

# Run
xlq sheets file.xlsx
xlq --mcp  # Run as MCP server
```

## Dependencies

- `github.com/qax-os/excelize/v2` - xlsx streaming library
- `github.com/spf13/cobra` - CLI framework
- `github.com/modelcontextprotocol/go-sdk` - MCP SDK
- `github.com/stretchr/testify` - Testing utilities

## Memory Management

- Streaming API only (never `GetRows()` which loads entire sheet)
- Ring buffers for tail operations (bounded size)
- Close file handles and iterators properly
- Future enhancement: LRU cache for shared strings to optimize repeated cell access

## CLI Design

### Read Operations
```bash
xlq sheets <file.xlsx>                    # List sheets
xlq info <file.xlsx> [sheet]              # Sheet metadata
xlq read <file.xlsx> [sheet] [range]      # Read range
xlq head <file.xlsx> [sheet] [-n 10]      # First N rows
xlq tail <file.xlsx> [sheet] [-n 10]      # Last N rows
xlq search <file.xlsx> <pattern>          # Search cells
xlq cell <file.xlsx> [sheet] <A1>         # Get cell value
```

### Write Operations
```bash
xlq write <file.xlsx> <cell> <value>      # Write cell value
xlq create <file.xlsx>                    # Create new file
xlq append <file.xlsx> <data.json>        # Append rows from JSON
```

### Server Mode
```bash
xlq --mcp                                 # Run as MCP server
```

## Output Formats

- Default: JSON (compact, token-efficient)
- `--format csv`: CSV with proper escaping
- `--format tsv`: Tab-separated values

## MCP Tools

Each CLI command maps to an MCP tool:

**Read Tools:**
- `sheets`, `info`, `read`, `head`, `tail`, `search`, `cell`

**Write Tools:**
- `write_cell`, `append_rows`, `create_file`, `write_range`
- `create_sheet`, `delete_sheet`, `rename_sheet`
- `insert_rows`, `delete_rows`

All tools use JSON schema for input validation.
