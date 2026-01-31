package cli

import (
	"context"
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
		filePath := ResolveFilePath(GetBasepathFromCmd(cmd), args[0])
		f, err := xlsx.OpenFile(filePath)
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

		ctx := context.Background()

		var rows []xlsx.Row
		var truncated bool

		if rangeStr != "" {
			// Specific range - no limit needed
			ch, err := xlsx.StreamRange(ctx, f, sheet, rangeStr)
			if err != nil {
				return err
			}
			rows, err = xlsx.CollectRows(ch)
			if err != nil {
				return err
			}
		} else {
			// Full sheet - apply limit
			limit, err := cmd.Flags().GetInt("limit")
			if err != nil {
				return err
			}

			ch, err := xlsx.StreamRows(ctx, f, sheet, 0, 0)
			if err != nil {
				return err
			}

			if limit <= 0 {
				rows, err = xlsx.CollectRows(ch)
				if err != nil {
					return err
				}
			} else {
				var total int
				rows, total, truncated, err = xlsx.CollectRowsWithLimit(ch, limit)
				_ = total
				if err != nil {
					return err
				}
			}
		}

		if truncated {
			fmt.Fprintf(os.Stderr, "Warning: Output truncated at limit (use --limit to adjust)\n")
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
	readCmd.Flags().IntP("limit", "l", 1000, "Maximum rows when no range specified (0 = unlimited)")
	rootCmd.AddCommand(readCmd)
}
