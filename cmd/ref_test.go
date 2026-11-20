package cmd

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/fjbalsamo/storyline/internal/backlog"
)

func refProject(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "storyline", "backlog.yaml")
	b := &backlog.Backlog{Version: 1, Items: []backlog.Item{
		{ID: "US-001", As: "a", Want: "first", SoThat: "c", Ref: "TKT-001", Status: backlog.StatusDone, Story: "storyline/done/x/story.md"},
		{ID: "US-002", As: "a", Want: "second", SoThat: "c", Ref: "TKT-001", Status: backlog.StatusTodo},
		{ID: "US-003", As: "a", Want: "third", SoThat: "c", Ref: "TKT-002", Status: backlog.StatusTodo},
		{ID: "US-004", As: "a", Want: "fourth", SoThat: "c", Status: backlog.StatusTodo},
	}}
	if err := b.Save(path); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestAddStoresRef(t *testing.T) {
	path := refProject(t)
	out := runRoot(t, "add", "--as", "a", "--want", "w", "--so-that", "s", "--ref", "  TKT-009 ", "--backlog", path)
	if !strings.Contains(out, "✓ added US-005") {
		t.Fatalf("out = %q", out)
	}
	b, err := backlog.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := b.Items[4].Ref; got != "TKT-009" {
		t.Errorf("ref = %q", got)
	}
}

func TestAddWithoutRefLeavesTheFieldOut(t *testing.T) {
	path := filepath.Join(t.TempDir(), "storyline", "backlog.yaml")
	runRoot(t, "add", "--as", "a", "--want", "w", "--so-that", "s", "--backlog", path)
	if raw := readFile(t, path); strings.Contains(raw, "ref") {
		t.Errorf("file mentions ref:\n%s", raw)
	}
}

func TestAddRejectsBadRef(t *testing.T) {
	for _, tt := range []struct{ ref, want string }{
		{"", "--ref must not be empty"},
		{"   ", "--ref must not be empty"},
		{"A\nB", "--ref must be a single line"},
	} {
		path := filepath.Join(t.TempDir(), "storyline", "backlog.yaml")
		out := runRoot(t, "add", "--as", "a", "--want", "w", "--so-that", "s", "--ref", tt.ref, "--backlog", path)
		if !strings.Contains(out, tt.want) {
			t.Errorf("ref %q: out = %q", tt.ref, out)
		}
	}
}

func TestListShowsRefColumnOnlyWhenNeeded(t *testing.T) {
	path := refProject(t)
	out := runRoot(t, "list", "--backlog", path)
	if !strings.Contains(out, "REF") || !strings.Contains(out, "TKT-001") || !strings.Contains(out, "TKT-002") {
		t.Errorf("list should show REF:\n%s", out)
	}
	if strings.Contains(out, "US-001") {
		t.Errorf("done story listed without -a:\n%s", out)
	}

	// only the story without a ref is listed: no column
	out = runRoot(t, "list", "-s", "todo", "--backlog", path)
	if !strings.Contains(out, "REF") {
		t.Fatalf("todo stories include refs:\n%s", out)
	}
	only := filepath.Join(t.TempDir(), "storyline", "backlog.yaml")
	runRoot(t, "add", "--as", "a", "--want", "w", "--so-that", "s", "--backlog", only)
	if out := runRoot(t, "list", "--backlog", only); strings.Contains(out, "REF") {
		t.Errorf("no story has a ref, yet:\n%s", out)
	}
}

func TestListRefFilterIncludesDoneStories(t *testing.T) {
	path := refProject(t)
	out := runRoot(t, "list", "--ref", "TKT-001", "--backlog", path)
	if !strings.Contains(out, "US-001") || !strings.Contains(out, "US-002") {
		t.Errorf("both stories of TKT-001 expected:\n%s", out)
	}
	if strings.Contains(out, "US-003") || strings.Contains(out, "US-004") {
		t.Errorf("other stories listed:\n%s", out)
	}
	if !strings.Contains(runRoot(t, "list", "--ref", "TKT-001", "-s", "done", "--backlog", path), "US-001") {
		t.Error("-s done should keep US-001")
	}
	if out := runRoot(t, "list", "--ref", "TKT-001", "-s", "done", "--backlog", path); strings.Contains(out, "US-002") {
		t.Errorf("-s done should drop US-002:\n%s", out)
	}
}

func TestListUnknownRef(t *testing.T) {
	out := runRoot(t, "list", "--ref", "NOPE-1", "--backlog", refProject(t))
	if out != "No stories to show.\n" {
		t.Errorf("out = %q", out)
	}
}

func TestEditAndMoveKeepTheRef(t *testing.T) {
	path := refProject(t)
	// edit and move validate the whole backlog, so the done story needs no file
	b0, err := backlog.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	b0.Items[0].Status, b0.Items[0].Story = backlog.StatusDiscarded, ""
	if err := b0.Save(path); err != nil {
		t.Fatal(err)
	}
	if _, _, err := runEditCmd(t, path, "US-002", "--want", "changed"); err != nil {
		t.Fatal(err)
	}
	if _, err := runMoveCmd(t, path, "US-003", "--top"); err != nil {
		t.Fatal(err)
	}
	b, err := backlog.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if b.Items[1].ID != "US-003" || b.Items[1].Ref != "TKT-002" || b.Items[2].Ref != "TKT-001" || b.Items[2].Want != "changed" {
		t.Errorf("items = %+v", b.Items)
	}
}
