package backlog

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

const sample = `version: 1
backlog:
  - id: US-001
    as: club player
    want: to book a court for a free time slot
    so_that: I don't have to call the club
    status: done
    story: stories/done/2026-11-18-US-001-book-court/story.md
  - id: US-002
    as: admin
    want: to block courts for maintenance
    so_that: nobody books an unusable court
    status: todo
`

func mustDecode(t *testing.T, s string) *Backlog {
	t.Helper()
	b, err := Decode(strings.NewReader(s))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	return b
}

func TestDecodeValid(t *testing.T) {
	b := mustDecode(t, sample)
	if err := b.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}
	if len(b.Items) != 2 || b.Items[1].SoThat != "nobody books an unusable court" {
		t.Fatalf("unexpected items: %+v", b.Items)
	}
}

func TestDecodeEmpty(t *testing.T) {
	b := mustDecode(t, "")
	if b.Version != CurrentVersion || len(b.Items) != 0 {
		t.Fatalf("expected empty backlog, got %+v", b)
	}
}

func TestDecodeRejectsUnknownFields(t *testing.T) {
	_, err := Decode(strings.NewReader("version: 1\nbacklog:\n  - id: US-001\n    sothat: typo\n"))
	if err == nil {
		t.Fatal("expected error for unknown field")
	}
}

func TestValidate(t *testing.T) {
	item := func(id string, st Status, story string) Item {
		return Item{ID: id, As: "a", Want: "w", SoThat: "s", Status: st, Story: story}
	}
	tests := []struct {
		name    string
		items   []Item
		wantErr string
	}{
		{"ok", []Item{item("US-001", StatusTodo, "")}, ""},
		{"bad id", []Item{item("US-1", StatusTodo, "")}, "invalid id"},
		{"duplicate", []Item{item("US-001", StatusTodo, ""), item("US-001", StatusTodo, "")}, "duplicate id"},
		{"bad status", []Item{item("US-001", "wip", "")}, "invalid status"},
		{"missing story", []Item{item("US-001", StatusReady, "")}, "requires a story path"},
		{"missing field", []Item{{ID: "US-001", Status: StatusTodo}}, `"as" is required`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := (&Backlog{Version: CurrentVersion, Items: tt.items}).Validate()
			switch {
			case tt.wantErr == "" && err != nil:
				t.Fatalf("unexpected error: %v", err)
			case tt.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErr)):
				t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestNextIDAndAdd(t *testing.T) {
	b := New()
	if got := b.NextID(); got != "US-001" {
		t.Fatalf("NextID on empty = %s", got)
	}
	b.Items = append(b.Items, Item{ID: "US-007", Status: StatusDiscarded})
	it := b.Add("a", "w", "s")
	if it.ID != "US-008" || it.Status != StatusTodo {
		t.Fatalf("Add = %+v", it)
	}
}

func TestNext(t *testing.T) {
	b := mustDecode(t, sample)
	it, ok := b.Next()
	if !ok || it.ID != "US-002" {
		t.Fatalf("Next = %+v, %v", it, ok)
	}
}

func TestSaveRoundTrip(t *testing.T) {
	b := mustDecode(t, sample)
	path := filepath.Join(t.TempDir(), "stories", "backlog.yaml")
	if err := b.Save(path); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	var want, have bytes.Buffer
	_ = b.Encode(&want)
	_ = got.Encode(&have)
	if want.String() != have.String() {
		t.Fatalf("round trip mismatch:\n%s\nvs\n%s", want.String(), have.String())
	}
	if !strings.HasPrefix(have.String(), "# yaml-language-server: $schema=") {
		t.Fatal("missing schema header")
	}
}
