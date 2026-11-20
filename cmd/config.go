package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/fjbalsamo/storyline/internal/backlog"
	"github.com/fjbalsamo/storyline/internal/config"
	"github.com/spf13/cobra"
)

func newConfigCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "config",
		Short: "Show and change storyline/config.yaml",
		Long: `Show and change storyline/config.yaml (this repo's description and type, and
the related repos it depends on) without editing it by hand.

metadata.name is the repo's identity and other repos refer to it, so it can't
be changed here. Comments and formatting in the file are kept, only the
affected lines change, and nothing is saved if the result would be invalid.`,
	}
	c.AddCommand(newConfigShowCmd(), newConfigSetCmd(), newConfigAddDepCmd(), newConfigRemoveDepCmd())
	return c
}

func newConfigShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Print the config",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			path := configPath(cmd)
			src, err := readConfig(path)
			if err != nil {
				return err
			}
			c, err := decodeValid(path, src)
			if err != nil {
				return err
			}
			printConfig(cmd.OutOrStdout(), c)
			return nil
		},
	}
}

func newConfigSetCmd() *cobra.Command {
	c := &cobra.Command{
		Use:     "set [--description <text>] [--type <text>]",
		Short:   "Set the description or type; an empty value clears it",
		Example: `  storyline config set --description "SPA using ReactJS" --type front`,
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			fields, err := setFromFlags(cmd)
			if err != nil {
				return err
			}
			return updateConfig(cmd, func(src []byte) ([]byte, string, error) {
				return applySet(src, fields)
			})
		},
	}
	c.Flags().String("description", "", "what this repo is (empty clears it)")
	c.Flags().String("type", "", "free text, e.g. front, back (empty clears it)")
	return c
}

func newConfigAddDepCmd() *cobra.Command {
	c := &cobra.Command{
		Use:     "add-dependency <name> [--url <url>]",
		Short:   "Add a related repo, or update the URL of an existing one",
		Example: `  storyline config add-dependency backend-repository --url https://github.com/acme/backend`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			url := ""
			if cmd.Flags().Changed("url") {
				raw, _ := cmd.Flags().GetString("url")
				if url = strings.TrimSpace(raw); url == "" {
					return errors.New("--url must not be empty")
				}
			}
			return updateConfig(cmd, func(src []byte) ([]byte, string, error) {
				return applyAdd(src, args[0], url)
			})
		},
	}
	c.Flags().String("url", "", "git-clonable URL: http(s) or git@host:path")
	return c
}

func newConfigRemoveDepCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove-dependency <name>",
		Short: "Remove a related repo",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return updateConfig(cmd, func(src []byte) ([]byte, string, error) {
				out, err := config.RemoveDependency(src, args[0])
				return out, "removed dependency " + args[0], err
			})
		},
	}
}

// configPath is config.yaml next to the backlog file.
func configPath(cmd *cobra.Command) string {
	path, _ := cmd.Flags().GetString("backlog")
	return filepath.Join(filepath.Dir(path), "config.yaml")
}

func readConfig(path string) ([]byte, error) {
	src, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("no %s: run storyline init", path)
	}
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	return src, nil
}

func decodeValid(path string, src []byte) (*config.Config, error) {
	c, err := config.Decode(bytes.NewReader(src))
	if err != nil {
		return nil, fmt.Errorf("%s is invalid:\n%w", path, err)
	}
	if err := c.Validate(); err != nil {
		return nil, fmt.Errorf("%s is invalid:\n%w", path, err)
	}
	return c, nil
}

// updateConfig reads config.yaml, runs change, and writes the result
// atomically. change returns the new text and the success message.
func updateConfig(cmd *cobra.Command, change func(src []byte) ([]byte, string, error)) error {
	path := configPath(cmd)
	src, err := readConfig(path)
	if err != nil {
		return err
	}
	out, msg, err := change(src)
	if err != nil {
		return err
	}
	if bytes.Equal(out, src) {
		fmt.Fprintln(cmd.OutOrStdout(), "nothing to change")
		return nil
	}
	if err := backlog.WriteFile(path, out); err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "✓ %s\n", msg)
	return nil
}

type setField struct {
	name  string
	value string
	set   func([]byte, string) ([]byte, error)
}

// setFromFlags collects the flags that were passed, trimmed.
func setFromFlags(cmd *cobra.Command) ([]setField, error) {
	var fields []setField
	for _, f := range []setField{
		{name: "description", set: config.SetDescription},
		{name: "type", set: config.SetType},
	} {
		if !cmd.Flags().Changed(f.name) {
			continue
		}
		raw, _ := cmd.Flags().GetString(f.name)
		f.value = strings.TrimSpace(raw)
		if strings.ContainsAny(f.value, "\r\n") {
			return nil, fmt.Errorf("--%s must be a single line", f.name)
		}
		fields = append(fields, f)
	}
	if len(fields) == 0 {
		return nil, errors.New("nothing to set: pass at least one of --description, --type")
	}
	return fields, nil
}

func applySet(src []byte, fields []setField) ([]byte, string, error) {
	var changed []string
	cur := src
	for _, f := range fields {
		out, err := f.set(cur, f.value)
		if err != nil {
			return nil, "", err
		}
		if !bytes.Equal(out, cur) {
			changed = append(changed, f.name)
		}
		cur = out
	}
	return cur, "updated: " + strings.Join(changed, ", "), nil
}

func applyAdd(src []byte, name, url string) ([]byte, string, error) {
	existed, err := hasDependency(src, name)
	if err != nil {
		return nil, "", err
	}
	out, err := config.AddDependency(src, name, url)
	if err != nil {
		return nil, "", err
	}
	if existed {
		return out, "updated dependency " + name, nil
	}
	return out, "added dependency " + name, nil
}

func hasDependency(src []byte, name string) (bool, error) {
	c, err := config.Decode(bytes.NewReader(src))
	if err != nil {
		return false, err
	}
	for _, d := range c.Spec.DependsOn {
		if d.Name == name {
			return true, nil
		}
	}
	return false, nil
}

func printConfig(w io.Writer, c *config.Config) {
	fmt.Fprintf(w, "name:        %s\n", c.Metadata.Name)
	fmt.Fprintf(w, "description: %s\n", orNotSet(c.Metadata.Description))
	fmt.Fprintf(w, "type:        %s\n", orNotSet(c.Spec.Type))
	if len(c.Spec.DependsOn) == 0 {
		fmt.Fprintln(w, "dependsOn:   (none)")
		return
	}
	fmt.Fprintln(w, "dependsOn:")
	for _, d := range c.Spec.DependsOn {
		if d.URL == "" {
			fmt.Fprintf(w, "  - %s\n", d.Name)
			continue
		}
		fmt.Fprintf(w, "  - %s  %s\n", d.Name, d.URL)
	}
}

func orNotSet(s string) string {
	if s == "" {
		return "(not set)"
	}
	return s
}

func init() {
	rootCmd.AddCommand(newConfigCmd())
}
