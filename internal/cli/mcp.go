package cli

import (
	"fmt"
	"log"

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

		basepath := GetBasepathFromCmd(cmd)

		// If basepath is set, include it in allowed paths
		if basepath != "" {
			allowedPaths = append(allowedPaths, basepath)
		}

		if len(allowedPaths) > 0 {
			// CLI flag takes precedence over env var
			if err := mcp.InitAllowedPaths(allowedPaths); err != nil {
				return fmt.Errorf("failed to initialize allowed paths: %w", err)
			}
		} else if err := mcp.LoadAllowedPathsFromEnv(); err != nil {
			// Fall back to XLQ_ALLOWED_PATHS environment variable
			return fmt.Errorf("failed to load allowed paths from environment: %w", err)
		}

		// Always ensure paths are explicitly initialized
		// (LoadAllowedPathsFromEnv is a no-op when env var is unset)
		if len(mcp.GetAllowedBasePaths()) == 0 {
			if err := mcp.InitAllowedPaths(nil); err != nil {
				return fmt.Errorf("failed to initialize default allowed paths: %w", err)
			}
		}

		log.Printf("xlq MCP server allowed paths: %v", mcp.GetAllowedBasePaths())

		srv := mcp.New(basepath)
		return srv.Run()
	},
}

func init() {
	rootCmd.AddCommand(mcpCmd)
	mcpCmd.Flags().StringSlice("allowed-paths", nil,
		"Additional directories to allow file access (comma-separated or repeated, e.g. --allowed-paths /tmp,/data)")
}
