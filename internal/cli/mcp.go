package cli

import (
	"fmt"

	"github.com/fuabioo/xlq/internal/mcp"
	"github.com/spf13/cobra"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Run as MCP server (stdio)",
	Long:  `Run xlq as a Model Context Protocol server using stdio transport.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		allowedPaths, err := cmd.Flags().GetStringSlice("allowed-paths")
		if err != nil {
			return fmt.Errorf("failed to get allowed-paths flag: %w", err)
		}

		if len(allowedPaths) > 0 {
			// CLI flag takes precedence over env var
			if err := mcp.InitAllowedPaths(allowedPaths); err != nil {
				return fmt.Errorf("failed to initialize allowed paths: %w", err)
			}
		} else {
			// Fall back to XLQ_ALLOWED_PATHS environment variable
			if err := mcp.LoadAllowedPathsFromEnv(); err != nil {
				return fmt.Errorf("failed to load allowed paths from environment: %w", err)
			}
		}

		srv := mcp.New()
		return srv.Run()
	},
}

func init() {
	rootCmd.AddCommand(mcpCmd)
	mcpCmd.Flags().StringSlice("allowed-paths", nil,
		"Additional directories to allow file access (comma-separated, e.g. --allowed-paths /tmp,/data)")
}
