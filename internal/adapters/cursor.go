package adapters

import (
	"fmt"
	"strings"

	"github.com/fjbalsamo/storyline/internal/templates"
)

// cursor renders Cursor commands: plain Markdown with no frontmatter, invoked
// as /story-<name>. Cursor has no argument placeholder, so the body refers to
// "the text the user typed after the command".
// Docs: https://cursor.com/docs (the commands page could not be fetched when
// this was written; the layout follows the project spec).
type cursor struct{}

func (cursor) ID() string   { return "cursor" }
func (cursor) Name() string { return "Cursor" }

func (cursor) Render(cmds []templates.Command) ([]File, error) {
	files := make([]File, 0, len(cmds))
	for _, c := range cmds {
		name := "story-" + c.Name
		var b strings.Builder
		fmt.Fprintf(&b, "# %s\n\n%s\n\n", name, c.Description)
		b.WriteString(fill(c.Body, "the text the user typed after the command"))
		files = append(files, File{Path: ".cursor/commands/" + name + ".md", Content: []byte(b.String())})
	}
	return files, nil
}
