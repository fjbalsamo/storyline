package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fjbalsamo/storyline/internal/backlog"
	"github.com/spf13/cobra"
)

const moveHead = "# header\nversion: 1\nbacklog:\n"

// moveBlocks[1..5] are the items with the comments directly above them.
var moveBlocks = map[int]string{
	1: "  # an old idea\n  - id: US-001\n    as: a\n    want: b\n    so_that: c\n    status: discarded\n",
	2: "  # second story\n  # on two lines\n  - id: US-002\n    as: a   # who\n    want: b\n    so_that: c\n    status: todo\n",
	3: "  - id: US-003\n    as: a\n    want: b\n    so_that: c\n    status: refining\n",
	4: "  # fourth\n  - id: US-004\n    as: a\n    want: b\n    so_that: c\n    status: todo\n",
	5: "  - id: US-005\n    as: a\n    want: b\n    so_that: c\n    status: todo\n",
}

// moveGaps[i] sits between slot i and slot i+1 and never moves.
var moveGaps = []string{"", "\n", "\n", "  # detached comment\n\n", ""}

func assemble(order ...int) string {
	var sb strings.Builder
	sb.WriteString(moveHead)
	for slot, n := range order {
		sb.WriteString(moveBlocks[n])
		if slot < len(order)-1 {
			sb.WriteString(moveGaps[slot+1])
		}
	}
	return sb.String()
}

func runMoveCmd(t *testing.T, path string, args ...string) (string, error) {
	t.Helper()
	root := &cobra.Command{Use: "storyline", SilenceUsage: true, SilenceErrors: true}
	root.PersistentFlags().String("backlog", backlog.DefaultPath, "")
	root.AddCommand(newMoveCmd())
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs(append([]string{"move", "--backlog", path}, args...))
	err := root.Execute()
	return out.String(), err
}

func writeBacklog(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "storyline", "backlog.yaml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestMoveModes(t *testing.T) {
	tests := []struct {
		name  string
		args  []string
		order []int
		out   string
	}{
		{"top", []string{"US-005", "--top"}, []int{1, 5, 2, 3, 4}, "✓ US-005 moved: position 5 → 2\n"},
		{"bottom", []string{"US-002", "--bottom"}, []int{1, 3, 4, 5, 2}, "✓ US-002 moved: position 2 → 5\n"},
		{"before", []string{"US-004", "--before", "US-002"}, []int{1, 4, 2, 3, 5}, "✓ US-004 moved: position 4 → 2\n"},
		{"after", []string{"US-002", "--after", "US-004"}, []int{1, 3, 4, 2, 5}, "✓ US-002 moved: position 2 → 4\n"},
		{"reference may be discarded", []string{"US-002", "--before", "US-001"}, []int{2, 1, 3, 4, 5}, "✓ US-002 moved: position 2 → 1\n"},
		{"after a discarded story", []string{"US-003", "--after", "US-001"}, []int{1, 3, 2, 4, 5}, "✓ US-003 moved: position 3 → 2\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeBacklog(t, assemble(1, 2, 3, 4, 5))
			out, err := runMoveCmd(t, path, tt.args...)
			if err != nil {
				t.Fatal(err)
			}
			if out != tt.out {
				t.Errorf("stdout = %q, want %q", out, tt.out)
			}
			if got, want := readFile(t, path), assemble(tt.order...); got != want {
				t.Errorf("file =\n%s\nwant\n%s", got, want)
			}
		})
	}
}

func TestMoveAlreadyInPlace(t *testing.T) {
	tests := [][]string{
		{"US-002", "--top"}, // first open story; the discarded one above doesn't count
		{"US-005", "--bottom"},
		{"US-003", "--after", "US-002"},
		{"US-003", "--before", "US-004"},
	}
	for _, args := range tests {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			path := writeBacklog(t, assemble(1, 2, 3, 4, 5))
			out, err := runMoveCmd(t, path, args...)
			if err != nil || out != "nothing to change\n" {
				t.Errorf("out = %q, err = %v", out, err)
			}
			if readFile(t, path) != assemble(1, 2, 3, 4, 5) {
				t.Error("file changed")
			}
		})
	}
}

func TestMoveErrors(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{"no flag", []string{"US-002"}, "pass exactly one of --top, --bottom, --before, --after"},
		{"two flags", []string{"US-002", "--top", "--bottom"}, "pass exactly one of"},
		{"flag and reference", []string{"US-002", "--top", "--after", "US-003"}, "pass exactly one of"},
		{"empty reference", []string{"US-002", "--before", ""}, "--before needs a story id"},
		{"discarded", []string{"US-001", "--bottom"}, "discarded stories are history and can't be moved"},
		{"unknown id", []string{"US-099", "--top"}, "story US-099 not found"},
		{"unknown reference", []string{"US-002", "--after", "US-099"}, "story US-099 not found"},
		{"same id", []string{"US-002", "--before", "US-002"}, "can't move US-002 relative to itself"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeBacklog(t, assemble(1, 2, 3, 4, 5))
			_, err := runMoveCmd(t, path, tt.args...)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("err = %v, want %q", err, tt.wantErr)
			}
			if readFile(t, path) != assemble(1, 2, 3, 4, 5) {
				t.Error("file changed despite the error")
			}
		})
	}
}

func TestMoveRejectsDone(t *testing.T) {
	_, path, _ := project(t, backlog.StatusDone, "")
	before := readFile(t, path)
	_, err := runMoveCmd(t, path, "US-001", "--top")
	if err == nil || err.Error() != "done stories are history and can't be moved" {
		t.Fatalf("err = %v", err)
	}
	if readFile(t, path) != before {
		t.Error("file changed")
	}
}

func TestMoveRejectsFlowStyle(t *testing.T) {
	src := "version: 1\nbacklog:\n  - {id: US-001, as: a, want: b, so_that: c, status: todo}\n" +
		"  - {id: US-002, as: a, want: b, so_that: c, status: todo}\n"
	path := writeBacklog(t, src)
	_, err := runMoveCmd(t, path, "US-001", "--bottom")
	if err == nil || !strings.Contains(err.Error(), "flow style") {
		t.Fatalf("err = %v", err)
	}
	if readFile(t, path) != src {
		t.Error("file changed")
	}
}

func TestMoveWritesNothingWhenBacklogInvalid(t *testing.T) {
	bad := strings.Replace(assemble(1, 2, 3, 4, 5), "status: refining", "status: ready", 1) // ready without a story path
	path := writeBacklog(t, bad)
	_, err := runMoveCmd(t, path, "US-005", "--top")
	if err == nil || !strings.Contains(err.Error(), "not saved") {
		t.Fatalf("err = %v", err)
	}
	if readFile(t, path) != bad {
		t.Error("file changed despite the invalid backlog")
	}
}

func TestListShowsNewNextAfterMoveToTop(t *testing.T) {
	path := writeBacklog(t, assemble(1, 2, 3, 4, 5))
	if next := listNext(t, path); !strings.Contains(next, "US-002") {
		t.Fatalf("before: ▶ line = %q", next)
	}
	if _, err := runMoveCmd(t, path, "US-005", "--top"); err != nil {
		t.Fatal(err)
	}
	if next := listNext(t, path); !strings.Contains(next, "US-005") {
		t.Errorf("after: ▶ line = %q", next)
	}
}

// listNext runs the real list command and returns the line marked ▶.
func listNext(t *testing.T, path string) string {
	t.Helper()
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetArgs([]string{"list", "--backlog", path})
	t.Cleanup(func() { rootCmd.SetOut(nil); rootCmd.SetArgs(nil) })
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(out.String(), "\n") {
		if strings.Contains(line, "▶") {
			return line
		}
	}
	t.Fatalf("no ▶ in list output:\n%s", out.String())
	return ""
}
