package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/fuabioo/xlq/internal/output"
	"github.com/fuabioo/xlq/internal/xlsx"
	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search <file.xlsx> <pattern>",
	Short: "Search for cells matching pattern",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		ignoreCase, _ := cmd.Flags().GetBool("ignore-case")
		regex, _ := cmd.Flags().GetBool("regex")
		sheet, _ := cmd.Flags().GetString("sheet")
		max, _ := cmd.Flags().GetInt("max")

		f, err := xlsx.OpenFile(args[0])
		if err != nil {
			return err
		}
		defer f.Close()

		opts := xlsx.SearchOptions{
			Sheet:           sheet,
			CaseInsensitive: ignoreCase,
			Regex:           regex,
			MaxResults:      max,
		}

		ctx := context.Background()

		ch, err := xlsx.Search(ctx, f, args[1], opts)
		if err != nil {
			return err
		}

		results, err := xlsx.CollectSearchResults(ch)
		if err != nil {
			return err
		}

		out, err := output.FormatSingle(GetFormatFromCmd(cmd), results)
		if err != nil {
			return err
		}

		fmt.Fprint(os.Stdout, string(out))
		return nil
	},
}

func init() {
	searchCmd.Flags().BoolP("ignore-case", "i", false, "Case-insensitive search")
	searchCmd.Flags().BoolP("regex", "r", false, "Treat pattern as regex")
	searchCmd.Flags().StringP("sheet", "s", "", "Search only in specific sheet")
	searchCmd.Flags().IntP("max", "m", 0, "Maximum results (0 = unlimited)")
	rootCmd.AddCommand(searchCmd)
}
