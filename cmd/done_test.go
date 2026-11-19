package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fjbalsamo/storyline/internal/backlog"
)

const storyMD = "# US-001: T\n\n### AC-1.1: a\n- Given x\n- When y\n- Then z\n"

func fixedClock() time.Time { return time.Date(2026, 11, 19, 10, 0, 0, 0, time.Local) }

// project builds a temp project with US-001 in the given status.
func project(t *testing.T, status backlog.Status, tasks string) (b *backlog.Backlog, blPath, root string) {
	t.Helper()
	root = t.TempDir()
	dir := filepath.Join(root, "storyline/active/US-001-x")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "story.md"), []byte(storyMD), 0o644); err != nil {
		t.Fatal(err)
	}
	if tasks != "" {
		if err := os.WriteFile(filepath.Join(dir, "tasks.md"), []byte(tasks), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	b = &backlog.Backlog{Version: 1, Items: []backlog.Item{{
		ID: "US-001", As: "a", Want: "b", SoThat: "c", Status: status, Story: "storyline/active/US-001-x/story.md",
	}}}
	blPath = filepath.Join(root, "storyline/backlog.yaml")
	if err := b.Save(blPath); err != nil {
		t.Fatal(err)
	}
	return b, blPath, root
}

func TestCloseStory(t *testing.T) {
	const wantDir = "storyline/done/2026-11-19-US-001-x"
	tests := []struct {
		name    string
		status  backlog.Status
		tasks   string
		force   bool
		wantErr string
		warning string
	}{
		{"ok", backlog.StatusInProgress, "- [x] 1. (AC-1.1) t\n", false, "", ""},
		{"no tasks file", backlog.StatusInProgress, "", false, "no tasks.md: run the slice command first", ""},
		{"no tasks file forced", backlog.StatusInProgress, "", true, "", "without tasks.md"},
		{"not in progress", backlog.StatusReady, "", false, "only in-progress", ""},
		{"unchecked", backlog.StatusInProgress, "- [x] 1. (AC-1.1) a\n- [ ] 2. (AC-1.1) pending\n", false, "1 unchecked task(s)", ""},
		{"unchecked forced", backlog.StatusInProgress, "- [ ] 1. (AC-1.1) pending\n", true, "", "pending"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, blPath, root := project(t, tt.status, tt.tasks)
			var warn bytes.Buffer
			got, err := closeStory(b, blPath, "US-001", tt.force, fixedClock, &warn)

			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want %q", err, tt.wantErr)
				}
				if _, statErr := os.Stat(filepath.Join(root, "storyline/active/US-001-x")); statErr != nil {
					t.Error("folder moved despite failure")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != wantDir+"/story.md" {
				t.Errorf("path = %s", got)
			}
			if _, err := os.Stat(filepath.Join(root, wantDir, "story.md")); err != nil {
				t.Errorf("story not moved: %v", err)
			}
			if _, err := os.Stat(filepath.Join(root, "storyline/active/US-001-x")); !os.IsNotExist(err) {
				t.Error("old folder still exists")
			}
			saved, err := backlog.Load(blPath)
			if err != nil {
				t.Fatal(err)
			}
			if it := saved.Items[0]; it.Status != backlog.StatusDone || it.Story != got {
				t.Errorf("saved item = %+v", it)
			}
			if !strings.Contains(warn.String(), tt.warning) {
				t.Errorf("warning = %q, want %q", warn.String(), tt.warning)
			}
		})
	}
}

func TestCloseStoryRollsBackOnSaveFailure(t *testing.T) {
	b, blPath, root := project(t, backlog.StatusInProgress, "- [x] 1. (AC-1.1) t\n")
	// A directory where the backlog file should be makes the rename in Save fail.
	if err := os.Remove(blPath); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(blPath, 0o755); err != nil {
		t.Fatal(err)
	}

	_, err := closeStory(b, blPath, "US-001", false, fixedClock, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error")
	}
	if _, statErr := os.Stat(filepath.Join(root, "storyline/active/US-001-x/story.md")); statErr != nil {
		t.Errorf("folder not moved back: %v", statErr)
	}
	if it := b.Items[0]; it.Status != backlog.StatusInProgress || it.Story != "storyline/active/US-001-x/story.md" {
		t.Errorf("in-memory item not restored: %+v", it)
	}
}

func TestCloseStoryRejectsNonActiveFolder(t *testing.T) {
	b, blPath, _ := project(t, backlog.StatusInProgress, "")
	b.Items[0].Story = "elsewhere/US-001-x/story.md"
	_, err := closeStory(b, blPath, "US-001", false, fixedClock, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "not under storyline/active/") {
		t.Errorf("err = %v", err)
	}
}
