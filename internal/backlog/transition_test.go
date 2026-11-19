package backlog

import (
	"strings"
	"testing"
)

func TestTransition(t *testing.T) {
	all := []Status{StatusTodo, StatusRefining, StatusReady, StatusInProgress, StatusDone, StatusDiscarded}
	allowed := map[Status][]Status{
		StatusTodo:       {StatusRefining, StatusDiscarded},
		StatusRefining:   {StatusReady, StatusTodo, StatusDiscarded},
		StatusReady:      {StatusInProgress, StatusRefining, StatusDiscarded},
		StatusInProgress: {StatusReady, StatusDiscarded},
		StatusDiscarded:  {StatusTodo},
	}
	for _, from := range all {
		for _, to := range all {
			want := false
			for _, a := range allowed[from] {
				if a == to {
					want = true
				}
			}
			t.Run(string(from)+"→"+string(to), func(t *testing.T) {
				b := &Backlog{Version: 1, Items: []Item{{ID: "US-001", Status: from}}}
				got, err := b.Transition("US-001", to)
				if want != (err == nil) {
					t.Fatalf("err = %v, want ok=%v", err, want)
				}
				if got != from {
					t.Errorf("from = %s, want %s", got, from)
				}
				if want && b.Items[0].Status != to {
					t.Errorf("status = %s, want %s", b.Items[0].Status, to)
				}
				if !want && b.Items[0].Status != from {
					t.Errorf("status changed to %s on failure", b.Items[0].Status)
				}
			})
		}
	}
}

func TestTransitionMessages(t *testing.T) {
	b := &Backlog{Items: []Item{{ID: "US-001", Status: StatusDone}, {ID: "US-002", Status: StatusTodo}, {ID: "US-003", Status: StatusInProgress}}}
	for _, tt := range []struct {
		id   string
		to   Status
		want string
	}{
		{"US-001", StatusTodo, "done has no transitions"},
		{"US-002", StatusReady, "allowed from todo: refining, discarded"},
		{"US-003", StatusDone, "use `storyline done US-003`"},
		{"US-002", "bogus", "invalid status"},
		{"US-999", StatusTodo, "not found"},
	} {
		_, err := b.Transition(tt.id, tt.to)
		if err == nil || !strings.Contains(err.Error(), tt.want) {
			t.Errorf("%s → %s: err = %v, want %q", tt.id, tt.to, err, tt.want)
		}
	}
}
