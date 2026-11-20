package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/fjbalsamo/storyline/internal/backlog"
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:     "add",
	Short:   "Add a user story to the bottom of the backlog",
	Example: `  storyline add --as "club player" --want "to cancel a booking" --so-that "the court is freed for others"`,
	Args:    cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		path, _ := cmd.Flags().GetString("backlog")
		as, _ := cmd.Flags().GetString("as")
		want, _ := cmd.Flags().GetString("want")
		soThat, _ := cmd.Flags().GetString("so-that")

		as, want, soThat = strings.TrimSpace(as), strings.TrimSpace(want), strings.TrimSpace(soThat)
		if as == "" || want == "" || soThat == "" {
			return errors.New("--as, --want and --so-that must not be empty")
		}
		ref, err := refFromFlag(cmd)
		if err != nil {
			return err
		}

		b, err := backlog.Load(path)
		switch {
		case errors.Is(err, os.ErrNotExist):
			b = backlog.New()
		case err != nil:
			return err
		}

		if err := b.Validate(); err != nil {
			return fmt.Errorf("%s is invalid, fix it before adding stories:\n%w", path, err)
		}

		it := b.Add(as, want, soThat, ref)
		if err := b.Save(path); err != nil {
			return err
		}

		fmt.Fprintf(cmd.OutOrStdout(), "✓ added %s: As %s, I want %s\n", it.ID, it.As, it.Want)
		return nil
	},
}

// refFromFlag returns the trimmed --ref, or "" when the flag wasn't passed.
func refFromFlag(cmd *cobra.Command) (string, error) {
	if !cmd.Flags().Changed("ref") {
		return "", nil
	}
	raw, _ := cmd.Flags().GetString("ref")
	ref := strings.TrimSpace(raw)
	switch {
	case ref == "":
		return "", errors.New("--ref must not be empty")
	case strings.ContainsAny(ref, "\r\n"):
		return "", errors.New("--ref must be a single line")
	}
	return ref, nil
}

func init() {
	addCmd.Flags().String("as", "", "who benefits (role)")
	addCmd.Flags().String("want", "", "what they want (goal)")
	addCmd.Flags().String("so-that", "", "why they want it (benefit)")
	addCmd.Flags().String("ref", "", "where the story came from, e.g. an issue tracker key (TKT-001)")
	_ = addCmd.MarkFlagRequired("as")
	_ = addCmd.MarkFlagRequired("want")
	_ = addCmd.MarkFlagRequired("so-that")
	rootCmd.AddCommand(addCmd)
}
