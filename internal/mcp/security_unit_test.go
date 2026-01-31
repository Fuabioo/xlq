package mcp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateFilePath(t *testing.T) {
	// Save original allowedBasePaths
	allowedPathsMu.RLock()
	originalPaths := make([]string, len(allowedBasePaths))
	copy(originalPaths, allowedBasePaths)
	allowedPathsMu.RUnlock()
	defer func() {
		allowedPathsMu.Lock()
		allowedBasePaths = originalPaths
		allowedPathsMu.Unlock()
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
			// Set allowedBasePaths for this test
			allowedPathsMu.Lock()
			allowedBasePaths = tt.basePaths
			allowedPathsMu.Unlock()

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

func TestInitAllowedPaths(t *testing.T) {
	allowedPathsMu.RLock()
	originalPaths := make([]string, len(allowedBasePaths))
	copy(originalPaths, allowedBasePaths)
	allowedPathsMu.RUnlock()
	defer func() {
		allowedPathsMu.Lock()
		allowedBasePaths = originalPaths
		allowedPathsMu.Unlock()
	}()

	// Resolve CWD canonically (same as InitAllowedPaths does)
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	realCWD, err := filepath.EvalSymlinks(cwd)
	if err != nil {
		t.Fatalf("Failed to resolve working directory: %v", err)
	}

	// Create temp dirs for testing (these are real, existing directories)
	tmpDir1 := t.TempDir()
	tmpDir2 := t.TempDir()

	// Resolve them canonically for comparison
	realDir1, _ := filepath.EvalSymlinks(tmpDir1)

	t.Run("No extra paths", func(t *testing.T) {
		allowedPathsMu.Lock()
		allowedBasePaths = nil
		allowedPathsMu.Unlock()
		err := InitAllowedPaths(nil)
		if err != nil {
			t.Fatalf("InitAllowedPaths returned error: %v", err)
		}
		if len(allowedBasePaths) != 1 {
			t.Errorf("expected 1 path, got %d: %v", len(allowedBasePaths), allowedBasePaths)
		}
		if allowedBasePaths[0] != realCWD {
			t.Errorf("expected CWD %q, got %q", realCWD, allowedBasePaths[0])
		}
	})

	t.Run("One extra path", func(t *testing.T) {
		allowedPathsMu.Lock()
		allowedBasePaths = nil
		allowedPathsMu.Unlock()
		err := InitAllowedPaths([]string{tmpDir1})
		if err != nil {
			t.Fatalf("InitAllowedPaths returned error: %v", err)
		}
		if len(allowedBasePaths) != 2 {
			t.Errorf("expected 2 paths, got %d: %v", len(allowedBasePaths), allowedBasePaths)
		}
		if allowedBasePaths[0] != realCWD {
			t.Errorf("expected first path CWD %q, got %q", realCWD, allowedBasePaths[0])
		}
		if allowedBasePaths[1] != realDir1 {
			t.Errorf("expected second path %q, got %q", realDir1, allowedBasePaths[1])
		}
	})

	t.Run("Multiple extra paths", func(t *testing.T) {
		allowedPathsMu.Lock()
		allowedBasePaths = nil
		allowedPathsMu.Unlock()
		err := InitAllowedPaths([]string{tmpDir1, tmpDir2})
		if err != nil {
			t.Fatalf("InitAllowedPaths returned error: %v", err)
		}
		if len(allowedBasePaths) != 3 {
			t.Errorf("expected 3 paths, got %d: %v", len(allowedBasePaths), allowedBasePaths)
		}
	})

	t.Run("Empty strings filtered out", func(t *testing.T) {
		allowedPathsMu.Lock()
		allowedBasePaths = nil
		allowedPathsMu.Unlock()
		err := InitAllowedPaths([]string{"", tmpDir1, "  ", tmpDir2})
		if err != nil {
			t.Fatalf("InitAllowedPaths returned error: %v", err)
		}
		if len(allowedBasePaths) != 3 {
			t.Errorf("expected 3 paths, got %d: %v", len(allowedBasePaths), allowedBasePaths)
		}
	})

	t.Run("Duplicate paths deduplicated", func(t *testing.T) {
		allowedPathsMu.Lock()
		allowedBasePaths = nil
		allowedPathsMu.Unlock()
		err := InitAllowedPaths([]string{tmpDir1, tmpDir1, tmpDir1})
		if err != nil {
			t.Fatalf("InitAllowedPaths returned error: %v", err)
		}
		if len(allowedBasePaths) != 2 {
			t.Errorf("expected 2 paths (CWD + 1 unique), got %d: %v", len(allowedBasePaths), allowedBasePaths)
		}
	})

	t.Run("Filesystem root rejected", func(t *testing.T) {
		allowedPathsMu.Lock()
		allowedBasePaths = nil
		allowedPathsMu.Unlock()
		err := InitAllowedPaths([]string{"/"})
		if err == nil {
			t.Error("Expected error for filesystem root, got none")
		}
		if err != nil && !strings.Contains(err.Error(), "filesystem root") {
			t.Errorf("Expected 'filesystem root' error, got: %v", err)
		}
	})

	t.Run("Non-existent path rejected", func(t *testing.T) {
		allowedPathsMu.Lock()
		allowedBasePaths = nil
		allowedPathsMu.Unlock()
		err := InitAllowedPaths([]string{"/nonexistent/path/that/does/not/exist"})
		if err == nil {
			t.Error("Expected error for non-existent path, got none")
		}
		if err != nil && !strings.Contains(err.Error(), "does not exist or cannot be resolved") {
			t.Errorf("Expected 'does not exist' error, got: %v", err)
		}
	})

	t.Run("File path rejected (not a directory)", func(t *testing.T) {
		tmpFile := filepath.Join(tmpDir1, "notadir.txt")
		err := os.WriteFile(tmpFile, []byte("test"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
		allowedPathsMu.Lock()
		allowedBasePaths = nil
		allowedPathsMu.Unlock()
		err = InitAllowedPaths([]string{tmpFile})
		if err == nil {
			t.Error("Expected error for file path, got none")
		}
		if err != nil && !strings.Contains(err.Error(), "not a directory") {
			t.Errorf("Expected 'not a directory' error, got: %v", err)
		}
	})

	t.Run("Paths are canonicalized", func(t *testing.T) {
		allowedPathsMu.Lock()
		allowedBasePaths = nil
		allowedPathsMu.Unlock()
		// Use a relative-ish path with trailing components
		err := InitAllowedPaths([]string{tmpDir1 + "/"})
		if err != nil {
			t.Fatalf("InitAllowedPaths returned error: %v", err)
		}
		if len(allowedBasePaths) < 2 {
			t.Fatalf("expected at least 2 paths, got %d", len(allowedBasePaths))
		}
		// The stored path should be the canonical form
		if allowedBasePaths[1] != realDir1 {
			t.Errorf("expected canonical path %q, got %q", realDir1, allowedBasePaths[1])
		}
	})
}

func TestLoadAllowedPathsFromEnv(t *testing.T) {
	allowedPathsMu.RLock()
	originalPaths := make([]string, len(allowedBasePaths))
	copy(originalPaths, allowedBasePaths)
	allowedPathsMu.RUnlock()
	defer func() {
		allowedPathsMu.Lock()
		allowedBasePaths = originalPaths
		allowedPathsMu.Unlock()
	}()

	// Resolve CWD canonically
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	realCWD, err := filepath.EvalSymlinks(cwd)
	if err != nil {
		t.Fatalf("Failed to resolve CWD: %v", err)
	}

	// Create temp dirs for env var tests
	tmpDir1 := t.TempDir()
	tmpDir2 := t.TempDir()

	t.Run("Empty env var leaves paths unchanged", func(t *testing.T) {
		allowedPathsMu.Lock()
		allowedBasePaths = nil
		allowedPathsMu.Unlock()
		t.Setenv("XLQ_ALLOWED_PATHS", "")
		err := LoadAllowedPathsFromEnv()
		if err != nil {
			t.Fatalf("LoadAllowedPathsFromEnv returned error: %v", err)
		}
		if len(allowedBasePaths) != 0 {
			t.Errorf("expected 0 paths (unchanged), got %d: %v", len(allowedBasePaths), allowedBasePaths)
		}
	})

	t.Run("Single path", func(t *testing.T) {
		allowedPathsMu.Lock()
		allowedBasePaths = nil
		allowedPathsMu.Unlock()
		t.Setenv("XLQ_ALLOWED_PATHS", tmpDir1)
		err := LoadAllowedPathsFromEnv()
		if err != nil {
			t.Fatalf("LoadAllowedPathsFromEnv returned error: %v", err)
		}
		if len(allowedBasePaths) != 2 {
			t.Errorf("expected 2 paths, got %d: %v", len(allowedBasePaths), allowedBasePaths)
		}
		if len(allowedBasePaths) > 0 && allowedBasePaths[0] != realCWD {
			t.Errorf("expected first path CWD %q, got %q", realCWD, allowedBasePaths[0])
		}
	})

	t.Run("Multiple paths separated", func(t *testing.T) {
		allowedPathsMu.Lock()
		allowedBasePaths = nil
		allowedPathsMu.Unlock()
		t.Setenv("XLQ_ALLOWED_PATHS", tmpDir1+string(os.PathListSeparator)+tmpDir2)
		err := LoadAllowedPathsFromEnv()
		if err != nil {
			t.Fatalf("LoadAllowedPathsFromEnv returned error: %v", err)
		}
		if len(allowedBasePaths) != 3 {
			t.Errorf("expected 3 paths, got %d: %v", len(allowedBasePaths), allowedBasePaths)
		}
	})

	t.Run("Trailing separator ignored", func(t *testing.T) {
		allowedPathsMu.Lock()
		allowedBasePaths = nil
		allowedPathsMu.Unlock()
		t.Setenv("XLQ_ALLOWED_PATHS", tmpDir1+string(os.PathListSeparator))
		err := LoadAllowedPathsFromEnv()
		if err != nil {
			t.Fatalf("LoadAllowedPathsFromEnv returned error: %v", err)
		}
		if len(allowedBasePaths) != 2 {
			t.Errorf("expected 2 paths, got %d: %v", len(allowedBasePaths), allowedBasePaths)
		}
	})

	t.Run("Only separators treated as unset", func(t *testing.T) {
		allowedPathsMu.Lock()
		allowedBasePaths = nil
		allowedPathsMu.Unlock()
		t.Setenv("XLQ_ALLOWED_PATHS", ":::")
		err := LoadAllowedPathsFromEnv()
		if err != nil {
			t.Fatalf("LoadAllowedPathsFromEnv returned error: %v", err)
		}
		if len(allowedBasePaths) != 0 {
			t.Errorf("expected 0 paths (treated as unset), got %d: %v", len(allowedBasePaths), allowedBasePaths)
		}
	})
}

func TestGetAllowedBasePaths(t *testing.T) {
	allowedPathsMu.RLock()
	originalPaths := make([]string, len(allowedBasePaths))
	copy(originalPaths, allowedBasePaths)
	allowedPathsMu.RUnlock()
	defer func() {
		allowedPathsMu.Lock()
		allowedBasePaths = originalPaths
		allowedPathsMu.Unlock()
	}()

	tmpDir := t.TempDir()
	err := InitAllowedPaths([]string{tmpDir})
	if err != nil {
		t.Fatalf("InitAllowedPaths error: %v", err)
	}

	got := GetAllowedBasePaths()
	if len(got) != len(allowedBasePaths) {
		t.Errorf("GetAllowedBasePaths returned %d paths, expected %d", len(got), len(allowedBasePaths))
	}

	// Modifying the returned slice should not affect the internal state
	got[0] = "/hacked"
	if allowedBasePaths[0] == "/hacked" {
		t.Error("GetAllowedBasePaths returned a reference, not a copy")
	}
}

func TestInitAllowedPathsWithValidation(t *testing.T) {
	allowedPathsMu.RLock()
	originalPaths := make([]string, len(allowedBasePaths))
	copy(originalPaths, allowedBasePaths)
	allowedPathsMu.RUnlock()
	defer func() {
		allowedPathsMu.Lock()
		allowedBasePaths = originalPaths
		allowedPathsMu.Unlock()
	}()

	// Create a temp dir and file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.xlsx")
	err := os.WriteFile(tmpFile, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Without extra paths, temp file should be denied
	allowedPathsMu.Lock()
	allowedBasePaths = nil
	allowedPathsMu.Unlock()
	_, err = ValidateFilePath(tmpFile)
	if err == nil {
		t.Error("Expected access denied for temp file with default paths")
	}

	// After InitAllowedPaths with tmpDir, it should work
	err = InitAllowedPaths([]string{tmpDir})
	if err != nil {
		t.Fatalf("InitAllowedPaths error: %v", err)
	}

	result, err := ValidateFilePath(tmpFile)
	if err != nil {
		t.Errorf("Expected access allowed after InitAllowedPaths, got error: %v", err)
	}
	if result == "" {
		t.Error("Expected non-empty result path")
	}

	// Files outside both CWD and tmpDir should still be denied
	otherDir := t.TempDir()
	otherFile := filepath.Join(otherDir, "other.xlsx")
	err = os.WriteFile(otherFile, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create other file: %v", err)
	}
	_, err = ValidateFilePath(otherFile)
	if err == nil {
		t.Error("Expected access denied for file outside allowed directories")
	}
}

func TestValidateFilePathSymlinks(t *testing.T) {
	// Save original allowedBasePaths
	allowedPathsMu.RLock()
	originalPaths := make([]string, len(allowedBasePaths))
	copy(originalPaths, allowedBasePaths)
	allowedPathsMu.RUnlock()
	defer func() {
		allowedPathsMu.Lock()
		allowedBasePaths = originalPaths
		allowedPathsMu.Unlock()
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
			allowedPathsMu.Lock()
			allowedBasePaths = tt.basePaths
			allowedPathsMu.Unlock()

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
