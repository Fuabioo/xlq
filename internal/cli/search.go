package cli

import (
	"fmt"
	"os"

	"github.com/fuabioo/xlq/internal/output"
	"github.com/fuabioo/xlq/internal/xlsx"
	"github.com/spf13/cobra"
)

var (
	searchIgnoreCase bool
	searchRegex      bool
	searchSheet      string
	searchMax        int
)

var searchCmd = &cobra.Command{
	Use:   "search <file.xlsx> <pattern>",
	Short: "Search for cells matching pattern",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		f, err := xlsx.OpenFile(args[0])
		if err != nil {
			return err
		}
		defer f.Close()

		opts := xlsx.SearchOptions{
			Sheet:           searchSheet,
			CaseInsensitive: searchIgnoreCase,
			Regex:           searchRegex,
			MaxResults:      searchMax,
		}

		ch, err := xlsx.Search(f, args[1], opts)
		if err != nil {
			return err
		}

		results, err := xlsx.CollectSearchResults(ch)
		if err != nil {
			return err
		}

		out, err := output.FormatSingle(GetFormat(), results)
		if err != nil {
			return err
		}

		fmt.Fprint(os.Stdout, string(out))
		return nil
	},
}

func init() {
	searchCmd.Flags().BoolVarP(&searchIgnoreCase, "ignore-case", "i", false, "Case-insensitive search")
	searchCmd.Flags().BoolVarP(&searchRegex, "regex", "r", false, "Treat pattern as regex")
	searchCmd.Flags().StringVarP(&searchSheet, "sheet", "s", "", "Search only in specific sheet")
	searchCmd.Flags().IntVarP(&searchMax, "max", "m", 0, "Maximum results (0 = unlimited)")
	rootCmd.AddCommand(searchCmd)
}
