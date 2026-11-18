package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fjbalsamo/storyline/internal/adapters"
)

var commands = []string{"context", "add", "refine", "slice", "apply", "verify", "done"}

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
	if !contains(res.Created, "AGENTS.md") || contains(res.Updated, "AGENTS.md") {
		t.Errorf("Created = %v, Updated = %v", res.Created, res.Updated)
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

func TestAgentsMDAlwaysCreated(t *testing.T) {
	// Even a tool with no AGENTS.md of its own gets one.
	root := t.TempDir()
	if _, err := Run(Options{Root: root, Tools: tools(t, "cursor")}); err != nil {
		t.Fatal(err)
	}
	got := mustRead(t, filepath.Join(root, "AGENTS.md"))
	if !strings.Contains(got, adapters.MarkerStart) {
		t.Errorf("AGENTS.md has no managed block:\n%s", got)
	}
}

func TestAgentsMDAliasIsNoOp(t *testing.T) {
	root := t.TempDir()
	res, err := Run(Options{Root: root, Tools: tools(t, "claude", "agents-md")})
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, p := range append(res.Created, res.Updated...) {
		if p == "AGENTS.md" {
			n++
		}
	}
	if n != 1 {
		t.Errorf("AGENTS.md reported %d times: created %v, updated %v", n, res.Created, res.Updated)
	}
}

func TestClaudeMDCreatedWithImport(t *testing.T) {
	root := t.TempDir()
	res, err := Run(Options{Root: root, Tools: tools(t, "claude")})
	if err != nil {
		t.Fatal(err)
	}
	if got := mustRead(t, filepath.Join(root, "CLAUDE.md")); got != "@AGENTS.md\n" {
		t.Errorf("CLAUDE.md = %q", got)
	}
	if !contains(res.Created, "CLAUDE.md") {
		t.Errorf("CLAUDE.md not reported as created: %v", res.Created)
	}
}

func TestExistingClaudeMDGetsImportAppended(t *testing.T) {
	for name, existing := range map[string]string{
		"newline":    "# Mine\n\nKeep this.\n",
		"no newline": "# Mine\n\nKeep this.",
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, "CLAUDE.md")
			mustWrite(t, path, existing)
			res, err := Run(Options{Root: root, Tools: tools(t, "claude")})
			if err != nil {
				t.Fatal(err)
			}
			got := mustRead(t, path)
			if !strings.HasPrefix(got, existing) || !strings.HasSuffix(got, "\n@AGENTS.md\n") {
				t.Errorf("CLAUDE.md = %q", got)
			}
			if !contains(res.Updated, "CLAUDE.md") {
				t.Errorf("CLAUDE.md not reported as updated: %v", res.Updated)
			}
		})
	}
}

func TestExistingImportIsLeftAlone(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "CLAUDE.md")
	const existing = "# Mine\n  @AGENTS.md  \nmore\n"
	mustWrite(t, path, existing)
	if _, err := Run(Options{Root: root, Tools: tools(t, "claude")}); err != nil {
		t.Fatal(err)
	}
	if got := mustRead(t, path); got != existing {
		t.Errorf("CLAUDE.md changed: %q", got)
	}
}

func TestGeminiMDPointer(t *testing.T) {
	root := t.TempDir()
	if _, err := Run(Options{Root: root, Tools: tools(t, "gemini")}); err != nil {
		t.Fatal(err)
	}
	if got := mustRead(t, filepath.Join(root, "GEMINI.md")); got != "@AGENTS.md\n" {
		t.Errorf("GEMINI.md = %q", got)
	}
	if _, err := os.Stat(filepath.Join(root, ".gemini", "settings.json")); err == nil {
		t.Error("settings.json must not be created")
	}
}

func TestNativeToolsGetNoPointerFile(t *testing.T) {
	root := t.TempDir()
	if _, err := Run(Options{Root: root, Tools: tools(t, "cursor", "copilot", "codex")}); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{"CLAUDE.md", "GEMINI.md", ".github/copilot-instructions.md"} {
		if _, err := os.Stat(filepath.Join(root, p)); err == nil {
			t.Errorf("%s should not exist", p)
		}
	}
}

func TestRerunIsByteIdentical(t *testing.T) {
	root := t.TempDir()
	opts := Options{Root: root, Tools: tools(t, "all")}
	mustWrite(t, filepath.Join(root, "CLAUDE.md"), "# Mine\n")
	if _, err := Run(opts); err != nil {
		t.Fatal(err)
	}
	before := snapshot(t, root)
	res, err := Run(opts)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Created)+len(res.Updated) != 0 {
		t.Errorf("rerun changed files: created %v, updated %v", res.Created, res.Updated)
	}
	after := snapshot(t, root)
	for p, content := range before {
		if after[p] != content {
			t.Errorf("%s changed on rerun", p)
		}
	}
	if len(after) != len(before) {
		t.Errorf("file count changed: %d -> %d", len(before), len(after))
	}
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// snapshot maps every file under root to its content.
func snapshot(t *testing.T, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := os.ReadFile(p)
		out[p] = string(b)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestAgentsMDCreatedVsUpdated(t *testing.T) {
	t.Run("missing is created", func(t *testing.T) {
		res, err := Run(Options{Root: t.TempDir(), Tools: tools(t, "cursor")})
		if err != nil {
			t.Fatal(err)
		}
		if !contains(res.Created, "AGENTS.md") || contains(res.Updated, "AGENTS.md") {
			t.Errorf("Created = %v, Updated = %v", res.Created, res.Updated)
		}
	})
	t.Run("existing is updated", func(t *testing.T) {
		root := t.TempDir()
		mustWrite(t, filepath.Join(root, "AGENTS.md"), "# Mine\n")
		res, err := Run(Options{Root: root, Tools: tools(t, "cursor")})
		if err != nil {
			t.Fatal(err)
		}
		if !contains(res.Updated, "AGENTS.md") || contains(res.Created, "AGENTS.md") {
			t.Errorf("Created = %v, Updated = %v", res.Created, res.Updated)
		}
	})
}
