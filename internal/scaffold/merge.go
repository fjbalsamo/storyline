package scaffold

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/fjbalsamo/storyline/internal/adapters"
)

// mergeFile writes the managed block of f into the file at f.Path, leaving
// everything outside the markers untouched.
func mergeFile(root string, f adapters.File, res *Result) error {
	path := filepath.Join(root, filepath.FromSlash(f.Path))
	existing, err := os.ReadFile(path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		existing = nil
	case err != nil:
		return fmt.Errorf("read %s: %w", f.Path, err)
	}

	merged, err := mergeBlock(existing, f.Content)
	if err != nil {
		return fmt.Errorf("%s: %w", f.Path, err)
	}
	if existing != nil && bytes.Equal(existing, merged) {
		res.Skipped = append(res.Skipped, Skipped{f.Path, "up to date"})
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create %s: %w", filepath.Dir(f.Path), err)
	}
	if err := os.WriteFile(path, merged, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", f.Path, err)
	}
	res.Updated = append(res.Updated, f.Path)
	return nil
}

// mergeBlock returns existing with the block (which includes its markers)
// appended, or replacing the previous block when the markers are present.
func mergeBlock(existing, block []byte) ([]byte, error) {
	start := []byte(adapters.MarkerStart)
	end := []byte(adapters.MarkerEnd)

	i := bytes.Index(existing, start)
	if i < 0 {
		return appendBlock(existing, block), nil
	}
	j := bytes.Index(existing[i:], end)
	if j < 0 {
		return nil, fmt.Errorf("found %s without %s; fix the file by hand", start, end)
	}
	j += i + len(end)
	// Swallow the newline that terminated the old block; block brings its own.
	if j < len(existing) && existing[j] == '\n' {
		j++
	}

	var out bytes.Buffer
	out.Write(existing[:i])
	out.Write(block)
	out.Write(existing[j:])
	return out.Bytes(), nil
}

func appendBlock(existing, block []byte) []byte {
	if len(existing) == 0 {
		return block
	}
	var out bytes.Buffer
	out.Write(existing)
	if !bytes.HasSuffix(existing, []byte("\n")) {
		out.WriteByte('\n')
	}
	out.WriteByte('\n')
	out.Write(block)
	return out.Bytes()
}
