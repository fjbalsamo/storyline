package backlog

import (
	"fmt"
	"strings"
)

// transitions is the workflow state machine. done is terminal and is
// reached only through `storyline done`, so no row lists it as a target.
var transitions = map[Status][]Status{
	StatusTodo:       {StatusRefining, StatusDiscarded},
	StatusRefining:   {StatusReady, StatusTodo, StatusDiscarded},
	StatusReady:      {StatusInProgress, StatusRefining, StatusDiscarded},
	StatusInProgress: {StatusReady, StatusDiscarded},
	StatusDiscarded:  {StatusTodo},
	StatusDone:       {},
}

// Allowed returns the statuses reachable from s.
func Allowed(s Status) []Status { return transitions[s] }

// CanTransition reports whether from → to is allowed.
func CanTransition(from, to Status) bool {
	for _, s := range transitions[from] {
		if s == to {
			return true
		}
	}
	return false
}

// Transition moves the item to a new status, enforcing the state machine.
// It does not validate the rest of the backlog; callers do that before saving.
func (b *Backlog) Transition(id string, to Status) (from Status, err error) {
	it, ok := b.Find(id)
	if !ok {
		return "", fmt.Errorf("%s not found in backlog", id)
	}
	if !to.Valid() {
		return it.Status, fmt.Errorf("invalid status %q", to)
	}
	if to == StatusDone {
		return it.Status, fmt.Errorf("cannot set %s to done: use `storyline done %s`", id, id)
	}
	from = it.Status
	if !CanTransition(from, to) {
		return from, fmt.Errorf("%s: cannot move %s → %s; %s", id, from, to, allowedMsg(from))
	}
	it.Status = to
	return from, nil
}

func allowedMsg(from Status) string {
	next := transitions[from]
	if len(next) == 0 {
		return fmt.Sprintf("%s has no transitions", from)
	}
	names := make([]string, len(next))
	for i, s := range next {
		names[i] = string(s)
	}
	return fmt.Sprintf("allowed from %s: %s", from, strings.Join(names, ", "))
}
