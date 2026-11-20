package adapters

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/fjbalsamo/storyline/internal/templates"
)

var update = flag.Bool("update", false, "regenerate golden files")

func commands(t *testing.T) []templates.Command {
	t.Helper()
	cmds, err := templates.Commands()
	if err != nil {
		t.Fatal(err)
	}
	return cmds
}

func TestGolden(t *testing.T) {
	cmds := commands(t)
	for _, a := range All() {
		t.Run(a.ID(), func(t *testing.T) {
			files, err := a.Render(cmds)
			if err != nil {
				t.Fatal(err)
			}
			if len(files) == 0 {
				t.Fatal("no files rendered")
			}
			for _, f := range files {
				if bytes.Contains(f.Content, []byte(templates.ArgsPlaceholder)) {
					t.Errorf("%s still contains %s", f.Path, templates.ArgsPlaceholder)
				}
				golden := filepath.Join("testdata", "golden", a.ID(), filepath.FromSlash(f.Path))
				if *update {
					if err := os.MkdirAll(filepath.Dir(golden), 0o755); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(golden, f.Content, 0o644); err != nil {
						t.Fatal(err)
					}
					continue
				}
				want, err := os.ReadFile(golden)
				if err != nil {
					t.Fatalf("read golden (run with -update): %v", err)
				}
				if !bytes.Equal(want, f.Content) {
					t.Errorf("%s differs from %s (run with -update)", f.Path, golden)
				}
			}
		})
	}
}

func TestOnlyAgentsMDMerges(t *testing.T) {
	for _, a := range All() {
		files, err := a.Render(commands(t))
		if err != nil {
			t.Fatal(err)
		}
		for _, f := range files {
			if f.Merge != (a.ID() == "agents-md") {
				t.Errorf("%s: Merge = %v", f.Path, f.Merge)
			}
		}
	}
}

func TestGeminiTOMLParses(t *testing.T) {
	cmds := commands(t)
	// Hostile input: backslashes, quotes and a closing delimiter.
	cmds = append(cmds, templates.Command{
		Name:         "evil",
		Description:  `say "hi" \ there`,
		ArgumentHint: "x",
		Body:         "a \\ b \"\"\" c \"\"\"\" {{ARGS}} \"\"",
	})
	files, err := gemini{}.Render(cmds)
	if err != nil {
		t.Fatal(err)
	}
	for i, f := range files {
		if f.Import != "" { // pointer file, not a command
			continue
		}
		var got struct{ Description, Prompt string }
		if _, err := toml.Decode(string(f.Content), &got); err != nil {
			t.Fatalf("%s: invalid TOML: %v", f.Path, err)
		}
		if !strings.Contains(got.Prompt, "{{args}}") {
			t.Errorf("%s: prompt lost {{args}}", f.Path)
		}
		if got.Description != cmds[i].Description {
			t.Errorf("%s: description = %q, want %q", f.Path, got.Description, cmds[i].Description)
		}
	}
}

func TestGeminiRejectsInjection(t *testing.T) {
	_, err := gemini{}.Render([]templates.Command{{Name: "x", Body: "run !{rm -rf /}"}})
	if err == nil {
		t.Error("expected error for !{ in body")
	}
}

func TestGet(t *testing.T) {
	all, err := Get([]string{"all"})
	if err != nil || len(all) != len(All()) {
		t.Fatalf("all: %d adapters, err %v", len(all), err)
	}
	two, err := Get([]string{"claude", " Gemini ", "claude"})
	if err != nil || len(two) != 2 || two[0].ID() != "claude" || two[1].ID() != "gemini" {
		t.Fatalf("list: %v, err %v", two, err)
	}
	_, err = Get([]string{"foo"})
	if err == nil {
		t.Fatal("expected error for unknown id")
	}
	for _, id := range append(IDs(), "all") {
		if !strings.Contains(err.Error(), id) {
			t.Errorf("error does not list %q: %v", id, err)
		}
	}
}

// Tools that do not read AGENTS.md natively get a pointer file; the others
// must not. Pointer files are never managed blocks.
func TestPointerFiles(t *testing.T) {
	want := map[string]string{"claude": "CLAUDE.md", "gemini": "GEMINI.md"}
	for _, a := range All() {
		files, err := a.Render(commands(t))
		if err != nil {
			t.Fatal(err)
		}
		var got string
		for _, f := range files {
			if f.Import == "" {
				continue
			}
			got = f.Path
			if f.Merge || !strings.Contains(string(f.Content), f.Import) {
				t.Errorf("%s: bad pointer file %+v", a.ID(), f)
			}
		}
		if got != want[a.ID()] {
			t.Errorf("%s: pointer file = %q, want %q", a.ID(), got, want[a.ID()])
		}
	}
}

func TestAgentsMDMentionsRelatedRepos(t *testing.T) {
	files, err := AgentsMD().Render(commands(t))
	if err != nil {
		t.Fatal(err)
	}
	body := string(files[0].Content)
	for _, want := range []string{"storyline/config.yaml", "spec.dependsOn", "never blocks"} {
		if !strings.Contains(body, want) {
			t.Errorf("AGENTS.md block lacks %q", want)
		}
	}
}
