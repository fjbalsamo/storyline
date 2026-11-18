package adapters

import (
	"fmt"
	"strings"

	"github.com/fjbalsamo/storyline/internal/templates"
)

// claude renders Claude Code custom commands. A subdirectory under
// .claude/commands becomes a namespace, so story/add.md is /story:add.
// Docs: https://code.claude.com/docs/en/slash-commands
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
	return files, nil
}
