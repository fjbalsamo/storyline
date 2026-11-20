package config

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func write(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestMinimalFile(t *testing.T) {
	c, err := Load(write(t, "version: 1\nmetadata:\n  name: my-app\n"))
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Validate(); err != nil {
		t.Errorf("minimal file should be valid: %v", err)
	}
}

func TestFullFile(t *testing.T) {
	c, err := Load(write(t, `# yaml-language-server: $schema=x
version: 1
metadata:
  name: frontend-repository
  description: SPA using ReactJS
spec:
  type: front
  dependsOn:
    - name: backend-repository
      url: https://github.com/acme/backend
    - name: infra
      url: git@github.com:acme/infra.git
    - name: docs
`))
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	if got := c.Spec.DependsOn; len(got) != 3 || got[1].URL != "git@github.com:acme/infra.git" {
		t.Errorf("dependsOn = %+v", got)
	}
}

func TestDefaultIsValid(t *testing.T) {
	c, err := Load(write(t, string(Default("my-app"))))
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Validate(); err != nil || c.Metadata.Name != "my-app" {
		t.Errorf("name=%q err=%v", c.Metadata.Name, err)
	}
}

func TestValidateRules(t *testing.T) {
	dep := func(n, u string) Dependency { return Dependency{Name: n, URL: u} }
	tests := []struct {
		name    string
		cfg     Config
		wantErr string
	}{
		{"no version", Config{Metadata: Metadata{Name: "a"}}, "version is required"},
		{"wrong version", Config{Version: 2, Metadata: Metadata{Name: "a"}}, "unsupported version 2"},
		{"no name", Config{Version: 1}, "metadata.name is required"},
		{"bad name", Config{Version: 1, Metadata: Metadata{Name: "My App"}}, "not a slug"},
		{"trailing hyphen", Config{Version: 1, Metadata: Metadata{Name: "app-"}}, "not a slug"},
		{"dep without name", Config{Version: 1, Metadata: Metadata{Name: "a"},
			Spec: Spec{DependsOn: []Dependency{dep("", "")}}}, "dependsOn[0].name is required"},
		{"bad dep name", Config{Version: 1, Metadata: Metadata{Name: "a"},
			Spec: Spec{DependsOn: []Dependency{dep("Back_End", "")}}}, "dependsOn[0].name"},
		{"duplicate dep", Config{Version: 1, Metadata: Metadata{Name: "a"},
			Spec: Spec{DependsOn: []Dependency{dep("b", ""), dep("b", "")}}}, `duplicate "b"`},
		{"own name", Config{Version: 1, Metadata: Metadata{Name: "a"},
			Spec: Spec{DependsOn: []Dependency{dep("a", "")}}}, "this repo itself"},
		{"bad url", Config{Version: 1, Metadata: Metadata{Name: "a"},
			Spec: Spec{DependsOn: []Dependency{dep("b", "github.com/acme/b")}}}, "must be an http(s) URL"},
		{"ftp url", Config{Version: 1, Metadata: Metadata{Name: "a"},
			Spec: Spec{DependsOn: []Dependency{dep("b", "ftp://x/y")}}}, "must be an http(s) URL"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("err = %v, want %q", err, tt.wantErr)
			}
		})
	}
}

func TestValidateReportsEverything(t *testing.T) {
	c := Config{Spec: Spec{DependsOn: []Dependency{{Name: "X"}}}}
	err := c.Validate()
	for _, want := range []string{"version is required", "metadata.name is required", "dependsOn[0].name"} {
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("missing %q in %v", want, err)
		}
	}
}

func TestLoadRejectsUnknownKeys(t *testing.T) {
	for _, src := range []string{
		"version: 1\nmetadata:\n  name: a\n  nmae: b\n",
		"version: 1\nmetadata:\n  name: a\nspec:\n  dependson: []\n",
		"version: 1\nmetadata:\n  name: a\nextra: 1\n",
	} {
		if _, err := Load(write(t, src)); err == nil {
			t.Errorf("expected error for %q", src)
		}
	}
}

func TestLoadMissingFile(t *testing.T) {
	_, err := Load(filepath.Join(t.TempDir(), "nope.yaml"))
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("err = %v, want ErrNotExist", err)
	}
}

func TestSlug(t *testing.T) {
	for in, want := range map[string]string{
		"My App_2": "my-app-2", "storyline": "storyline", "  --Foo..Bar  ": "foo-bar", "___": "project", "": "project",
	} {
		if got := Slug(in); got != want {
			t.Errorf("Slug(%q) = %q, want %q", in, got, want)
		}
	}
}

// The schema is for editors; this keeps it in step with the Go model.
func TestSchemaMatchesModel(t *testing.T) {
	raw, err := os.ReadFile("../../schema/config.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	var s struct {
		Required   []string `json:"required"`
		Properties map[string]struct {
			Properties map[string]struct {
				Items struct {
					Properties map[string]any `json:"properties"`
				} `json:"items"`
			} `json:"properties"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(raw, &s); err != nil {
		t.Fatalf("schema is not valid JSON: %v", err)
	}
	for _, k := range []string{"version", "metadata", "spec"} {
		if _, ok := s.Properties[k]; !ok {
			t.Errorf("schema lacks %q", k)
		}
	}
	if len(s.Required) != 2 {
		t.Errorf("required = %v", s.Required)
	}
	dep := s.Properties["spec"].Properties["dependsOn"].Items.Properties
	if _, ok := dep["name"]; !ok || dep["url"] == nil {
		t.Errorf("dependsOn item properties = %v", dep)
	}
	if _, ok := s.Properties["metadata"].Properties["name"]; !ok {
		t.Error("schema lacks metadata.name")
	}
}
