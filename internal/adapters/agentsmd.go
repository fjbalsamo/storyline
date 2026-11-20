package adapters

import (
	"fmt"
	"strings"

	"github.com/fjbalsamo/storyline/internal/templates"
)

// Markers delimit the storyline section of AGENTS.md. Everything outside
// them belongs to the user.
const (
	MarkerStart = "<!-- storyline:start -->"
	MarkerEnd   = "<!-- storyline:end -->"
)

// agentsMD renders a short AGENTS.md section so tools without slash commands
// still follow the workflow. AGENTS.md is the shared convention described at
// https://agents.md
type agentsMD struct{}

func (agentsMD) ID() string   { return "agents-md" }
func (agentsMD) Name() string { return "AGENTS.md" }

func (agentsMD) Render(cmds []templates.Command) ([]File, error) {
	var b strings.Builder
	b.WriteString(MarkerStart + "\n## Storyline\n\n")
	b.WriteString("This project is driven by a backlog of user stories managed with the `storyline` CLI.\n")
	b.WriteString("The backlog is `storyline/backlog.yaml`: never edit it or move story folders by hand.\n")
	b.WriteString("Story status flow: `todo → refining → ready → in-progress → done | discarded`.\n")
	b.WriteString("Change status with `storyline status <id> <status>` (`ready` needs `--story <path>`), close a story with `storyline done <id>`, and check which tests prove each acceptance criterion with `storyline trace <id>`.\n")
	b.WriteString("Related repos are listed in `storyline/config.yaml` (`spec.dependsOn`): the AI only suggests stories there and never blocks waiting for them.\n")
	b.WriteString("If a `storyline` command fails, show its error to the user and stop; never work around it by editing files.\n\n")
	b.WriteString("When the user asks for one of these steps, do only that step and stop:\n\n")
	for _, c := range cmds {
		fmt.Fprintf(&b, "- **%s**: %s. Input: %s\n", c.Name, strings.TrimRight(c.Description, "."), c.ArgumentHint)
	}
	b.WriteString("\nRun `storyline list` to see the backlog and `storyline validate` after changing it.\n")
	b.WriteString(MarkerEnd + "\n")
	return []File{{Path: "AGENTS.md", Content: []byte(b.String()), Merge: true}}, nil
}
