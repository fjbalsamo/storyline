package cmd

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fjbalsamo/storyline/internal/backlog"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var updateGolden = flag.Bool("update-golden", false, "regenerate cmd/testdata golden files")

// runRoot executes the real CLI (the global command tree) and returns its output.
func runRoot(t *testing.T, args ...string) string {
	t.Helper()
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs(args)
	t.Cleanup(func() { rootCmd.SetOut(nil); rootCmd.SetErr(nil); rootCmd.SetArgs(nil) })
	defer resetFlags(rootCmd)
	if err := rootCmd.Execute(); err != nil {
		out.WriteString("error: " + err.Error() + "\n")
	}
	return out.String()
}

// resetFlags puts every flag of the command tree back to its default, because
// the global commands keep flag values between Execute calls.
func resetFlags(c *cobra.Command) {
	reset := func(f *pflag.Flag) {
		_ = f.Value.Set(f.DefValue)
		f.Changed = false
	}
	c.Flags().VisitAll(reset)
	c.PersistentFlags().VisitAll(reset)
	for _, sub := range c.Commands() {
		resetFlags(sub)
	}
}

// TestAddAndListWithoutRefAreUnchanged pins the exact output and file bytes of
// add and list for backlogs that never use ref, so adding the optional field
// can't change what existing users see.
func TestAddAndListWithoutRefAreUnchanged(t *testing.T) {
	path := filepath.Join(t.TempDir(), "storyline", "backlog.yaml")
	seed := &backlog.Backlog{Version: 1, Items: []backlog.Item{
		{ID: "US-001", As: "club player", Want: "to book a court", SoThat: "I can play", Status: backlog.StatusDone, Story: "storyline/done/x/story.md"},
		{ID: "US-002", As: "admin", Want: "to see bookings", SoThat: "I can plan", Status: backlog.StatusTodo},
		{ID: "US-003", As: "coach", Want: "to cancel a class", SoThat: "players are told", Status: backlog.StatusInProgress, Story: "storyline/active/y/story.md"},
		{ID: "US-004", As: "owner", Want: "to close the club", SoThat: "nobody turns up", Status: backlog.StatusDiscarded},
	}}
	if err := seed.Save(path); err != nil {
		t.Fatal(err)
	}

	var got strings.Builder
	step := func(title string, args ...string) {
		got.WriteString("$ storyline " + title + "\n")
		got.WriteString(runRoot(t, append(args, "--backlog", path)...))
	}
	step("list", "list")
	step("list -a", "list", "-a")
	step("list -s todo", "list", "-s", "todo")
	step("list -s done", "list", "-s", "done")
	step("list -s nope", "list", "-s", "nope")
	step("add (1)", "add", "--as", "referee", "--want", "to log a result", "--so-that", "scores are fair")
	step("add (2, padded)", "add", "--as", "  fan ", "--want", " to follow a match", "--so-that", "I don't miss it ")
	step("add (empty)", "add", "--as", "", "--want", "x", "--so-that", "y")
	step("list after adds", "list")
	step("list -a after adds", "list", "-a")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got.WriteString("$ cat storyline/backlog.yaml\n" + string(raw))

	golden := filepath.Join("testdata", "add_list_no_ref.golden")
	if *updateGolden {
		if err := os.WriteFile(golden, []byte(got.String()), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("read golden (run with -update-golden): %v", err)
	}
	if got.String() != string(want) {
		t.Errorf("output changed:\n--- got ---\n%s\n--- want ---\n%s", got.String(), want)
	}
}
