---
description: "Plan the thinnest vertical slice that satisfies a ready story"
argument-hint: "[story-id] (defaults to the first ready story)"
---

Story: $ARGUMENTS
If empty, use the first story with `status: ready` in `storyline list -s ready`.

1. Read its `story.md` and the code it will touch.
2. Write `slice.md` next to `story.md`:

   ```markdown
   # Slice: <ID>

   ## Layers touched
   - UI: <screens/components>
   - API: <endpoints/handlers>
   - Data: <tables/fields/storage>

   ## Design
   <the minimum change that satisfies the criteria; reuse what exists>

   ## Not doing (yet)
   <abstractions, fields or generalizations no criterion needs>
   ```

3. Write `tasks.md` next to it: an ordered checklist where each task is one small, reviewable step and names the criteria it serves:

   ```markdown
   - [ ] 1. (AC-12.1) <task>
   - [ ] 2. (AC-12.1, AC-12.2) <task>
   ```

   - Task 1 should cut a thin path through every layer end to end, even if hardcoded. Later tasks fill it in.
   - At most 8 tasks. More means the story is too big: propose a split.
   - No layer-by-layer plans ("all the DB first, then all the API").
4. Show the plan and wait for the user's approval. Do not write code.

If a `storyline` command fails, show its error to the user and stop. Never work around it by editing `stories/backlog.yaml` or moving story folders by hand.
