# storyline

[![CI](https://github.com/fjbalsamo/storyline/actions/workflows/ci.yml/badge.svg)](https://github.com/fjbalsamo/storyline/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/fjbalsamo/storyline)](https://github.com/fjbalsamo/storyline/releases/latest)
[![Go version](https://img.shields.io/github/go-mod/go-version/fjbalsamo/storyline)](go.mod)
[![License](https://img.shields.io/github/license/fjbalsamo/storyline)](LICENSE)

**User-story driven development for AI coding assistants.**

![storyline demo](docs/demo.gif)

storyline turns a backlog of one-line user stories into working software, one thin **vertical slice** at a time. Your AI assistant writes the code; you steer, review small diffs and try each step by hand.

It works with Claude Code, Cursor, GitHub Copilot, Gemini CLI, Codex, and any tool that reads `AGENTS.md`.

---

## Why

Spec-driven workflows ask you to specify everything up front, then let the AI build it layer by layer: first the whole database, then the whole API, then the UI. Nothing runs until the end, and you review 3,000-line diffs.

storyline takes the agile route instead:

- **A backlog of small user stories.** Each is one line: *As … I want … so that …*. They are cheap to add, reorder or discard.
- **Vertical slices.** Each story cuts through every layer and ends with something you can run and use.
- **AI as pair programmer.** The AI is the driver and writes code in small steps. You are the navigator: you decide, review and test. The AI stops after every task.
- **Emergent architecture.** No abstractions "just in case". Only what the acceptance criteria need.

## How it works

```
context ──▶ add ──▶ refine ──▶ slice ──▶ apply ──▶ verify ──▶ done
 (once)                                 ▲   │
                                        └───┘  one task at a time,
                                               you test and commit in between
```

| Step | What happens | Status after |
|---|---|---|
| **context** | The AI inspects the repo, asks what it can't know, and writes the `## Project` section of `AGENTS.md`: stack, commands, rules, Do / Don't. Run it once, and again when the project changes | none (no story) |
| **add** | A request becomes one or more one-line stories in the backlog | `todo` |
| **refine** | The AI asks clarifying questions (it never invents business rules) and writes 1–5 acceptance criteria in Given/When/Then | `ready` |
| **slice** | The AI plans the thinnest change that meets the criteria, as up to 8 small tasks. Task 1 goes end to end, even if hardcoded | `ready` |
| **apply** | One task per run: a failing test named after its criterion, the minimum code to pass it, then **stop** | `in-progress` |
| **verify** | Every criterion is mapped to its tests, the suite runs, and you get a manual test checklist | `in-progress` |
| **done** | The criteria move to the living docs, the story is archived, the backlog is updated | `done` |

Stories can also be `refining` or `discarded`.

## Install

**Prebuilt binaries** for macOS, Linux and Windows (amd64 and arm64) are on the [Releases page](https://github.com/fjbalsamo/storyline/releases). Download the archive for your platform, check it against the checksums file, and put `storyline` somewhere in your `PATH`.

**From source**, with Go 1.24+:

```bash
go install github.com/fjbalsamo/storyline@latest
```

Make sure `$(go env GOPATH)/bin` is in your `PATH`.

Check the install with `storyline --version`.

## Quick start

```bash
cd my-project
storyline init --tools claude          # or: --tools cursor,copilot  /  --tools all
```

From then on, everything happens in your AI assistant:

```
/story:context    → describe the project once (stack, commands, rules)
/story:add players want to book a court without calling the club
/story:refine     → answer its questions
/story:slice      → review the plan, approve it
/story:apply      → try it by hand, commit, say "continue"   (repeat per task)
/story:verify
/story:done
```

`/story:context` reads your repo, asks at most five questions and writes the project directives into `AGENTS.md`. Do it once, before your first story.

`/story:add` rewrites your request as one or more user stories, asks when something is ambiguous, and adds them through the CLI. The assistant calls `storyline` under the hood, so it must be installed on the machine.

You can also manage the backlog from the terminal:

```bash
storyline add --as "club player" --want "to book a court for a free time slot" --so-that "I don't have to call the club"
storyline list
```

## AGENTS.md: one source of project directives

`AGENTS.md` is the single place for **general, durable project directives**: purpose, stack, exact build/test/lint commands, layout, engineering rules, test conventions, and short Do / Don't lists. `/story:context` writes it in a `## Project` section, outside the storyline block, and keeps your own edits when you run it again.

Everything about a specific feature lives in `storyline/` (`story.md`, `slice.md`, `tasks.md`, the living docs) and is never copied into `AGENTS.md`. That file is loaded into every conversation, so it stays short (under 60 lines) and only holds what will still be true after the next ten stories.

`storyline init` always writes `AGENTS.md`, and makes sure each tool you pick reads it:

- Cursor, Copilot and Codex read `AGENTS.md` natively.
- Claude Code gets a `CLAUDE.md` containing `@AGENTS.md`. An existing `CLAUDE.md` only gets that line appended; the rest of it is never touched.
- Gemini CLI gets a `GEMINI.md` containing `@AGENTS.md`.

## The backlog

`storyline/backlog.yaml` is plain YAML. **Order is priority**: the first open story is the next one to work on. Use `storyline move` to reorder.

```yaml
# yaml-language-server: $schema=https://raw.githubusercontent.com/fjbalsamo/storyline/main/schema/backlog.schema.json
version: 1
backlog:
  - id: US-001
    as: club player
    want: to book a court for a free time slot
    so_that: I don't have to call the club
    status: in-progress
    story: storyline/active/US-001-book-court/story.md
  - id: US-002
    as: admin
    want: to block courts for maintenance
    so_that: nobody books an unusable court
    status: todo
```

- **IDs are stable and never reused**, even after a story is discarded. Tests and docs reference them.
- `story` is required from `ready` on.
- Unknown keys are rejected, so a typo like `sothat` fails loudly.
- The header line gives VS Code and Cursor autocompletion and validation through the [JSON Schema](schema/backlog.schema.json).

## Project layout

`storyline init` creates:

```
storyline/
├── backlog.yaml        # one-line stories, in priority order
├── config.yaml         # this repo's name and the related repos it depends on
├── active/             # stories being worked on
│   └── US-001-book-court/
│       ├── story.md    # user story + acceptance criteria (AC-1.1, AC-1.2…)
│       ├── slice.md    # layers touched, minimal design, what is NOT being done
│       └── tasks.md    # ordered checklist, each task tied to its criteria
├── done/               # finished stories: YYYY-MM-DD-US-001-book-court/
└── living/             # how the product behaves NOW, grouped by capability
```

…plus `AGENTS.md` and the command files for the tools you picked (see below).

The folder is called `storyline/`, after the tool, because `stories/` collides with Storybook (and JBehave) and confuses people and AI assistants in frontend projects.

### Upgrading from the `stories/` layout

Earlier versions used `stories/`. Every command except `init` and `migrate` fails on that layout with `found the old layout (stories/). Run: storyline migrate`. `storyline migrate` moves `stories/backlog.yaml`, `active/`, `done/` and `living/` into `storyline/`, rewrites the `story:` paths in the backlog and validates the result, moving everything back if that fails. Anything else in `stories/` (a Storybook `Button.stories.tsx`, say) is left alone, and `stories/` is removed only if it ends up empty. Afterwards run `storyline init --tools <your tools> --force` to refresh the AI commands, and check the `## Project` section of `AGENTS.md` for references to `stories/`.

## Related repos

A story often needs something that lives in another repo: a frontend story that needs data from the backend, say. List those repos in `storyline/config.yaml`, which is versioned with the project:

```yaml
# yaml-language-server: $schema=https://raw.githubusercontent.com/fjbalsamo/storyline/main/schema/config.schema.json
version: 1
metadata:
  name: frontend-repository          # required, slug
  description: SPA using ReactJS     # optional
spec:
  type: front                        # optional, free text
  dependsOn:                         # optional
    - name: backend-repository       # required in each item, slug
      url: https://github.com/acme/backend   # optional, http(s) or git@host:path
```

- `version` and `metadata.name` are required. Names are slugs (`^[a-z0-9]+(-[a-z0-9]+)*$`); `dependsOn` names are unique and can't be this repo's own name. Unknown keys are rejected, and `storyline validate` checks the file.
- `storyline init` creates it with the folder name as `metadata.name` (`My App_2` → `my-app-2`) and never overwrites it, even with `--force`. **Projects created before this file existed** keep working without it; re-run `storyline init` to add it.
- storyline never creates, links or tracks stories in other repos, and never checks that they exist. The file only gives the AI context. People decide.

**It never blocks.** When `refine` or `slice` finds that a story needs something from a repo in `dependsOn`, the AI:

1. does not implement it here or pretend it exists;
2. suggests the story for the other repo, with a line you can paste there;
3. keeps going with this story, using a mock based on an assumed contract (`apply` uses that mock);
4. records the dependency under `## External dependencies` in `story.md`, so it isn't lost in the chat.

A suggestion looks like this:

```
Related repo: backend-repository (https://github.com/acme/backend)
Needed: an endpoint to look up a tire by code.
Paste it there:
/story:add As a frontend app, I want GET /tires?code=… to return the tire's vehicle, number and category, so that the SPA can show it
I'm not waiting for it: I'll mock GET /tires?code=… → { code, vehicle, number, category } and carry on.
```

and in `story.md`:

```markdown
## External dependencies
- backend-repository: needs a tire lookup endpoint
  Suggested: /story:add As a frontend app, I want GET /tires?code=… …
  Assumed contract (mock): GET /tires?code=… → { code, vehicle, number, category }
```

The section is optional, and `validate` ignores its content. If the needed repo isn't in `dependsOn`, the AI says so and suggests adding it.

## Acceptance criteria and tests

Criteria are numbered after their story: `US-012` gets `AC-12.1`, `AC-12.2`, and so on. Every test that proves a criterion carries its ID in the name:

```ts
it("AC-12.1 shows only free slots", () => { … })
```

When the language doesn't allow `-` or `.` in test names (Rust, for example), use `ac_12_1_shows_only_free_slots`.

This is what makes a story verifiable: each criterion → its tests → green or red. `storyline trace` checks the first half.

### How `trace` finds tests

`storyline trace` scans test files under `--root` (default `.`) and reports which tests reference each criterion:

```
US-001: Cancel a booking
  ✓ AC-1.1  cancel own booking  go/booking_test.go:7
  ✗ AC-1.4  observation         no tests
1/2 criteria covered
```

- **Test files:** Go `*_test.go`; JS/TS `*.test.*`, `*.spec.*` and files under `__tests__/`; Python `test_*.py` and `*_test.py`; Rust `.rs` files under `tests/` or containing `#[test]` / `#[cfg(test)]`; Java/Kotlin `*Test.*` and `*Tests.*`. The directories `.git`, `node_modules`, `target`, `vendor`, `dist`, `build` and `storyline` are skipped.
- **References:** `AC-12.1` and `ac_12_1`, in any case. The numbers are compared exactly, so `AC-1.1` never matches `AC-1.10`, and the reference must end at `_`, a non-word character or the end of the line.
- **No boundary before `ac`:** `TestAC_1_1_Cancel` and `test_ac_1_1` both match. The price is that a longer word ending in `ac`, such as `SPAC-1.1`, would match too.
- **Comments don't count:** a line that, once trimmed, starts with `//`, `#`, `/*`, `*` or `--` is ignored, so a criterion that only appears in a comment is reported as uncovered. The check is per line: inside a `/* … */` block only lines starting with `*` are ignored, and Python docstrings are not recognized as comments. A comment at the end of a code line (`x() // AC-1.1`) still counts.
- **Exit codes:** `0` when every criterion in scope has at least one test, `1` when any has none (or on any other error). Use it as a CI gate: `storyline trace --all`.

## CLI reference

| Command | Description |
|---|---|
| `storyline init [dir] [--tools ids] [--force]` | Set up a project: stories tree, `AGENTS.md`, pointer files and commands. Also creates `storyline/config.yaml` if it is missing. Safe to re-run: existing files are kept, so re-running it is how an older project gets `config.yaml`. `--force` regenerates command files but **never** overwrites the backlog or `config.yaml` |
| `storyline migrate` | Move a project from the old `stories/` layout to `storyline/` (see above). Fails if `storyline/` already exists |
| `storyline add --as … --want … --so-that …` | Append a story to the bottom of the backlog with the next ID |
| `storyline list [-a] [-s status]` | Open stories in priority order. `▶` marks the next one. `-a` includes done/discarded |
| `storyline edit <id> [--as …] [--want …] [--so-that …]` | Fix the wording of a story. **Only `as`, `want` and `so_that` are editable**; `id`, `status` and `story` never are (use `storyline status` for status). At least one flag is required and values can't be empty. Allowed on `todo` and `refining`; on `ready` and `in-progress` it warns that `story.md` may need the same change; `done` and `discarded` stories are rejected. Comments and formatting in the backlog are kept, only the edited values change, and nothing is saved if the backlog would be invalid |
| `storyline move <id> --top \| --bottom \| --before <id> \| --after <id>` | Reorder the backlog to change priority. Pass exactly one flag. `--top` puts the story before the first open story. Only open stories can be moved (`done` and `discarded` are history), but the `--before`/`--after` reference can have any status. The story moves with the comments directly above it and the rest of the file is left as it is. Nothing is saved if the backlog would be invalid |
| `storyline validate` | Check `storyline/config.yaml` when it exists (a missing file is fine), the backlog (ID format and uniqueness, required fields, valid status, story path when needed) and every story it points to: 1–5 criteria numbered `.1, .2, …` each with Given/When/Then, at most 8 numbered tasks that reference existing criteria, and a folder that matches the status |
| `storyline status <id> <status> [--story <path>]` | Move a story along the allowed transitions (table below). Moving to `ready` needs a story file that passes `validate`; `--story` sets its path and is only accepted with `ready`. Nothing is saved if the backlog would be invalid |
| `storyline done <id> [--force]` | Close an `in-progress` story: needs a `tasks.md` with every task checked, moves `storyline/active/<dir>` to `storyline/done/<YYYY-MM-DD>-<dir>`, sets `done` and updates the story path. `--force` skips only the tasks check and prints a warning. Living docs are not touched |
| `storyline trace [id] [--all] [--root dir]` | Show which tests prove each acceptance criterion. Defaults to `in-progress` stories; `[id]` picks one, `--all` takes every story with a `story.md`. Exits `1` if any criterion has no tests |

Global flag: `--backlog <path>` (default `storyline/backlog.yaml`).

### Status transitions

| From | To |
|---|---|
| `todo` | `refining`, `discarded` |
| `refining` | `ready`, `todo`, `discarded` |
| `ready` | `in-progress`, `refining`, `discarded` |
| `in-progress` | `ready`, `discarded` |
| `discarded` | `todo` |
| `done` | none |

Anything else fails and lists the allowed targets. `done` is only reachable with `storyline done <id>`.

### CI

`storyline validate` and `storyline trace --all` both exit `1` on failure, so they can run as CI steps:

```sh
storyline validate && storyline trace --all
```

## AI tool support

The seven commands are written once and rendered into each tool's native format.

| Tool | `--tools` | Generated files | Invoke as |
|---|---|---|---|
| Claude Code | `claude` | `.claude/commands/story/*.md` | `/story:refine` |
| Cursor | `cursor` | `.cursor/commands/story-*.md` | `/story-refine` |
| GitHub Copilot | `copilot` | `.github/prompts/story-*.prompt.md` | `/story-refine` |
| Gemini CLI | `gemini` | `.gemini/commands/story/*.toml` | `/story:refine` |
| Codex | `codex` | `.agents/skills/story-*/SKILL.md` | `$story-refine` |
| Any other tool | none needed | a managed section in `AGENTS.md` | ask in plain words: *"refine the next story"* |

`--tools all` sets them all up. `AGENTS.md` is written for every selection (`agents-md` is still accepted in `--tools`, as a no-op). Pointer files: `CLAUDE.md` (`claude`) and `GEMINI.md` (`gemini`). The storyline section of `AGENTS.md` lives between `<!-- storyline:start -->` and `<!-- storyline:end -->`. storyline only rewrites that block and never touches the rest of your file.

To customize the commands, edit the files storyline generated in your project. Running `init --force` restores the defaults.

## Roadmap

- [ ] A single `skills` adapter, as tools converge on `SKILL.md`
- [ ] Per-command model hints (a cheap model to read, a strong one to code)
- [ ] More ways to install: a Homebrew tap and `npx storyline`

## Inspiration

- [OpenSpec](https://github.com/Fission-AI/OpenSpec): the CLI + slash commands split, and how it supports many tools
- [hola-ai](https://github.com/nschurmann/hola-ai): agile, vertical-slice development with AI as a pair programmer

## License

[MIT](LICENSE) © 2026 Franco Bálsamo
