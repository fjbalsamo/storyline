package cmd

import (
	"os"

	"github.com/fjbalsamo/storyline/internal/backlog"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "storyline",
	Short: "User-story driven development for AI coding assistants",
	Long: `storyline drives development from a backlog of user stories,
delivering each one as a small vertical slice.`,
	SilenceUsage: true, // don't dump usage on validation errors
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().String("backlog", backlog.DefaultPath, "path to the backlog file")
}
