package cmd

import (
	"errors"
	"fmt"

	"github.com/fjbalsamo/storyline/internal/backlog"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status <id> <status>",
	Short: "Move a story to another status, enforcing the workflow",
	Long: `Move a story to another status. Only these transitions are allowed:

  todo        → refining, discarded
  refining    → ready, todo, discarded
  ready       → in-progress, refining, discarded
  in-progress → ready, discarded
  discarded   → todo
  done        → (none; reach done with "storyline done <id>")

Moving to ready requires a story.md that passes "storyline validate".`,
	Example: `  storyline status US-001 refining
  storyline status US-001 ready --story storyline/active/US-001-book-court/story.md`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		path, _ := cmd.Flags().GetString("backlog")
		storyPath, _ := cmd.Flags().GetString("story")
		to := backlog.Status(args[1])

		if storyPath != "" && to != backlog.StatusReady {
			return errors.New("--story is only accepted when moving to ready")
		}

		b, err := backlog.Load(path)
		if err != nil {
			return err
		}
		from, err := applyStatus(b, args[0], to, storyPath)
		if err != nil {
			return err
		}
		if err := validateAll(b, projectRoot(path)); err != nil {
			return fmt.Errorf("not saved, %s would be invalid:\n%w", path, err)
		}
		if err := b.Save(path); err != nil {
			return err
		}

		fmt.Fprintf(cmd.OutOrStdout(), "✓ %s: %s → %s\n", args[0], from, to)
		return nil
	},
}

// applyStatus performs the transition and the --story bookkeeping in memory.
func applyStatus(b *backlog.Backlog, id string, to backlog.Status, storyPath string) (backlog.Status, error) {
	from, err := b.Transition(id, to)
	if err != nil {
		return from, err
	}
	it, _ := b.Find(id)
	if storyPath != "" {
		it.Story = storyPath
	}
	if to == backlog.StatusReady && it.Story == "" {
		return from, fmt.Errorf("%s: moving to ready requires a story path (pass --story <path to story.md>)", id)
	}
	return from, nil
}

func init() {
	statusCmd.Flags().String("story", "", "path to story.md (only with ready)")
	rootCmd.AddCommand(statusCmd)
}
