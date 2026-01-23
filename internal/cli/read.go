package cli

import (
	"fmt"
	"os"

	"github.com/fuabioo/xlq/internal/output"
	"github.com/fuabioo/xlq/internal/xlsx"
	"github.com/spf13/cobra"
)

var readCmd = &cobra.Command{
	Use:   "read <file.xlsx> [sheet] [range]",
	Short: "Read cell range",
	Long:  `Read cells from a range (e.g., A1:C10). If no range specified, reads entire sheet.`,
	Args:  cobra.RangeArgs(1, 3),
	RunE: func(cmd *cobra.Command, args []string) error {
		f, err := xlsx.OpenFile(args[0])
		if err != nil {
			return err
		}
		defer f.Close()

		sheet := ""
		rangeStr := ""

		if len(args) > 1 {
			// Could be sheet name or range
			if xlsx.IsValidRange(args[1]) {
				rangeStr = args[1]
			} else {
				sheet = args[1]
			}
		}
		if len(args) > 2 {
			rangeStr = args[2]
		}

		// Resolve sheet name
		if sheet == "" {
			sheet, err = xlsx.GetDefaultSheet(f)
			if err != nil {
				return err
			}
		}

		var rows []xlsx.Row
		if rangeStr != "" {
			ch, err := xlsx.StreamRange(f, sheet, rangeStr)
			if err != nil {
				return err
			}
			rows, err = xlsx.CollectRows(ch)
			if err != nil {
				return err
			}
		} else {
			ch, err := xlsx.StreamRows(f, sheet, 0, 0)
			if err != nil {
				return err
			}
			rows, err = xlsx.CollectRows(ch)
			if err != nil {
				return err
			}
		}

		data := xlsx.RowsToStringSlice(rows)
		out, err := output.FormatRows(GetFormat(), data)
		if err != nil {
			return err
		}

		fmt.Fprint(os.Stdout, string(out))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(readCmd)
}
