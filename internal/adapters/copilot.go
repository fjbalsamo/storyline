package adapters

import (
	"fmt"
	"strings"

	"github.com/fjbalsamo/storyline/internal/templates"
)

// copilot renders GitHub Copilot prompt files. Frontmatter fields
// description, argument-hint and agent (ask|agent|plan) are documented, and
// ${input:name:placeholder} asks the user for a value.
// Docs: https://code.visualstudio.com/docs/agent-customization/prompt-files
//
// Copilot reads AGENTS.md (the nearest one wins), so it needs no pointer file.
// If a surface turns out not to read it, add .github/copilot-instructions.md.
// Docs: https://docs.github.com/en/copilot/how-tos/configure-custom-instructions/add-repository-instructions
type copilot struct{}

func (copilot) ID() string   { return "copilot" }
func (copilot) Name() string { return "GitHub Copilot" }

var inputHint = strings.NewReplacer(":", " ", "}", ")", "{", "(", "\n", " ")

func (copilot) Render(cmds []templates.Command) ([]File, error) {
	files := make([]File, 0, len(cmds))
	for _, c := range cmds {
		var b strings.Builder
		fmt.Fprintf(&b, "---\ndescription: %s\nargument-hint: %s\nagent: agent\n---\n\n", yamlString(c.Description), yamlString(c.ArgumentHint))
		b.WriteString(fill(c.Body, "${input:args:"+inputHint.Replace(c.ArgumentHint)+"}"))
		files = append(files, File{Path: ".github/prompts/story-" + c.Name + ".prompt.md", Content: []byte(b.String())})
	}
	return files, nil
}
