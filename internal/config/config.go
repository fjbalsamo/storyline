// Package config models storyline/config.yaml: the repo's own name and the
// related repos it depends on. storyline only reads it to give the AI
// context; it never checks that the related repos exist and never blocks.
package config

import (
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	Path           = "storyline/config.yaml"
	SchemaURL      = "https://raw.githubusercontent.com/fjbalsamo/storyline/main/schema/config.schema.json"
	CurrentVersion = 1
)

type Config struct {
	Version  int      `yaml:"version"`
	Metadata Metadata `yaml:"metadata"`
	Spec     Spec     `yaml:"spec"`
}

type Metadata struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description,omitempty"`
}

type Spec struct {
	Type      string       `yaml:"type,omitempty"`
	DependsOn []Dependency `yaml:"dependsOn,omitempty"`
}

type Dependency struct {
	Name string `yaml:"name"`
	URL  string `yaml:"url,omitempty"`
}

var (
	slugRe   = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
	httpRe   = regexp.MustCompile(`^https?://[^\s/]+\S*$`)
	sshRe    = regexp.MustCompile(`^[A-Za-z0-9._-]+@[A-Za-z0-9._-]+:\S+$`)
	nonSlug  = regexp.MustCompile(`[^a-z0-9]+`)
	fallback = "project"
)

// Load reads and parses a config file, rejecting unknown keys. A missing file
// is reported with an error that wraps os.ErrNotExist; callers decide whether
// that is a problem.
func Load(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open config: %w", err)
	}
	defer f.Close()
	return Decode(f)
}

// Decode parses a config, rejecting unknown keys. An empty document gives an
// empty Config, which Validate then reports.
func Decode(r io.Reader) (*Config, error) {
	dec := yaml.NewDecoder(r)
	dec.KnownFields(true)
	var c Config
	if err := dec.Decode(&c); err != nil {
		if errors.Is(err, io.EOF) {
			return &Config{}, nil
		}
		return nil, fmt.Errorf("decode config: %w", err)
	}
	return &c, nil
}

// Validate checks every rule and reports all violations at once.
func (c *Config) Validate() error {
	var errs []error
	switch {
	case c.Version == 0:
		errs = append(errs, errors.New("version is required"))
	case c.Version != CurrentVersion:
		errs = append(errs, fmt.Errorf("unsupported version %d (expected %d)", c.Version, CurrentVersion))
	}
	if err := checkSlug("metadata.name", c.Metadata.Name); err != nil {
		errs = append(errs, err)
	}
	errs = append(errs, c.checkDependsOn()...)
	return errors.Join(errs...)
}

func (c *Config) checkDependsOn() []error {
	var errs []error
	seen := map[string]bool{}
	for i, d := range c.Spec.DependsOn {
		field := fmt.Sprintf("spec.dependsOn[%d]", i)
		if err := checkSlug(field+".name", d.Name); err != nil {
			errs = append(errs, err)
		}
		switch {
		case d.Name != "" && d.Name == c.Metadata.Name:
			errs = append(errs, fmt.Errorf("%s.name: %q is this repo itself", field, d.Name))
		case d.Name != "" && seen[d.Name]:
			errs = append(errs, fmt.Errorf("%s.name: duplicate %q", field, d.Name))
		}
		seen[d.Name] = true
		if err := checkURL(field+".url", d.URL); err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

func checkSlug(field, value string) error {
	switch {
	case value == "":
		return fmt.Errorf("%s is required", field)
	case !slugRe.MatchString(value):
		return fmt.Errorf("%s: %q is not a slug (lowercase letters, digits and single hyphens)", field, value)
	}
	return nil
}

// checkURL accepts an http(s) URL or a git@host:path SSH URL; empty is fine.
func checkURL(field, value string) error {
	if value == "" || httpRe.MatchString(value) || sshRe.MatchString(value) {
		return nil
	}
	return fmt.Errorf("%s: %q must be an http(s) URL or git@host:path", field, value)
}

// Slug turns a folder name into a valid repo name ("My App_2" → "my-app-2").
func Slug(folder string) string {
	s := strings.Trim(nonSlug.ReplaceAllString(strings.ToLower(folder), "-"), "-")
	if s == "" {
		return fallback
	}
	return s
}

// Default is the file `storyline init` writes: the name and an empty spec
// with a commented example.
func Default(name string) []byte {
	return []byte(fmt.Sprintf(`# yaml-language-server: $schema=%s
version: 1
metadata:
  name: %s
spec:
  # type: front
  # dependsOn:
  #   - name: backend-repository
  #     url: https://github.com/acme/backend
`, SchemaURL, name))
}
