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

const editFixture = `# yaml-language-server: $schema=https://example.test/schema.json
# top-level note

version: 1
backlog:
  # the first story
  - id: US-001
    as: "club player"   # who
    want: 'to book a court'
    so_that: the court is mine
    status: todo

  # the second story
  - id: US-002
    as: admin
    want: to see bookings
    so_that: I can plan   # why
    status: refining
`

// runEditCmd executes a fresh edit command the way the CLI would.
func runEditCmd(t *testing.T, path string, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	root := &cobra.Command{Use: "storyline", SilenceUsage: true, SilenceErrors: true}
	root.PersistentFlags().String("backlog", backlog.DefaultPath, "")
	root.AddCommand(newEditCmd())
	var out, errOut bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errOut)
	root.SetArgs(append([]string{"edit", "--backlog", path}, args...))
	err = root.Execute()
	return out.String(), errOut.String(), err
}

func writeEditFixture(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "storyline", "backlog.yaml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(editFixture), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestEditFields(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantOut string
		replace [][2]string // edits expected in the file text
	}{
		{"as", []string{"US-001", "--as", "regular"}, "✓ US-001 updated: as\n",
			[][2]string{{`as: "club player"   # who`, `as: regular   # who`}}},
		{"want", []string{"US-001", "--want", "to cancel"}, "✓ US-001 updated: want\n",
			[][2]string{{`want: 'to book a court'`, `want: to cancel`}}},
		{"so-that", []string{"US-001", "--so-that", "it is free"}, "✓ US-001 updated: so_that\n",
			[][2]string{{`so_that: the court is mine`, `so_that: it is free`}}},
		{"all three", []string{"US-001", "--so-that", "s", "--as", "a", "--want", "w"}, "✓ US-001 updated: as, want, so_that\n",
			[][2]string{
				{`as: "club player"   # who`, `as: a   # who`},
				{`want: 'to book a court'`, `want: w`},
				{`so_that: the court is mine`, `so_that: s`},
			}},
		{"values are trimmed", []string{"US-001", "--as", "  regular  "}, "✓ US-001 updated: as\n",
			[][2]string{{`as: "club player"   # who`, `as: regular   # who`}}},
		{"only changed fields are listed", []string{"US-002", "--as", "admin", "--want", "to plan"}, "✓ US-002 updated: want\n",
			[][2]string{{`want: to see bookings`, `want: to plan`}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeEditFixture(t)
			out, _, err := runEditCmd(t, path, tt.args...)
			if err != nil {
				t.Fatal(err)
			}
			if out != tt.wantOut {
				t.Errorf("stdout = %q, want %q", out, tt.wantOut)
			}
			want := editFixture
			for _, r := range tt.replace {
				if !strings.Contains(want, r[0]) {
					t.Fatalf("bad test: %q not in fixture", r[0])
				}
				want = strings.Replace(want, r[0], r[1], 1)
			}
			if got := readFile(t, path); got != want {
				t.Errorf("file differs outside the edited lines:\n%s", got)
			}
		})
	}
}

func TestEditRejectsAndLeavesFileAlone(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{"no flags", []string{"US-001"}, "at least one of --as, --want, --so-that"},
		{"empty value", []string{"US-001", "--want", ""}, "--want must not be empty"},
		{"blank value", []string{"US-001", "--so-that", "   "}, "--so-that must not be empty"},
		{"multi-line value", []string{"US-001", "--as", "a\nb"}, "--as must be a single line"},
		{"status flag", []string{"US-001", "--status", "done"}, "unknown flag: --status"},
		{"id flag", []string{"US-001", "--id", "US-009"}, "unknown flag: --id"},
		{"story flag", []string{"US-001", "--story", "x.md"}, "unknown flag: --story"},
		{"unknown id", []string{"US-099", "--as", "x"}, "story US-099 not found"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeEditFixture(t)
			_, _, err := runEditCmd(t, path, tt.args...)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("err = %v, want %q", err, tt.wantErr)
			}
			if got := readFile(t, path); got != editFixture {
				t.Error("file changed despite the error")
			}
		})
	}
}

func TestEditByStatus(t *testing.T) {
	tests := []struct {
		status   backlog.Status
		wantErr  string
		wantWarn bool
	}{
		{backlog.StatusTodo, "", false},
		{backlog.StatusRefining, "", false},
		{backlog.StatusReady, "", true},
		{backlog.StatusInProgress, "", true},
		{backlog.StatusDone, "done stories are history and can't be edited", false},
		{backlog.StatusDiscarded, "discarded stories are history and can't be edited", false},
	}
	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			_, path, _ := project(t, tt.status, "")
			before := readFile(t, path)
			out, errOut, err := runEditCmd(t, path, "US-001", "--want", "something new")

			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Fatalf("err = %v, want %q", err, tt.wantErr)
				}
				if readFile(t, path) != before {
					t.Error("file changed")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(out, "✓ US-001 updated: want") {
				t.Errorf("stdout = %q", out)
			}
			if warned := strings.Contains(errOut, "story.md may need the same change"); warned != tt.wantWarn {
				t.Errorf("warning = %v (stderr %q), want %v", warned, errOut, tt.wantWarn)
			}
			b, err := backlog.Load(path)
			if err != nil {
				t.Fatal(err)
			}
			if b.Items[0].Want != "something new" || b.Items[0].Status != tt.status {
				t.Errorf("saved item = %+v", b.Items[0])
			}
		})
	}
}

func TestEditNothingToChange(t *testing.T) {
	path := writeEditFixture(t)
	out, errOut, err := runEditCmd(t, path, "US-001", "--as", "club player", "--want", " to book a court ")
	if err != nil {
		t.Fatal(err)
	}
	if out != "nothing to change\n" || errOut != "" {
		t.Errorf("stdout = %q, stderr = %q", out, errOut)
	}
	if readFile(t, path) != editFixture {
		t.Error("file changed")
	}
}

func TestEditNothingToChangeDoesNotWarn(t *testing.T) {
	_, path, _ := project(t, backlog.StatusReady, "")
	_, errOut, err := runEditCmd(t, path, "US-001", "--as", "a")
	if err != nil || errOut != "" {
		t.Errorf("err = %v, stderr = %q", err, errOut)
	}
}

func TestEditWritesNothingWhenBacklogInvalid(t *testing.T) {
	path := writeEditFixture(t)
	bad := strings.Replace(editFixture, "status: refining", "status: ready", 1) // ready without a story path
	if err := os.WriteFile(path, []byte(bad), 0o644); err != nil {
		t.Fatal(err)
	}
	_, _, err := runEditCmd(t, path, "US-001", "--as", "regular")
	if err == nil || !strings.Contains(err.Error(), "not saved") {
		t.Fatalf("err = %v", err)
	}
	if readFile(t, path) != bad {
		t.Error("file changed despite the invalid backlog")
	}
}

func TestEditQuotesValuesThatNeedIt(t *testing.T) {
	path := writeEditFixture(t)
	const tricky = `to: say "hi" # not a comment`
	if _, _, err := runEditCmd(t, path, "US-001", "--want", tricky); err != nil {
		t.Fatal(err)
	}
	b, err := backlog.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if b.Items[0].Want != tricky {
		t.Errorf("want = %q", b.Items[0].Want)
	}
}
