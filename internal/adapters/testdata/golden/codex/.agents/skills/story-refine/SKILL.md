---
name: story-refine
description: "Refine a backlog story into acceptance criteria"
---

Story: the story id or text the user provided
If empty, use the story marked ▶ in `storyline list`.

1. Read the item in `storyline/backlog.yaml` and the current behavior in `storyline/living/`. Read code only to understand what already exists.
2. If the item is `todo` or `ready`, run `storyline status <ID> refining`.
3. Ask the user the questions needed to remove ambiguity: at most 5, in one batch. Do not invent business rules.
4. Create `storyline/active/<ID>-<kebab-slug>/story.md`:

   ```markdown
   # <ID>: <short title>

   As <as>, I want <want>, so that <so_that>.

   ## Acceptance criteria

   ### AC-<n>.1: <short name>
   - Given <context>
   - When <action>
   - Then <observable outcome>

   ## Out of scope
   - <what this story deliberately does not do>
   ```

   `<n>` is the number in the ID without leading zeros (US-012 → AC-12.1, AC-12.2…).
5. Rules for criteria:
   - Observable from the user's point of view and testable.
   - 1 to 5 criteria. More than 5 means the story is too big: propose a split with `/story:add` instead.
   - If the conversation changes the meaning of the story, update its wording with `storyline edit <ID> [--as …] [--want …] [--so-that …]` and keep `story.md` consistent with it. Never edit `backlog.yaml` by hand.
6. Run `storyline status <ID> ready --story <path to story.md>`. It checks the story file and updates the backlog; do not edit the backlog yourself.

Stop here. Do not plan or write code.

If a `storyline` command fails, show its error to the user and stop. Never work around it by editing `storyline/backlog.yaml` or moving story folders by hand.
