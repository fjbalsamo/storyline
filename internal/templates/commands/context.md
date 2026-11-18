---
description: "Describe the project in AGENTS.md: stack, commands, rules, Do / Don't"
argument_hint: "[anything the AI should know about the project]"
---

Write or update the `## Project` section of `AGENTS.md`, the single source of general project directives.

Extra context from the user: {{ARGS}}

1. **Inspect the repo.** Find languages, frameworks, package manager, folder layout, and the exact commands to build, test, lint, format and run. Read the real files (Cargo.toml, package.json, go.mod, Makefile, CI workflows, linter configs, existing docs). Do not guess.
2. **Ask the user** for what the repo cannot answer, in one batch of at most 5 questions. Always include: "Anything the AI must always do or never do in this project?" Never invent rules.
3. **Pick the test naming convention** for acceptance criteria in this language: `AC-12.1` in the test name when the language allows it, otherwise `ac_12_1_` (Rust, Python, Go…).
4. **Write the `## Project` section** in `AGENTS.md`, outside the `<!-- storyline:start -->` / `<!-- storyline:end -->` block. If the section already exists, update it and keep the user's own edits. Never touch the managed block.
5. **Show the result and stop.**

If the repo is empty, write only what the user answers, plus a note to run this command again once there is code.

## What goes in `## Project`

- **Purpose:** what the project is, in 2–3 lines.
- **Stack:** languages, frameworks, key versions.
- **Commands:** build, test, lint, format, run. Exact commands in code blocks.
- **Layout:** what kind of code lives where, one line per top-level folder.
- **Engineering rules:** error handling, logging, dependencies, anything that always applies.
- **Test conventions:** framework, location, and the criterion-ID naming from step 3.
- **Workflow:** one line saying work is driven by storyline (`stories/backlog.yaml`) and that current behavior is documented in `stories/living/`.
- **References:** links to existing docs (briefs, ADRs, glossary) instead of copying them.
- **Do / Don't:** always the last block of the section.

## Do / Don't

- Two lists of short imperative one-liners, under `### Do` and `### Don't`.
- 3 to 7 items each. Every item is concrete and checkable: "Run `cargo clippy -- -D warnings` before finishing a task", never "write clean code".
- Source every item from the repo or the user's answers. If there are fewer than 3 real ones, write fewer. Never pad.
- Always include these two, adapted to the project:
  - Do: "Name tests after the acceptance criterion they prove (`AC-12.1` / `ac_12_1_`)."
  - Don't: "Don't implement anything not covered by the current story's acceptance criteria."

## What must NOT go in

- Anything about a specific story or feature: acceptance criteria, tasks, designs, decisions made while slicing. That lives in `stories/`.
- Temporary state: "currently working on…", TODOs, known bugs.
- Content already in other docs. Link to it instead.

## Rules for the whole section

- **Durability:** keep a line only if it will still be true after the next ten stories. If unsure, leave it out or ask.
- **Size:** under 60 lines. This file is loaded into every conversation, so shorter is better.

Do not write code or touch any file other than `AGENTS.md`.
