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
	want := []string{"add", "refine", "slice", "apply", "verify", "done"}
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
