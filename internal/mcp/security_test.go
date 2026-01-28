package mcp

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

// TestPathTraversalVulnerability tests whether MCP handlers accept arbitrary file paths
// This test PROVES or DISPROVES the security claim about path traversal vulnerabilities
func TestPathTraversalVulnerability(t *testing.T) {
	// Create a temporary directory structure to simulate the scenario
	tmpDir := t.TempDir()

	// Create a test xlsx file in /tmp (outside the working directory)
	tmpFile := filepath.Join(tmpDir, "sensitive.xlsx")

	// Copy the test xlsx file to temp location
	err := copyTestFile(tmpFile)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Verify the file exists
	if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
		t.Fatalf("Test file not created: %s", tmpFile)
	}

	t.Logf("Created test file at: %s", tmpFile)

	// Create MCP server
	srv := New()
	if srv == nil {
		t.Fatal("Failed to create MCP server")
	}

	// Test cases for path traversal attempts
	tests := []struct {
		name        string
		filePath    string
		shouldAllow bool // true if vulnerability exists, false if blocked
		description string
	}{
		{
			name:        "Absolute path outside working directory",
			filePath:    tmpFile,
			shouldAllow: true, // EXPECTED: Currently allows (vulnerability)
			description: "Test if absolute paths to files outside working directory are allowed",
		},
		{
			name:        "Relative path traversal with ../",
			filePath:    filepath.Join("..", "..", "tmp", filepath.Base(tmpFile)),
			shouldAllow: true, // EXPECTED: Currently allows (vulnerability)
			description: "Test if relative path traversal using ../ is allowed",
		},
		{
			name:        "Path with /tmp/ prefix",
			filePath:    filepath.Join("/tmp", filepath.Base(tmpFile)),
			shouldAllow: false, // File doesn't exist at /tmp, but tests absolute path handling
			description: "Test if absolute /tmp paths are processed (file won't exist but tests validation)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Testing: %s", tt.description)
			t.Logf("File path: %s", tt.filePath)

			// Create a mock MCP request for the 'sheets' tool
			request := createMockRequest("sheets", map[string]any{
				"file": tt.filePath,
			})

			// Call the handler
			result, err := srv.handleSheets(context.Background(), request)

			// Analyze the result
			if err != nil {
				t.Logf("Handler returned error: %v", err)
			}

			if result == nil {
				t.Fatal("Handler returned nil result")
			}

			// Check if the operation succeeded or failed
			accessible := !result.IsError

			t.Logf("File accessible: %v", accessible)
			if result.IsError && len(result.Content) > 0 {
				if textContent, ok := result.Content[0].(mcp.TextContent); ok {
					t.Logf("Error message: %s", textContent.Text)
				}
			}

			// Verify behavior matches expectation
			if accessible && tt.shouldAllow {
				t.Logf("VULNERABILITY CONFIRMED: File outside working directory was accessible via path: %s", tt.filePath)
			} else if accessible && !tt.shouldAllow {
				t.Errorf("UNEXPECTED: File should not be accessible but was: %s", tt.filePath)
			} else if !accessible && tt.shouldAllow {
				t.Logf("VULNERABILITY NOT PRESENT: Access blocked for: %s", tt.filePath)
			} else {
				t.Logf("EXPECTED: Access correctly blocked for: %s", tt.filePath)
			}
		})
	}
}

// TestAllHandlersPathTraversal tests path traversal for all MCP handlers that accept file parameter
func TestAllHandlersPathTraversal(t *testing.T) {
	// Create a temporary file outside working directory
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.xlsx")

	err := copyTestFile(tmpFile)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	srv := New()
	if srv == nil {
		t.Fatal("Failed to create MCP server")
	}

	// Test all handlers that accept file paths
	handlers := []struct {
		name    string
		handler func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)
		params  map[string]any
	}{
		{
			name:    "sheets",
			handler: srv.handleSheets,
			params:  map[string]any{"file": tmpFile},
		},
		{
			name:    "info",
			handler: srv.handleInfo,
			params:  map[string]any{"file": tmpFile, "sheet": "Sheet1"},
		},
		{
			name:    "read",
			handler: srv.handleRead,
			params:  map[string]any{"file": tmpFile, "sheet": "Sheet1"},
		},
		{
			name:    "head",
			handler: srv.handleHead,
			params:  map[string]any{"file": tmpFile, "sheet": "Sheet1", "n": 5},
		},
		{
			name:    "tail",
			handler: srv.handleTail,
			params:  map[string]any{"file": tmpFile, "sheet": "Sheet1", "n": 5},
		},
		{
			name:    "search",
			handler: srv.handleSearch,
			params:  map[string]any{"file": tmpFile, "pattern": "test"},
		},
		{
			name:    "cell",
			handler: srv.handleCell,
			params:  map[string]any{"file": tmpFile, "address": "A1", "sheet": "Sheet1"},
		},
	}

	for _, h := range handlers {
		t.Run(h.name, func(t *testing.T) {
			request := createMockRequest(h.name, h.params)
			result, err := h.handler(context.Background(), request)

			if err != nil {
				t.Logf("Handler error: %v", err)
			}

			if result == nil {
				t.Fatal("Handler returned nil result")
			}

			accessible := !result.IsError

			if accessible {
				t.Errorf("VULNERABILITY: Handler '%s' allows access to file outside working directory: %s", h.name, tmpFile)
			} else {
				t.Logf("Handler '%s' blocked access (expected behavior)", h.name)
			}
		})
	}
}

// TestSymbolicLinkPathTraversal tests if symbolic links can be used for path traversal
func TestSymbolicLinkPathTraversal(t *testing.T) {
	// Create temp directory with test file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "target.xlsx")

	err := copyTestFile(tmpFile)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create a symlink in current directory pointing to the temp file
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	symlinkPath := filepath.Join(cwd, "symlink_test.xlsx")

	// Clean up any existing symlink
	os.Remove(symlinkPath)

	// Create symlink
	err = os.Symlink(tmpFile, symlinkPath)
	if err != nil {
		t.Skipf("Cannot create symlink (may need permissions): %v", err)
		return
	}
	defer os.Remove(symlinkPath)

	t.Logf("Created symlink: %s -> %s", symlinkPath, tmpFile)

	srv := New()
	request := createMockRequest("sheets", map[string]any{
		"file": symlinkPath,
	})

	result, err := srv.handleSheets(context.Background(), request)
	if err != nil {
		t.Logf("Handler error: %v", err)
	}

	if result == nil {
		t.Fatal("Handler returned nil result")
	}

	accessible := !result.IsError

	if accessible {
		t.Logf("VULNERABILITY: Symlink path traversal possible - file accessible via: %s", symlinkPath)
	} else {
		t.Logf("Symlink access blocked")
	}
}

// Helper function to create a mock MCP request
// This creates a CallToolRequest compatible with the handler signature
func createMockRequest(tool string, params map[string]any) mcp.CallToolRequest {
	// The MCP SDK CallToolRequest has embedded params
	// We construct it with the arguments directly
	req := mcp.CallToolRequest{}
	req.Params.Name = tool
	req.Params.Arguments = params
	return req
}

// Helper function to copy the test xlsx file to a target location
func copyTestFile(dst string) error {
	// Use the test file from testdata package
	src := "../../testdata/test.xlsx"

	// Check if source exists
	if _, err := os.Stat(src); os.IsNotExist(err) {
		return fmt.Errorf("source test file not found: %s", src)
	}

	// Open source file
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source: %w", err)
	}
	defer sourceFile.Close()

	// Create destination file
	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination: %w", err)
	}
	defer destFile.Close()

	// Copy content
	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	return nil
}
