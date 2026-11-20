package templates

import (
	"strings"
	"testing"
)

func TestCommands(t *testing.T) {
	cmds, err := Commands()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"context", "add", "refine", "slice", "apply", "verify", "done"}
	if len(cmds) != len(want) {
		t.Fatalf("got %d commands", len(cmds))
	}
	for i, c := range cmds {
		if c.Name != want[i] {
			t.Errorf("order[%d] = %s, want %s", i, c.Name, want[i])
		}
		if !strings.Contains(c.Body, ArgsPlaceholder) {
			t.Errorf("%s: body has no %s", c.Name, ArgsPlaceholder)
		}
		if strings.Contains(c.Body, "$ARGUMENTS") {
			t.Errorf("%s: tool-specific placeholder left", c.Name)
		}
	}
}

func TestParseMissingField(t *testing.T) {
	if _, err := parse("x", []byte("---\ndescription: d\n---\nbody")); err == nil {
		t.Error("expected error for missing argument_hint")
	}
}

// TestContextKeepsItsGuardrails stops future edits from dropping the rules
// that keep AGENTS.md short, durable and free of feature details.
func TestContextKeepsItsGuardrails(t *testing.T) {
	cmds, err := Commands()
	if err != nil {
		t.Fatal(err)
	}
	body := cmds[0].Body
	for _, want := range []string{
		"What must NOT go in",
		"next ten stories",
		"### Do",
		"### Don't",
		"under 60 lines",
		"storyline/living/",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("context body lost %q", want)
		}
	}
}

func commandBody(t *testing.T, name string) string {
	t.Helper()
	cmds, err := Commands()
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range cmds {
		if c.Name == name {
			return c.Body
		}
	}
	t.Fatalf("no command %q", name)
	return ""
}

func TestRelatedReposInRefineAndSlice(t *testing.T) {
	for _, name := range []string{"refine", "slice"} {
		body := commandBody(t, name)
		for _, want := range []string{
			"storyline/config.yaml",
			"dependsOn",
			"ready-to-paste `/story:add",
			"Don't wait for an answer",
			"External dependencies",
			"storyline config add-dependency",
		} {
			if !strings.Contains(body, want) {
				t.Errorf("%s lacks %q", name, want)
			}
		}
	}
	if !strings.Contains(commandBody(t, "slice"), "mock") {
		t.Error("slice does not plan a mock")
	}
}

func TestApplyUsesTheMock(t *testing.T) {
	body := commandBody(t, "apply")
	for _, want := range []string{"mock", "assumed contract", "Never block"} {
		if !strings.Contains(body, want) {
			t.Errorf("apply lacks %q", want)
		}
	}
}
