package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fjbalsamo/storyline/internal/backlog"
)

func runValidate(t *testing.T, path string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"validate", "--backlog", path})
	t.Cleanup(func() { rootCmd.SetOut(nil); rootCmd.SetErr(nil); rootCmd.SetArgs(nil) })
	err := rootCmd.Execute()
	return out.String(), err
}

func validateProject(t *testing.T, config string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "storyline", "backlog.yaml")
	if err := backlog.New().Save(path); err != nil {
		t.Fatal(err)
	}
	if config != "" {
		if err := os.WriteFile(filepath.Join(filepath.Dir(path), "config.yaml"), []byte(config), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return path
}

func TestValidateWithoutConfig(t *testing.T) {
	out, err := runValidate(t, validateProject(t, ""))
	if err != nil {
		t.Fatalf("a missing config.yaml must be fine: %v", err)
	}
	if strings.Contains(out, "config.yaml") {
		t.Errorf("stdout mentions config.yaml: %q", out)
	}
}

func TestValidateWithValidConfig(t *testing.T) {
	out, err := runValidate(t, validateProject(t, "version: 1\nmetadata:\n  name: app\n"))
	if err != nil || !strings.Contains(out, "config.yaml is valid") {
		t.Errorf("out = %q, err = %v", out, err)
	}
}

func TestValidateReportsConfigErrors(t *testing.T) {
	cfg := "version: 1\nmetadata:\n  name: app\nspec:\n  dependsOn:\n    - name: app\n    - name: Back_End\n"
	_, err := runValidate(t, validateProject(t, cfg))
	if err == nil {
		t.Fatal("expected error")
	}
	for _, want := range []string{"config.yaml is invalid", "this repo itself", "dependsOn[1].name"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("err lacks %q: %v", want, err)
		}
	}
}

func TestValidateRejectsUnknownConfigKey(t *testing.T) {
	_, err := runValidate(t, validateProject(t, "version: 1\nmetadata:\n  name: app\nbogus: 1\n"))
	if err == nil || !strings.Contains(err.Error(), "config.yaml is invalid") {
		t.Errorf("err = %v", err)
	}
}
