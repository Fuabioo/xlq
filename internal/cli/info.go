package cli

import (
	"fmt"
	"os"

	"github.com/fuabioo/xlq/internal/output"
	"github.com/fuabioo/xlq/internal/xlsx"
	"github.com/spf13/cobra"
)

var infoCmd = &cobra.Command{
	Use:   "info <file.xlsx> [sheet]",
	Short: "Get sheet metadata",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		f, err := xlsx.OpenFile(args[0])
		if err != nil {
			return err
		}
		defer f.Close()

		sheet := ""
		if len(args) > 1 {
			sheet = args[1]
		} else {
			// Get default sheet
			sheet, err = xlsx.GetDefaultSheet(f)
			if err != nil {
				return err
			}
		}

		info, err := xlsx.GetSheetInfo(f, sheet)
		if err != nil {
			return err
		}

		out, err := output.FormatSingle(GetFormatFromCmd(cmd), info)
		if err != nil {
			return err
		}

		fmt.Fprint(os.Stdout, string(out))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(infoCmd)
}
