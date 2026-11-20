---
description: "Import stories from an issue tracker into the backlog (read-only)"
argument-hint: "\u003cwhat to import, in free text, e.g. 'connect to Jira and bring the tickets of project ABCD'\u003e"
---

Import stories from an issue tracker into the storyline backlog. storyline never connects to a tracker: you do, through an MCP the user has connected. This command only reads from the tracker and writes to the backlog.

Request: $ARGUMENTS

1. Find a connected MCP for the tracker the user mentions (Jira, Linear, GitHub Issues, Azure DevOps…). If there isn't one, say which kind of MCP is needed and stop.
2. **Read only.** Never create, update, transition, assign, comment on or close anything in the tracker, even if the request asks for it.
3. Fetch the requested items (a project, a filter, or specific keys). If there are more than 10, show how many and ask whether to import all of them or filter first.
4. For each item, run `storyline list --ref <KEY>`, where `<KEY>` is the item's key as the tracker shows it (`TKT-001`, `owner/repo#123`…; one line). If it already has stories, tell the user which ones and ask before importing it again.
5. Turn each item into one or more stories in **As** <role> / **I want** <goal> / **So that** <benefit> form, with the same rules as the add command:
   - `as` is a real user role, never "developer" or "system".
   - `want` is observable behavior, not an implementation detail.
   - `so_that` is the value the user gets.
   - If an item holds several needs, propose one story per need.
   - Do not invent missing information. If an item is too vague, say so and propose the closest story, marked as needing refinement.
6. Show a preview table and ask **once** for confirmation. Waiting here is fine: the user asked for the import.

   | Key | Proposed stories |
   |---|---|
   | TKT-001 | As a club player, I want … so that … |
   | TKT-002 | As an admin, I want … so that … *(needs refinement: <what is missing>)* |
7. Add each confirmed story with the CLI, one call per story. Stories from the same item share its key as `--ref`. Never edit `storyline/backlog.yaml` by hand for this:
   `storyline add --as "..." --want "..." --so-that "..." --ref <KEY>`
8. Show the result of `storyline list`.

Do not refine, plan or write code.

If a `storyline` command fails, show its error to the user and stop. Never work around it by editing `storyline/backlog.yaml` or moving story folders by hand.
