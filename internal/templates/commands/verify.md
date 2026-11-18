---
description: "Check that every acceptance criterion of a story is covered and passing"
argument_hint: "[story-id] (defaults to the story in progress)"
---

Story: {{ARGS}}
If empty, use the first story with `status: in-progress`.

1. Check that every task in `tasks.md` is checked.
2. For each criterion in `story.md`, find the tests whose name contains its ID (e.g. `AC-12.1`).
3. Run the full test suite.
4. Report as a table: criterion | tests | passing?
5. Turn the criteria into a short manual test checklist (Given/When/Then as steps) for the user to try.

Do not fix anything. If there are gaps, list them and suggest `/story:apply`.
