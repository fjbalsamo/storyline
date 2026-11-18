package adapters

import (
	"fmt"
	"strings"

	"github.com/fjbalsamo/storyline/internal/templates"
)

// claude renders Claude Code custom commands. A subdirectory under
// .claude/commands becomes a namespace, so story/add.md is /story:add.
// Docs: https://code.claude.com/docs/en/slash-commands
//
// Claude Code reads AGENTS.md only when there is no CLAUDE.md, so it also gets
// a CLAUDE.md that imports it.
// Docs: https://code.claude.com/docs/en/memory#agents-md
type claude struct{}

func (claude) ID() string   { return "claude" }
func (claude) Name() string { return "Claude Code" }

func (claude) Render(cmds []templates.Command) ([]File, error) {
	files := make([]File, 0, len(cmds))
	for _, c := range cmds {
		var b strings.Builder
		fmt.Fprintf(&b, "---\ndescription: %s\nargument-hint: %s\n---\n\n", yamlString(c.Description), yamlString(c.ArgumentHint))
		b.WriteString(fill(c.Body, "$ARGUMENTS"))
		files = append(files, File{Path: ".claude/commands/story/" + c.Name + ".md", Content: []byte(b.String())})
	}
	files = append(files, File{
		Path:    "CLAUDE.md",
		Content: []byte(AgentsImport + "\n"),
		Import:  AgentsImport,
	})
	return files, nil
}
