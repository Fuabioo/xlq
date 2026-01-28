package output

import (
	"fmt"
	"os"
)

// Print outputs any result in the specified format to stdout.
// This is a convenience function for CLI commands.
func Print(result any, format string) error {
	out, err := FormatSingle(format, result)
	if err != nil {
		return fmt.Errorf("failed to format output: %w", err)
	}

	fmt.Fprint(os.Stdout, string(out))
	return nil
}
