package cmd

import (
	"fmt"
	"strings"

	"github.com/fjbalsamo/storyline/internal/adapters"
	"github.com/fjbalsamo/storyline/internal/scaffold"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init [dir]",
	Short: "Set up storyline in a project",
	Long: `Set up storyline in a project (current directory by default):
the stories/ tree, an empty backlog and the story commands (context, add,
refine, slice, apply, verify, done) for the AI tools you pick with --tools.
AGENTS.md is always written, and each tool that does not read it natively
gets a one-line pointer file (CLAUDE.md, GEMINI.md) that imports it.
Safe to run again: existing files are kept, the storyline section of
AGENTS.md is updated in place, and pointer files only gain the missing line.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		root := "."
		if len(args) == 1 {
			root = args[0]
		}
		force, _ := cmd.Flags().GetBool("force")

		toolIDs, _ := cmd.Flags().GetString("tools")
		tools, err := adapters.Get(strings.Split(toolIDs, ","))
		if err != nil {
			return err
		}

		res, err := scaffold.Run(scaffold.Options{Root: root, Force: force, Tools: tools})
		if err != nil {
			return err
		}

		out := cmd.OutOrStdout()
		for _, p := range res.Created {
			fmt.Fprintf(out, "  created  %s\n", p)
		}
		for _, p := range res.Updated {
			fmt.Fprintf(out, "  updated  %s\n", p)
		}
		for _, s := range res.Skipped {
			fmt.Fprintf(out, "  skipped  %s (%s)\n", s.Path, s.Reason)
		}
		fmt.Fprintf(out, "\n✓ storyline is ready. Tools configured: %s.\n", toolNames(tools))
		fmt.Fprintln(out, "Next: run the context command in your AI tool to describe your project.")
		return nil
	},
}

func toolNames(tools []adapters.Adapter) string {
	names := make([]string, len(tools))
	for i, t := range tools {
		names[i] = t.Name()
	}
	return strings.Join(names, ", ")
}

func init() {
	initCmd.Flags().String("tools", "claude",
		"comma-separated AI tools to set up: "+strings.Join(adapters.IDs(), ", ")+", or all (agents-md is always set up)")
	initCmd.Flags().Bool("force", false, "overwrite existing command files (the backlog is never overwritten)")
	rootCmd.AddCommand(initCmd)
}
