package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/fjbalsamo/storyline/internal/backlog"
	"github.com/spf13/cobra"
)

const editWarning = "warning: story.md may need the same change"

func newEditCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "edit <id>",
		Short: "Fix the wording of a story (as, want, so_that only)",
		Long: `Change the wording of a story in the backlog. Only --as, --want and --so-that
can be edited; id, status and story never can (use "storyline status" for
status). At least one flag is required.

todo and refining stories can be edited freely. ready and in-progress stories
can too, with a warning that story.md may need the same change. done and
discarded stories are history and are rejected.

The backlog is validated before saving, and comments and formatting outside
the edited values are kept as they are.`,
		Example: `  storyline edit US-003 --want "to cancel a booking up to 2h before"`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path, _ := cmd.Flags().GetString("backlog")
			edits, err := editsFromFlags(cmd)
			if err != nil {
				return err
			}
			return runEdit(path, args[0], edits, cmd.OutOrStdout(), cmd.ErrOrStderr())
		},
	}
	c.Flags().String("as", "", "new role (who benefits)")
	c.Flags().String("want", "", "new goal (what they want)")
	c.Flags().String("so-that", "", "new benefit (why they want it)")
	return c
}

// editsFromFlags collects the flags that were set, trimmed and checked.
func editsFromFlags(cmd *cobra.Command) (backlog.Edits, error) {
	var e backlog.Edits
	targets := []struct {
		flag string
		dst  **string
	}{{"as", &e.As}, {"want", &e.Want}, {"so-that", &e.SoThat}}

	for _, t := range targets {
		if !cmd.Flags().Changed(t.flag) {
			continue
		}
		raw, _ := cmd.Flags().GetString(t.flag)
		v, err := cleanValue(t.flag, raw)
		if err != nil {
			return e, err
		}
		*t.dst = &v
	}
	if e.As == nil && e.Want == nil && e.SoThat == nil {
		return e, errors.New("nothing to edit: pass at least one of --as, --want, --so-that")
	}
	return e, nil
}

func cleanValue(flag, raw string) (string, error) {
	v := strings.TrimSpace(raw)
	if v == "" {
		return "", fmt.Errorf("--%s must not be empty", flag)
	}
	if strings.ContainsAny(v, "\r\n") {
		return "", fmt.Errorf("--%s must be a single line", flag)
	}
	return v, nil
}

func runEdit(path, id string, edits backlog.Edits, stdout, stderr io.Writer) error {
	src, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("open backlog: %w", err)
	}
	b, err := backlog.Decode(bytes.NewReader(src))
	if err != nil {
		return err
	}
	it, ok := b.Find(id)
	if !ok {
		return fmt.Errorf("story %s not found", id)
	}
	if err := backlog.CheckEditable(it.Status); err != nil {
		return err
	}

	out, changed, err := backlog.ApplyEdits(src, id, edits)
	if err != nil {
		return err
	}
	if len(changed) == 0 {
		fmt.Fprintln(stdout, "nothing to change")
		return nil
	}
	if err := saveEdited(path, out); err != nil {
		return err
	}

	if it.Status == backlog.StatusReady || it.Status == backlog.StatusInProgress {
		fmt.Fprintln(stderr, editWarning)
	}
	fmt.Fprintf(stdout, "✓ %s updated: %s\n", id, strings.Join(changed, ", "))
	return nil
}

// saveEdited validates the whole edited backlog and only then writes it.
func saveEdited(path string, data []byte) error {
	b, err := backlog.Decode(bytes.NewReader(data))
	if err != nil {
		return err
	}
	if err := validateAll(b, projectRoot(path)); err != nil {
		return fmt.Errorf("not saved, %s would be invalid:\n%w", path, err)
	}
	return backlog.WriteFile(path, data)
}

func init() {
	rootCmd.AddCommand(newEditCmd())
}
