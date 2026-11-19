package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/fjbalsamo/storyline/internal/backlog"
	"github.com/fjbalsamo/storyline/internal/story"
	"github.com/spf13/cobra"
)

const (
	activeDir = "stories/active"
	doneDir   = "stories/done"
)

var doneCmd = &cobra.Command{
	Use:   "done <id>",
	Short: "Close an in-progress story and archive its folder",
	Long: `Close an in-progress story: moves stories/active/<dir> to
stories/done/<YYYY-MM-DD>-<dir>, sets its status to done and updates its
story path. tasks.md must exist and every task in it must be checked
(--force skips only that check). Living docs are not touched.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		blPath, _ := cmd.Flags().GetString("backlog")
		force, _ := cmd.Flags().GetBool("force")

		b, err := backlog.Load(blPath)
		if err != nil {
			return err
		}
		newPath, err := closeStory(b, blPath, args[0], force, time.Now, cmd.ErrOrStderr())
		if err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "✓ %s done → %s/\n", args[0], path.Dir(newPath))
		return nil
	},
}

// closeStory checks the preconditions, moves the folder, updates and saves
// the backlog, and moves the folder back if anything after the move fails.
// It returns the new story path.
func closeStory(b *backlog.Backlog, blPath, id string, force bool, now func() time.Time, warn io.Writer) (string, error) {
	root := projectRoot(blPath)
	it, err := doneItem(b, id)
	if err != nil {
		return "", err
	}
	oldDir := path.Dir(filepath.ToSlash(it.Story))
	if path.Dir(oldDir) != activeDir {
		return "", fmt.Errorf("%s: story folder %s is not under %s/", id, oldDir, activeDir)
	}
	if err := checkTasksDone(id, filepath.Join(root, filepath.FromSlash(oldDir)), force, warn); err != nil {
		return "", err
	}

	newDir := path.Join(doneDir, now().Format("2006-01-02")+"-"+path.Base(oldDir))
	src, dst := filepath.Join(root, filepath.FromSlash(oldDir)), filepath.Join(root, filepath.FromSlash(newDir))
	if _, err := os.Stat(dst); err == nil {
		return "", fmt.Errorf("%s: destination %s already exists", id, newDir)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return "", fmt.Errorf("%s: create %s: %w", id, doneDir, err)
	}
	if err := os.Rename(src, dst); err != nil {
		return "", fmt.Errorf("%s: move story folder: %w", id, err)
	}

	oldStory, oldStatus := it.Story, it.Status
	it.Story, it.Status = path.Join(newDir, path.Base(filepath.ToSlash(oldStory))), backlog.StatusDone
	if err := saveClosed(b, blPath, root); err != nil {
		it.Story, it.Status = oldStory, oldStatus
		if rbErr := os.Rename(dst, src); rbErr != nil {
			err = errors.Join(err, fmt.Errorf("could not move %s back: %w", newDir, rbErr))
		}
		return "", fmt.Errorf("%s: %w", id, err)
	}
	return path.Join(newDir, path.Base(filepath.ToSlash(oldStory))), nil
}

func doneItem(b *backlog.Backlog, id string) (*backlog.Item, error) {
	it, ok := b.Find(id)
	switch {
	case !ok:
		return nil, fmt.Errorf("%s not found in backlog", id)
	case it.Status != backlog.StatusInProgress:
		return nil, fmt.Errorf("%s is %s; only in-progress stories can be closed (use `storyline status %s in-progress` first)", id, it.Status, id)
	case it.Story == "":
		return nil, fmt.Errorf("%s has no story path", id)
	}
	return it, nil
}

// checkTasksDone fails on unchecked tasks unless force, in which case it
// lists them as a warning.
func checkTasksDone(id, dir string, force bool, warn io.Writer) error {
	tasksPath := filepath.Join(dir, "tasks.md")
	if _, err := os.Stat(tasksPath); errors.Is(err, os.ErrNotExist) {
		if force {
			fmt.Fprintf(warn, "warning: closing %s without tasks.md\n", id)
			return nil
		}
		return fmt.Errorf("%s: no tasks.md: run the slice command first (or use --force)", id)
	}
	tk, err := story.ParseTasks(tasksPath)
	if err != nil {
		return fmt.Errorf("%s: %w", id, err)
	}
	var open []string
	for _, t := range tk.Tasks {
		if !t.Done {
			open = append(open, fmt.Sprintf("  %s:%d: %d. %s", tasksPath, t.Line, t.Number, t.Text))
		}
	}
	switch {
	case len(open) == 0:
		return nil
	case force:
		fmt.Fprintf(warn, "warning: closing %s with %d unchecked task(s):\n%s\n", id, len(open), strings.Join(open, "\n"))
		return nil
	}
	return fmt.Errorf("%s has %d unchecked task(s) (check them or use --force):\n%s", id, len(open), strings.Join(open, "\n"))
}

func saveClosed(b *backlog.Backlog, blPath, root string) error {
	if err := validateAll(b, root); err != nil {
		return fmt.Errorf("not saved, %s would be invalid:\n%w", blPath, err)
	}
	return b.Save(blPath)
}

func init() {
	doneCmd.Flags().Bool("force", false, "close even if some tasks are unchecked (they are listed as a warning)")
	rootCmd.AddCommand(doneCmd)
}
