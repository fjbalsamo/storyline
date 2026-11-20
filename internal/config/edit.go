package config

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/fjbalsamo/storyline/internal/yamltext"
	"gopkg.in/yaml.v3"
)

// The operations below change config.yaml in its own text, so comments, the
// schema header and formatting stay as they are. Each one returns the new
// text, or src itself when there is nothing to change. Every result is
// validated, then re-read and compared with the intended config before it is
// returned, so a bad splice is an error and never a corrupted file.

var (
	typeExample  = regexp.MustCompile(`^(\s*)#\s*type:\s*\S.*$`)
	depsExample  = regexp.MustCompile(`^(\s*)#\s*dependsOn:\s*$`)
	depsExItem   = regexp.MustCompile(`^\s*#\s+(-\s+name:|url:)`)
	errFlowStyle = errors.New("write it in block style (not {…} or […]) so it can be edited")
)

// SetDescription sets metadata.description; an empty value removes it.
func SetDescription(src []byte, value string) ([]byte, error) {
	return apply(src,
		func(c *Config) error { c.Metadata.Description = value; return nil },
		func(src []byte) ([]byte, error) { return editDescription(src, value) })
}

// SetType sets spec.type; an empty value removes it.
func SetType(src []byte, value string) ([]byte, error) {
	return apply(src,
		func(c *Config) error { c.Spec.Type = value; return nil },
		func(src []byte) ([]byte, error) {
			return editSpec(src, func(src []byte, k, v *yaml.Node) ([]byte, error) {
				return editType(src, k, v, value)
			})
		})
}

// AddDependency adds a dependency, or sets the URL of an existing one when
// url is not empty.
func AddDependency(src []byte, name, url string) ([]byte, error) {
	return apply(src,
		func(c *Config) error {
			for i, d := range c.Spec.DependsOn {
				if d.Name != name {
					continue
				}
				if url != "" {
					c.Spec.DependsOn[i].URL = url
				}
				return nil
			}
			c.Spec.DependsOn = append(c.Spec.DependsOn, Dependency{Name: name, URL: url})
			return nil
		},
		func(src []byte) ([]byte, error) {
			return editSpec(src, func(src []byte, k, v *yaml.Node) ([]byte, error) {
				return addDependency(src, k, v, name, url)
			})
		})
}

// RemoveDependency removes a dependency by name.
func RemoveDependency(src []byte, name string) ([]byte, error) {
	return apply(src,
		func(c *Config) error {
			for i, d := range c.Spec.DependsOn {
				if d.Name == name {
					c.Spec.DependsOn = append(c.Spec.DependsOn[:i], c.Spec.DependsOn[i+1:]...)
					return nil
				}
			}
			return fmt.Errorf("dependency %s not found", name)
		},
		func(src []byte) ([]byte, error) {
			return editSpec(src, func(src []byte, k, v *yaml.Node) ([]byte, error) {
				return removeDependency(src, k, v, name)
			})
		})
}

// apply runs the config-level change, validates it, runs the text edit and
// checks that the edited text means exactly the intended config.
func apply(src []byte, change func(*Config) error, edit func([]byte) ([]byte, error)) ([]byte, error) {
	cur, err := Decode(bytes.NewReader(src))
	if err != nil {
		return nil, err
	}
	want := *cur
	want.Spec.DependsOn = append([]Dependency(nil), cur.Spec.DependsOn...)
	if err := change(&want); err != nil {
		return nil, err
	}
	if equal(*cur, want) {
		return src, nil
	}
	if err := want.Validate(); err != nil {
		return nil, err
	}
	out, err := edit(src)
	if err != nil {
		return nil, err
	}
	got, err := Decode(bytes.NewReader(out))
	if err != nil || !equal(*got, want) {
		return nil, errors.New("could not rewrite config.yaml safely; edit it by hand")
	}
	return out, nil
}

func equal(a, b Config) bool {
	if a.Version != b.Version || a.Metadata != b.Metadata || a.Spec.Type != b.Spec.Type ||
		len(a.Spec.DependsOn) != len(b.Spec.DependsOn) {
		return false
	}
	for i := range a.Spec.DependsOn {
		if a.Spec.DependsOn[i] != b.Spec.DependsOn[i] {
			return false
		}
	}
	return true
}

func parseRoot(src []byte) (*yaml.Node, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(src, &doc); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 || doc.Content[0].Kind != yaml.MappingNode {
		return nil, errors.New("config.yaml is not a mapping")
	}
	return doc.Content[0], nil
}

func editDescription(src []byte, value string) ([]byte, error) {
	root, err := parseRoot(src)
	if err != nil {
		return nil, err
	}
	_, meta := yamltext.MapPair(root, "metadata")
	if meta == nil || meta.Kind != yaml.MappingNode || meta.Style&yaml.FlowStyle != 0 || len(meta.Content) == 0 {
		return nil, errFlowStyle
	}
	if k, v := yamltext.MapPair(meta, "description"); v != nil {
		return replaceOrRemove(src, k, v, value)
	}
	text, err := keyLine(meta.Content[0].Column-1, "description", value)
	if err != nil {
		return nil, err
	}
	return insertAfterLine(src, lastChildEnd(src, meta), text), nil
}

// specOp edits the spec mapping. By the time it runs, spec exists and is
// either a block mapping or an empty `spec:`.
type specOp func(src []byte, specKey, spec *yaml.Node) ([]byte, error)

func editSpec(src []byte, op specOp) ([]byte, error) {
	root, err := parseRoot(src)
	if err != nil {
		return nil, err
	}
	k, v := yamltext.MapPair(root, "spec")
	if k == nil {
		s := withNewline(src)
		return reparseSpec(append(s[:len(s):len(s)], "spec:\n"...), op)
	}
	if isEmpty(v) {
		if src, err = stripValue(src, k, v); err != nil {
			return nil, err
		}
		return reparseSpec(src, op)
	}
	if v.Kind != yaml.MappingNode || v.Style&yaml.FlowStyle != 0 {
		return nil, fmt.Errorf("spec: %w", errFlowStyle)
	}
	return op(src, k, v)
}

func reparseSpec(src []byte, op specOp) ([]byte, error) {
	root, err := parseRoot(src)
	if err != nil {
		return nil, err
	}
	k, v := yamltext.MapPair(root, "spec")
	if k == nil {
		return nil, errors.New("could not create spec")
	}
	return op(src, k, v)
}

func editType(src []byte, specKey, spec *yaml.Node, value string) ([]byte, error) {
	if k, v := yamltext.MapPair(spec, "type"); v != nil {
		return replaceOrRemove(src, k, v, value)
	}
	if start, m, ok := findExample(src, specKey, typeExample); ok {
		text, err := keyLine(len(m[1]), "type", value)
		if err != nil {
			return nil, err
		}
		return splice(src, start, yamltext.NextOrEnd(src, start), text), nil
	}
	text, err := keyLine(childIndent(specKey, spec), "type", value)
	if err != nil {
		return nil, err
	}
	return insertAfterLine(src, yamltext.OffsetOf(src, specKey.Line, 1), text), nil
}

func addDependency(src []byte, specKey, spec *yaml.Node, name, url string) ([]byte, error) {
	k, deps := yamltext.MapPair(spec, "dependsOn")
	switch {
	case deps == nil:
		return addFirstDependency(src, specKey, spec, name, url)
	case isEmpty(deps):
		out, err := stripValue(src, k, deps)
		if err != nil {
			return nil, err
		}
		return insertAfterLine(out, yamltext.OffsetOf(out, k.Line, 1), itemLines(k.Column-1+4, 2, name, url)), nil
	case deps.Kind != yaml.SequenceNode || deps.Style&yaml.FlowStyle != 0:
		return nil, fmt.Errorf("spec.dependsOn: %w", errFlowStyle)
	}
	if item := findDependency(deps, name); item != nil {
		return setDependencyURL(src, item, url)
	}
	return appendDependency(src, deps, name, url)
}

// addFirstDependency creates spec.dependsOn. The commented example that init
// writes is replaced by the real block; otherwise it goes after the last key.
func addFirstDependency(src []byte, specKey, spec *yaml.Node, name, url string) ([]byte, error) {
	if start, end, indent, ok := findDepsExample(src, specKey); ok {
		block := strings.Repeat(" ", indent) + "dependsOn:\n" + itemLines(indent+4, 2, name, url)
		return splice(src, start, end, block), nil
	}
	indent := childIndent(specKey, spec)
	block := strings.Repeat(" ", indent) + "dependsOn:\n" + itemLines(indent+4, 2, name, url)
	if len(spec.Content) > 0 {
		return insertAfterLine(src, lastChildEnd(src, spec), block), nil
	}
	return insertAfterLine(src, yamltext.OffsetOf(src, specKey.Line, 1), block), nil
}

func findDependency(deps *yaml.Node, name string) *yaml.Node {
	for _, item := range deps.Content {
		if _, v := yamltext.MapPair(item, "name"); v != nil && v.Value == name {
			return item
		}
	}
	return nil
}

func setDependencyURL(src []byte, item *yaml.Node, url string) ([]byte, error) {
	if item.Style&yaml.FlowStyle != 0 {
		return nil, fmt.Errorf("spec.dependsOn: %w", errFlowStyle)
	}
	if k, v := yamltext.MapPair(item, "url"); v != nil {
		return replaceOrRemove(src, k, v, url)
	}
	nameKey, _ := yamltext.MapPair(item, "name")
	text, err := keyLine(nameKey.Column-1, "url", url)
	if err != nil {
		return nil, err
	}
	return insertAfterLine(src, lastChildEnd(src, item), text), nil
}

func appendDependency(src []byte, deps *yaml.Node, name, url string) ([]byte, error) {
	dashIndent := deps.Column - 1
	gap := 2
	if nameKey, _ := yamltext.MapPair(deps.Content[0], "name"); nameKey != nil && nameKey.Line == deps.Content[0].Line {
		gap = nameKey.Column - 1 - dashIndent
	}
	last := deps.Content[len(deps.Content)-1]
	end := yamltext.ItemEnd(src, itemStart(src, last), dashIndent)
	out := src
	if end >= len(src) {
		out = withNewline(src)
		end = len(out)
	}
	return splice(out, end, end, itemLines(dashIndent+gap, gap, name, url)), nil
}

func removeDependency(src []byte, specKey, spec *yaml.Node, name string) ([]byte, error) {
	depsKey, deps := yamltext.MapPair(spec, "dependsOn")
	if deps == nil || deps.Kind != yaml.SequenceNode {
		return nil, fmt.Errorf("dependency %s not found", name)
	}
	dashIndent := deps.Column - 1
	floor := yamltext.NextOrEnd(src, yamltext.OffsetOf(src, depsKey.Line, 1))
	for _, item := range deps.Content {
		first := itemStart(src, item)
		end := yamltext.ItemEnd(src, first, dashIndent)
		if _, v := yamltext.MapPair(item, "name"); v == nil || v.Value != name {
			floor = end
			continue
		}
		if len(deps.Content) == 1 {
			return splice(src, yamltext.OffsetOf(src, depsKey.Line, 1), end, ""), nil
		}
		return splice(src, yamltext.LeadingComments(src, first, floor), end, ""), nil
	}
	return nil, fmt.Errorf("dependency %s not found", name)
}

// itemStart is the offset of the line holding an item's dash.
func itemStart(src []byte, item *yaml.Node) int {
	pos := yamltext.OffsetOf(src, item.Line, 1)
	line := bytes.TrimSpace(src[pos:yamltext.LineEnd(src, pos)])
	if !bytes.HasPrefix(line, []byte("-")) {
		if p := yamltext.PrevLine(src, pos); p >= 0 {
			return p
		}
	}
	return pos
}

// itemLines renders "- name: n" and "url: u" with the keys at keyIndent and
// the dash gap columns before them.
func itemLines(keyIndent, gap int, name, url string) string {
	n, _ := yamltext.RenderScalar(name)
	dash := strings.Repeat(" ", keyIndent-gap) + "-" + strings.Repeat(" ", gap-1)
	out := dash + "name: " + n + "\n"
	if url != "" {
		u, _ := yamltext.RenderScalar(url)
		out += strings.Repeat(" ", keyIndent) + "url: " + u + "\n"
	}
	return out
}

func keyLine(indent int, key, value string) (string, error) {
	rendered, err := yamltext.RenderScalar(value)
	if err != nil {
		return "", fmt.Errorf("%s: %w", key, err)
	}
	return strings.Repeat(" ", indent) + key + ": " + rendered + "\n", nil
}

// replaceOrRemove rewrites the value of key k, or deletes its whole line when
// value is empty.
func replaceOrRemove(src []byte, k, v *yaml.Node, value string) ([]byte, error) {
	if v.Kind != yaml.ScalarNode {
		return nil, fmt.Errorf("%s is not a plain text value", k.Value)
	}
	start, end, err := yamltext.ScalarSpan(src, k, v)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", k.Value, err)
	}
	if value == "" {
		return splice(src, yamltext.OffsetOf(src, k.Line, 1), yamltext.NextOrEnd(src, end), ""), nil
	}
	text, err := yamltext.RenderScalar(value)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", k.Value, err)
	}
	return splice(src, start, end, text), nil
}

// isEmpty reports whether a value is absent in practice: null, `~`, `{}` or `[]`.
func isEmpty(v *yaml.Node) bool {
	switch v.Kind {
	case yaml.ScalarNode:
		return v.Tag == "!!null"
	case yaml.MappingNode, yaml.SequenceNode:
		return len(v.Content) == 0 && v.Style&yaml.FlowStyle != 0
	}
	return false
}

// stripValue turns `key: {}` (or null, ~, []) into `key:`, keeping any
// comment on the line.
func stripValue(src []byte, k, v *yaml.Node) ([]byte, error) {
	if v.Kind == yaml.ScalarNode && v.Value == "" {
		return src, nil
	}
	keyEnd := yamltext.OffsetOf(src, k.Line, k.Column) + len(k.Value)
	if keyEnd >= len(src) || src[keyEnd] != ':' {
		return nil, fmt.Errorf("%s: unexpected key layout", k.Value)
	}
	start := yamltext.OffsetOf(src, v.Line, v.Column)
	end := start
	if v.Kind == yaml.ScalarNode {
		_, e, err := yamltext.ScalarSpan(src, k, v)
		if err != nil {
			return nil, err
		}
		end = e
	} else if i := bytes.IndexAny(src[start:], "}]"); i >= 0 {
		end = start + i + 1
	}
	return splice(src, keyEnd+1, end, ""), nil
}

// childIndent is the indentation of the keys inside spec.
func childIndent(specKey, spec *yaml.Node) int {
	if len(spec.Content) > 0 {
		return spec.Content[0].Column - 1
	}
	return specKey.Column - 1 + 2
}

// lastChildEnd is an offset on the last line of the mapping's last value.
func lastChildEnd(src []byte, m *yaml.Node) int {
	k, v := m.Content[len(m.Content)-2], m.Content[len(m.Content)-1]
	if v.Kind == yaml.ScalarNode {
		if _, end, err := yamltext.ScalarSpan(src, k, v); err == nil {
			return end
		}
	}
	return yamltext.ItemEnd(src, yamltext.OffsetOf(src, k.Line, 1), k.Column-1) - 1
}

// findExample finds a commented example line matching re in the lines that
// belong to spec.
func findExample(src []byte, specKey *yaml.Node, re *regexp.Regexp) (start int, m []string, ok bool) {
	first := yamltext.OffsetOf(src, specKey.Line, 1)
	regionEnd := yamltext.ItemEnd(src, first, specKey.Column-1)
	for pos := yamltext.NextOrEnd(src, first); pos < regionEnd; pos = yamltext.NextOrEnd(src, pos) {
		if m := re.FindStringSubmatch(string(src[pos:yamltext.LineEnd(src, pos)])); m != nil {
			return pos, m, true
		}
	}
	return 0, nil, false
}

// findDepsExample finds the commented dependsOn example that init writes: the
// `# dependsOn:` line and the commented `- name:` / `url:` lines under it.
func findDepsExample(src []byte, specKey *yaml.Node) (start, end, indent int, ok bool) {
	start, m, ok := findExample(src, specKey, depsExample)
	if !ok {
		return 0, 0, 0, false
	}
	regionEnd := yamltext.ItemEnd(src, yamltext.OffsetOf(src, specKey.Line, 1), specKey.Column-1)
	end = yamltext.NextOrEnd(src, start)
	for end < regionEnd && depsExItem.Match(src[end:yamltext.LineEnd(src, end)]) {
		end = yamltext.NextOrEnd(src, end)
	}
	return start, end, len(m[1]), true
}

func withNewline(src []byte) []byte {
	if len(src) > 0 && !bytes.HasSuffix(src, []byte("\n")) {
		return append(append([]byte(nil), src...), '\n')
	}
	return src
}

// insertAfterLine inserts text (ending in a newline) after the line holding pos.
func insertAfterLine(src []byte, pos int, text string) []byte {
	if yamltext.NextLine(src, pos) < 0 {
		src = withNewline(src)
		return splice(src, len(src), len(src), text)
	}
	n := yamltext.NextLine(src, pos)
	return splice(src, n, n, text)
}

func splice(src []byte, start, end int, text string) []byte {
	out := make([]byte, 0, len(src)-(end-start)+len(text))
	out = append(out, src[:start]...)
	out = append(out, text...)
	return append(out, src[end:]...)
}
