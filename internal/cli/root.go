package cli

import (
	"github.com/fuabioo/xlq/internal/mcp"
	"github.com/spf13/cobra"
)

var (
	formatFlag string
	mcpFlag    bool
)

// rootCmd is the base command
var rootCmd = &cobra.Command{
	Use:   "xlq",
	Short: "xlq - jq for Excel",
	Long:  `xlq is a streaming xlsx CLI tool that provides efficient Excel file operations.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if mcpFlag {
			srv := mcp.New()
			return srv.Run()
		}
		return cmd.Help()
	},
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&formatFlag, "format", "f", "json", "Output format (json, csv, tsv)")
	rootCmd.Flags().BoolVar(&mcpFlag, "mcp", false, "Run as MCP server")
}

// GetFormat returns the current format flag value
func GetFormat() string {
	return formatFlag
}
