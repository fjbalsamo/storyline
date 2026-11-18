package adapters

import (
	"fmt"
	"strings"

	"github.com/fjbalsamo/storyline/internal/templates"
)

// gemini renders Gemini CLI custom commands. A subdirectory under
// .gemini/commands becomes a namespace (story/add.toml is /story:add) and
// {{args}} is replaced with what the user typed.
// Docs: https://geminicli.com/docs/cli/custom-commands/
type gemini struct{}

func (gemini) ID() string   { return "gemini" }
func (gemini) Name() string { return "Gemini CLI" }

func (gemini) Render(cmds []templates.Command) ([]File, error) {
	files := make([]File, 0, len(cmds))
	for _, c := range cmds {
		// Gemini treats !{...} and @{...} as shell/file injection.
		if strings.Contains(c.Body, "!{") || strings.Contains(c.Body, "@{") {
			return nil, fmt.Errorf("gemini: command %q contains !{ or @{, which Gemini would execute", c.Name)
		}
		var b strings.Builder
		fmt.Fprintf(&b, "description = %s\nprompt = \"\"\"\n%s\"\"\"\n", tomlString(c.Description), tomlMultiline(fill(c.Body, "{{args}}")))
		files = append(files, File{Path: ".gemini/commands/story/" + c.Name + ".toml", Content: []byte(b.String())})
	}
	return files, nil
}

// tomlString returns s as a TOML basic string.
func tomlString(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`, "\r", `\r`, "\t", `\t`)
	return `"` + r.Replace(s) + `"`
}

// tomlMultiline escapes s for use inside a """ multi-line basic string.
// Backslashes are doubled and every """ is broken up so it cannot close the
// string. The caller guarantees s ends with a newline, so a trailing quote
// cannot merge with the closing delimiter.
func tomlMultiline(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	return strings.ReplaceAll(s, `"""`, `""\"`)
}
