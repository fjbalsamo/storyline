package cmd

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/fjbalsamo/storyline/internal/backlog"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List backlog stories in priority order",
	Long: `List backlog stories in priority order (top = highest).
By default only open stories are shown; ▶ marks the next one to work on.
A REF column appears when a listed story has a ref. --ref KEY lists the
stories with that ref, whatever their status.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		path, _ := cmd.Flags().GetString("backlog")
		f, err := listFilterFromFlags(cmd)
		if err != nil {
			return err
		}
		b, err := backlog.Load(path)
		if err != nil {
			return err
		}
		rows, nextIdx := listRows(b, f)
		return printList(cmd.OutOrStdout(), rows, nextIdx)
	},
}

type listFilter struct {
	all       bool
	status    backlog.Status
	statusSet bool
	ref       string
}

type listRow struct {
	idx  int
	item backlog.Item
}

func listFilterFromFlags(cmd *cobra.Command) (listFilter, error) {
	all, _ := cmd.Flags().GetBool("all")
	statusFlag, _ := cmd.Flags().GetString("status")
	ref, _ := cmd.Flags().GetString("ref")

	f := listFilter{all: all, status: backlog.Status(statusFlag), statusSet: statusFlag != "", ref: ref}
	if f.statusSet && !f.status.Valid() {
		return f, fmt.Errorf("invalid status %q", statusFlag)
	}
	return f, nil
}

// match reports whether an item is listed. A ref filter ignores whether the
// story is open, so imported stories that are done still show up.
func (f listFilter) match(it backlog.Item) bool {
	switch {
	case f.ref != "":
		return string(it.Ref) == f.ref && (!f.statusSet || it.Status == f.status)
	case f.statusSet:
		return it.Status == f.status
	}
	return f.all || it.Status.Open()
}

// listRows returns the listed rows and the index of the first open story of
// the whole backlog (-1 if none), which ▶ marks when it is listed.
func listRows(b *backlog.Backlog, f listFilter) (rows []listRow, nextIdx int) {
	nextIdx = -1
	for i, it := range b.Items {
		if nextIdx == -1 && it.Status.Open() {
			nextIdx = i
		}
		if f.match(it) {
			rows = append(rows, listRow{idx: i, item: it})
		}
	}
	return rows, nextIdx
}

func printList(out io.Writer, rows []listRow, nextIdx int) error {
	if len(rows) == 0 {
		fmt.Fprintln(out, "No stories to show.")
		return nil
	}
	withRef := anyRef(rows)
	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, listLine(withRef, "", "ID", "STATUS", "REF", "STORY"))
	for _, r := range rows {
		marker := ""
		if r.idx == nextIdx {
			marker = "▶"
		}
		it := r.item
		fmt.Fprintln(w, listLine(withRef, marker, it.ID, string(it.Status), string(it.Ref),
			fmt.Sprintf("As %s, I want %s", it.As, it.Want)))
	}
	return w.Flush()
}

func anyRef(rows []listRow) bool {
	for _, r := range rows {
		if r.item.Ref != "" {
			return true
		}
	}
	return false
}

// listLine joins the columns with tabs, leaving out REF when it isn't shown.
func listLine(withRef bool, marker, id, status, ref, story string) string {
	if withRef {
		return marker + "\t" + id + "\t" + status + "\t" + ref + "\t" + story
	}
	return marker + "\t" + id + "\t" + status + "\t" + story
}

func init() {
	listCmd.Flags().BoolP("all", "a", false, "include done and discarded stories")
	listCmd.Flags().StringP("status", "s", "", "only show stories with this status")
	listCmd.Flags().String("ref", "", "only show stories with this ref (any status)")
	rootCmd.AddCommand(listCmd)
}
