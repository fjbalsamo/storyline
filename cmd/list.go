package cmd

import (
	"fmt"
	"text/tabwriter"

	"github.com/fjbalsamo/storyline/internal/backlog"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List backlog stories in priority order",
	Long: `List backlog stories in priority order (top = highest).
By default only open stories are shown; ▶ marks the next one to work on.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		path, _ := cmd.Flags().GetString("backlog")
		all, _ := cmd.Flags().GetBool("all")
		statusFlag, _ := cmd.Flags().GetString("status")

		status := backlog.Status(statusFlag)
		if statusFlag != "" && !status.Valid() {
			return fmt.Errorf("invalid status %q", statusFlag)
		}

		b, err := backlog.Load(path)
		if err != nil {
			return err
		}

		type row struct {
			idx  int
			item backlog.Item
		}
		var rows []row
		nextIdx := -1
		for i, it := range b.Items {
			if nextIdx == -1 && it.Status.Open() {
				nextIdx = i
			}
			switch {
			case statusFlag != "" && it.Status != status:
				continue
			case statusFlag == "" && !all && !it.Status.Open():
				continue
			}
			rows = append(rows, row{idx: i, item: it})
		}

		out := cmd.OutOrStdout()
		if len(rows) == 0 {
			fmt.Fprintln(out, "No stories to show.")
			return nil
		}

		w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "\tID\tSTATUS\tSTORY")
		for _, r := range rows {
			marker := ""
			if r.idx == nextIdx {
				marker = "▶"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\tAs %s, I want %s\n", marker, r.item.ID, r.item.Status, r.item.As, r.item.Want)
		}
		return w.Flush()
	},
}

func init() {
	listCmd.Flags().BoolP("all", "a", false, "include done and discarded stories")
	listCmd.Flags().StringP("status", "s", "", "only show stories with this status")
	rootCmd.AddCommand(listCmd)
}
