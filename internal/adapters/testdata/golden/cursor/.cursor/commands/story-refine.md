# story-refine

Refine a backlog story into acceptance criteria

Story: the text the user typed after the command
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

   ## External dependencies
   - <repo name from spec.dependsOn>: <what's needed, one line>
     Suggested: /story:add <text for the other repo>
     Assumed contract (mock): <e.g. GET /tires?code=… → { code, vehicle, number, category }>
   ```

   Include `## External dependencies` only when the story needs another repo; omit it otherwise.

   `<n>` is the number in the ID without leading zeros (US-012 → AC-12.1, AC-12.2…).
5. Rules for criteria:
   - Observable from the user's point of view and testable.
   - 1 to 5 criteria. More than 5 means the story is too big: propose a split with `/story:add` instead.
   - If the conversation changes the meaning of the story, update its wording with `storyline edit <ID> [--as …] [--want …] [--so-that …]` and keep `story.md` consistent with it. Never edit `backlog.yaml` by hand.
6. Related repos: read `storyline/config.yaml`. If the story needs something that belongs to a repo listed in `spec.dependsOn` (data, an endpoint, infrastructure…):
   - Don't implement it here, and don't invent it as if it existed.
   - Tell the user, in one short block: which repo, its URL if known, what is needed, and a ready-to-paste `/story:add …` for that repo, written from that repo's point of view.
   - **Don't wait for an answer about it.** Keep going with this story.
   - Record it under `## External dependencies` in `story.md`, with an assumed contract the mock can follow.
   - If the needed repo isn't in `dependsOn`, say so and suggest adding it to `storyline/config.yaml`.
7. Run `storyline status <ID> ready --story <path to story.md>`. It checks the story file and updates the backlog; do not edit the backlog yourself.

Stop here. Do not plan or write code.

If a `storyline` command fails, show its error to the user and stop. Never work around it by editing `storyline/backlog.yaml` or moving story folders by hand.
