package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fjbalsamo/storyline/internal/adapters"
)

var commands = []string{"add", "refine", "slice", "apply", "verify", "done"}

func tools(t *testing.T, ids ...string) []adapters.Adapter {
	t.Helper()
	a, err := adapters.Get(ids)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func TestRunCreatesProject(t *testing.T) {
	root := t.TempDir()
	if _, err := Run(Options{Root: root, Tools: tools(t, "claude")}); err != nil {
		t.Fatalf("run: %v", err)
	}

	want := []string{
		"stories/backlog.yaml",
		"stories/active/.gitkeep",
		"stories/done/.gitkeep",
		"stories/living/.gitkeep",
	}
	for _, c := range commands {
		want = append(want, ".claude/commands/story/"+c+".md")
	}
	for _, p := range want {
		if _, err := os.Stat(filepath.Join(root, p)); err != nil {
			t.Errorf("missing %s: %v", p, err)
		}
	}
}

func TestRunIsIdempotent(t *testing.T) {
	root := t.TempDir()
	if _, err := Run(Options{Root: root, Tools: tools(t, "claude")}); err != nil {
		t.Fatal(err)
	}
	res, err := Run(Options{Root: root, Tools: tools(t, "claude")})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Created) != 0 {
		t.Fatalf("second run created files: %v", res.Created)
	}
}

func TestForceOverwritesCommandsButNotBacklog(t *testing.T) {
	root := t.TempDir()
	if _, err := Run(Options{Root: root, Tools: tools(t, "claude")}); err != nil {
		t.Fatal(err)
	}

	backlogPath := filepath.Join(root, "stories/backlog.yaml")
	cmdPath := filepath.Join(root, ".claude/commands/story/add.md")
	mustWrite(t, backlogPath, "user data")
	mustWrite(t, cmdPath, "stale")

	if _, err := Run(Options{Root: root, Force: true, Tools: tools(t, "claude")}); err != nil {
		t.Fatal(err)
	}
	if got := mustRead(t, backlogPath); got != "user data" {
		t.Errorf("backlog was overwritten: %q", got)
	}
	if got := mustRead(t, cmdPath); got == "stale" {
		t.Error("command file was not overwritten with --force")
	}
}

func mustWrite(t *testing.T, path, s string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(s), 0o644); err != nil {
		t.Fatal(err)
	}
}

func mustRead(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestAllToolsCreateFiles(t *testing.T) {
	root := t.TempDir()
	res, err := Run(Options{Root: root, Tools: tools(t, "all")})
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{
		".cursor/commands/story-refine.md",
		".github/prompts/story-refine.prompt.md",
		".gemini/commands/story/refine.toml",
		".agents/skills/story-refine/SKILL.md",
		"AGENTS.md",
	} {
		if _, err := os.Stat(filepath.Join(root, p)); err != nil {
			t.Errorf("missing %s: %v", p, err)
		}
	}
	if len(res.Updated) != 1 || res.Updated[0] != "AGENTS.md" {
		t.Errorf("Updated = %v", res.Updated)
	}
}

func TestAgentsMDNewFileHasBlock(t *testing.T) {
	root := t.TempDir()
	if _, err := Run(Options{Root: root, Tools: tools(t, "agents-md")}); err != nil {
		t.Fatal(err)
	}
	got := mustRead(t, filepath.Join(root, "AGENTS.md"))
	if !strings.HasPrefix(got, adapters.MarkerStart) || !strings.HasSuffix(got, adapters.MarkerEnd+"\n") {
		t.Errorf("unexpected AGENTS.md:\n%s", got)
	}
}

func TestAgentsMDKeepsExistingContent(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "AGENTS.md")
	mustWrite(t, path, "# My rules\n\nBe nice.")

	opts := Options{Root: root, Tools: tools(t, "agents-md")}
	if _, err := Run(opts); err != nil {
		t.Fatal(err)
	}
	first := mustRead(t, path)
	if !strings.HasPrefix(first, "# My rules\n\nBe nice.\n\n"+adapters.MarkerStart) {
		t.Errorf("existing content not preserved:\n%s", first)
	}

	res, err := Run(opts)
	if err != nil {
		t.Fatal(err)
	}
	if second := mustRead(t, path); second != first {
		t.Errorf("second run changed the file:\n%s", second)
	}
	if len(res.Updated) != 0 {
		t.Errorf("second run reported updates: %v", res.Updated)
	}
}

func TestAgentsMDReplacesOnlyTheBlock(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "AGENTS.md")
	before, after := "# Top\n\n", "\n## Bottom\nkeep me\n"
	mustWrite(t, path, before+adapters.MarkerStart+"\nstale\n"+adapters.MarkerEnd+"\n"+after)

	for _, force := range []bool{false, true} {
		if _, err := Run(Options{Root: root, Force: force, Tools: tools(t, "agents-md")}); err != nil {
			t.Fatal(err)
		}
		got := mustRead(t, path)
		if !strings.HasPrefix(got, before+adapters.MarkerStart) || !strings.HasSuffix(got, adapters.MarkerEnd+"\n"+after) {
			t.Errorf("force=%v: content outside markers changed:\n%s", force, got)
		}
		if strings.Contains(got, "stale") {
			t.Errorf("force=%v: old block not replaced", force)
		}
	}
}

func TestAgentsMDUnterminatedBlockIsAnError(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "AGENTS.md"), adapters.MarkerStart+"\nhalf")
	if _, err := Run(Options{Root: root, Tools: tools(t, "agents-md")}); err == nil {
		t.Error("expected error for start marker without end marker")
	}
}
