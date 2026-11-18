package cmd

import (
	"fmt"

	"github.com/fjbalsamo/storyline/internal/backlog"
	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate the backlog file",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		path, _ := cmd.Flags().GetString("backlog")

		b, err := backlog.Load(path)
		if err != nil {
			return err
		}
		if err := b.Validate(); err != nil {
			return fmt.Errorf("%s is invalid:\n%w", path, err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "✓ %s is valid (%d stories)\n", path, len(b.Items))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(validateCmd)
}
