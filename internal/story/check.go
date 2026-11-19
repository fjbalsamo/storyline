package story

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const maxCriteria, maxTasks = 5, 8

// Check validates the story.md (and tasks.md, if present) at storyPath for
// the backlog item itemID with the given status. storyPath is the path as
// written in the backlog (relative to root); root is the project root.
// Errors are joined and prefixed with the story ID and file:line.
func Check(root, itemID, status, storyPath string) error {
	full := filepath.Join(root, filepath.FromSlash(storyPath))
	s, err := ParseStory(full)
	if err != nil {
		return fmt.Errorf("%s: %w", itemID, err)
	}

	var errs []error
	errs = append(errs, checkStory(itemID, s)...)
	errs = append(errs, checkLocation(itemID, status, storyPath)...)

	tasksPath := filepath.Join(filepath.Dir(full), "tasks.md")
	if _, statErr := os.Stat(tasksPath); statErr == nil {
		errs = append(errs, checkTasks(itemID, s, tasksPath)...)
	}
	return errors.Join(errs...)
}

func checkStory(itemID string, s *Story) []error {
	var errs []error
	at := func(line int, format string, a ...any) error {
		return fmt.Errorf("%s: %s:%d: %s", itemID, s.Path, line, fmt.Sprintf(format, a...))
	}

	if s.ID != itemID {
		errs = append(errs, at(s.Line, "story ID %s does not match backlog item %s", s.ID, itemID))
	}
	if n := len(s.Criteria); n < 1 || n > maxCriteria {
		errs = append(errs, at(s.Line, "%d acceptance criteria (expected 1 to %d)", n, maxCriteria))
	}

	prefix := ACPrefix(itemID)
	for i, c := range s.Criteria {
		if want := fmt.Sprintf("%s%d", prefix, i+1); c.ID != want {
			errs = append(errs, at(c.Line, "criterion %s should be %s (numbering must be sequential from .1)", c.ID, want))
		}
		var missing []string
		for _, p := range []struct {
			name string
			ok   bool
		}{{"Given", c.Given}, {"When", c.When}, {"Then", c.Then}} {
			if !p.ok {
				missing = append(missing, p.name)
			}
		}
		if len(missing) > 0 {
			errs = append(errs, at(c.Line, "%s is missing %s", c.ID, strings.Join(missing, ", ")))
		}
	}
	return errs
}

func checkTasks(itemID string, s *Story, path string) []error {
	tk, err := ParseTasks(path)
	var errs []error
	if err != nil {
		errs = append(errs, fmt.Errorf("%s: %w", itemID, err))
	}

	known := make(map[string]bool, len(s.Criteria))
	for _, c := range s.Criteria {
		known[c.ID] = true
	}
	if len(tk.Tasks) > maxTasks {
		errs = append(errs, fmt.Errorf("%s: %s: %d tasks (maximum %d)", itemID, path, len(tk.Tasks), maxTasks))
	}
	for i, t := range tk.Tasks {
		at := func(format string, a ...any) error {
			return fmt.Errorf("%s: %s:%d: %s", itemID, path, t.Line, fmt.Sprintf(format, a...))
		}
		if t.Number != i+1 {
			errs = append(errs, at("task numbered %d, expected %d", t.Number, i+1))
		}
		if len(t.ACs) == 0 {
			errs = append(errs, at("task %d references no criterion", t.Number))
		}
		for _, ac := range t.ACs {
			if !known[ac] {
				errs = append(errs, at("task %d references %s, which is not in story.md", t.Number, ac))
			}
		}
	}
	return errs
}

// checkLocation enforces that the folder matches the status:
// ready/in-progress → storyline/active/, done → storyline/done/.
func checkLocation(itemID, status, storyPath string) []error {
	p := filepath.ToSlash(storyPath)
	var want string
	switch status {
	case "ready", "in-progress":
		want = "storyline/active/"
	case "done":
		want = "storyline/done/"
	default:
		return nil
	}
	if !strings.HasPrefix(p, want) {
		return []error{fmt.Errorf("%s: %s: status %q requires the story under %s", itemID, storyPath, status, want)}
	}
	return nil
}
