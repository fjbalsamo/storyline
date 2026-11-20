package backlog

import (
	"bytes"
	"strings"
	"testing"
)

func str(s string) *string { return &s }

func decodeItem(t *testing.T, src []byte, id string) Item {
	t.Helper()
	b, err := Decode(bytes.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}
	it, ok := b.Find(id)
	if !ok {
		t.Fatalf("%s missing", id)
	}
	return *it
}

func TestApplyEditsValueStyles(t *testing.T) {
	const head = "version: 1\nbacklog:\n  - id: US-001\n"
	const tail = "    so_that: c\n    status: todo\n  - id: US-002\n    as: z\n    want: z\n    so_that: z\n    status: todo\n"
	tests := []struct {
		name string
		want string // the original `want` lines
	}{
		{"plain", "    want: b\n"},
		{"plain with comment", "    want: b   # keep me\n"},
		{"double quoted", "    want: \"b: x\"\n"},
		{"single quoted", "    want: 'it''s b'\n"},
		{"plain multi-line", "    want: first\n      second\n"},
		{"double quoted multi-line", "    want: \"first\n      second\"\n"},
		{"literal block", "    want: |\n      first\n      second\n"},
		{"folded block", "    want: >-\n      first\n      second\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := []byte(head + "    as: a\n" + tt.want + tail)
			out, changed, err := ApplyEdits(src, "US-001", Edits{Want: str("new: value # x")})
			if err != nil {
				t.Fatal(err)
			}
			if len(changed) != 1 || changed[0] != "want" {
				t.Errorf("changed = %v", changed)
			}
			if got := decodeItem(t, out, "US-001"); got.Want != "new: value # x" || got.As != "a" || got.SoThat != "c" {
				t.Errorf("item = %+v", got)
			}
			if got := decodeItem(t, out, "US-002"); got.Want != "z" {
				t.Errorf("neighbour changed: %+v", got)
			}
			if strings.Contains(tt.name, "comment") && !strings.Contains(string(out), "# keep me") {
				t.Error("trailing comment lost")
			}
			if !strings.HasPrefix(string(out), head+"    as: a\n") || !strings.HasSuffix(string(out), tail) {
				t.Errorf("text around the value changed:\n%s", out)
			}
		})
	}
}

func TestApplyEditsErrors(t *testing.T) {
	src := []byte("version: 1\nbacklog:\n  - id: US-001\n    as: a\n    want: b\n    so_that: c\n    status: todo\n")
	if _, _, err := ApplyEdits(src, "US-009", Edits{As: str("x")}); err == nil || !strings.Contains(err.Error(), "story US-009 not found") {
		t.Errorf("err = %v", err)
	}
	flow := []byte("version: 1\nbacklog:\n  - {id: US-001, as: a, want: b, so_that: c, status: todo}\n")
	if _, _, err := ApplyEdits(flow, "US-001", Edits{As: str("x")}); err == nil {
		t.Error("expected error for flow style")
	}
}

func TestApplyEditsSameValueIsUntouched(t *testing.T) {
	src := []byte("version: 1\nbacklog:\n  - id: US-001\n    as: \"a\"\n    want: b\n    so_that: c\n    status: todo\n")
	out, changed, err := ApplyEdits(src, "US-001", Edits{As: str("a")})
	if err != nil || len(changed) != 0 || !bytes.Equal(out, src) {
		t.Errorf("out=%q changed=%v err=%v", out, changed, err)
	}
}

func TestCheckEditable(t *testing.T) {
	for _, s := range []Status{StatusTodo, StatusRefining, StatusReady, StatusInProgress} {
		if err := CheckEditable(s); err != nil {
			t.Errorf("%s: %v", s, err)
		}
	}
	for _, s := range []Status{StatusDone, StatusDiscarded} {
		want := string(s) + " stories are history and can't be edited"
		if err := CheckEditable(s); err == nil || err.Error() != want {
			t.Errorf("%s: err = %v", s, err)
		}
	}
}
