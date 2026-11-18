---
description: "Refine a backlog story into acceptance criteria"
argument-hint: "[story-id] (defaults to the next open story)"
agent: agent
---

Story: ${input:args:[story-id] (defaults to the next open story)}
If empty, use the story marked ▶ in `storyline list`.

1. Read the item in `stories/backlog.yaml` and the current behavior in `stories/living/`. Read code only to understand what already exists.
2. Set the item's `status` to `refining` and run `storyline validate`.
3. Ask the user the questions needed to remove ambiguity: at most 5, in one batch. Do not invent business rules.
4. Create `stories/active/<ID>-<kebab-slug>/story.md`:

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
6. In the backlog item, set `status: ready` and `story: <path to story.md>`. Change only those two fields, then run `storyline validate`.

Stop here. Do not plan or write code.
