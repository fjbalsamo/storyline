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
		Short: "Fix the wording of a story, or set where it came from",
		Long: `Change the wording of a story in the backlog: --as, --want and --so-that.
You can also set or change its ref with --ref, or drop it with --remove-ref
(not both). id, status and story can never be edited (use "storyline status"
for status). At least one flag is required.

todo and refining stories can be edited freely. ready and in-progress stories
can too, with a warning that story.md may need the same change. done and
discarded stories are history: their wording can't be edited, but their ref
can, because it records where the story came from and is not part of it.

The backlog is validated before saving, and comments and formatting outside
the edited values are kept as they are.`,
		Example: `  storyline edit US-003 --want "to cancel a booking up to 2h before"
  storyline edit US-003 --ref TKT-001
  storyline edit US-003 --remove-ref`,
		Args: cobra.ExactArgs(1),
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
	c.Flags().String("ref", "", "set or change where the story came from (any status)")
	c.Flags().Bool("remove-ref", false, "remove the ref (any status)")
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
	if err := refFromEditFlags(cmd, &e); err != nil {
		return e, err
	}
	if !e.TouchesText() && e.Ref == nil && !e.RemoveRef {
		return e, errors.New("nothing to edit: pass at least one of --as, --want, --so-that, --ref, --remove-ref")
	}
	return e, nil
}

// refFromEditFlags reads --ref and --remove-ref, which exclude each other.
func refFromEditFlags(cmd *cobra.Command, e *backlog.Edits) error {
	remove, _ := cmd.Flags().GetBool("remove-ref")
	if !cmd.Flags().Changed("ref") {
		e.RemoveRef = remove
		return nil
	}
	if remove {
		return errors.New("--ref and --remove-ref can't be used together")
	}
	raw, _ := cmd.Flags().GetString("ref")
	v, err := cleanValue("ref", raw)
	if err != nil {
		return err
	}
	e.Ref = &v
	return nil
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
	if edits.TouchesText() {
		if err := backlog.CheckEditable(it.Status); err != nil {
			return err
		}
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

	if changedText(changed) && (it.Status == backlog.StatusReady || it.Status == backlog.StatusInProgress) {
		fmt.Fprintln(stderr, editWarning)
	}
	fmt.Fprintf(stdout, "✓ %s updated: %s\n", id, strings.Join(changed, ", "))
	return nil
}

// changedText reports whether any changed field is part of the story's
// wording; a ref change alone doesn't touch story.md.
func changedText(changed []string) bool {
	for _, f := range changed {
		if f != "ref" {
			return true
		}
	}
	return false
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
