package cli

import (
	"fmt"

	"github.com/fuabioo/xlq/internal/output"
	"github.com/fuabioo/xlq/internal/xlsx"
	"github.com/spf13/cobra"
)

var writeCmd = &cobra.Command{
	Use:   "write <file> <cell> <value>",
	Short: "Write a value to a cell",
	Long:  "Write a value to a specific cell in an xlsx file. Use --sheet to specify sheet.",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		file := ResolveFilePath(GetBasepathFromCmd(cmd), args[0])
		cell := args[1]
		value := args[2]

		sheet, err := cmd.Flags().GetString("sheet")
		if err != nil {
			return fmt.Errorf("failed to get sheet flag: %w", err)
		}

		valueType, err := cmd.Flags().GetString("type")
		if err != nil {
			return fmt.Errorf("failed to get type flag: %w", err)
		}

		result, err := xlsx.WriteCell(file, sheet, cell, value, valueType)
		if err != nil {
			return err
		}

		format := GetFormatFromCmd(cmd)
		return output.Print(result, format)
	},
}

func init() {
	writeCmd.Flags().StringP("sheet", "s", "", "Sheet name (default: first sheet)")
	writeCmd.Flags().StringP("type", "t", "auto", "Value type: auto, string, number, bool, formula")
	rootCmd.AddCommand(writeCmd)
}
