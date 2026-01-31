package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// ResolveFilePath resolves a file path relative to a basepath.
// If basepath is empty or file is absolute, file is returned unchanged.
// Otherwise, filepath.Join(basepath, file) is returned after verifying
// the resolved path does not escape the basepath via path traversal.
func ResolveFilePath(basepath, file string) (string, error) {
	if basepath == "" {
		return file, nil
	}
	if filepath.IsAbs(file) {
		return file, nil
	}

	resolved := filepath.Join(basepath, file)

	// Verify resolved path stays within basepath
	cleanBase := filepath.Clean(basepath)
	cleanResolved := filepath.Clean(resolved)

	rel, err := filepath.Rel(cleanBase, cleanResolved)
	if err != nil {
		return "", fmt.Errorf("failed to check path containment: %w", err)
	}
	if strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || rel == ".." {
		return "", fmt.Errorf("path traversal denied: %q escapes basepath %q", file, basepath)
	}

	return resolved, nil
}

// GetBasepathFromCmd returns the basepath from the command flag,
// falling back to the XLQ_BASEPATH environment variable.
func GetBasepathFromCmd(cmd *cobra.Command) string {
	basepath, err := cmd.Flags().GetString("basepath")
	if err != nil {
		// Flag not registered or other error, fall back to env
		basepath = ""
	}
	if basepath == "" {
		basepath = os.Getenv("XLQ_BASEPATH")
	}
	return basepath
}
