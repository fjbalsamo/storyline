// Package trace finds which test files reference which acceptance criteria,
// so each criterion can be traced to the tests that prove it.
package trace

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Criterion identifies AC-<Story>.<N> by its numbers, so comparison is exact
// (AC-1.1 never matches AC-1.10) and leading zeros don't matter.
type Criterion struct{ Story, N int }

// Ref is one place where a test references a criterion.
type Ref struct {
	File string // slash-separated, relative to the scan root
	Line int
}

func (r Ref) String() string { return fmt.Sprintf("%s:%d", r.File, r.Line) }

// Index maps criteria to the test lines that reference them.
type Index map[Criterion][]Ref

var skipDirs = map[string]bool{
	".git": true, "node_modules": true, "target": true, "vendor": true,
	"dist": true, "build": true, "stories": true,
}

// refRe matches AC-12.1 and ac_12_1 (any case). Numbers are captured and
// compared as integers; the trailing group requires the reference to end
// at "_", a non-word character or end of line (Go has no lookahead).
var refRe = regexp.MustCompile(`(?i)ac[-_](\d+)[._](\d+)(?:_|\W|$)`)

var commentPrefixes = []string{"//", "#", "/*", "*", "--"}

// ParseCriterion parses "AC-12.1" (or "ac_12_1").
func ParseCriterion(id string) (Criterion, bool) {
	m := refRe.FindStringSubmatch(id)
	if m == nil {
		return Criterion{}, false
	}
	return toCriterion(m[1], m[2])
}

func toCriterion(a, b string) (Criterion, bool) {
	s, err1 := strconv.Atoi(a)
	n, err2 := strconv.Atoi(b)
	return Criterion{s, n}, err1 == nil && err2 == nil
}

// Scan walks root and indexes references found in test files. Lines that
// consist only of a comment do not count as references.
func Scan(root string) (Index, error) {
	idx := Index{}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, relErr := filepath.Rel(root, p)
		if relErr != nil {
			return relErr
		}
		if d.IsDir() {
			if p != root && skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		rel = filepath.ToSlash(rel)
		kind := testKind(rel)
		if kind == notTest {
			return nil
		}
		return scanFile(p, rel, kind, idx)
	})
	if err != nil {
		return nil, fmt.Errorf("scan %s: %w", root, err)
	}
	for _, refs := range idx {
		sort.Slice(refs, func(i, j int) bool {
			if refs[i].File != refs[j].File {
				return refs[i].File < refs[j].File
			}
			return refs[i].Line < refs[j].Line
		})
	}
	return idx, nil
}

type kind int

const (
	notTest kind = iota
	isTest
	maybeRust // .rs outside tests/: a test file only if it holds #[test] or #[cfg(test)]
)

// testKind classifies a slash-separated relative path.
func testKind(rel string) kind {
	name := path.Base(rel)
	ext := path.Ext(name)
	stem := strings.TrimSuffix(name, ext)

	switch {
	case strings.HasSuffix(name, "_test.go"):
		return isTest
	case strings.Contains(name, ".test.") || strings.Contains(name, ".spec."):
		return isTest
	case hasDir(rel, "__tests__"):
		return isTest
	case ext == ".py" && (strings.HasPrefix(name, "test_") || strings.HasSuffix(stem, "_test")):
		return isTest
	case ext == ".rs" && hasDir(rel, "tests"):
		return isTest
	case ext == ".rs":
		return maybeRust
	case (ext == ".java" || ext == ".kt" || ext == ".kts") &&
		(strings.HasSuffix(stem, "Test") || strings.HasSuffix(stem, "Tests")):
		return isTest
	}
	return notTest
}

func hasDir(rel, dir string) bool {
	parts := strings.Split(rel, "/")
	for _, p := range parts[:len(parts)-1] {
		if p == dir {
			return true
		}
	}
	return false
}

func scanFile(full, rel string, k kind, idx Index) error {
	f, err := os.Open(full)
	if err != nil {
		return err
	}
	defer f.Close()

	var lines []string
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	if err := sc.Err(); err != nil {
		return fmt.Errorf("%s: %w", rel, err)
	}
	if k == maybeRust && !hasRustTests(lines) {
		return nil
	}

	for i, line := range lines {
		if isComment(line) {
			continue
		}
		seen := map[Criterion]bool{}
		for _, m := range refRe.FindAllStringSubmatch(line, -1) {
			c, ok := toCriterion(m[1], m[2])
			if !ok || seen[c] {
				continue
			}
			seen[c] = true
			idx[c] = append(idx[c], Ref{File: rel, Line: i + 1})
		}
	}
	return nil
}

func hasRustTests(lines []string) bool {
	for _, l := range lines {
		if strings.Contains(l, "#[test]") || strings.Contains(l, "#[cfg(test)]") {
			return true
		}
	}
	return false
}

// isComment reports whether the line is only a comment.
func isComment(line string) bool {
	t := strings.TrimSpace(line)
	for _, p := range commentPrefixes {
		if strings.HasPrefix(t, p) {
			return true
		}
	}
	return false
}
