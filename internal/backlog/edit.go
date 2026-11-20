package backlog

import (
	"bytes"
	"errors"
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

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
		start, end, err := scalarSpan(src, k, v)
		if err != nil {
			return nil, nil, fmt.Errorf("edit %s: %w", f.key, err)
		}
		text, err := renderScalar(*f.value)
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

// renderScalar returns the YAML form of s on a single line, quoted if needed.
func renderScalar(s string) (string, error) {
	out, err := yaml.Marshal(s)
	if err != nil {
		return "", fmt.Errorf("render value: %w", err)
	}
	text := strings.TrimSuffix(string(out), "\n")
	if strings.Contains(text, "\n") {
		return "", errors.New("value cannot be written on a single line")
	}
	return text, nil
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

// mapPair returns the key and value nodes stored under key in a mapping node.
func mapPair(m *yaml.Node, key string) (k, v *yaml.Node) {
	if m.Kind != yaml.MappingNode {
		return nil, nil
	}
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			return m.Content[i], m.Content[i+1]
		}
	}
	return nil, nil
}

func mapValue(m *yaml.Node, key string) *yaml.Node {
	_, v := mapPair(m, key)
	return v
}

// fieldNodes returns the nodes of an editable text field.
func fieldNodes(m *yaml.Node, key string) (k, v *yaml.Node, err error) {
	k, v = mapPair(m, key)
	switch {
	case v == nil:
		return nil, nil, fmt.Errorf("edit: field %q not found", key)
	case v.Kind != yaml.ScalarNode || v.Tag == "!!null":
		return nil, nil, fmt.Errorf("edit: %q is not a plain text value", key)
	}
	return k, v, nil
}

// scalarSpan returns the byte range of a value as written in src, from its
// first character to its last, excluding any trailing comment.
func scalarSpan(src []byte, k, v *yaml.Node) (start, end int, err error) {
	start = offsetOf(src, v.Line, v.Column)
	keyIndent := k.Column - 1
	switch {
	case v.Style&(yaml.LiteralStyle|yaml.FoldedStyle) != 0:
		return start, blockEnd(src, start, keyIndent), nil
	case v.Style&yaml.DoubleQuotedStyle != 0:
		end, err = quotedEnd(src, start, '"')
	case v.Style&yaml.SingleQuotedStyle != 0:
		end, err = quotedEnd(src, start, '\'')
	default:
		end = plainEnd(src, start, keyIndent)
	}
	return start, end, err
}

// offsetOf converts a 1-based line and character column to a byte offset.
func offsetOf(src []byte, line, col int) int {
	off := 0
	for l := 1; l < line; l++ {
		i := bytes.IndexByte(src[off:], '\n')
		if i < 0 {
			return len(src)
		}
		off += i + 1
	}
	for c := 1; c < col && off < len(src); c++ {
		_, n := utf8.DecodeRune(src[off:])
		off += n
	}
	return off
}

// lineEnd is the offset of the end of the line holding pos, without \r or \n.
func lineEnd(src []byte, pos int) int {
	i := bytes.IndexByte(src[pos:], '\n')
	if i < 0 {
		i = len(src) - pos
	}
	end := pos + i
	if end > pos && src[end-1] == '\r' {
		end--
	}
	return end
}

// nextLine is the offset of the start of the line after the one holding pos,
// or -1 at the end of the file.
func nextLine(src []byte, pos int) int {
	i := bytes.IndexByte(src[pos:], '\n')
	if i < 0 {
		return -1
	}
	return pos + i + 1
}

func indentOf(line []byte) int {
	return len(line) - len(bytes.TrimLeft(line, " \t"))
}

// quotedEnd finds the end of a quoted scalar that may span lines.
func quotedEnd(src []byte, start int, q byte) (int, error) {
	for i := start + 1; i < len(src); i++ {
		switch {
		case q == '"' && src[i] == '\\':
			i++
		case src[i] != q:
		case q == '\'' && i+1 < len(src) && src[i+1] == '\'':
			i++
		default:
			return i + 1, nil
		}
	}
	return 0, errors.New("unterminated quoted value")
}

// blockEnd finds the end of a | or > block: the last line indented deeper
// than its key. The header line is replaced together with the body.
func blockEnd(src []byte, start, keyIndent int) int {
	end := lineEnd(src, start)
	for pos := nextLine(src, start); pos >= 0 && pos < len(src); pos = nextLine(src, pos) {
		line := src[pos:lineEnd(src, pos)]
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		if indentOf(line) <= keyIndent {
			break
		}
		end = lineEnd(src, pos)
	}
	return end
}

// plainEnd finds the end of an unquoted scalar, which may continue on lines
// indented deeper than its key. A trailing " #" comment ends it.
func plainEnd(src []byte, start, keyIndent int) int {
	end, commented := trimComment(src, start, lineEnd(src, start))
	for pos := nextLine(src, start); !commented && pos >= 0 && pos < len(src); pos = nextLine(src, pos) {
		line := src[pos:lineEnd(src, pos)]
		trimmed := bytes.TrimSpace(line)
		if len(trimmed) == 0 || trimmed[0] == '#' || indentOf(line) <= keyIndent {
			break
		}
		end, commented = trimComment(src, pos+indentOf(line), lineEnd(src, pos))
	}
	return end
}

// trimComment returns the end of the text in src[from:to] without a trailing
// comment or spaces, and whether a comment was found.
func trimComment(src []byte, from, to int) (int, bool) {
	text := src[from:to]
	commented := false
	for i := 1; i < len(text); i++ {
		if text[i] == '#' && (text[i-1] == ' ' || text[i-1] == '\t') {
			text, commented = text[:i], true
			break
		}
	}
	return from + len(bytes.TrimRight(text, " \t")), commented
}
