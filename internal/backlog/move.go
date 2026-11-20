package backlog

import (
	"bytes"
	"errors"
	"fmt"

	"gopkg.in/yaml.v3"
)

// Mode says where `storyline move` puts a story.
type Mode int

const (
	MoveTop Mode = iota + 1
	MoveBottom
	MoveBefore
	MoveAfter
)

// Placement is a target position; Ref is the reference id for before/after.
type Placement struct {
	Mode Mode
	Ref  string
}

// PlanMove returns the new order of the items as indexes into items, plus the
// 0-based position of id before (from) and after (to) the move. Top means
// before the first open story, so done and discarded stories stay above it.
func PlanMove(items []Item, id string, p Placement) (order []int, from, to int, err error) {
	from = indexOfID(items, id)
	if from < 0 {
		return nil, 0, 0, fmt.Errorf("story %s not found", id)
	}
	if !items[from].Status.Open() {
		return nil, 0, 0, fmt.Errorf("%s stories are history and can't be moved", items[from].Status)
	}
	rest := make([]int, 0, len(items)-1)
	for i := range items {
		if i != from {
			rest = append(rest, i)
		}
	}
	to, err = insertPos(items, rest, from, p)
	if err != nil {
		return nil, 0, 0, err
	}
	order = append(append(append([]int{}, rest[:to]...), from), rest[to:]...)
	return order, from, to, nil
}

func indexOfID(items []Item, id string) int {
	for i, it := range items {
		if it.ID == id {
			return i
		}
	}
	return -1
}

// insertPos is where the moved story lands inside rest (all other items).
func insertPos(items []Item, rest []int, from int, p Placement) (int, error) {
	switch p.Mode {
	case MoveTop:
		return topPos(items, rest, from), nil
	case MoveBottom:
		return len(rest), nil
	case MoveBefore, MoveAfter:
		return refPos(items, rest, from, p)
	}
	return 0, errors.New("move: no target position")
}

func topPos(items []Item, rest []int, from int) int {
	for _, it := range items[:from] {
		if it.Status.Open() {
			for pos, i := range rest {
				if items[i].Status.Open() {
					return pos
				}
			}
		}
	}
	return from // already the first open story
}

func refPos(items []Item, rest []int, from int, p Placement) (int, error) {
	ref := indexOfID(items, p.Ref)
	switch {
	case ref < 0:
		return 0, fmt.Errorf("story %s not found", p.Ref)
	case ref == from:
		return 0, fmt.Errorf("can't move %s relative to itself", p.Ref)
	}
	pos := ref
	if ref > from {
		pos--
	}
	if p.Mode == MoveAfter {
		pos++
	}
	return pos, nil
}

// ApplyMove moves item id in the YAML text without re-encoding it: the item's
// block (its lines plus the comments directly above) swaps places with the
// others, while everything between blocks stays where it is. from and to are
// 1-based positions; when they are equal the text is returned untouched.
func ApplyMove(src []byte, id string, p Placement) (out []byte, from, to int, err error) {
	b, err := Decode(bytes.NewReader(src))
	if err != nil {
		return nil, 0, 0, err
	}
	order, f, t, err := PlanMove(b.Items, id, p)
	if err != nil {
		return nil, 0, 0, err
	}
	if f == t {
		return src, f + 1, t + 1, nil
	}
	seq, err := itemsSeq(src, len(b.Items), indexOfID(b.Items, id))
	if err != nil {
		return nil, 0, 0, err
	}
	out = permuteBlocks(src, blockSpans(src, seq), order)
	if err := verifyOrder(out, b.Items, order); err != nil {
		return nil, 0, 0, err
	}
	return out, f + 1, t + 1, nil
}

// itemsSeq returns the backlog sequence node, rejecting flow style.
func itemsSeq(src []byte, count, moved int) (*yaml.Node, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(src, &doc); err != nil {
		return nil, fmt.Errorf("move: parse backlog: %w", err)
	}
	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return nil, errors.New("move: backlog is empty")
	}
	seq := mapValue(doc.Content[0], "backlog")
	if seq == nil || seq.Kind != yaml.SequenceNode || len(seq.Content) != count {
		return nil, errors.New("move: unexpected backlog structure")
	}
	if seq.Style&yaml.FlowStyle != 0 || seq.Content[moved].Style&yaml.FlowStyle != 0 {
		return nil, errors.New("move: the backlog or the story is written in flow style ({…}); move it by hand")
	}
	return seq, nil
}

type span struct{ start, end int }

// blockSpans returns, per item, the byte range of its block.
func blockSpans(src []byte, seq *yaml.Node) []span {
	dashIndent := seq.Column - 1
	spans := make([]span, len(seq.Content))
	floor := 0
	for i, item := range seq.Content {
		first := itemStart(src, item)
		end := itemEnd(src, first, dashIndent)
		spans[i] = span{leadingComments(src, first, floor), end}
		floor = end
	}
	return spans
}

// itemStart is the offset of the line holding the item's dash.
func itemStart(src []byte, item *yaml.Node) int {
	pos := offsetOf(src, item.Line, 1)
	if !bytes.HasPrefix(bytes.TrimSpace(src[pos:lineEnd(src, pos)]), []byte("-")) {
		if p := prevLine(src, pos); p >= 0 {
			return p
		}
	}
	return pos
}

func prevLine(src []byte, pos int) int {
	if pos <= 0 {
		return -1
	}
	return bytes.LastIndexByte(src[:pos-1], '\n') + 1
}

func nextOrEnd(src []byte, pos int) int {
	if n := nextLine(src, pos); n >= 0 {
		return n
	}
	return len(src)
}

// itemEnd is the offset just past the item's last line: lines deeper than
// the dash belong to it; blank lines only count when more of it follows.
func itemEnd(src []byte, first, dashIndent int) int {
	end := nextOrEnd(src, first)
	for pos := end; pos < len(src); pos = nextOrEnd(src, pos) {
		line := src[pos:lineEnd(src, pos)]
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		if indentOf(line) <= dashIndent {
			break
		}
		end = nextOrEnd(src, pos)
	}
	return end
}

// leadingComments extends the block upwards over the comment lines directly
// above the item, never past floor (the end of the previous block).
func leadingComments(src []byte, first, floor int) int {
	start := first
	for {
		p := prevLine(src, start)
		if p < floor || p < 0 || !bytes.HasPrefix(bytes.TrimSpace(src[p:lineEnd(src, p)]), []byte("#")) {
			return start
		}
		start = p
	}
}

// permuteBlocks puts block order[i] into slot i; the gaps between slots, and
// everything before and after them, are copied as they were.
func permuteBlocks(src []byte, spans []span, order []int) []byte {
	var out bytes.Buffer
	out.Write(src[:spans[0].start])
	for slot, idx := range order {
		block := src[spans[idx].start:spans[idx].end]
		out.Write(block)
		if !bytes.HasSuffix(block, []byte("\n")) {
			out.WriteByte('\n')
		}
		if slot < len(spans)-1 {
			out.Write(src[spans[slot].end:spans[slot+1].start])
		}
	}
	tail := src[spans[len(spans)-1].end:]
	out.Write(tail)
	if len(tail) == 0 && !bytes.HasSuffix(src, []byte("\n")) {
		return bytes.TrimSuffix(out.Bytes(), []byte("\n"))
	}
	return out.Bytes()
}

// verifyOrder re-reads the result and checks the items came out as planned.
func verifyOrder(out []byte, items []Item, order []int) error {
	b, err := Decode(bytes.NewReader(out))
	if err != nil {
		return fmt.Errorf("move produced an unreadable backlog: %w", err)
	}
	if len(b.Items) != len(order) {
		return errors.New("move changed the number of stories")
	}
	for slot, idx := range order {
		if b.Items[slot] != items[idx] {
			return errors.New("move could not reorder the file safely")
		}
	}
	return nil
}
