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

// importFile makes sure the pointer file at f.Path contains the line f.Import.
// A missing file is created from f.Content. An existing file only gets the line
// appended when it is absent; nothing else in it is touched.
func importFile(root string, f adapters.File, res *Result) error {
	path := filepath.Join(root, filepath.FromSlash(f.Path))
	existing, err := os.ReadFile(path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		if err := writeNew(path, f.Path, f.Content); err != nil {
			return err
		}
		res.Created = append(res.Created, f.Path)
		return nil
	case err != nil:
		return fmt.Errorf("read %s: %w", f.Path, err)
	}

	if hasLine(existing, f.Import) {
		res.Skipped = append(res.Skipped, Skipped{f.Path, "up to date"})
		return nil
	}
	if err := os.WriteFile(path, appendLine(existing, f.Import), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", f.Path, err)
	}
	res.Updated = append(res.Updated, f.Path)
	return nil
}

func writeNew(path, rel string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create %s: %w", filepath.Dir(rel), err)
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", rel, err)
	}
	return nil
}

// hasLine reports whether any line of data is exactly line, ignoring
// surrounding whitespace.
func hasLine(data []byte, line string) bool {
	for _, l := range bytes.Split(data, []byte("\n")) {
		if string(bytes.TrimSpace(l)) == line {
			return true
		}
	}
	return false
}

// appendLine adds line at the end of data on its own line.
func appendLine(data []byte, line string) []byte {
	out := append([]byte{}, data...)
	if len(out) > 0 && !bytes.HasSuffix(out, []byte("\n")) {
		out = append(out, '\n')
	}
	return append(out, line+"\n"...)
}
