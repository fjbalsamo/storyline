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

const refDoc = `# top
version: 1
backlog:
  # first
  - id: US-001
    as: a
    want: b
    so_that: c   # why
    status: todo

  - id: US-002
    as: a
    want: b
    so_that: |
      line one
      line two
    ref: TKT-001   # origin
    status: todo
`

func TestApplyEditsRefInsertsAfterSoThat(t *testing.T) {
	out, changed, err := ApplyEdits([]byte(refDoc), "US-001", Edits{Ref: str("TKT-9")})
	if err != nil || len(changed) != 1 || changed[0] != "ref" {
		t.Fatalf("changed = %v, err = %v", changed, err)
	}
	want := strings.Replace(refDoc, "    so_that: c   # why\n", "    so_that: c   # why\n    ref: TKT-9\n", 1)
	if string(out) != want {
		t.Errorf("out:\n%s", out)
	}
}

func TestApplyEditsRefInsertsAfterAMultiLineSoThat(t *testing.T) {
	src := strings.Replace(refDoc, "    ref: TKT-001   # origin\n", "", 1)
	out, _, err := ApplyEdits([]byte(src), "US-002", Edits{Ref: str("X-1")})
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Replace(src, "      line two\n", "      line two\n    ref: X-1\n", 1)
	if string(out) != want {
		t.Errorf("out:\n%s", out)
	}
}

func TestApplyEditsRefReplaceKeepsTheComment(t *testing.T) {
	out, changed, err := ApplyEdits([]byte(refDoc), "US-002", Edits{Ref: str("TKT-2")})
	if err != nil || len(changed) != 1 {
		t.Fatal(changed, err)
	}
	if want := strings.Replace(refDoc, "TKT-001", "TKT-2", 1); string(out) != want {
		t.Errorf("out:\n%s", out)
	}
}

func TestApplyEditsRefRemove(t *testing.T) {
	out, changed, err := ApplyEdits([]byte(refDoc), "US-002", Edits{RemoveRef: true})
	if err != nil || len(changed) != 1 || changed[0] != "ref" {
		t.Fatal(changed, err)
	}
	if want := strings.Replace(refDoc, "    ref: TKT-001   # origin\n", "", 1); string(out) != want {
		t.Errorf("out:\n%s", out)
	}
}

func TestApplyEditsRefNoChange(t *testing.T) {
	for name, tt := range map[string]struct {
		id string
		e  Edits
	}{
		"remove when absent": {"US-001", Edits{RemoveRef: true}},
		"same value":         {"US-002", Edits{Ref: str("TKT-001")}},
	} {
		out, changed, err := ApplyEdits([]byte(refDoc), tt.id, tt.e)
		if err != nil || len(changed) != 0 || string(out) != refDoc {
			t.Errorf("%s: changed = %v, err = %v", name, changed, err)
		}
	}
}

func TestApplyEditsRefWithText(t *testing.T) {
	out, changed, err := ApplyEdits([]byte(refDoc), "US-001", Edits{Want: str("new"), Ref: str("T-1")})
	if err != nil || strings.Join(changed, ",") != "want,ref" {
		t.Fatalf("changed = %v, err = %v", changed, err)
	}
	it := decodeItem(t, out, "US-001")
	if it.Want != "new" || it.Ref != "T-1" || it.SoThat != "c" {
		t.Errorf("item = %+v", it)
	}
}

func TestEditsTouchesText(t *testing.T) {
	if (Edits{Ref: str("x")}).TouchesText() || (Edits{RemoveRef: true}).TouchesText() {
		t.Error("a ref-only edit does not touch the wording")
	}
	if !(Edits{Want: str("x")}).TouchesText() {
		t.Error("want is wording")
	}
}
