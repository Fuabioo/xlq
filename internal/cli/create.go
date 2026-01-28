package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/fuabioo/xlq/internal/output"
	"github.com/fuabioo/xlq/internal/xlsx"
	"github.com/spf13/cobra"
)

var createCmd = &cobra.Command{
	Use:   "create <file>",
	Short: "Create a new Excel file",
	Long:  "Create a new xlsx file with optional headers and initial data.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		file := args[0]

		sheetName, err := cmd.Flags().GetString("sheet")
		if err != nil {
			return fmt.Errorf("failed to get sheet flag: %w", err)
		}

		headersStr, err := cmd.Flags().GetString("headers")
		if err != nil {
			return fmt.Errorf("failed to get headers flag: %w", err)
		}

		overwrite, err := cmd.Flags().GetBool("overwrite")
		if err != nil {
			return fmt.Errorf("failed to get overwrite flag: %w", err)
		}

		dataFile, err := cmd.Flags().GetString("data")
		if err != nil {
			return fmt.Errorf("failed to get data flag: %w", err)
		}

		var headers []string
		if headersStr != "" {
			headers = strings.Split(headersStr, ",")
		}

		var rows [][]any
		if dataFile != "" {
			// Read JSON data from file
			data, err := os.ReadFile(dataFile)
			if err != nil {
				return fmt.Errorf("failed to read data file: %w", err)
			}
			if err := json.Unmarshal(data, &rows); err != nil {
				return fmt.Errorf("failed to parse data file as JSON array: %w", err)
			}
		}

		result, err := xlsx.CreateFile(file, sheetName, headers, rows, overwrite)
		if err != nil {
			return err
		}

		format := GetFormatFromCmd(cmd)
		return output.Print(result, format)
	},
}

func init() {
	createCmd.Flags().StringP("sheet", "s", "Sheet1", "Name for the first sheet")
	createCmd.Flags().StringP("headers", "H", "", "Comma-separated header row")
	createCmd.Flags().BoolP("overwrite", "o", false, "Overwrite existing file")
	createCmd.Flags().StringP("data", "d", "", "JSON file with initial data (array of arrays)")
	rootCmd.AddCommand(createCmd)
}
