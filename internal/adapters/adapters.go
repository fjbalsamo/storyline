// Package adapters renders the canonical storyline commands for each
// supported AI tool.
package adapters

import (
	"fmt"
	"strings"

	"github.com/fjbalsamo/storyline/internal/templates"
)

// File is one generated file.
type File struct {
	Path    string // relative to project root, slash-separated
	Content []byte
	Merge   bool // true only for AGENTS.md (managed block)
}

// Adapter renders the canonical commands for one AI tool.
type Adapter interface {
	ID() string   // used in --tools
	Name() string // human name
	Render(cmds []templates.Command) ([]File, error)
}

// All returns every adapter in a stable order.
func All() []Adapter {
	return []Adapter{claude{}, cursor{}, copilot{}, gemini{}, codex{}, agentsMD{}}
}

// IDs returns the IDs of every adapter, in order.
func IDs() []string {
	var ids []string
	for _, a := range All() {
		ids = append(ids, a.ID())
	}
	return ids
}

// Get resolves adapter IDs. "all" selects every adapter. Duplicates are
// dropped and the order of first appearance is kept.
func Get(ids []string) ([]Adapter, error) {
	byID := map[string]Adapter{}
	for _, a := range All() {
		byID[a.ID()] = a
	}
	var out []Adapter
	seen := map[string]bool{}
	add := func(a Adapter) {
		if !seen[a.ID()] {
			seen[a.ID()] = true
			out = append(out, a)
		}
	}
	for _, raw := range ids {
		id := strings.ToLower(strings.TrimSpace(raw))
		switch a, ok := byID[id]; {
		case id == "all":
			for _, a := range All() {
				add(a)
			}
		case ok:
			add(a)
		default:
			return nil, fmt.Errorf("unknown tool %q: valid ids are %s, all",
				raw, strings.Join(IDs(), ", "))
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no tools given: valid ids are %s, all", strings.Join(IDs(), ", "))
	}
	return out, nil
}

// fill replaces the args placeholder and normalizes leading/trailing blank
// lines so the body ends with exactly one newline.
func fill(body, args string) string {
	body = strings.ReplaceAll(body, templates.ArgsPlaceholder, args)
	return strings.Trim(body, "\n") + "\n"
}

// yamlString quotes s as a YAML double-quoted scalar (JSON strings are valid YAML).
func yamlString(s string) string {
	return quoteString(s)
}
