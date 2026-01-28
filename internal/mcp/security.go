package mcp

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// AllowedBasePaths contains directories from which files can be read.
// If empty, defaults to current working directory.
var AllowedBasePaths []string

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

	basePaths := AllowedBasePaths
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
