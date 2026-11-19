package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fjbalsamo/storyline/internal/backlog"
	"github.com/fjbalsamo/storyline/internal/story"
	"github.com/fjbalsamo/storyline/internal/trace"
)

const traceStory = `# US-001: Cancel bookings

### AC-1.1: cancel own
- Given a
- When b
- Then c

### AC-1.2: cancel late
- Given a
- When b
- Then c

### AC-1.4: comment only
- Given a
- When b
- Then c
`

func TestRenderTrace(t *testing.T) {
	path := filepath.Join(t.TempDir(), "story.md")
	if err := os.WriteFile(path, []byte(traceStory), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := story.ParseStory(path)
	if err != nil {
		t.Fatal(err)
	}
	idx, err := trace.Scan("../internal/trace/testdata/repo")
	if err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	uncovered := renderTrace(&out, s, idx)

	if uncovered != 1 {
		t.Errorf("uncovered = %d, want 1 (AC-1.4 only appears in comments)", uncovered)
	}
	for _, want := range []string{
		"US-001: Cancel bookings",
		"✓ AC-1.1  cancel own    go/booking_test.go:7",
		"✓ AC-1.2  cancel late   web/__tests__/helper.js:4, web/cancel.test.ts:2",
		"✗ AC-1.4  comment only  no tests",
		"2/3 criteria covered",
	} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("output missing %q:\n%s", want, out.String())
		}
	}
}

func TestTraceScope(t *testing.T) {
	b := &backlog.Backlog{Items: []backlog.Item{
		{ID: "US-001", Status: backlog.StatusInProgress, Story: "stories/active/US-001-x/story.md"},
		{ID: "US-002", Status: backlog.StatusDone, Story: "stories/done/d-US-002-x/story.md"},
		{ID: "US-003", Status: backlog.StatusTodo},
	}}
	for _, tt := range []struct {
		name    string
		args    []string
		all     bool
		want    int
		wantErr string
	}{
		{"default: in-progress", nil, false, 1, ""},
		{"all with stories", nil, true, 2, ""},
		{"one id", []string{"US-002"}, false, 1, ""},
		{"id without story", []string{"US-003"}, false, 0, "no story.md yet"},
		{"unknown id", []string{"US-009"}, false, 0, "not found"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			items, err := traceScope(b, tt.args, tt.all)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v", err)
				}
				return
			}
			if err != nil || len(items) != tt.want {
				t.Errorf("items = %d, err = %v, want %d", len(items), err, tt.want)
			}
		})
	}
}
