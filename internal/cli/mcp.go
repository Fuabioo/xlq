package cli

import (
	"github.com/fuabioo/xlq/internal/mcp"
	"github.com/spf13/cobra"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Run as MCP server (stdio)",
	Long:  `Run xlq as a Model Context Protocol server using stdio transport.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		srv := mcp.New()
		return srv.Run()
	},
}

func init() {
	rootCmd.AddCommand(mcpCmd)
}
