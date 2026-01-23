package cli

import (
	"fmt"
	"os"

	"github.com/fuabioo/xlq/internal/output"
	"github.com/fuabioo/xlq/internal/xlsx"
	"github.com/spf13/cobra"
)

var (
	tailN int
)

var tailCmd = &cobra.Command{
	Use:   "tail <file.xlsx> [sheet]",
	Short: "Show last N rows",
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
			var err error
			sheet, err = xlsx.GetDefaultSheet(f)
			if err != nil {
				return err
			}
		}

		rows, err := xlsx.StreamTail(f, sheet, tailN)
		if err != nil {
			return err
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
	tailCmd.Flags().IntVarP(&tailN, "number", "n", 10, "Number of rows to show")
	rootCmd.AddCommand(tailCmd)
}
