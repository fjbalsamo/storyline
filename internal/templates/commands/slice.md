---
description: "Plan the thinnest vertical slice that satisfies a ready story"
argument_hint: "[story-id] (defaults to the first ready story)"
---

Story: {{ARGS}}
If empty, use the first story with `status: ready` in `storyline list -s ready`.

1. Read its `story.md` and the code it will touch.
2. Related repos: read `storyline/config.yaml`. If the story needs something that belongs to a repo listed in `spec.dependsOn` (data, an endpoint, infrastructure…):
   - Don't implement it here, and don't invent it as if it existed.
   - Tell the user, in one short block: which repo, its URL if known, what is needed, and a ready-to-paste `/story:add …` for that repo, written from that repo's point of view.
   - **Don't wait for an answer about it.** Keep going with this story.
   - Plan a mock based on the explicit assumed contract from `## External dependencies` in `story.md`. If the story lacks one, write the assumed contract into `story.md` first.
   - If the needed repo isn't in `dependsOn`, say so and suggest the exact command, e.g. `storyline config add-dependency <repo-name> --url <url>`. Only suggest it: don't run it, don't wait for an answer, and never edit `storyline/config.yaml` by hand.
3. Write `slice.md` next to `story.md`:

   ```markdown
   # Slice: <ID>

   ## Layers touched
   - UI: <screens/components>
   - API: <endpoints/handlers>
   - Data: <tables/fields/storage>

   ## Design
   <the minimum change that satisfies the criteria; reuse what exists>

   ## Mocked (external)
   <repo, what is mocked, and the assumed contract; omit if none>

   ## Not doing (yet)
   <abstractions, fields or generalizations no criterion needs>
   ```

4. Write `tasks.md` next to it: an ordered checklist where each task is one small, reviewable step and names the criteria it serves:

   ```markdown
   - [ ] 1. (AC-12.1) <task>
   - [ ] 2. (AC-12.1, AC-12.2) <task>
   ```

   - Task 1 should cut a thin path through every layer end to end, even if hardcoded. Later tasks fill it in.
   - At most 8 tasks. More means the story is too big: propose a split.
   - A task that depends on another repo uses the mock, so it never waits for that repo.
   - No layer-by-layer plans ("all the DB first, then all the API").
5. Show the plan and wait for the user's approval. Do not write code.

If a `storyline` command fails, show its error to the user and stop. Never work around it by editing `storyline/backlog.yaml` or moving story folders by hand.
