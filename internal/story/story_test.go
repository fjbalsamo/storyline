package story

import (
	"strings"
	"testing"
)

func TestParseStory(t *testing.T) {
	tests := []struct {
		name    string
		file    string
		wantErr string
		check   func(t *testing.T, s *Story)
	}{
		{"valid", "testdata/valid_story.md", "", func(t *testing.T, s *Story) {
			if s.ID != "US-012" || s.Title != "Cancel a booking" {
				t.Errorf("got %q %q", s.ID, s.Title)
			}
			if len(s.Criteria) != 2 {
				t.Fatalf("criteria = %d", len(s.Criteria))
			}
			c := s.Criteria[1]
			if c.ID != "AC-12.2" || !c.Given || !c.When || !c.Then || c.Line != 12 {
				t.Errorf("criterion = %+v", c)
			}
		}},
		{"missing then", "testdata/incomplete_story.md", "", func(t *testing.T, s *Story) {
			c := s.Criteria[0]
			if !c.Given || !c.When || c.Then {
				t.Errorf("criterion = %+v", c)
			}
		}},
		{"external dependencies", "testdata/valid_story_external.md", "", func(t *testing.T, s *Story) {
			if len(s.Criteria) != 1 || s.Criteria[0].ID != "AC-20.1" {
				t.Errorf("criteria = %+v", s.Criteria)
			}
		}},
		{"no title", "testdata/no_title.md", "no_title.md:1: missing title", nil},
		{"missing file", "testdata/nope.md", "open story", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := ParseStory(tt.file)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			tt.check(t, s)
		})
	}
}

func TestParseTasks(t *testing.T) {
	tk, err := ParseTasks("testdata/valid_tasks.md")
	if err != nil {
		t.Fatal(err)
	}
	if len(tk.Tasks) != 2 {
		t.Fatalf("tasks = %d", len(tk.Tasks))
	}
	if tk.Tasks[0].Done || !tk.Tasks[1].Done || tk.Tasks[1].Number != 2 ||
		strings.Join(tk.Tasks[1].ACs, ",") != "AC-12.1,AC-12.2" || tk.Tasks[1].Text != "Fill in" {
		t.Errorf("tasks = %+v", tk.Tasks)
	}

	_, err = ParseTasks("testdata/bad_tasks.md")
	if err == nil || !strings.Contains(err.Error(), "bad_tasks.md:2:") || !strings.Contains(err.Error(), "bad_tasks.md:3:") {
		t.Errorf("err = %v", err)
	}
}

func TestACPrefix(t *testing.T) {
	for in, want := range map[string]string{"US-012": "AC-12.", "US-001": "AC-1.", "US-100": "AC-100.", "bad": ""} {
		if got := ACPrefix(in); got != want {
			t.Errorf("ACPrefix(%q) = %q, want %q", in, got, want)
		}
	}
}
