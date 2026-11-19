package backlog

import (
	"errors"
	"os"
	"path/filepath"
)

const (
	// Dir is the project folder storyline owns. It is named after the tool,
	// not "stories/", which collides with Storybook.
	Dir = "storyline"
	// OldDir is the folder used before the rename; see `storyline migrate`.
	OldDir = "stories"
)

// ErrOldLayout means the project still uses stories/ instead of storyline/.
var ErrOldLayout = errors.New("found the old layout (stories/). Run: storyline migrate")

// CheckLayout returns ErrOldLayout when root has stories/backlog.yaml but no
// storyline/backlog.yaml.
func CheckLayout(root string) error {
	if exists(filepath.Join(root, Dir, "backlog.yaml")) {
		return nil
	}
	if exists(filepath.Join(root, OldDir, "backlog.yaml")) {
		return ErrOldLayout
	}
	return nil
}

func exists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
