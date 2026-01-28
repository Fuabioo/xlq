package mcp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateFilePath(t *testing.T) {
	// Save original AllowedBasePaths
	originalPaths := AllowedBasePaths
	defer func() {
		AllowedBasePaths = originalPaths
	}()

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	// Create a temporary directory outside working directory
	tmpDir := t.TempDir()

	// Create a test file in the temporary directory
	tmpFile := filepath.Join(tmpDir, "test.xlsx")
	err = os.WriteFile(tmpFile, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create a test file in the working directory
	cwdFile := filepath.Join(cwd, "test_in_cwd.xlsx")
	err = os.WriteFile(cwdFile, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file in cwd: %v", err)
	}
	defer os.Remove(cwdFile)

	tests := []struct {
		name        string
		path        string
		basePaths   []string
		shouldError bool
		errorMsg    string
	}{
		{
			name:        "Empty path",
			path:        "",
			basePaths:   nil,
			shouldError: true,
			errorMsg:    "file path cannot be empty",
		},
		{
			name:        "File in working directory",
			path:        cwdFile,
			basePaths:   nil, // Uses cwd by default
			shouldError: false,
		},
		{
			name:        "Relative path in working directory",
			path:        filepath.Base(cwdFile),
			basePaths:   nil,
			shouldError: false,
		},
		{
			name:        "File outside working directory (default)",
			path:        tmpFile,
			basePaths:   nil,
			shouldError: true,
			errorMsg:    "access denied: path outside allowed directories",
		},
		{
			name:        "File outside working directory (explicit cwd)",
			path:        tmpFile,
			basePaths:   []string{cwd},
			shouldError: true,
			errorMsg:    "access denied: path outside allowed directories",
		},
		{
			name:        "File in allowed directory",
			path:        tmpFile,
			basePaths:   []string{tmpDir},
			shouldError: false,
		},
		{
			name:        "File with multiple allowed paths",
			path:        tmpFile,
			basePaths:   []string{cwd, tmpDir},
			shouldError: false,
		},
		{
			name:        "Non-existent file",
			path:        "/nonexistent/file.xlsx",
			basePaths:   nil,
			shouldError: true,
			errorMsg:    "file not found",
		},
		{
			name:        "Path traversal with ../",
			path:        filepath.Join(cwd, "..", "..", "etc", "passwd"),
			basePaths:   nil,
			shouldError: true,
			errorMsg:    "file not found", // Will fail because the file doesn't exist or access denied
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set AllowedBasePaths for this test
			AllowedBasePaths = tt.basePaths

			result, err := ValidateFilePath(tt.path)

			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error but got none, result: %s", result)
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s' but got: %v", tt.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
					return
				}
				if result == "" {
					t.Error("Expected non-empty result path")
				}
			}
		})
	}
}

func TestValidateFilePathSymlinks(t *testing.T) {
	// Save original AllowedBasePaths
	originalPaths := AllowedBasePaths
	defer func() {
		AllowedBasePaths = originalPaths
	}()

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	// Create a temporary directory outside working directory
	tmpDir := t.TempDir()

	// Create a test file in the temporary directory
	tmpFile := filepath.Join(tmpDir, "target.xlsx")
	err = os.WriteFile(tmpFile, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create a symlink in the working directory
	symlinkPath := filepath.Join(cwd, "symlink_test.xlsx")
	os.Remove(symlinkPath) // Clean up any existing symlink

	err = os.Symlink(tmpFile, symlinkPath)
	if err != nil {
		t.Skipf("Cannot create symlink (may need permissions): %v", err)
		return
	}
	defer os.Remove(symlinkPath)

	tests := []struct {
		name        string
		path        string
		basePaths   []string
		shouldError bool
		errorMsg    string
	}{
		{
			name:        "Symlink pointing outside working directory (default)",
			path:        symlinkPath,
			basePaths:   nil, // Uses cwd by default
			shouldError: true,
			errorMsg:    "access denied: path outside allowed directories",
		},
		{
			name:        "Symlink allowed when target is in allowed paths",
			path:        symlinkPath,
			basePaths:   []string{tmpDir},
			shouldError: false,
		},
		{
			name:        "Symlink allowed when both symlink and target are in allowed paths",
			path:        symlinkPath,
			basePaths:   []string{cwd, tmpDir},
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			AllowedBasePaths = tt.basePaths

			result, err := ValidateFilePath(tt.path)

			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error but got none, result: %s", result)
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s' but got: %v", tt.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
					return
				}
				// The result should be the real path (after symlink resolution)
				if result != tmpFile {
					t.Errorf("Expected result to be real path %s but got %s", tmpFile, result)
				}
			}
		})
	}
}
