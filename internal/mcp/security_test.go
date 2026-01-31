package mcp

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
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

// TestIsBlockedWritePath tests the blocked write path pattern matching
func TestIsBlockedWritePath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		blocked bool
	}{
		// .git patterns
		{name: "git directory", path: "/home/user/.git/config", blocked: true},
		{name: "git file in subdir", path: "/home/user/project/.git/HEAD", blocked: true},
		{name: "dot git file", path: "/home/user/.git", blocked: true},
		{name: "git in path", path: "/home/user/.github/workflows/test.yml", blocked: false},

		// node_modules
		{name: "node_modules directory", path: "/project/node_modules/pkg/file.js", blocked: true},
		{name: "node_modules-like but different", path: "/project/node_modules_backup/file.js", blocked: false},

		// Environment files
		{name: "env file", path: "/project/.env", blocked: true},
		{name: "env file in subdir", path: "/project/config/.env", blocked: true},
		{name: "env example not blocked", path: "/project/.env.example", blocked: false},

		// Key files
		{name: "pem key", path: "/home/user/certs/server.pem", blocked: true},
		{name: "private key", path: "/home/user/.ssh/id_rsa", blocked: true},
		{name: "ed25519 key", path: "/home/user/.ssh/id_ed25519", blocked: true},
		{name: "p12 certificate", path: "/certs/cert.p12", blocked: true},
		{name: "pfx certificate", path: "/certs/cert.pfx", blocked: true},
		{name: "key extension", path: "/config/api.key", blocked: true},

		// Database files
		{name: "sqlite database", path: "/data/app.sqlite", blocked: true},
		{name: "db file", path: "/data/database.db", blocked: true},

		// Safe paths
		{name: "xlsx file", path: "/data/report.xlsx", blocked: false},
		{name: "txt file", path: "/docs/readme.txt", blocked: false},
		{name: "json config", path: "/config/settings.json", blocked: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isBlockedWritePath(tt.path)
			if result != tt.blocked {
				t.Errorf("isBlockedWritePath(%q) = %v, want %v", tt.path, result, tt.blocked)
			}
		})
	}
}

// TestValidateWritePath tests write path validation
func TestValidateWritePath(t *testing.T) {
	// Setup test directory structure
	tmpDir := t.TempDir()

	// Create a writable directory
	writeDir := filepath.Join(tmpDir, "writable")
	err := os.Mkdir(writeDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// Create an existing file
	existingFile := filepath.Join(writeDir, "existing.xlsx")
	err = os.WriteFile(existingFile, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create existing file: %v", err)
	}

	// Create a read-only directory (will be cleaned up by t.TempDir())
	readOnlyDir := filepath.Join(tmpDir, "readonly")
	err = os.Mkdir(readOnlyDir, 0555)
	if err != nil {
		t.Fatalf("Failed to create read-only directory: %v", err)
	}

	// Save original allowedBasePaths and restore after test
	origBasePaths := allowedBasePaths
	defer func() { allowedBasePaths = origBasePaths }()

	// Set allowed base paths to our test directory
	allowedBasePaths = []string{tmpDir}

	tests := []struct {
		name           string
		path           string
		allowOverwrite bool
		expectError    bool
		errorContains  string
	}{
		{
			name:           "New file in writable directory",
			path:           filepath.Join(writeDir, "new.xlsx"),
			allowOverwrite: false,
			expectError:    false,
		},
		{
			name:           "Existing file without overwrite",
			path:           existingFile,
			allowOverwrite: false,
			expectError:    true,
			errorContains:  "file already exists",
		},
		{
			name:           "Existing file with overwrite",
			path:           existingFile,
			allowOverwrite: true,
			expectError:    false,
		},
		{
			name:           "Empty path",
			path:           "",
			allowOverwrite: false,
			expectError:    true,
			errorContains:  "cannot be empty",
		},
		{
			name:           "Parent directory doesn't exist",
			path:           filepath.Join(tmpDir, "nonexistent", "file.xlsx"),
			allowOverwrite: false,
			expectError:    true,
			errorContains:  "parent directory does not exist",
		},
		{
			name:           "Path outside allowed directories",
			path:           "/tmp/outside.xlsx",
			allowOverwrite: false,
			expectError:    true,
			errorContains:  "outside allowed directories",
		},
		{
			name:           "Blocked pattern - .env",
			path:           filepath.Join(writeDir, ".env"),
			allowOverwrite: false,
			expectError:    true,
			errorContains:  "sensitive path",
		},
		{
			name:           "Blocked pattern - .key file",
			path:           filepath.Join(writeDir, "secret.key"),
			allowOverwrite: false,
			expectError:    true,
			errorContains:  "sensitive path",
		},
		{
			name:           "Blocked pattern - .pem file",
			path:           filepath.Join(writeDir, "cert.pem"),
			allowOverwrite: false,
			expectError:    true,
			errorContains:  "sensitive path",
		},
		{
			name:           "Blocked pattern - .git directory",
			path:           filepath.Join(tmpDir, ".git", "config"),
			allowOverwrite: false,
			expectError:    true,
			errorContains:  "sensitive path",
		},
		{
			name:           "Read-only parent directory",
			path:           filepath.Join(readOnlyDir, "file.xlsx"),
			allowOverwrite: false,
			expectError:    true,
			errorContains:  "not writable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ValidateWritePath(tt.path, tt.allowOverwrite)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none. Path: %s", result)
				} else if tt.errorContains != "" && !contains(err.Error(), tt.errorContains) {
					t.Errorf("Error message %q does not contain %q", err.Error(), tt.errorContains)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result == "" {
					t.Error("Expected non-empty result path")
				}
			}
		})
	}
}

// TestCheckFileSize tests file size validation
func TestCheckFileSize(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a small file (1KB)
	smallFile := filepath.Join(tmpDir, "small.xlsx")
	smallData := make([]byte, 1024) // 1KB
	err := os.WriteFile(smallFile, smallData, 0644)
	if err != nil {
		t.Fatalf("Failed to create small file: %v", err)
	}

	// Create a large file (100KB)
	largeFile := filepath.Join(tmpDir, "large.xlsx")
	largeData := make([]byte, 100*1024) // 100KB
	err = os.WriteFile(largeFile, largeData, 0644)
	if err != nil {
		t.Fatalf("Failed to create large file: %v", err)
	}

	tests := []struct {
		name        string
		path        string
		maxSize     int64
		expectError bool
	}{
		{
			name:        "Small file under limit",
			path:        smallFile,
			maxSize:     10 * 1024, // 10KB limit
			expectError: false,
		},
		{
			name:        "Large file over limit",
			path:        largeFile,
			maxSize:     50 * 1024, // 50KB limit
			expectError: true,
		},
		{
			name:        "File at exact limit",
			path:        largeFile,
			maxSize:     100 * 1024, // Exactly 100KB
			expectError: false,
		},
		{
			name:        "Non-existent file",
			path:        filepath.Join(tmpDir, "nonexistent.xlsx"),
			maxSize:     10 * 1024,
			expectError: false, // No error for non-existent files
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckFileSize(tt.path, tt.maxSize)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				} else {
					t.Logf("Got expected error: %v", err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

// TestValidateWritePathSymlinks tests symlink handling in write validation
func TestValidateWritePathSymlinks(t *testing.T) {
	tmpDir := t.TempDir()

	// Create allowed directory
	allowedDir := filepath.Join(tmpDir, "allowed")
	err := os.Mkdir(allowedDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create allowed directory: %v", err)
	}

	// Create disallowed directory
	disallowedDir := filepath.Join(tmpDir, "disallowed")
	err = os.Mkdir(disallowedDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create disallowed directory: %v", err)
	}

	// Create a target file in disallowed directory
	targetFile := filepath.Join(disallowedDir, "target.xlsx")
	err = os.WriteFile(targetFile, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create target file: %v", err)
	}

	// Create symlink in allowed directory pointing to disallowed location
	symlinkPath := filepath.Join(allowedDir, "symlink.xlsx")
	err = os.Symlink(targetFile, symlinkPath)
	if err != nil {
		t.Skipf("Cannot create symlink: %v", err)
	}

	// Set allowed base paths
	origBasePaths := allowedBasePaths
	defer func() { allowedBasePaths = origBasePaths }()
	allowedBasePaths = []string{allowedDir}

	// Try to write via symlink - should be blocked because real path is outside allowed
	_, err = ValidateWritePath(symlinkPath, true)
	if err == nil {
		t.Error("Expected symlink to disallowed location to be blocked, but it was allowed")
	} else {
		t.Logf("Symlink correctly blocked: %v", err)
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
