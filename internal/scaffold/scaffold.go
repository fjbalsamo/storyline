// Package scaffold sets up storyline in a project: the stories/ tree, an
// empty backlog, AGENTS.md and the command files of each selected AI tool.
package scaffold

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/fjbalsamo/storyline/internal/adapters"
	"github.com/fjbalsamo/storyline/internal/backlog"
	"github.com/fjbalsamo/storyline/internal/templates"
)

type Options struct {
	Root  string             // project root
	Force bool               // overwrite existing command files (never the backlog)
	Tools []adapters.Adapter // AI tools to generate command files for
}

type Skipped struct {
	Path   string
	Reason string
}

type Result struct {
	Created []string
	Updated []string // merged files (AGENTS.md, pointer files) changed in place
	Skipped []Skipped
}

var storyDirs = []string{"stories/active", "stories/done", "stories/living"}

func Run(opts Options) (Result, error) {
	var res Result

	for _, dir := range storyDirs {
		keep := filepath.Join(dir, ".gitkeep")
		created, err := writeIfMissing(opts.Root, keep, nil, false)
		if err != nil {
			return res, err
		}
		if created {
			res.Created = append(res.Created, keep)
		}
	}

	backlogPath := filepath.Join(opts.Root, backlog.DefaultPath)
	switch _, err := os.Stat(backlogPath); {
	case errors.Is(err, fs.ErrNotExist):
		if err := backlog.New().Save(backlogPath); err != nil {
			return res, err
		}
		res.Created = append(res.Created, backlog.DefaultPath)
	case err != nil:
		return res, fmt.Errorf("check backlog: %w", err)
	default:
		res.Skipped = append(res.Skipped, Skipped{backlog.DefaultPath, "exists, never overwritten"})
	}

	if err := writeTools(opts, &res); err != nil {
		return res, err
	}
	return res, nil
}

// writeTools renders every selected adapter and writes its files.
func writeTools(opts Options, res *Result) error {
	cmds, err := templates.Commands()
	if err != nil {
		return fmt.Errorf("load commands: %w", err)
	}
	for _, a := range withAgentsMD(opts.Tools) {
		files, err := a.Render(cmds)
		if err != nil {
			return fmt.Errorf("render %s: %w", a.ID(), err)
		}
		for _, f := range files {
			if err := writeFile(opts, f, res); err != nil {
				return fmt.Errorf("%s: %w", a.ID(), err)
			}
		}
	}
	return nil
}

// withAgentsMD returns tools with the agents-md adapter first: AGENTS.md is the
// single source of project directives, so it is written whatever tools are
// picked. An explicit agents-md in tools is a no-op alias.
func withAgentsMD(tools []adapters.Adapter) []adapters.Adapter {
	out := []adapters.Adapter{adapters.AgentsMD()}
	for _, t := range tools {
		if t.ID() != out[0].ID() {
			out = append(out, t)
		}
	}
	return out
}

func writeFile(opts Options, f adapters.File, res *Result) error {
	switch {
	case f.Merge:
		return mergeFile(opts.Root, f, res)
	case f.Import != "":
		return importFile(opts.Root, f, res)
	}
	created, err := writeIfMissing(opts.Root, f.Path, f.Content, opts.Force)
	if err != nil {
		return err
	}
	if created {
		res.Created = append(res.Created, f.Path)
	} else {
		res.Skipped = append(res.Skipped, Skipped{f.Path, "exists, use --force to overwrite"})
	}
	return nil
}

// writeIfMissing writes rel under root unless it already exists (or force is
// set). It reports whether the file was written.
func writeIfMissing(root, rel string, content []byte, force bool) (bool, error) {
	path := filepath.Join(root, rel)
	if !force {
		if _, err := os.Stat(path); err == nil {
			return false, nil
		} else if !errors.Is(err, fs.ErrNotExist) {
			return false, fmt.Errorf("check %s: %w", rel, err)
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, fmt.Errorf("create %s: %w", filepath.Dir(rel), err)
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		return false, fmt.Errorf("write %s: %w", rel, err)
	}
	return true, nil
}
