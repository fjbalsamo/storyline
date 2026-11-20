package adapters

import (
	"fmt"
	"strings"

	"github.com/fjbalsamo/storyline/internal/templates"
)

// codex renders OpenAI Codex skills. Codex reads repo skills from
// .agents/skills (not .codex/skills); SKILL.md needs name and description, and
// a skill is invoked explicitly as $story-<name>.
// Docs: https://learn.chatgpt.com/docs/build-skills (was developers.openai.com/codex/skills)
//
// Antigravity CLI (agy) reads the same files: workspace skills in
// .agents/skills/<name>/SKILL.md (name + description frontmatter), exposed as
// /<name>, plus AGENTS.md. --tools antigravity is an alias of this adapter.
// Docs: https://antigravity.google/docs/skills/
// Docs: https://antigravity.google/docs/rules/
//
// Codex reads AGENTS.md natively, so it needs no pointer file.
// Docs: https://learn.chatgpt.com/docs/agent-configuration/agents-md
type codex struct{}

func (codex) ID() string   { return "codex" }
func (codex) Name() string { return "Codex" }

func (codex) Render(cmds []templates.Command) ([]File, error) {
	files := make([]File, 0, len(cmds))
	for _, c := range cmds {
		name := "story-" + c.Name
		var b strings.Builder
		fmt.Fprintf(&b, "---\nname: %s\ndescription: %s\n---\n\n", name, yamlString(c.Description))
		b.WriteString(fill(c.Body, "the story id or text the user provided"))
		files = append(files, File{Path: ".agents/skills/" + name + "/SKILL.md", Content: []byte(b.String())})
	}
	return files, nil
}
