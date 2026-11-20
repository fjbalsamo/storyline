# Antigravity CLI test checklist

storyline has not been tested against a real Antigravity CLI (`agy`). `--tools antigravity` is an alias of `codex`, based on the official docs. Run this checklist when someone has `agy` installed, and fill in the results table at the end.

Expectations come from:

- Skills: https://antigravity.google/docs/skills/
- Rules (`AGENTS.md`, `GEMINI.md`): https://antigravity.google/docs/rules/
- Discovery and `/skills reload`: https://antigravity.google/docs/customization-discovery/
- Migration from Gemini CLI: https://antigravity.google/docs/cli/gcli-migration/
- MCP: https://antigravity.google/docs/mcp/

Items marked **UNVERIFIED** are not documented, so a failure there is a finding, not necessarily a bug.

## Prerequisites

- [ ] `agy` installed and signed in (`agy --version` noted below)
- [ ] `storyline` built from this branch (`go build -o storyline .`)
- [ ] A scratch directory that is a git repo (`git init`); Antigravity looks up to the repo root

## 1. Files generated

```sh
storyline init --tools antigravity
```

- [ ] `AGENTS.md` exists and contains the `<!-- storyline:start -->` block
- [ ] `.agents/skills/story-*/SKILL.md` exist, each with `name` and `description` frontmatter
- [ ] No `GEMINI.md`, no `.gemini/`

## 2. Skills are discovered

Start `agy` in the scratch directory, then run `/skills reload`.

- [ ] Every `story-*` skill is listed
- [ ] Each shows up as a slash command (`/story-refine`, `/story-add`, ...)

## 3. Arguments reach the skill (UNVERIFIED)

Add a story with `storyline add "Show a greeting"`, then run `/story-refine <its id>`.

- [ ] The agent uses the id you typed, without asking for it again
- [ ] If not, does a plain prompt work (`/story-refine` then the id in the next message)? Note which.

## 4. AGENTS.md is loaded

Ask: "Which project rules do you have loaded? Quote the storyline section."

- [ ] The agent quotes the storyline block from `AGENTS.md`
- [ ] The file is under the limits (24,000 bytes per file, 20,000 tokens total)

## 5. Workflow commands work

During `/story-refine`:

- [ ] The agent runs `storyline` commands in the shell (a permission prompt is expected; `toolPermission` defaults to `request-review`)
- [ ] The agent reads and edits files under `storyline/`
- [ ] It stops after the one step, as the skill says

## 6. GEMINI.md duplication (UNVERIFIED, do not fix here)

```sh
storyline init --tools antigravity,gemini --force
```

This writes a `GEMINI.md` containing `@AGENTS.md`.

- [ ] Ask the same question as in step 4. Is the storyline section loaded once or twice?
- [ ] Is the literal `@AGENTS.md` line shown as text, or expanded?

## 7. Import through MCP

Configure one tracker MCP server in `.agents/mcp_config.json` (remote servers need Streamable HTTP; see the MCP doc), then run `/mcp` and `/story-import`.

- [ ] The server shows as connected in `/mcp`
- [ ] The import reads the tracker and creates stories through `storyline`
- [ ] Nothing is written to the tracker

## 8. Gemini CLI leftovers (optional)

- [ ] Put a file in `.gemini/commands/story/x.toml`. Confirm Antigravity ignores it.
- [ ] Run `agy plugin import gemini`. Note what it does with project-level `.gemini/commands/`.

## Results

| Field | Value |
|---|---|
| `agy` version | |
| OS | |
| Date | |
| Tester | |

| Step | Pass / fail | Notes |
|---|---|---|
| 1 Files generated | | |
| 2 Skills discovered | | |
| 3 Arguments | | |
| 4 AGENTS.md loaded | | |
| 5 Workflow commands | | |
| 6 GEMINI.md duplication | | |
| 7 MCP import | | |
| 8 Gemini leftovers | | |

When done, update the README note ("untested against a real `agy`") with the outcome and remove this caveat.
