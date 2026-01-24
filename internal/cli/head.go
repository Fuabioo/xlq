package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/fuabioo/xlq/internal/output"
	"github.com/fuabioo/xlq/internal/xlsx"
	"github.com/spf13/cobra"
)

var (
	headN int
)

var headCmd = &cobra.Command{
	Use:   "head <file.xlsx> [sheet]",
	Short: "Show first N rows",
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
			sheet, err = xlsx.GetDefaultSheet(f)
			if err != nil {
				return err
			}
		}

		ctx := context.Background()

		ch, err := xlsx.StreamHead(ctx, f, sheet, headN)
		if err != nil {
			return err
		}

		rows, err := xlsx.CollectRows(ch)
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
	headCmd.Flags().IntVarP(&headN, "number", "n", 10, "Number of rows to show")
	rootCmd.AddCommand(headCmd)
}
