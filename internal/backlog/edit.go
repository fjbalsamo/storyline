package backlog

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/fjbalsamo/storyline/internal/yamltext"

	"gopkg.in/yaml.v3"
)

// Edits holds the new text for the only fields `storyline edit` may change.
// A nil pointer means "leave the field alone".
type Edits struct {
	As, Want, SoThat *string
}

type fieldEdit struct {
	key   string
	value *string
}

// fields lists the requested edits in file order (as, want, so_that).
func (e Edits) fields() []fieldEdit {
	all := []fieldEdit{{"as", e.As}, {"want", e.Want}, {"so_that", e.SoThat}}
	var out []fieldEdit
	for _, f := range all {
		if f.value != nil {
			out = append(out, f)
		}
	}
	return out
}

// CheckEditable rejects statuses whose stories are history.
func CheckEditable(s Status) error {
	if s == StatusDone || s == StatusDiscarded {
		return fmt.Errorf("%s stories are history and can't be edited", s)
	}
	return nil
}

type patch struct {
	start, end int
	text       string
}

// ApplyEdits changes the as/want/so_that values of item id directly in the
// YAML text, so comments, key order and formatting outside the edited values
// are kept byte for byte. It returns the new text and the names of the fields
// whose value really changed; when none did, the text is returned untouched.
func ApplyEdits(src []byte, id string, e Edits) ([]byte, []string, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(src, &doc); err != nil {
		return nil, nil, fmt.Errorf("edit: parse backlog: %w", err)
	}
	item, err := findItemNode(&doc, id)
	if err != nil {
		return nil, nil, err
	}

	patches, changed, err := buildPatches(src, item, e)
	if err != nil {
		return nil, nil, err
	}
	if len(patches) == 0 {
		return src, nil, nil
	}
	out := applyPatches(src, patches)
	if err := verifyEdits(out, id, e); err != nil {
		return nil, nil, err
	}
	return out, changed, nil
}

func buildPatches(src []byte, item *yaml.Node, e Edits) ([]patch, []string, error) {
	var patches []patch
	var changed []string
	for _, f := range e.fields() {
		k, v, err := fieldNodes(item, f.key)
		if err != nil {
			return nil, nil, err
		}
		if v.Value == *f.value {
			continue
		}
		start, end, err := yamltext.ScalarSpan(src, k, v)
		if err != nil {
			return nil, nil, fmt.Errorf("edit %s: %w", f.key, err)
		}
		text, err := yamltext.RenderScalar(*f.value)
		if err != nil {
			return nil, nil, fmt.Errorf("edit %s: %w", f.key, err)
		}
		patches = append(patches, patch{start, end, text})
		changed = append(changed, f.key)
	}
	return patches, changed, nil
}

// applyPatches splices from the end so earlier offsets stay valid.
func applyPatches(src []byte, patches []patch) []byte {
	sort.Slice(patches, func(i, j int) bool { return patches[i].start > patches[j].start })
	out := append([]byte(nil), src...)
	for _, p := range patches {
		out = append(out[:p.start], append([]byte(p.text), out[p.end:]...)...)
	}
	return out
}

// verifyEdits re-reads the spliced text and checks every edited value landed.
func verifyEdits(out []byte, id string, e Edits) error {
	b, err := Decode(bytes.NewReader(out))
	if err != nil {
		return fmt.Errorf("edit produced an unreadable backlog: %w", err)
	}
	it, ok := b.Find(id)
	if !ok {
		return fmt.Errorf("edit lost story %s", id)
	}
	got := map[string]string{"as": it.As, "want": it.Want, "so_that": it.SoThat}
	for _, f := range e.fields() {
		if got[f.key] != *f.value {
			return fmt.Errorf("edit %s: could not rewrite the value safely", f.key)
		}
	}
	return nil
}

func findItemNode(doc *yaml.Node, id string) (*yaml.Node, error) {
	notFound := fmt.Errorf("story %s not found", id)
	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return nil, notFound
	}
	seq := mapValue(doc.Content[0], "backlog")
	if seq == nil || seq.Kind != yaml.SequenceNode {
		return nil, notFound
	}
	for _, item := range seq.Content {
		if v := mapValue(item, "id"); v != nil && v.Value == id {
			if item.Style&yaml.FlowStyle != 0 {
				return nil, fmt.Errorf("story %s is written in flow style ({…}); edit it by hand", id)
			}
			return item, nil
		}
	}
	return nil, notFound
}

func mapValue(m *yaml.Node, key string) *yaml.Node {
	_, v := yamltext.MapPair(m, key)
	return v
}

// fieldNodes returns the nodes of an editable text field.
func fieldNodes(m *yaml.Node, key string) (k, v *yaml.Node, err error) {
	k, v = yamltext.MapPair(m, key)
	switch {
	case v == nil:
		return nil, nil, fmt.Errorf("edit: field %q not found", key)
	case v.Kind != yaml.ScalarNode || v.Tag == "!!null":
		return nil, nil, fmt.Errorf("edit: %q is not a plain text value", key)
	}
	return k, v, nil
}
