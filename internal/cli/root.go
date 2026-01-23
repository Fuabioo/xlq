package cli

import (
	"github.com/spf13/cobra"
)

var (
	formatFlag string
)

// rootCmd is the base command
var rootCmd = &cobra.Command{
	Use:   "xlq",
	Short: "xlq - jq for Excel",
	Long:  `xlq is a streaming xlsx CLI tool that provides efficient Excel file operations.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&formatFlag, "format", "f", "json", "Output format (json, csv, tsv)")
}

// GetFormat returns the current format flag value
func GetFormat() string {
	return formatFlag
}
