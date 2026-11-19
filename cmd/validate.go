package cmd

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/fjbalsamo/storyline/internal/backlog"
	"github.com/fjbalsamo/storyline/internal/story"
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
		if err := validateAll(b, projectRoot(path)); err != nil {
			return fmt.Errorf("%s is invalid:\n%w", path, err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "✓ %s is valid (%d stories)\n", path, len(b.Items))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(validateCmd)
}

// projectRoot is the directory story paths are relative to: the parent of
// the directory holding the backlog (stories/backlog.yaml → project root).
func projectRoot(backlogPath string) string {
	return filepath.Dir(filepath.Dir(backlogPath))
}

// validateAll runs the backlog checks and then the story checks for every
// item that points to a story.md.
func validateAll(b *backlog.Backlog, root string) error {
	errs := []error{b.Validate()}
	for _, it := range b.Items {
		if it.Story == "" {
			continue
		}
		errs = append(errs, story.Check(root, it.ID, string(it.Status), it.Story))
	}
	return errors.Join(errs...)
}
