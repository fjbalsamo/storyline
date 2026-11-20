package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fjbalsamo/storyline/internal/backlog"
	"github.com/fjbalsamo/storyline/internal/config"
	"github.com/spf13/cobra"
)

func runConfigCmd(t *testing.T, backlogPath string, args ...string) (string, error) {
	t.Helper()
	root := &cobra.Command{Use: "storyline", SilenceUsage: true, SilenceErrors: true}
	root.PersistentFlags().String("backlog", backlog.DefaultPath, "")
	root.AddCommand(newConfigCmd())
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs(append([]string{"config"}, append(args, "--backlog", backlogPath)...))
	err := root.Execute()
	return out.String(), err
}

// configProject returns the backlog path of a temp project whose config.yaml
// holds content ("" means no config file).
func configProject(t *testing.T, content string) (backlogPath, configFile string) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "storyline")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	configFile = filepath.Join(dir, "config.yaml")
	if content != "" {
		if err := os.WriteFile(configFile, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return filepath.Join(dir, "backlog.yaml"), configFile
}

var freshConfig = string(config.Default("my-app"))

const fullConfig = `# yaml-language-server: $schema=x
# keep this note
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
`

func TestConfigShow(t *testing.T) {
	bl, _ := configProject(t, fullConfig)
	out, err := runConfigCmd(t, bl, "show")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"name:        frontend-repository", "description: SPA using ReactJS", "type:        front",
		"  - backend-repository  https://github.com/acme/backend", "  - infra\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("show lacks %q:\n%s", want, out)
		}
	}
	bl, _ = configProject(t, freshConfig)
	out, _ = runConfigCmd(t, bl, "show")
	if !strings.Contains(out, "description: (not set)") || !strings.Contains(out, "(none)") {
		t.Errorf("fresh show:\n%s", out)
	}
}

func TestConfigMissingFile(t *testing.T) {
	bl, _ := configProject(t, "")
	for _, args := range [][]string{
		{"show"}, {"set", "--type", "x"}, {"add-dependency", "b"}, {"remove-dependency", "b"},
	} {
		_, err := runConfigCmd(t, bl, args...)
		if err == nil || !strings.Contains(err.Error(), "config.yaml: run storyline init") {
			t.Errorf("%v: err = %v", args, err)
		}
	}
}

func TestConfigSet(t *testing.T) {
	bl, file := configProject(t, freshConfig)
	out, err := runConfigCmd(t, bl, "set", "--description", "  SPA  ", "--type", "front")
	if err != nil || out != "✓ updated: description, type\n" {
		t.Fatalf("out = %q, err = %v", out, err)
	}
	want := strings.Replace(freshConfig, "  name: my-app\n", "  name: my-app\n  description: SPA\n", 1)
	want = strings.Replace(want, "  # type: front\n", "  type: front\n", 1)
	if got := readFile(t, file); got != want {
		t.Errorf("file:\n%s", got)
	}
}

func TestConfigSetListsOnlyWhatChanged(t *testing.T) {
	bl, _ := configProject(t, fullConfig)
	out, err := runConfigCmd(t, bl, "set", "--description", "SPA using ReactJS", "--type", "web")
	if err != nil || out != "✓ updated: type\n" {
		t.Errorf("out = %q, err = %v", out, err)
	}
	out, _ = runConfigCmd(t, bl, "set", "--type", "web")
	if out != "nothing to change\n" {
		t.Errorf("out = %q", out)
	}
}

func TestConfigSetClears(t *testing.T) {
	bl, file := configProject(t, fullConfig)
	if _, err := runConfigCmd(t, bl, "set", "--type", ""); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, file); got != strings.Replace(fullConfig, "  type: front\n", "", 1) {
		t.Errorf("file:\n%s", got)
	}
}

func TestConfigSetRejects(t *testing.T) {
	tests := []struct {
		args    []string
		wantErr string
	}{
		{[]string{"set"}, "pass at least one of --description, --type"},
		{[]string{"set", "--name", "other"}, "unknown flag: --name"},
		{[]string{"set", "--description", "a\nb"}, "--description must be a single line"},
	}
	for _, tt := range tests {
		bl, file := configProject(t, fullConfig)
		_, err := runConfigCmd(t, bl, tt.args...)
		if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
			t.Errorf("%v: err = %v", tt.args, err)
		}
		if readFile(t, file) != fullConfig {
			t.Errorf("%v: file changed", tt.args)
		}
	}
}

func TestConfigFirstDependencyOnFreshInitFile(t *testing.T) {
	bl, file := configProject(t, freshConfig)
	out, err := runConfigCmd(t, bl, "add-dependency", "backend-repository", "--url", "https://github.com/acme/backend")
	if err != nil || out != "✓ added dependency backend-repository\n" {
		t.Fatalf("out = %q, err = %v", out, err)
	}
	want := strings.Replace(freshConfig, "  # dependsOn:\n  #   - name: backend-repository\n  #     url: https://github.com/acme/backend\n",
		"  dependsOn:\n    - name: backend-repository\n      url: https://github.com/acme/backend\n", 1)
	if got := readFile(t, file); got != want {
		t.Errorf("file:\n%s\nwant:\n%s", got, want)
	}
	if _, err := config.Load(file); err != nil {
		t.Error(err)
	}
}

func TestConfigAddDuplicate(t *testing.T) {
	bl, file := configProject(t, fullConfig)

	out, err := runConfigCmd(t, bl, "add-dependency", "backend-repository")
	if err != nil || out != "nothing to change\n" || readFile(t, file) != fullConfig {
		t.Errorf("without --url: out = %q, err = %v", out, err)
	}
	out, err = runConfigCmd(t, bl, "add-dependency", "backend-repository", "--url", "https://gitlab.com/acme/api")
	if err != nil || out != "✓ updated dependency backend-repository\n" {
		t.Errorf("with --url: out = %q, err = %v", out, err)
	}
	want := strings.Replace(fullConfig, "https://github.com/acme/backend", "https://gitlab.com/acme/api", 1)
	if got := readFile(t, file); got != want {
		t.Errorf("file:\n%s", got)
	}
}

func TestConfigAddRejects(t *testing.T) {
	tests := []struct {
		args    []string
		wantErr string
	}{
		{[]string{"add-dependency", "frontend-repository"}, "this repo itself"},
		{[]string{"add-dependency", "Bad_Name"}, "not a slug"},
		{[]string{"add-dependency", "ok", "--url", "github.com/x"}, "must be an http(s) URL"},
		{[]string{"add-dependency", "ok", "--url", "  "}, "--url must not be empty"},
	}
	for _, tt := range tests {
		bl, file := configProject(t, fullConfig)
		_, err := runConfigCmd(t, bl, tt.args...)
		if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
			t.Errorf("%v: err = %v", tt.args, err)
		}
		if readFile(t, file) != fullConfig {
			t.Errorf("%v: file changed", tt.args)
		}
	}
}

func TestConfigRemove(t *testing.T) {
	bl, file := configProject(t, fullConfig)
	out, err := runConfigCmd(t, bl, "remove-dependency", "infra")
	if err != nil || out != "✓ removed dependency infra\n" {
		t.Fatalf("out = %q, err = %v", out, err)
	}
	if got := readFile(t, file); got != strings.Replace(fullConfig, "    - name: infra\n", "", 1) {
		t.Errorf("file:\n%s", got)
	}
	_, err = runConfigCmd(t, bl, "remove-dependency", "nope")
	if err == nil || !strings.Contains(err.Error(), "dependency nope not found") {
		t.Errorf("err = %v", err)
	}
}

func TestConfigWritesNothingWhenResultInvalid(t *testing.T) {
	bad := strings.Replace(fullConfig, "name: frontend-repository", "name: Front End", 1)
	bl, file := configProject(t, bad)
	_, err := runConfigCmd(t, bl, "set", "--type", "web")
	if err == nil || !strings.Contains(err.Error(), "not a slug") {
		t.Fatalf("err = %v", err)
	}
	if readFile(t, file) != bad {
		t.Error("file changed despite the invalid config")
	}
}

func TestConfigSetPreservesEverythingElse(t *testing.T) {
	bl, file := configProject(t, fullConfig)
	if _, err := runConfigCmd(t, bl, "set", "--description", "New text"); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, file); got != strings.Replace(fullConfig, "SPA using ReactJS", "New text", 1) {
		t.Errorf("file:\n%s", got)
	}
}
