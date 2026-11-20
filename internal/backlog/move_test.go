package backlog

import (
	"strings"
	"testing"
)

func TestPlanMoveReferenceMayBeDone(t *testing.T) {
	items := []Item{
		{ID: "US-001", Status: StatusDone},
		{ID: "US-002", Status: StatusTodo},
		{ID: "US-003", Status: StatusTodo},
	}
	order, from, to, err := PlanMove(items, "US-003", Placement{Mode: MoveBefore, Ref: "US-001"})
	if err != nil || from != 2 || to != 0 || order[0] != 2 {
		t.Errorf("order=%v from=%d to=%d err=%v", order, from, to, err)
	}
	if _, _, _, err := PlanMove(items, "US-001", Placement{Mode: MoveTop}); err == nil ||
		err.Error() != "done stories are history and can't be moved" {
		t.Errorf("err = %v", err)
	}
}

func TestApplyMoveWithoutFinalNewline(t *testing.T) {
	src := "version: 1\nbacklog:\n  - id: US-001\n    as: a\n    want: b\n    so_that: c\n    status: todo\n" +
		"  - id: US-002\n    as: a\n    want: b\n    so_that: c\n    status: todo"
	out, _, _, err := ApplyMove([]byte(src), "US-001", Placement{Mode: MoveBottom})
	if err != nil {
		t.Fatal(err)
	}
	want := "version: 1\nbacklog:\n  - id: US-002\n    as: a\n    want: b\n    so_that: c\n    status: todo\n" +
		"  - id: US-001\n    as: a\n    want: b\n    so_that: c\n    status: todo"
	if string(out) != want {
		t.Errorf("out =\n%q\nwant\n%q", out, want)
	}
	if strings.HasSuffix(string(out), "\n") {
		t.Error("final newline appeared")
	}
}
