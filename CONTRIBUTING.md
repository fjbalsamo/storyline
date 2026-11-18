# Contributing to storyline

Thanks for your interest in storyline! Bug reports, ideas and pull requests are all welcome.

## Before you start

- **Bugs:** open an issue with the command you ran, what you expected and what happened. Include `storyline --help` output if it's relevant.
- **Ideas and features:** open an issue first, written as a user story. That's what this tool is about:

  > **As** a developer using Windsurf, **I want** `storyline init --tools windsurf`, **so that** I can use the workflow there.

  Agreeing on the story before writing code saves everyone a rewrite.
- **Small fixes** (typos, docs, obvious bugs) can go straight to a pull request.

## Development setup

Requires Go 1.24+.

```bash
git clone https://github.com/fjbalsamo/storyline.git
cd storyline
go build ./...
go test ./...
go run . --help
```

To try your build in another project:

```bash
go install .
cd /tmp && mkdir demo && cd demo
storyline init --tools all
```

## Project layout

```
main.go                       # entry point
cmd/                          # cobra commands: thin, no business logic
internal/
├── backlog/                  # backlog model: load, save, validate, add
├── templates/
│   ├── commands/*.md         # the six workflow commands, written once
│   └── templates.go          # parses them into templates.Command
├── adapters/                 # render the commands into each tool's format
│   └── testdata/golden/      # expected output per tool
└── scaffold/                 # what `init` writes, incl. the AGENTS.md merge
schema/backlog.schema.json    # JSON Schema for editors
```

## Common changes

### Changing a workflow command

Edit `internal/templates/commands/<name>.md`. It has a frontmatter with `description` and `argument_hint`. In the body, use `{{ARGS}}` where the user's input goes, and each adapter replaces it with its tool's syntax.

Then regenerate the golden files and review the diff:

```bash
go test ./internal/adapters -update
git diff internal/adapters/testdata
```

### Adding support for a new AI tool

1. Read the tool's official docs: where command files live, their format, and how arguments are passed.
2. Add `internal/adapters/<tool>.go` implementing `Adapter` (`ID`, `Name`, `Render`). Put a link to the docs in a comment.
3. Register it in `All()` in `adapters.go`.
4. Run `go test ./internal/adapters -update`, check the generated files, and add tests for anything special about the format.
5. Add a row to the **AI tool support** table in the README.

### Changing the backlog format

Update `internal/backlog`, `schema/backlog.schema.json` and the README together. Backlogs are files in users' repos, so a change that breaks existing ones needs a new `version` and a migration path. Open an issue first.

## Pull requests

- Keep each PR focused on one change.
- Add or update tests. `go test ./...` must pass.
- Run `gofmt -l .` (it should print nothing) and `go vet ./...`.
- Write code, comments, templates and messages in English.
- Use [Conventional Commits](https://www.conventionalcommits.org/): `feat:`, `fix:`, `docs:`, `test:`, `refactor:`, `chore:`.

## License

By contributing, you agree that your contributions are licensed under the [MIT License](LICENSE).
