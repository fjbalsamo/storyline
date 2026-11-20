package cmd

import (
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/fjbalsamo/storyline/internal/backlog"
)

var refLine = regexp.MustCompile(`(?m)^\s*ref\s*:`)

// TestStoriesWithoutRefNeverWriteARefLine pins that ref is optional: a story
// without one has no `ref:` line (never `ref: ""`, never an empty value), whichever
// command last rewrote the file.
func TestStoriesWithoutRefNeverWriteARefLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), "storyline", "backlog.yaml")
	noRef := func(step string) {
		t.Helper()
		if raw := readFile(t, path); refLine.MatchString(raw) {
			t.Fatalf("after %s the file has a ref line:\n%s", step, raw)
		}
	}

	runRoot(t, "add", "--as", "a", "--want", "first", "--so-that", "c", "--backlog", path)
	noRef("add")
	runRoot(t, "add", "--as", "a", "--want", "second", "--so-that", "c", "--backlog", path)
	noRef("second add")

	runRoot(t, "status", "US-001", "refining", "--backlog", path) // re-encodes the whole file
	noRef("status")
	if _, _, err := runEditCmd(t, path, "US-002", "--want", "reworded"); err != nil {
		t.Fatal(err)
	}
	noRef("edit")
	if _, err := runMoveCmd(t, path, "US-002", "--top"); err != nil {
		t.Fatal(err)
	}
	noRef("move")
	if out, _, err := runEditCmd(t, path, "US-001", "--remove-ref"); err != nil || out != "nothing to change\n" {
		t.Fatalf("remove-ref without a ref: out = %q, err = %v", out, err)
	}
	noRef("edit --remove-ref on a story without ref")
}

func TestRemovingARefDeletesTheLineInsteadOfBlankingIt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "storyline", "backlog.yaml")
	runRoot(t, "add", "--as", "a", "--want", "w", "--so-that", "c", "--ref", "TKT-1", "--backlog", path)
	if !refLine.MatchString(readFile(t, path)) {
		t.Fatal("setup: the ref should be there")
	}
	if _, _, err := runEditCmd(t, path, "US-001", "--remove-ref"); err != nil {
		t.Fatal(err)
	}
	raw := readFile(t, path)
	if refLine.MatchString(raw) || strings.Contains(raw, "TKT-1") {
		t.Errorf("the ref line must be gone, not emptied:\n%s", raw)
	}
	// a later full re-encode (status) must not bring it back as `ref: ""`
	runRoot(t, "status", "US-001", "refining", "--backlog", path)
	if refLine.MatchString(readFile(t, path)) {
		t.Errorf("status re-encoded an empty ref:\n%s", readFile(t, path))
	}
}

func TestOnlyStoriesWithARefGetTheLine(t *testing.T) {
	b := backlog.New()
	b.AddWithRef("a", "w", "s", "TKT-1")
	b.AddWithRef("a", "w", "s", "")
	path := filepath.Join(t.TempDir(), "storyline", "backlog.yaml")
	if err := b.Save(path); err != nil {
		t.Fatal(err)
	}
	if n := len(refLine.FindAllString(readFile(t, path), -1)); n != 1 {
		t.Errorf("%d ref lines, want exactly 1:\n%s", n, readFile(t, path))
	}
}
