package cli

import (
	"fmt"
	"os"

	"github.com/fuabioo/xlq/internal/output"
	"github.com/fuabioo/xlq/internal/xlsx"
	"github.com/spf13/cobra"
)

var sheetsCmd = &cobra.Command{
	Use:   "sheets <file.xlsx>",
	Short: "List all sheets in workbook",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath, err := ResolveFilePath(GetBasepathFromCmd(cmd), args[0])
		if err != nil {
			return err
		}
		f, err := xlsx.OpenFile(filePath)
		if err != nil {
			return err
		}
		defer f.Close()

		sheets, err := xlsx.GetSheets(f)
		if err != nil {
			return err
		}

		out, err := output.FormatSingle(GetFormatFromCmd(cmd), sheets)
		if err != nil {
			return err
		}

		fmt.Fprint(os.Stdout, string(out))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(sheetsCmd)
}
