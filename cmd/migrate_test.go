package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fjbalsamo/storyline/internal/backlog"
)

const oldBacklog = `# yaml-language-server: $schema=x
version: 1
backlog:
  # a user comment that must survive
  - id: US-001
    as: a
    want: b
    so_that: c
    status: done
    story: stories/done/2026-11-19-US-001-x/story.md
  - id: US-002
    as: a
    want: b
    so_that: c
    status: in-progress
    story: stories/active/US-002-y/story.md
  - id: US-003
    as: a
    want: b
    so_that: c
    status: todo
`

func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func storyFor(id string) string {
	return "# " + id + ": T\n\n### AC-" + id[len(id)-1:] + ".1: a\n- Given x\n- When y\n- Then z\n"
}

func oldProject(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeFile(t, root, "stories/backlog.yaml", oldBacklog)
	writeFile(t, root, "stories/done/2026-11-19-US-001-x/story.md", storyFor("US-001"))
	writeFile(t, root, "stories/active/US-002-y/story.md", storyFor("US-002"))
	writeFile(t, root, "stories/living/bookings.md", "living")
	return root
}

func exists(p string) bool { _, err := os.Stat(p); return err == nil }

func TestMigrate(t *testing.T) {
	root := oldProject(t)
	if err := migrate(root); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(filepath.Join(root, "storyline/backlog.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"story: storyline/done/2026-11-19-US-001-x/story.md",
		"story: storyline/active/US-002-y/story.md",
		"# a user comment that must survive",
	} {
		if !strings.Contains(string(got), want) {
			t.Errorf("backlog missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(string(got), "stories/") {
		t.Errorf("backlog still mentions stories/:\n%s", got)
	}
	for _, p := range []string{
		"storyline/done/2026-11-19-US-001-x/story.md", "storyline/active/US-002-y/story.md", "storyline/living/bookings.md",
	} {
		if !exists(filepath.Join(root, p)) {
			t.Errorf("%s missing", p)
		}
	}
	if exists(filepath.Join(root, "stories")) {
		t.Error("empty stories/ was not removed")
	}
	b, err := backlog.Load(filepath.Join(root, "storyline/backlog.yaml"))
	if err != nil || validateAll(b, root) != nil {
		t.Errorf("migrated backlog invalid: %v", err)
	}
}

func TestMigrateTargetExists(t *testing.T) {
	root := oldProject(t)
	writeFile(t, root, "storyline/other.txt", "x")
	err := migrate(root)
	if err == nil || !strings.Contains(err.Error(), "storyline/ already exists") {
		t.Fatalf("err = %v", err)
	}
	if !exists(filepath.Join(root, "stories/backlog.yaml")) {
		t.Error("old layout was modified")
	}
}

func TestMigrateNoOldBacklog(t *testing.T) {
	err := migrate(t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "stories/backlog.yaml") {
		t.Fatalf("err = %v", err)
	}
}

func TestMigrateKeepsOtherFiles(t *testing.T) {
	root := oldProject(t)
	writeFile(t, root, "stories/Button.stories.tsx", "export default {}")
	if err := migrate(root); err != nil {
		t.Fatal(err)
	}
	if !exists(filepath.Join(root, "stories/Button.stories.tsx")) {
		t.Error("Storybook file was moved or removed")
	}
	for _, name := range []string{"backlog.yaml", "active", "done", "living"} {
		if exists(filepath.Join(root, "stories", name)) {
			t.Errorf("stories/%s was left behind", name)
		}
	}
}

func TestMigrateRollsBackWhenInvalid(t *testing.T) {
	root := oldProject(t)
	writeFile(t, root, "stories/Button.stories.tsx", "export default {}")
	if err := os.Remove(filepath.Join(root, "stories/active/US-002-y/story.md")); err != nil {
		t.Fatal(err)
	}

	err := migrate(root)
	if err == nil || !strings.Contains(err.Error(), "US-002") {
		t.Fatalf("err = %v", err)
	}
	if exists(filepath.Join(root, "storyline")) {
		t.Error("storyline/ was not removed")
	}
	got, readErr := os.ReadFile(filepath.Join(root, "stories/backlog.yaml"))
	if readErr != nil || string(got) != oldBacklog {
		t.Errorf("original backlog not restored (err %v):\n%s", readErr, got)
	}
	for _, p := range []string{"stories/done/2026-11-19-US-001-x/story.md", "stories/living/bookings.md", "stories/Button.stories.tsx"} {
		if !exists(filepath.Join(root, p)) {
			t.Errorf("%s not restored", p)
		}
	}
	if !exists(filepath.Join(root, "stories/active")) {
		t.Error("stories/active not restored")
	}
}

func TestOldLayoutDetection(t *testing.T) {
	root := oldProject(t)
	err := backlog.CheckLayout(root)
	if !errors.Is(err, backlog.ErrOldLayout) {
		t.Fatalf("err = %v", err)
	}
	if err.Error() != "found the old layout (stories/). Run: storyline migrate" {
		t.Errorf("message = %q", err)
	}

	if err := migrate(root); err != nil {
		t.Fatal(err)
	}
	if err := backlog.CheckLayout(root); err != nil {
		t.Errorf("after migrate: %v", err)
	}
	if err := backlog.CheckLayout(t.TempDir()); err != nil {
		t.Errorf("empty project: %v", err)
	}
}
