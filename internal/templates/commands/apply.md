---
description: "Implement the next task of a story, one step at a time"
argument_hint: "[story-id] (defaults to the story in progress)"
---

Story: {{ARGS}}
If empty, use the first story with `status: in-progress`, otherwise the first with `status: ready`.

You are the driver; the user is the navigator. Work in small steps and stop after each one.

1. If the story is `ready`, run `storyline status <ID> in-progress`.
2. Take the **first unchecked task** in `tasks.md`. Only that one.
3. Write a failing test whose name contains the criterion ID (e.g. `AC-12.1 shows free slots`). A reference only in a comment does not count for `storyline trace`. Run it and show that it fails.
4. Write the minimum code that makes it pass. Run the test suite.
5. Check the task off in `tasks.md`.
6. Report briefly: files changed and why, and how the user can try it by hand.
7. **Stop and wait.** Do not start the next task until the user says so.

If a task turns out wrong or too big, stop and propose a change to `tasks.md` instead of improvising.

If a `storyline` command fails, show its error to the user and stop. Never work around it by editing `storyline/backlog.yaml` or moving story folders by hand.
