// Package yamltext holds the small text-level helpers behind the commands
// that edit YAML files without re-encoding them: finding where a value sits
// in the original bytes so only those bytes change.
package yamltext

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"gopkg.in/yaml.v3"
)

// RenderScalar returns the YAML form of s on a single line, quoted if needed.
func RenderScalar(s string) (string, error) {
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

// MapPair returns the key and value nodes stored under key in a mapping node.
func MapPair(m *yaml.Node, key string) (k, v *yaml.Node) {
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

// ScalarSpan returns the byte range of a value as written in src, from its
// first character to its last, excluding any trailing comment.
func ScalarSpan(src []byte, k, v *yaml.Node) (start, end int, err error) {
	start = OffsetOf(src, v.Line, v.Column)
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

// OffsetOf converts a 1-based line and character column to a byte offset.
func OffsetOf(src []byte, line, col int) int {
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

// LineEnd is the offset of the end of the line holding pos, without \r or \n.
func LineEnd(src []byte, pos int) int {
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

// NextLine is the offset of the start of the line after the one holding pos,
// or -1 at the end of the file.
func NextLine(src []byte, pos int) int {
	i := bytes.IndexByte(src[pos:], '\n')
	if i < 0 {
		return -1
	}
	return pos + i + 1
}

// IndentOf is the number of leading spaces and tabs of line.
func IndentOf(line []byte) int {
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
	end := LineEnd(src, start)
	for pos := NextLine(src, start); pos >= 0 && pos < len(src); pos = NextLine(src, pos) {
		line := src[pos:LineEnd(src, pos)]
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		if IndentOf(line) <= keyIndent {
			break
		}
		end = LineEnd(src, pos)
	}
	return end
}

// plainEnd finds the end of an unquoted scalar, which may continue on lines
// indented deeper than its key. A trailing " #" comment ends it.
func plainEnd(src []byte, start, keyIndent int) int {
	end, commented := trimComment(src, start, LineEnd(src, start))
	for pos := NextLine(src, start); !commented && pos >= 0 && pos < len(src); pos = NextLine(src, pos) {
		line := src[pos:LineEnd(src, pos)]
		trimmed := bytes.TrimSpace(line)
		if len(trimmed) == 0 || trimmed[0] == '#' || IndentOf(line) <= keyIndent {
			break
		}
		end, commented = trimComment(src, pos+IndentOf(line), LineEnd(src, pos))
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

// PrevLine is the offset of the start of the line before the one holding
// pos, or -1 on the first line.
func PrevLine(src []byte, pos int) int {
	if pos <= 0 {
		return -1
	}
	return bytes.LastIndexByte(src[:pos-1], '\n') + 1
}

// NextOrEnd is like NextLine but returns len(src) instead of -1.
func NextOrEnd(src []byte, pos int) int {
	if n := NextLine(src, pos); n >= 0 {
		return n
	}
	return len(src)
}

// ItemEnd is the offset just past the item's last line: lines deeper than
// the dash belong to it; blank lines only count when more of it follows.
func ItemEnd(src []byte, first, dashIndent int) int {
	end := NextOrEnd(src, first)
	for pos := end; pos < len(src); pos = NextOrEnd(src, pos) {
		line := src[pos:LineEnd(src, pos)]
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		if IndentOf(line) <= dashIndent {
			break
		}
		end = NextOrEnd(src, pos)
	}
	return end
}

// LeadingComments extends the block upwards over the comment lines directly
// above the item, never past floor (the end of the previous block).
func LeadingComments(src []byte, first, floor int) int {
	start := first
	for {
		p := PrevLine(src, start)
		if p < floor || p < 0 || !bytes.HasPrefix(bytes.TrimSpace(src[p:LineEnd(src, p)]), []byte("#")) {
			return start
		}
		start = p
	}
}
