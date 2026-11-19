package cmd

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/fjbalsamo/storyline/internal/backlog"
	"github.com/fjbalsamo/storyline/internal/story"
	"github.com/fjbalsamo/storyline/internal/trace"
	"github.com/spf13/cobra"
)

var traceCmd = &cobra.Command{
	Use:   "trace [id]",
	Short: "Show which tests prove each acceptance criterion",
	Long: `Show, for each acceptance criterion, the tests that reference it.

By default only in-progress stories are traced; pass an ID for one story or
--all for every story that has a story.md. Test files are found under --root
(Go, JS/TS, Python, Rust, Java/Kotlin). A criterion is referenced as AC-12.1
or ac_12_1; references on comment-only lines do not count.

Exit code 1 if any criterion in scope has no tests, so it can gate CI.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		blPath, _ := cmd.Flags().GetString("backlog")
		all, _ := cmd.Flags().GetBool("all")
		scanRoot, _ := cmd.Flags().GetString("root")

		if all && len(args) == 1 {
			return errors.New("pass either an ID or --all, not both")
		}
		b, err := backlog.Load(blPath)
		if err != nil {
			return err
		}
		items, err := traceScope(b, args, all)
		if err != nil {
			return err
		}
		out := cmd.OutOrStdout()
		if len(items) == 0 {
			fmt.Fprintln(out, "no in-progress stories to trace (pass an ID or --all)")
			return nil
		}

		idx, err := trace.Scan(scanRoot)
		if err != nil {
			return err
		}
		root := projectRoot(blPath)
		var uncovered int
		for i, it := range items {
			s, err := story.ParseStory(filepath.Join(root, filepath.FromSlash(it.Story)))
			if err != nil {
				return fmt.Errorf("%s: %w", it.ID, err)
			}
			if i > 0 {
				fmt.Fprintln(out)
			}
			uncovered += renderTrace(out, s, idx)
		}
		if uncovered > 0 {
			noun := "criteria"
			if uncovered == 1 {
				noun = "criterion"
			}
			return fmt.Errorf("%d %s without tests", uncovered, noun)
		}
		return nil
	},
}

// traceScope picks the items to trace: one by ID, every story, or in-progress.
func traceScope(b *backlog.Backlog, args []string, all bool) ([]backlog.Item, error) {
	if len(args) == 1 {
		it, ok := b.Find(args[0])
		switch {
		case !ok:
			return nil, fmt.Errorf("%s not found in backlog", args[0])
		case it.Story == "":
			return nil, fmt.Errorf("%s has no story.md yet", it.ID)
		}
		return []backlog.Item{*it}, nil
	}
	var items []backlog.Item
	for _, it := range b.Items {
		if it.Story != "" && (all || it.Status == backlog.StatusInProgress) {
			items = append(items, it)
		}
	}
	return items, nil
}

// renderTrace prints one story and returns how many criteria are uncovered.
func renderTrace(w io.Writer, s *story.Story, idx trace.Index) int {
	fmt.Fprintf(w, "%s: %s\n", s.ID, s.Title)
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	covered := 0
	for _, c := range s.Criteria {
		var refs []string
		if key, ok := trace.ParseCriterion(c.ID); ok {
			for _, r := range idx[key] {
				refs = append(refs, r.String())
			}
		}
		if len(refs) == 0 {
			fmt.Fprintf(tw, "  ✗ %s\t%s\tno tests\n", c.ID, c.Name)
			continue
		}
		covered++
		fmt.Fprintf(tw, "  ✓ %s\t%s\t%s\n", c.ID, c.Name, strings.Join(refs, ", "))
	}
	_ = tw.Flush()
	fmt.Fprintf(w, "%d/%d criteria covered\n", covered, len(s.Criteria))
	return len(s.Criteria) - covered
}

func init() {
	traceCmd.Flags().Bool("all", false, "trace every story that has a story.md")
	traceCmd.Flags().String("root", ".", "directory to scan for test files")
	rootCmd.AddCommand(traceCmd)
}
