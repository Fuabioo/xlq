package cli

import (
	"fmt"
	"os"

	"github.com/fuabioo/xlq/internal/output"
	"github.com/fuabioo/xlq/internal/xlsx"
	"github.com/spf13/cobra"
)

var tailCmd = &cobra.Command{
	Use:   "tail <file.xlsx> [sheet]",
	Short: "Show last N rows",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		n, _ := cmd.Flags().GetInt("number")

		filePath := ResolveFilePath(GetBasepathFromCmd(cmd), args[0])
		f, err := xlsx.OpenFile(filePath)
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

		rows, err := xlsx.StreamTail(f, sheet, n)
		if err != nil {
			return err
		}

		data := xlsx.RowsToStringSlice(rows)
		out, err := output.FormatRows(GetFormatFromCmd(cmd), data)
		if err != nil {
			return err
		}

		fmt.Fprint(os.Stdout, string(out))
		return nil
	},
}

func init() {
	tailCmd.Flags().IntP("number", "n", 10, "Number of rows to show")
	rootCmd.AddCommand(tailCmd)
}
