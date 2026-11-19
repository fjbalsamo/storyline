package story

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const goodStory = `# US-001: Title

### AC-1.1: a
- Given x
- When y
- Then z
`

func write(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCheck(t *testing.T) {
	const path = "storyline/active/US-001-x/story.md"
	tests := []struct {
		name, status, storyPath, story, tasks string
		wantErr                               []string // substrings; empty = valid
	}{
		{"valid", "ready", path, goodStory, "- [ ] 1. (AC-1.1) t\n", nil},
		{"no tasks file", "in-progress", path, goodStory, "", nil},
		{"id mismatch", "ready", path, strings.Replace(goodStory, "US-001", "US-002", 1), "",
			[]string{"US-001:", "story ID US-002 does not match"}},
		{"no criteria", "ready", path, "# US-001: T\n", "", []string{"0 acceptance criteria"}},
		{"gap", "ready", path, goodStory + "\n### AC-1.3: c\n- Given a\n- When b\n- Then c\n", "",
			[]string{"story.md:8:", "should be AC-1.2"}},
		{"missing then", "ready", path, strings.Replace(goodStory, "- Then z\n", "", 1), "",
			[]string{"story.md:3:", "missing Then"}},
		{"too many tasks", "ready", path, goodStory,
			strings.Repeat("- [ ] 1. (AC-1.1) t\n", 9), []string{"9 tasks (maximum 8)"}},
		{"task numbering", "ready", path, goodStory, "- [ ] 2. (AC-1.1) t\n", []string{"tasks.md:1:", "expected 1"}},
		{"task no ref", "ready", path, goodStory, "- [ ] 1. t\n", []string{"references no criterion"}},
		{"task unknown ref", "ready", path, goodStory, "- [ ] 1. (AC-1.2) t\n", []string{"AC-1.2, which is not in story.md"}},
		{"wrong location active", "done", path, goodStory, "", []string{"requires the story under storyline/done/"}},
		{"wrong location done", "ready", "storyline/done/2026-01-01-US-001-x/story.md", goodStory, "",
			[]string{"requires the story under storyline/active/"}},
		{"missing file", "ready", "storyline/active/nope/story.md", "", "", []string{"US-001:", "open story"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			if tt.story != "" {
				write(t, root, tt.storyPath, tt.story)
			}
			if tt.tasks != "" {
				write(t, root, filepath.Join(filepath.Dir(tt.storyPath), "tasks.md"), tt.tasks)
			}
			err := Check(root, "US-001", tt.status, tt.storyPath)
			if len(tt.wantErr) == 0 {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("expected error")
			}
			for _, w := range tt.wantErr {
				if !strings.Contains(err.Error(), w) {
					t.Errorf("error %q missing %q", err, w)
				}
			}
		})
	}
}
