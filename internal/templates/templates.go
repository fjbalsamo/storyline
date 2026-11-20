package templates

import (
	"bytes"
	"fmt"
	"io/fs"
	"strings"

	"gopkg.in/yaml.v3"
)

// Command is one canonical storyline command.
type Command struct {
	Name         string // "refine"
	Description  string
	ArgumentHint string
	Body         string // contains {{ARGS}}
}

// ArgsPlaceholder marks where the user's arguments go in a Body.
const ArgsPlaceholder = "{{ARGS}}"

// workflow is the fixed order commands are returned in.
var workflow = []string{"context", "add", "import", "refine", "slice", "apply", "verify", "done"}

type frontmatter struct {
	Description  string `yaml:"description"`
	ArgumentHint string `yaml:"argument_hint"`
}

// Commands parses the embedded command files in workflow order.
func Commands() ([]Command, error) {
	cmds := make([]Command, 0, len(workflow))
	for _, name := range workflow {
		path := "commands/" + name + ".md"
		raw, err := fs.ReadFile(FS, path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
		c, err := parse(name, raw)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		cmds = append(cmds, c)
	}
	return cmds, nil
}

func parse(name string, raw []byte) (Command, error) {
	raw = bytes.ReplaceAll(raw, []byte("\r\n"), []byte("\n"))
	rest, ok := strings.CutPrefix(string(raw), "---\n")
	if !ok {
		return Command{}, fmt.Errorf("missing frontmatter")
	}
	var fm, body string
	if i := strings.Index(rest, "\n---\n"); i >= 0 {
		fm, body = rest[:i], rest[i+len("\n---\n"):]
	} else if i := strings.Index(rest, "---\n"); i == 0 {
		body = rest[len("---\n"):]
	} else {
		return Command{}, fmt.Errorf("unterminated frontmatter")
	}

	var meta frontmatter
	if err := yaml.Unmarshal([]byte(fm), &meta); err != nil {
		return Command{}, fmt.Errorf("parse frontmatter: %w", err)
	}
	if meta.Description == "" {
		return Command{}, fmt.Errorf("frontmatter: missing description")
	}
	if meta.ArgumentHint == "" {
		return Command{}, fmt.Errorf("frontmatter: missing argument_hint")
	}
	return Command{
		Name:         name,
		Description:  meta.Description,
		ArgumentHint: meta.ArgumentHint,
		Body:         body,
	}, nil
}
