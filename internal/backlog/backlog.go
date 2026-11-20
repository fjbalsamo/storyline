// Package backlog models the storyline backlog file (storyline/backlog.yaml):
// an ordered list of one-line user stories. Order is priority — the first
// item that is not done or discarded is the next one to work on.
package backlog

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	DefaultPath    = Dir + "/backlog.yaml"
	SchemaURL      = "https://raw.githubusercontent.com/fjbalsamo/storyline/main/schema/backlog.schema.json"
	CurrentVersion = 1
)

type Status string

const (
	StatusTodo       Status = "todo"
	StatusRefining   Status = "refining"
	StatusReady      Status = "ready"
	StatusInProgress Status = "in-progress"
	StatusDone       Status = "done"
	StatusDiscarded  Status = "discarded"
)

var statuses = map[Status]struct{}{
	StatusTodo: {}, StatusRefining: {}, StatusReady: {},
	StatusInProgress: {}, StatusDone: {}, StatusDiscarded: {},
}

func (s Status) Valid() bool {
	_, ok := statuses[s]
	return ok
}

// Open reports whether the story still has work pending.
func (s Status) Open() bool {
	return s != StatusDone && s != StatusDiscarded
}

// Ref says where a story came from, e.g. an issue tracker key (TKT-001).
// It is free single-line text, and several stories may share one.
type Ref string

// UnmarshalYAML rejects a ref that is present but empty or spans lines. (A
// bare `ref:` is a null and counts as absent.)
func (r *Ref) UnmarshalYAML(n *yaml.Node) error {
	var s string
	if err := n.Decode(&s); err != nil {
		return fmt.Errorf("line %d: ref must be text: %w", n.Line, err)
	}
	if strings.TrimSpace(s) == "" {
		return fmt.Errorf("line %d: ref must not be empty", n.Line)
	}
	if strings.ContainsAny(s, "\r\n") {
		return fmt.Errorf("line %d: ref must be a single line", n.Line)
	}
	*r = Ref(s)
	return nil
}

type Item struct {
	ID     string `yaml:"id"`
	As     string `yaml:"as"`
	Want   string `yaml:"want"`
	SoThat string `yaml:"so_that"`
	Ref    Ref    `yaml:"ref,omitempty"`
	Status Status `yaml:"status"`
	Story  string `yaml:"story,omitempty"`
}

// RequiresStory reports whether the item must point to a refined story.md.
func (i Item) RequiresStory() bool {
	switch i.Status {
	case StatusReady, StatusInProgress, StatusDone:
		return true
	}
	return false
}

type Backlog struct {
	Version int    `yaml:"version"`
	Items   []Item `yaml:"backlog"`
}

var idPattern = regexp.MustCompile(`^US-(\d{3,})$`)

// New returns an empty backlog at the current schema version.
func New() *Backlog {
	return &Backlog{Version: CurrentVersion, Items: []Item{}}
}

func Load(path string) (*Backlog, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open backlog: %w", err)
	}
	defer f.Close()
	return Decode(f)
}

// Decode parses a backlog, rejecting unknown keys so typos (e.g. "sothat")
// fail loudly instead of being silently dropped.
func Decode(r io.Reader) (*Backlog, error) {
	dec := yaml.NewDecoder(r)
	dec.KnownFields(true)

	var b Backlog
	if err := dec.Decode(&b); err != nil {
		if errors.Is(err, io.EOF) {
			return New(), nil
		}
		return nil, fmt.Errorf("decode backlog: %w", err)
	}
	if b.Items == nil {
		b.Items = []Item{}
	}
	return &b, nil
}

func (b *Backlog) Encode(w io.Writer) error {
	if _, err := fmt.Fprintf(w, "# yaml-language-server: $schema=%s\n", SchemaURL); err != nil {
		return fmt.Errorf("encode backlog: %w", err)
	}
	enc := yaml.NewEncoder(w)
	enc.SetIndent(2)
	if err := enc.Encode(b); err != nil {
		return fmt.Errorf("encode backlog: %w", err)
	}
	return enc.Close()
}

// Save writes the backlog atomically (temp file + rename) so an interrupted
// write never leaves a truncated backlog behind.
func (b *Backlog) Save(path string) error {
	var buf bytes.Buffer
	if err := b.Encode(&buf); err != nil {
		return err
	}
	return WriteFile(path, buf.Bytes())
}

// WriteFile atomically replaces the backlog file at path with data.
func WriteFile(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("save backlog: %w", err)
	}

	tmp, err := os.CreateTemp(dir, ".backlog-*.yaml")
	if err != nil {
		return fmt.Errorf("save backlog: %w", err)
	}
	defer os.Remove(tmp.Name()) // no-op once renamed

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("save backlog: %w", err)
	}
	if err := tmp.Chmod(0o644); err != nil {
		tmp.Close()
		return fmt.Errorf("save backlog: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("save backlog: %w", err)
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		return fmt.Errorf("save backlog: %w", err)
	}
	return nil
}

// Validate checks rules the JSON Schema can't express (unique IDs) plus the
// structural ones, so the CLI works without the schema at hand.
func (b *Backlog) Validate() error {
	var errs []error
	if b.Version != CurrentVersion {
		errs = append(errs, fmt.Errorf("unsupported version %d (expected %d)", b.Version, CurrentVersion))
	}

	seen := make(map[string]int, len(b.Items))
	for i, it := range b.Items {
		pos := fmt.Sprintf("backlog[%d]", i)

		switch prev, dup := seen[it.ID]; {
		case !idPattern.MatchString(it.ID):
			errs = append(errs, fmt.Errorf("%s: invalid id %q (expected US-NNN)", pos, it.ID))
		case dup:
			errs = append(errs, fmt.Errorf("%s: duplicate id %q (first at backlog[%d])", pos, it.ID, prev))
		default:
			seen[it.ID] = i
		}

		for _, f := range []struct{ name, value string }{
			{"as", it.As}, {"want", it.Want}, {"so_that", it.SoThat},
		} {
			if f.value == "" {
				errs = append(errs, fmt.Errorf("%s (%s): %q is required", pos, it.ID, f.name))
			}
		}

		if r := string(it.Ref); r != strings.TrimSpace(r) || strings.ContainsAny(r, "\r\n") {
			errs = append(errs, fmt.Errorf("%s (%s): \"ref\" must be trimmed, single-line text", pos, it.ID))
		}

		if !it.Status.Valid() {
			errs = append(errs, fmt.Errorf("%s (%s): invalid status %q", pos, it.ID, it.Status))
		} else if it.RequiresStory() && it.Story == "" {
			errs = append(errs, fmt.Errorf("%s (%s): status %q requires a story path", pos, it.ID, it.Status))
		}
	}
	return errors.Join(errs...)
}

func (b *Backlog) Find(id string) (*Item, bool) {
	for i := range b.Items {
		if b.Items[i].ID == id {
			return &b.Items[i], true
		}
	}
	return nil, false
}

// Next returns the highest-priority open item.
func (b *Backlog) Next() (*Item, bool) {
	for i := range b.Items {
		if b.Items[i].Status.Open() {
			return &b.Items[i], true
		}
	}
	return nil, false
}

// NextID returns the ID following the highest existing one. IDs are never
// reused, even after a story is discarded.
func (b *Backlog) NextID() string {
	highest := 0
	for _, it := range b.Items {
		if m := idPattern.FindStringSubmatch(it.ID); m != nil {
			if n, err := strconv.Atoi(m[1]); err == nil && n > highest {
				highest = n
			}
		}
	}
	return fmt.Sprintf("US-%03d", highest+1)
}

// Add appends a new todo item at the bottom (lowest priority).
func (b *Backlog) Add(as, want, soThat string) Item {
	return b.AddWithRef(as, want, soThat, "")
}

// AddWithRef is Add for a story that came from somewhere else (ref may be "").
func (b *Backlog) AddWithRef(as, want, soThat, ref string) Item {
	it := Item{ID: b.NextID(), As: as, Want: want, SoThat: soThat, Ref: Ref(ref), Status: StatusTodo}
	b.Items = append(b.Items, it)
	return it
}
