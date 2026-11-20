package cmd

import (
	"os"
	"runtime/debug"

	"github.com/fjbalsamo/storyline/internal/backlog"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "storyline",
	Short: "User-story driven development for AI coding assistants",
	Long: `storyline drives development from a backlog of user stories,
delivering each one as a small vertical slice.`,
	SilenceUsage: true, // don't dump usage on validation errors
	PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
		if !usesBacklog[cmd.Name()] || cmd.Flags().Changed("backlog") {
			return nil
		}
		return backlog.CheckLayout(".")
	},
}

// usesBacklog lists the commands that fail on the old stories/ layout.
// init checks its own target directory; migrate is the way out.
var usesBacklog = map[string]bool{
	"add": true, "edit": true, "move": true, "list": true, "validate": true, "status": true, "done": true, "trace": true,
	"show": true, "set": true, "add-dependency": true, "remove-dependency": true,
}

// Execute runs the CLI; version is the one injected into main at build time.
func Execute(version string) {
	info, _ := debug.ReadBuildInfo()
	rootCmd.Version = ResolveVersion(version, info)
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().String("backlog", backlog.DefaultPath, "path to the backlog file")
}
