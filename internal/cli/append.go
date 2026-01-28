package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/fuabioo/xlq/internal/output"
	"github.com/fuabioo/xlq/internal/xlsx"
	"github.com/spf13/cobra"
)

var appendCmd = &cobra.Command{
	Use:   "append <file> <data-file>",
	Short: "Append rows to a sheet",
	Long:  "Append rows from a JSON file to the end of a sheet.",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		file := args[0]
		dataFile := args[1]

		sheet, err := cmd.Flags().GetString("sheet")
		if err != nil {
			return fmt.Errorf("failed to get sheet flag: %w", err)
		}

		// Read JSON data
		data, err := os.ReadFile(dataFile)
		if err != nil {
			return fmt.Errorf("failed to read data file: %w", err)
		}

		var rows [][]any
		if err := json.Unmarshal(data, &rows); err != nil {
			return fmt.Errorf("failed to parse data as JSON array: %w", err)
		}

		result, err := xlsx.AppendRows(file, sheet, rows)
		if err != nil {
			return err
		}

		format := GetFormatFromCmd(cmd)
		return output.Print(result, format)
	},
}

func init() {
	appendCmd.Flags().StringP("sheet", "s", "", "Sheet name (default: first sheet)")
	rootCmd.AddCommand(appendCmd)
}
