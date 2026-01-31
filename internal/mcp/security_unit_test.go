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

func TestInitAllowedPaths(t *testing.T) {
	originalPaths := AllowedBasePaths
	defer func() { AllowedBasePaths = originalPaths }()

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	tests := []struct {
		name      string
		extra     []string
		wantLen   int
		wantFirst string
		wantPaths []string
	}{
		{
			name:      "No extra paths",
			extra:     nil,
			wantLen:   1,
			wantFirst: cwd,
		},
		{
			name:      "One extra path",
			extra:     []string{"/tmp"},
			wantLen:   2,
			wantFirst: cwd,
			wantPaths: []string{cwd, "/tmp"},
		},
		{
			name:      "Multiple extra paths",
			extra:     []string{"/tmp", "/data"},
			wantLen:   3,
			wantFirst: cwd,
			wantPaths: []string{cwd, "/tmp", "/data"},
		},
		{
			name:      "Empty strings filtered out",
			extra:     []string{"", "/tmp", "  ", "/data"},
			wantLen:   3,
			wantFirst: cwd,
			wantPaths: []string{cwd, "/tmp", "/data"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			AllowedBasePaths = nil

			err := InitAllowedPaths(tt.extra)
			if err != nil {
				t.Fatalf("InitAllowedPaths returned error: %v", err)
			}

			if len(AllowedBasePaths) != tt.wantLen {
				t.Errorf("expected %d paths, got %d: %v", tt.wantLen, len(AllowedBasePaths), AllowedBasePaths)
			}

			if len(AllowedBasePaths) > 0 && AllowedBasePaths[0] != tt.wantFirst {
				t.Errorf("expected first path to be CWD %q, got %q", tt.wantFirst, AllowedBasePaths[0])
			}

			if tt.wantPaths != nil {
				for i, want := range tt.wantPaths {
					if i >= len(AllowedBasePaths) {
						t.Errorf("missing expected path at index %d: %q", i, want)
						continue
					}
					if AllowedBasePaths[i] != want {
						t.Errorf("path[%d] = %q, want %q", i, AllowedBasePaths[i], want)
					}
				}
			}
		})
	}
}

func TestLoadAllowedPathsFromEnv(t *testing.T) {
	originalPaths := AllowedBasePaths
	defer func() { AllowedBasePaths = originalPaths }()

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	tests := []struct {
		name    string
		envVal  string
		wantLen int
		wantCWD bool
	}{
		{
			name:    "Empty env var leaves paths unchanged",
			envVal:  "",
			wantLen: 0,
			wantCWD: false,
		},
		{
			name:    "Single path",
			envVal:  "/tmp",
			wantLen: 2,
			wantCWD: true,
		},
		{
			name:    "Multiple paths colon-separated",
			envVal:  "/tmp:/data",
			wantLen: 3,
			wantCWD: true,
		},
		{
			name:    "Trailing separator ignored",
			envVal:  "/tmp:",
			wantLen: 2,
			wantCWD: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			AllowedBasePaths = nil

			// Set env var for this test
			origEnv := os.Getenv("XLQ_ALLOWED_PATHS")
			os.Setenv("XLQ_ALLOWED_PATHS", tt.envVal)
			defer os.Setenv("XLQ_ALLOWED_PATHS", origEnv)

			err := LoadAllowedPathsFromEnv()
			if err != nil {
				t.Fatalf("LoadAllowedPathsFromEnv returned error: %v", err)
			}

			if len(AllowedBasePaths) != tt.wantLen {
				t.Errorf("expected %d paths, got %d: %v", tt.wantLen, len(AllowedBasePaths), AllowedBasePaths)
			}

			if tt.wantCWD && len(AllowedBasePaths) > 0 && AllowedBasePaths[0] != cwd {
				t.Errorf("expected first path to be CWD %q, got %q", cwd, AllowedBasePaths[0])
			}
		})
	}
}

func TestInitAllowedPathsIntegration(t *testing.T) {
	originalPaths := AllowedBasePaths
	defer func() { AllowedBasePaths = originalPaths }()

	// Create a temp dir and file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.xlsx")
	err := os.WriteFile(tmpFile, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Without extra paths, /tmp file should be denied
	AllowedBasePaths = nil
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
