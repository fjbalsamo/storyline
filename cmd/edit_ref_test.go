package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fjbalsamo/storyline/internal/backlog"
)

const refFixture = `# header
version: 1
backlog:
  # first
  - id: US-001
    as: a
    want: b
    so_that: c   # why
    status: todo

  - id: US-002
    as: a
    want: b
    so_that: c
    ref: TKT-001   # origin
    status: todo
`

func refFixturePath(t *testing.T) string { return writeBacklog(t, refFixture) }

func TestEditRefAddsChangesAndRemoves(t *testing.T) {
	tests := []struct {
		name string
		args []string
		out  string
		file string
	}{
		{"add", []string{"US-001", "--ref", " TKT-9 "}, "✓ US-001 updated: ref\n",
			strings.Replace(refFixture, "    so_that: c   # why\n", "    so_that: c   # why\n    ref: TKT-9\n", 1)},
		{"change", []string{"US-002", "--ref", "TKT-2"}, "✓ US-002 updated: ref\n",
			strings.Replace(refFixture, "TKT-001", "TKT-2", 1)},
		{"remove", []string{"US-002", "--remove-ref"}, "✓ US-002 updated: ref\n",
			strings.Replace(refFixture, "    ref: TKT-001   # origin\n", "", 1)},
		{"remove when absent", []string{"US-001", "--remove-ref"}, "nothing to change\n", refFixture},
		{"same value", []string{"US-002", "--ref", "TKT-001"}, "nothing to change\n", refFixture},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := refFixturePath(t)
			out, _, err := runEditCmd(t, path, tt.args...)
			if err != nil {
				t.Fatal(err)
			}
			if out != tt.out {
				t.Errorf("stdout = %q, want %q", out, tt.out)
			}
			if got := readFile(t, path); got != tt.file {
				t.Errorf("file:\n%s\nwant:\n%s", got, tt.file)
			}
		})
	}
}

func TestEditRefRejects(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{"both flags", []string{"US-001", "--ref", "X-1", "--remove-ref"}, "--ref and --remove-ref can't be used together"},
		{"empty", []string{"US-001", "--ref", ""}, "--ref must not be empty"},
		{"blank", []string{"US-001", "--ref", "   "}, "--ref must not be empty"},
		{"multi-line", []string{"US-001", "--ref", "A\nB"}, "--ref must be a single line"},
		{"no flags", []string{"US-001"}, "at least one of --as, --want, --so-that, --ref, --remove-ref"},
		{"unknown id", []string{"US-099", "--ref", "X"}, "story US-099 not found"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := refFixturePath(t)
			_, _, err := runEditCmd(t, path, tt.args...)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("err = %v, want %q", err, tt.wantErr)
			}
			if readFile(t, path) != refFixture {
				t.Error("file changed despite the error")
			}
		})
	}
}

// historyProject has US-001 done (with its story in storyline/done) and US-002 discarded.
func historyProject(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "storyline/done/2026-11-19-US-001-x")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "story.md"), []byte(storyMD), 0o644); err != nil {
		t.Fatal(err)
	}
	b := &backlog.Backlog{Version: 1, Items: []backlog.Item{
		{ID: "US-001", As: "a", Want: "b", SoThat: "c", Status: backlog.StatusDone, Story: "storyline/done/2026-11-19-US-001-x/story.md"},
		{ID: "US-002", As: "a", Want: "b", SoThat: "c", Ref: "OLD-1", Status: backlog.StatusDiscarded},
	}}
	path := filepath.Join(root, "storyline/backlog.yaml")
	if err := b.Save(path); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestEditRefWorksOnDoneAndDiscarded(t *testing.T) {
	path := historyProject(t)
	if out, _, err := runEditCmd(t, path, "US-001", "--ref", "TKT-5"); err != nil || out != "✓ US-001 updated: ref\n" {
		t.Fatalf("done: out = %q, err = %v", out, err)
	}
	if out, _, err := runEditCmd(t, path, "US-002", "--remove-ref"); err != nil || out != "✓ US-002 updated: ref\n" {
		t.Fatalf("discarded: out = %q, err = %v", out, err)
	}
	b, err := backlog.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if b.Items[0].Ref != "TKT-5" || b.Items[1].Ref != "" {
		t.Errorf("items = %+v", b.Items)
	}
}

func TestEditWordingOfHistoryStaysRejected(t *testing.T) {
	path := historyProject(t)
	before := readFile(t, path)
	for _, args := range [][]string{
		{"US-001", "--want", "x"},
		{"US-001", "--want", "x", "--ref", "TKT-5"}, // the wording part blocks the whole edit
		{"US-002", "--as", "x", "--remove-ref"},
	} {
		_, _, err := runEditCmd(t, path, args...)
		if err == nil || !strings.Contains(err.Error(), "stories are history and can't be edited") {
			t.Errorf("%v: err = %v", args, err)
		}
		if readFile(t, path) != before {
			t.Errorf("%v: file changed", args)
		}
	}
}

func TestEditRefWarnsOnlyWhenTheWordingChanges(t *testing.T) {
	_, path, _ := project(t, backlog.StatusReady, "")

	_, errOut, err := runEditCmd(t, path, "US-001", "--ref", "TKT-1")
	if err != nil || errOut != "" {
		t.Errorf("ref only: err = %v, stderr = %q", err, errOut)
	}
	out, errOut, err := runEditCmd(t, path, "US-001", "--want", "new wording", "--ref", "TKT-2")
	if err != nil || out != "✓ US-001 updated: want, ref\n" || !strings.Contains(errOut, "story.md may need the same change") {
		t.Errorf("want+ref: out = %q, stderr = %q, err = %v", out, errOut, err)
	}
}

func TestEditRefIsSeenByList(t *testing.T) {
	path := refFixturePath(t)
	if _, _, err := runEditCmd(t, path, "US-001", "--ref", "TKT-001"); err != nil {
		t.Fatal(err)
	}
	out := runRoot(t, "list", "--ref", "TKT-001", "--backlog", path)
	if !strings.Contains(out, "US-001") || !strings.Contains(out, "US-002") {
		t.Errorf("both stories expected under TKT-001:\n%s", out)
	}
}

func TestEditStillHasNoStatusIDOrStoryFlags(t *testing.T) {
	for _, flag := range []string{"--status", "--id", "--story"} {
		path := refFixturePath(t)
		_, _, err := runEditCmd(t, path, "US-001", "--ref", "X", flag, "v")
		if err == nil || !strings.Contains(err.Error(), "unknown flag: "+flag) {
			t.Errorf("%s: err = %v", flag, err)
		}
		if readFile(t, path) != refFixture {
			t.Errorf("%s: file changed", flag)
		}
	}
}
