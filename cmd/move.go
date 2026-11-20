package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/fjbalsamo/storyline/internal/backlog"
	"github.com/spf13/cobra"
)

func newMoveCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "move <id> --top | --bottom | --before <id> | --after <id>",
		Short: "Change a story's priority by moving it in the backlog",
		Long: `Move an open story to another position. Order is priority: the first open
story is the next one to work on. Pass exactly one of the four flags.

--top puts the story before the first open story (done and discarded stories
stay above it). --before and --after take any story as the reference, whatever
its status. done and discarded stories can't be moved.

The story moves with the comments directly above it; the rest of the file is
left as it is. Nothing is saved if the backlog would be invalid.`,
		Example: `  storyline move US-005 --top
  storyline move US-002 --after US-004`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path, _ := cmd.Flags().GetString("backlog")
			place, err := placementFromFlags(cmd)
			if err != nil {
				return err
			}
			return runMove(path, args[0], place, cmd.OutOrStdout())
		},
	}
	c.Flags().Bool("top", false, "move before the first open story")
	c.Flags().Bool("bottom", false, "move to the end of the backlog")
	c.Flags().String("before", "", "move right before this story")
	c.Flags().String("after", "", "move right after this story")
	return c
}

// placementFromFlags requires exactly one of the four position flags.
func placementFromFlags(cmd *cobra.Command) (backlog.Placement, error) {
	var set []backlog.Placement
	for _, f := range []struct {
		name string
		mode backlog.Mode
	}{{"top", backlog.MoveTop}, {"bottom", backlog.MoveBottom}} {
		if on, _ := cmd.Flags().GetBool(f.name); on {
			set = append(set, backlog.Placement{Mode: f.mode})
		}
	}
	for _, f := range []struct {
		name string
		mode backlog.Mode
	}{{"before", backlog.MoveBefore}, {"after", backlog.MoveAfter}} {
		if !cmd.Flags().Changed(f.name) {
			continue
		}
		raw, _ := cmd.Flags().GetString(f.name)
		ref := strings.TrimSpace(raw)
		if ref == "" {
			return backlog.Placement{}, fmt.Errorf("--%s needs a story id", f.name)
		}
		set = append(set, backlog.Placement{Mode: f.mode, Ref: ref})
	}
	if len(set) != 1 {
		return backlog.Placement{}, errors.New("pass exactly one of --top, --bottom, --before, --after")
	}
	return set[0], nil
}

func runMove(path, id string, p backlog.Placement, stdout io.Writer) error {
	src, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("open backlog: %w", err)
	}
	out, from, to, err := backlog.ApplyMove(src, id, p)
	if err != nil {
		return err
	}
	if from == to {
		fmt.Fprintln(stdout, "nothing to change")
		return nil
	}
	if err := saveEdited(path, out); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "✓ %s moved: position %d → %d\n", id, from, to)
	return nil
}

func init() {
	rootCmd.AddCommand(newMoveCmd())
}
