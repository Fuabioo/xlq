package mcp

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Error types for security validation
var (
	ErrWriteDenied  = errors.New("write operation denied")
	ErrFileTooLarge = errors.New("file exceeds size limit for write operations")
	ErrFileExists   = errors.New("file already exists")
)

// allowedBasePaths contains directories from which files can be accessed.
// If empty, defaults to current working directory.
// Must only be modified via InitAllowedPaths before the server starts.
var allowedBasePaths []string

// GetAllowedBasePaths returns a copy of the current allowed base paths.
func GetAllowedBasePaths() []string {
	out := make([]string, len(allowedBasePaths))
	copy(out, allowedBasePaths)
	return out
}

// InitAllowedPaths sets allowedBasePaths to the current working directory
// plus any additional paths provided. Each path is canonicalized (resolved to
// absolute path with symlinks evaluated) and validated to be an existing
// directory. CWD is always included as the first entry.
// This function replaces any previously configured paths.
func InitAllowedPaths(extraPaths []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("cannot determine working directory: %w", err)
	}

	// Canonicalize CWD
	realCWD, err := filepath.EvalSymlinks(cwd)
	if err != nil {
		return fmt.Errorf("cannot resolve working directory: %w", err)
	}

	seen := map[string]bool{realCWD: true}
	paths := []string{realCWD}

	for _, p := range extraPaths {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}

		// Canonicalize: absolute path + resolve symlinks
		absP, err := filepath.Abs(p)
		if err != nil {
			return fmt.Errorf("cannot resolve path %q: %w", p, err)
		}

		realP, err := filepath.EvalSymlinks(absP)
		if err != nil {
			return fmt.Errorf("allowed path %q does not exist or cannot be resolved: %w", p, err)
		}

		// Must be a directory
		info, err := os.Stat(realP)
		if err != nil {
			return fmt.Errorf("cannot stat allowed path %q: %w", p, err)
		}
		if !info.IsDir() {
			return fmt.Errorf("allowed path %q is not a directory", p)
		}

		// Reject filesystem root
		if filepath.Dir(realP) == realP {
			return fmt.Errorf("allowed path %q resolves to filesystem root, which is too broad", p)
		}

		// Deduplicate
		if seen[realP] {
			continue
		}
		seen[realP] = true
		paths = append(paths, realP)
	}

	allowedBasePaths = paths
	return nil
}

// LoadAllowedPathsFromEnv reads the XLQ_ALLOWED_PATHS environment variable
// and initializes allowedBasePaths. Paths are separated by os.PathListSeparator
// (colon on Unix, semicolon on Windows).
// If the env var is not set or empty, AllowedBasePaths is left unchanged.
func LoadAllowedPathsFromEnv() error {
	envPaths := os.Getenv("XLQ_ALLOWED_PATHS")
	if envPaths == "" {
		return nil
	}

	parts := strings.Split(envPaths, string(os.PathListSeparator))
	var extra []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			extra = append(extra, p)
		}
	}

	// If env var was set but contained only separators/whitespace, treat as unset
	if len(extra) == 0 {
		return nil
	}

	return InitAllowedPaths(extra)
}

// ValidateFilePath ensures the path is safe to access.
func ValidateFilePath(requestedPath string) (string, error) {
	if requestedPath == "" {
		return "", fmt.Errorf("file path cannot be empty")
	}

	// Get absolute path
	absPath, err := filepath.Abs(requestedPath)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}

	// Resolve symlinks to prevent bypass
	realPath, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("file not found: %s", requestedPath)
		}
		return "", fmt.Errorf("cannot resolve path: %w", err)
	}

	// Determine allowed base paths
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("cannot determine working directory: %w", err)
	}

	basePaths := allowedBasePaths
	if len(basePaths) == 0 {
		basePaths = []string{cwd}
	}

	// Check if path is within allowed directories
	for _, base := range basePaths {
		absBase, err := filepath.Abs(base)
		if err != nil {
			continue
		}
		realBase, err := filepath.EvalSymlinks(absBase)
		if err != nil {
			continue
		}
		if strings.HasPrefix(realPath, realBase+string(os.PathSeparator)) || realPath == realBase {
			return realPath, nil
		}
	}

	return "", fmt.Errorf("access denied: path outside allowed directories")
}

// blockedWritePatterns contains file patterns that should never be written to.
var blockedWritePatterns = []string{
	".git/",
	".git",
	"node_modules/",
	".env",
	"*.key",
	"*.pem",
	"*.p12",
	"*.pfx",
	"id_rsa",
	"id_ed25519",
	"*.sqlite",
	"*.db",
}

// isBlockedWritePath checks if a path matches any blocked write pattern.
func isBlockedWritePath(path string) bool {
	cleanPath := filepath.Clean(path)
	base := filepath.Base(cleanPath)

	// Split path into components for exact matching
	pathComponents := strings.Split(cleanPath, string(os.PathSeparator))

	for _, pattern := range blockedWritePatterns {
		// Check if pattern is a directory pattern (ends with /)
		if strings.HasSuffix(pattern, "/") {
			dirPattern := strings.TrimSuffix(pattern, "/")
			// Check if any path component exactly matches the directory name
			for _, component := range pathComponents {
				if component == dirPattern {
					return true
				}
			}
		} else if strings.Contains(pattern, "*") {
			// Handle glob patterns (e.g., *.key)
			matched, err := filepath.Match(pattern, base)
			if err == nil && matched {
				return true
			}
		} else {
			// Exact match on base filename
			if base == pattern {
				return true
			}
		}
	}

	return false
}

// ValidateWritePath validates a path for write operations.
// It performs all read validations plus:
// - Checks parent directory is writable
// - Blocks sensitive file patterns
// - Handles overwrite flag
func ValidateWritePath(path string, allowOverwrite bool) (string, error) {
	if path == "" {
		return "", fmt.Errorf("file path cannot be empty")
	}

	// Check if path matches blocked patterns
	if isBlockedWritePath(path) {
		return "", fmt.Errorf("%w: cannot write to sensitive path %s", ErrWriteDenied, path)
	}

	// Get absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}

	// Check if path matches blocked patterns after resolving to absolute path
	if isBlockedWritePath(absPath) {
		return "", fmt.Errorf("%w: cannot write to sensitive path %s", ErrWriteDenied, absPath)
	}

	// Check if file exists
	_, err = os.Stat(absPath)
	if err == nil {
		// File exists
		if !allowOverwrite {
			return "", fmt.Errorf("%w: %s", ErrFileExists, absPath)
		}
		// If overwrite is allowed, resolve symlinks
		realPath, err := filepath.EvalSymlinks(absPath)
		if err != nil {
			return "", fmt.Errorf("cannot resolve path: %w", err)
		}
		absPath = realPath
	} else if !os.IsNotExist(err) {
		// Some other error occurred
		return "", fmt.Errorf("cannot stat path: %w", err)
	}
	// If file doesn't exist, that's okay for write operations

	// Get parent directory
	parentDir := filepath.Dir(absPath)

	// Check if parent directory exists
	parentInfo, err := os.Stat(parentDir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("parent directory does not exist: %s", parentDir)
		}
		return "", fmt.Errorf("cannot access parent directory: %w", err)
	}

	// Verify parent is a directory
	if !parentInfo.IsDir() {
		return "", fmt.Errorf("parent path is not a directory: %s", parentDir)
	}

	// Check if parent directory is writable by attempting to create a temp file
	tempFile := filepath.Join(parentDir, ".xlq_write_test_"+filepath.Base(absPath))
	f, err := os.OpenFile(tempFile, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0600)
	if err != nil {
		return "", fmt.Errorf("%w: parent directory not writable: %s", ErrWriteDenied, parentDir)
	}
	f.Close()
	os.Remove(tempFile) // Clean up test file

	// Resolve parent directory symlinks to get real path
	realParent, err := filepath.EvalSymlinks(parentDir)
	if err != nil {
		return "", fmt.Errorf("cannot resolve parent directory: %w", err)
	}

	// Determine allowed base paths
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("cannot determine working directory: %w", err)
	}

	basePaths := allowedBasePaths
	if len(basePaths) == 0 {
		basePaths = []string{cwd}
	}

	// Check if parent path is within allowed directories
	for _, base := range basePaths {
		absBase, err := filepath.Abs(base)
		if err != nil {
			continue
		}
		realBase, err := filepath.EvalSymlinks(absBase)
		if err != nil {
			continue
		}
		if strings.HasPrefix(realParent, realBase+string(os.PathSeparator)) || realParent == realBase {
			return absPath, nil
		}
	}

	return "", fmt.Errorf("%w: path outside allowed directories", ErrWriteDenied)
}

// CheckFileSize validates file size for write operations.
func CheckFileSize(path string, maxSize int64) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // New file, no size check needed
		}
		return fmt.Errorf("cannot stat file: %w", err)
	}

	if info.Size() > maxSize {
		return fmt.Errorf("%w: %d bytes exceeds limit of %d", ErrFileTooLarge, info.Size(), maxSize)
	}

	return nil
}
