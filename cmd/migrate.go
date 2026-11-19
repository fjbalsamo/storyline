package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"

	"github.com/fjbalsamo/storyline/internal/backlog"
	"github.com/spf13/cobra"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Move a project from the old stories/ layout to storyline/",
	Long: `Move stories/backlog.yaml, stories/active/, stories/done/ and
stories/living/ into storyline/ and rewrite the story paths in the backlog.
Nothing else in stories/ is touched (it may belong to Storybook), and stories/
is removed only if it ends up empty. If the result does not validate,
everything is moved back.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		if err := migrate("."); err != nil {
			return err
		}
		printMigrated(cmd.OutOrStdout())
		return nil
	},
}

func printMigrated(w io.Writer) {
	fmt.Fprintln(w, "✓ migrated to storyline/")
	fmt.Fprintln(w, "Next: storyline init --tools <your tools> --force   (refreshes the AI commands)")
	fmt.Fprintln(w, `Check the "## Project" section of AGENTS.md for references to stories/.`)
}

// migrated lists what moves; anything else in stories/ stays where it is.
var migrated = []string{"backlog.yaml", "active", "done", "living"}

var storyPathRe = regexp.MustCompile(`(?m)^(\s*story:\s*["']?)stories/`)

type move struct{ from, to string }

// migrate converts the layout under root. On any failure after the first
// move it restores the original layout and backlog bytes.
func migrate(root string) error {
	oldDir, newDir := filepath.Join(root, backlog.OldDir), filepath.Join(root, backlog.Dir)
	oldBacklog := filepath.Join(oldDir, "backlog.yaml")

	original, err := os.ReadFile(oldBacklog)
	if err != nil {
		return fmt.Errorf("migrate: need %s/backlog.yaml: %w", backlog.OldDir, err)
	}
	if _, err := os.Lstat(newDir); err == nil {
		return fmt.Errorf("migrate: %s/ already exists; move or remove it first", backlog.Dir)
	}

	moves, err := moveIn(oldDir, newDir)
	if err != nil {
		rollback(moves, newDir, original)
		return fmt.Errorf("migrate: %w", err)
	}

	newBacklog := filepath.Join(newDir, "backlog.yaml")
	rewritten := storyPathRe.ReplaceAll(original, []byte("${1}"+backlog.Dir+"/"))
	if err := os.WriteFile(newBacklog, rewritten, 0o644); err != nil {
		rollback(moves, newDir, original)
		return fmt.Errorf("migrate: rewrite backlog: %w", err)
	}
	if err := validateFile(newBacklog); err != nil {
		rollback(moves, newDir, original)
		return fmt.Errorf("migrate: %s would be invalid, nothing was changed:\n%w", newBacklog, err)
	}

	removeIfEmpty(oldDir)
	return nil
}

// moveIn creates newDir and moves the known entries of oldDir into it.
func moveIn(oldDir, newDir string) ([]move, error) {
	var done []move
	if err := os.Mkdir(newDir, 0o755); err != nil {
		return nil, err
	}
	for _, name := range migrated {
		from := filepath.Join(oldDir, name)
		if _, err := os.Lstat(from); errors.Is(err, os.ErrNotExist) {
			continue
		}
		to := filepath.Join(newDir, name)
		if err := os.Rename(from, to); err != nil {
			return done, err
		}
		done = append(done, move{from, to})
	}
	return done, nil
}

func validateFile(path string) error {
	b, err := backlog.Load(path)
	if err != nil {
		return err
	}
	return validateAll(b, projectRoot(path))
}

// rollback undoes migrate: the original backlog bytes are put back before the
// file moves home, then the storyline/ folder it created is removed.
func rollback(moves []move, newDir string, original []byte) {
	for i := len(moves) - 1; i >= 0; i-- {
		m := moves[i]
		if filepath.Base(m.to) == "backlog.yaml" {
			_ = os.WriteFile(m.to, original, 0o644)
		}
		_ = os.Rename(m.to, m.from)
	}
	_ = os.Remove(newDir)
}

func removeIfEmpty(dir string) {
	if entries, err := os.ReadDir(dir); err == nil && len(entries) == 0 {
		_ = os.Remove(dir)
	}
}

func init() {
	rootCmd.AddCommand(migrateCmd)
}
