package cli

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// ResolveFilePath resolves a file path relative to a basepath.
// If basepath is empty or file is absolute, file is returned unchanged.
// Otherwise, filepath.Join(basepath, file) is returned.
func ResolveFilePath(basepath, file string) string {
	if basepath == "" {
		return file
	}
	if filepath.IsAbs(file) {
		return file
	}
	return filepath.Join(basepath, file)
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
