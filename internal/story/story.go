// Package story parses the per-story files written by /story:refine and
// /story:slice: story.md (acceptance criteria) and tasks.md (checklist).
// Parsing is line-based and does no I/O beyond reading the given file.
package story

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// Criterion is one "### AC-<n>.<m>: <name>" block.
type Criterion struct {
	ID    string
	Name  string
	Line  int
	Given bool
	When  bool
	Then  bool
}

// Story is a parsed story.md.
type Story struct {
	Path     string
	ID       string // e.g. US-012
	Title    string
	Line     int // line of the title heading
	Criteria []Criterion
}

// Task is one checklist line of tasks.md.
type Task struct {
	Number int
	Done   bool
	ACs    []string
	Text   string
	Line   int
}

// Tasks is a parsed tasks.md.
type Tasks struct {
	Path  string
	Tasks []Task
}

var (
	titleRe = regexp.MustCompile(`^#\s+(US-\d{3,}):\s*(.*?)\s*$`)
	acRe    = regexp.MustCompile(`^###\s+(AC-\d+\.\d+):\s*(.*?)\s*$`)
	gwtRe   = regexp.MustCompile(`(?i)^\s*[-*]\s+(given|when|then)\b`)
	taskRe  = regexp.MustCompile(`^\s*-\s+\[([ xX])\]\s+(\d+)\.\s*(?:\(([^)]*)\)\s*)?(.*?)\s*$`)
	acIDRe  = regexp.MustCompile(`^AC-\d+\.\d+$`)
	usNumRe = regexp.MustCompile(`^US-(\d+)$`)
)

// ACPrefix returns the criterion ID prefix for a story ID
// (US-012 → "AC-12."), or "" if the ID is malformed.
func ACPrefix(storyID string) string {
	m := usNumRe.FindStringSubmatch(storyID)
	if m == nil {
		return ""
	}
	n, _ := strconv.Atoi(m[1])
	return fmt.Sprintf("AC-%d.", n)
}

// ParseStory reads and parses a story.md.
func ParseStory(path string) (*Story, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open story: %w", err)
	}
	defer f.Close()

	s := &Story{Path: path}
	var cur *Criterion
	inFence := false
	sc := bufio.NewScanner(f)
	for n := 1; sc.Scan(); n++ {
		line := sc.Text()
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		if m := titleRe.FindStringSubmatch(line); m != nil && s.ID == "" {
			s.ID, s.Title, s.Line = m[1], m[2], n
			continue
		}
		if m := acRe.FindStringSubmatch(line); m != nil {
			s.Criteria = append(s.Criteria, Criterion{ID: m[1], Name: m[2], Line: n})
			cur = &s.Criteria[len(s.Criteria)-1]
			continue
		}
		if strings.HasPrefix(line, "#") {
			cur = nil // any other heading ends the criterion
			continue
		}
		if cur != nil {
			if m := gwtRe.FindStringSubmatch(line); m != nil {
				switch strings.ToLower(m[1]) {
				case "given":
					cur.Given = true
				case "when":
					cur.When = true
				case "then":
					cur.Then = true
				}
			}
		}
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	if s.ID == "" {
		return nil, fmt.Errorf("%s:1: missing title heading \"# US-NNN: <title>\"", path)
	}
	return s, nil
}

// ParseTasks reads and parses a tasks.md. Lines that are not checklist
// items are ignored; malformed checklist items are reported with line numbers.
func ParseTasks(path string) (*Tasks, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open tasks: %w", err)
	}
	defer f.Close()

	t := &Tasks{Path: path}
	var errs []error
	sc := bufio.NewScanner(f)
	for n := 1; sc.Scan(); n++ {
		line := sc.Text()
		if !strings.Contains(line, "[") || !strings.HasPrefix(strings.TrimSpace(line), "- [") {
			continue
		}
		m := taskRe.FindStringSubmatch(line)
		if m == nil {
			errs = append(errs, fmt.Errorf("%s:%d: malformed task, expected \"- [ ] <n>. (AC-x.y) <text>\"", path, n))
			continue
		}
		num, _ := strconv.Atoi(m[2])
		task := Task{Number: num, Done: m[1] != " ", Text: m[4], Line: n}
		if m[3] != "" {
			for _, p := range strings.Split(m[3], ",") {
				id := strings.TrimSpace(p)
				if !acIDRe.MatchString(id) {
					errs = append(errs, fmt.Errorf("%s:%d: invalid criterion reference %q", path, n, id))
					continue
				}
				task.ACs = append(task.ACs, id)
			}
		}
		t.Tasks = append(t.Tasks, task)
	}
	if err := sc.Err(); err != nil {
		errs = append(errs, fmt.Errorf("read %s: %w", path, err))
	}
	if len(errs) > 0 {
		return t, errors.Join(errs...)
	}
	return t, nil
}
