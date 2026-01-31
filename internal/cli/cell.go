package cli

import (
	"fmt"
	"os"

	"github.com/fuabioo/xlq/internal/output"
	"github.com/fuabioo/xlq/internal/xlsx"
	"github.com/spf13/cobra"
)

var cellCmd = &cobra.Command{
	Use:   "cell <file.xlsx> [sheet] <address>",
	Short: "Get single cell value",
	Args:  cobra.RangeArgs(2, 3),
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath := ResolveFilePath(GetBasepathFromCmd(cmd), args[0])
		f, err := xlsx.OpenFile(filePath)
		if err != nil {
			return err
		}
		defer f.Close()

		var sheet, address string
		if len(args) == 2 {
			// Only file and address provided, use default sheet
			sheet, err = xlsx.GetDefaultSheet(f)
			if err != nil {
				return err
			}
			address = args[1]
		} else {
			// File, sheet, and address provided
			sheet = args[1]
			address = args[2]
		}

		cell, err := xlsx.GetCell(f, sheet, address)
		if err != nil {
			return err
		}

		out, err := output.FormatSingle(GetFormatFromCmd(cmd), cell)
		if err != nil {
			return err
		}

		fmt.Fprint(os.Stdout, string(out))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(cellCmd)
}
