package config

import (
	"bytes"
	"strings"
	"testing"
)

var fresh = string(Default("my-app"))

const handWritten = `# yaml-language-server: $schema=x
# my notes

version: 1
metadata:
  name: frontend-repository   # the SPA
  description: SPA using ReactJS
spec:
  type: front
  # who we talk to
  dependsOn:
    # the API
    - name: backend-repository
      url: https://github.com/acme/backend   # main
    - name: infra
`

func must(t *testing.T, out []byte, err error) string {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}

func TestFirstDependencyReplacesTheInitExample(t *testing.T) {
	got := must(t, mustAdd(t, fresh, "backend-repository", "https://github.com/acme/backend"), nil)
	want := strings.Replace(fresh, `  # dependsOn:
  #   - name: backend-repository
  #     url: https://github.com/acme/backend
`, `  dependsOn:
    - name: backend-repository
      url: https://github.com/acme/backend
`, 1)
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
	if !strings.Contains(got, "  # type: front\n") {
		t.Error("the commented type line must stay")
	}
}

func mustAdd(t *testing.T, src, name, url string) []byte {
	t.Helper()
	out, err := AddDependency([]byte(src), name, url)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestFirstDependencyWithoutURL(t *testing.T) {
	got := string(mustAdd(t, fresh, "docs", ""))
	if !strings.Contains(got, "  dependsOn:\n    - name: docs\n") || strings.Contains(got, "url: https") {
		t.Errorf("got:\n%s", got)
	}
}

func TestAddEverySecondDependencyAndRemoveBoth(t *testing.T) {
	one := string(mustAdd(t, fresh, "backend-repository", "https://github.com/acme/backend"))
	two := string(mustAdd(t, one, "infra", ""))
	if want := one + "    - name: infra\n"; two != want {
		t.Errorf("two =\n%s\nwant\n%s", two, want)
	}
	back, err := RemoveDependency([]byte(two), "infra")
	if got := must(t, back, err); got != one {
		t.Errorf("after removing infra:\n%s", got)
	}
	empty, err := RemoveDependency([]byte(one), "backend-repository")
	got := must(t, empty, err)
	if want := strings.Replace(fresh, "  # dependsOn:\n  #   - name: backend-repository\n  #     url: https://github.com/acme/backend\n", "", 1); got != want {
		t.Errorf("after removing everything:\n%s\nwant\n%s", got, want)
	}
}

func TestAddToHandWrittenFile(t *testing.T) {
	got := string(mustAdd(t, handWritten, "data-service", "git@github.com:acme/data.git"))
	want := handWritten + "    - name: data-service\n      url: git@github.com:acme/data.git\n"
	if got != want {
		t.Errorf("got:\n%s", got)
	}
}

func TestUpdateURL(t *testing.T) {
	got := string(mustAdd(t, handWritten, "backend-repository", "https://gitlab.com/acme/api"))
	want := strings.Replace(handWritten, "https://github.com/acme/backend", "https://gitlab.com/acme/api", 1)
	if got != want {
		t.Errorf("replace existing url:\n%s", got)
	}
	got = string(mustAdd(t, handWritten, "infra", "https://github.com/acme/infra"))
	want = handWritten + "      url: https://github.com/acme/infra\n"
	if got != want {
		t.Errorf("insert missing url:\n%s", got)
	}
}

func TestAddExistingWithoutURLChangesNothing(t *testing.T) {
	out, err := AddDependency([]byte(handWritten), "backend-repository", "")
	if err != nil || !bytes.Equal(out, []byte(handWritten)) {
		t.Errorf("err = %v, changed = %v", err, string(out) != handWritten)
	}
	out, err = AddDependency([]byte(handWritten), "backend-repository", "https://github.com/acme/backend")
	if err != nil || !bytes.Equal(out, []byte(handWritten)) {
		t.Errorf("same url: err = %v", err)
	}
}

func TestAddRejects(t *testing.T) {
	tests := []struct{ name, url, want string }{
		{"my-app", "", "this repo itself"},
		{"Bad_Name", "", "not a slug"},
		{"ok-name", "ftp://x", "must be an http(s) URL"},
	}
	for _, tt := range tests {
		out, err := AddDependency([]byte(fresh), tt.name, tt.url)
		if err == nil || !strings.Contains(err.Error(), tt.want) {
			t.Errorf("%s: err = %v", tt.name, err)
		}
		if out != nil {
			t.Errorf("%s: returned text despite the error", tt.name)
		}
	}
}

func TestRemoveInTheMiddleTakesItsComments(t *testing.T) {
	got, err := RemoveDependency([]byte(handWritten), "backend-repository")
	want := strings.Replace(handWritten, `    # the API
    - name: backend-repository
      url: https://github.com/acme/backend   # main
`, "", 1)
	if s := must(t, got, err); s != want {
		t.Errorf("got:\n%s", s)
	}
	if _, err := RemoveDependency([]byte(handWritten), "nope"); err == nil || !strings.Contains(err.Error(), "dependency nope not found") {
		t.Errorf("err = %v", err)
	}
}

func TestRemoveLastDropsTheKey(t *testing.T) {
	src := "version: 1\nmetadata:\n  name: a\nspec:\n  type: front\n  dependsOn:\n    - name: b\n"
	got, err := RemoveDependency([]byte(src), "b")
	if s := must(t, got, err); s != "version: 1\nmetadata:\n  name: a\nspec:\n  type: front\n" {
		t.Errorf("got:\n%s", s)
	}
}

func TestSetDescription(t *testing.T) {
	// replace in place, keeping the comment on the name line
	got, err := SetDescription([]byte(handWritten), "React SPA")
	if s := must(t, got, err); s != strings.Replace(handWritten, "SPA using ReactJS", "React SPA", 1) {
		t.Errorf("replace:\n%s", s)
	}
	// clear
	got, err = SetDescription([]byte(handWritten), "")
	if s := must(t, got, err); s != strings.Replace(handWritten, "  description: SPA using ReactJS\n", "", 1) {
		t.Errorf("clear:\n%s", s)
	}
	// add after name
	got, err = SetDescription([]byte(fresh), "My app: the #1 one")
	want := strings.Replace(fresh, "  name: my-app\n", "  name: my-app\n  description: 'My app: the #1 one'\n", 1)
	if s := must(t, got, err); s != want {
		t.Errorf("add:\n%s", s)
	}
	// clearing what is not set changes nothing
	if out, err := SetDescription([]byte(fresh), ""); err != nil || string(out) != fresh {
		t.Errorf("clear unset: %v", err)
	}
}

func TestSetType(t *testing.T) {
	// fresh init file: the commented example becomes the real key
	got, err := SetType([]byte(fresh), "front")
	want := strings.Replace(fresh, "  # type: front\n", "  type: front\n", 1)
	if s := must(t, got, err); s != want {
		t.Errorf("fresh:\n%s", s)
	}
	// replace, clear
	got, err = SetType([]byte(handWritten), "web")
	if s := must(t, got, err); s != strings.Replace(handWritten, "type: front", "type: web", 1) {
		t.Errorf("replace:\n%s", s)
	}
	got, err = SetType([]byte(handWritten), "")
	if s := must(t, got, err); s != strings.Replace(handWritten, "  type: front\n", "", 1) {
		t.Errorf("clear:\n%s", s)
	}
}

func TestSetTypeOnOtherSpecShapes(t *testing.T) {
	head := "version: 1\nmetadata:\n  name: a\n"
	tests := []struct{ name, src, want string }{
		{"no spec", head, head + "spec:\n  type: back\n"},
		{"no spec, no final newline", strings.TrimSuffix(head, "\n"), head + "spec:\n  type: back\n"},
		{"empty spec", head + "spec: {}\n", head + "spec:\n  type: back\n"},
		{"null spec with comment", head + "spec:  # later\n", head + "spec:  # later\n  type: back\n"},
		{"deps only", head + "spec:\n  dependsOn:\n    - name: b\n", head + "spec:\n  type: back\n  dependsOn:\n    - name: b\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SetType([]byte(tt.src), "back")
			if s := must(t, got, err); s != tt.want {
				t.Errorf("got:\n%s\nwant:\n%s", s, tt.want)
			}
		})
	}
}

func TestAddDependencyOnOtherSpecShapes(t *testing.T) {
	head := "version: 1\nmetadata:\n  name: a\n"
	tests := []struct{ name, src, want string }{
		{"no spec", head, head + "spec:\n  dependsOn:\n    - name: b\n"},
		{"empty spec", head + "spec: {}\n", head + "spec:\n  dependsOn:\n    - name: b\n"},
		{"type only", head + "spec:\n  type: front\n", head + "spec:\n  type: front\n  dependsOn:\n    - name: b\n"},
		{"empty list", head + "spec:\n  dependsOn: []\n", head + "spec:\n  dependsOn:\n    - name: b\n"},
		{"null list", head + "spec:\n  dependsOn:\n", head + "spec:\n  dependsOn:\n    - name: b\n"},
		{"indentless list", head + "spec:\n  dependsOn:\n  - name: c\n", head + "spec:\n  dependsOn:\n  - name: c\n  - name: b\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := AddDependency([]byte(tt.src), "b", "")
			if s := must(t, got, err); s != tt.want {
				t.Errorf("got:\n%s\nwant:\n%s", s, tt.want)
			}
		})
	}
}

func TestEditRefusesFlowStyleAndInvalidFiles(t *testing.T) {
	flow := "version: 1\nmetadata:\n  name: a\nspec: {type: front}\n"
	if _, err := SetType([]byte(flow), "x"); err == nil || !strings.Contains(err.Error(), "block style") {
		t.Errorf("flow spec: err = %v", err)
	}
	if _, err := SetDescription([]byte("version: 1\nmetadata:\n  name: a\nbogus: 1\n"), "x"); err == nil {
		t.Error("unknown key should fail")
	}
	if _, err := SetDescription([]byte("version: 1\nmetadata:\n  name: Bad Name\n"), "x"); err == nil ||
		!strings.Contains(err.Error(), "not a slug") {
		t.Errorf("invalid config: err = %v", err)
	}
}
