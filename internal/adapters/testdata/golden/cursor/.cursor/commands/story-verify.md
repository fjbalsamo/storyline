# story-verify

Check that every acceptance criterion of a story is covered and passing

Story: the text the user typed after the command
If empty, use the first story with `status: in-progress`.

1. Check that every task in `tasks.md` is checked.
2. Run `storyline trace <ID>` and `storyline validate`. Take the criterion-to-tests mapping from the trace output; do not search for tests yourself. `trace` exits with an error when a criterion has no tests: report that gap.
3. Run the full test suite.
4. Report as a table, from the trace output: criterion | tests | passing?
5. Turn the criteria into a short manual test checklist (Given/When/Then as steps) for the user to try.

Do not fix anything. If there are gaps, list them and suggest `/story:apply`.

If a `storyline` command fails, show its error to the user and stop. Never work around it by editing `storyline/backlog.yaml` or moving story folders by hand. The one exception: `storyline trace` exiting 1 because a criterion has no tests is a result to report, not a failure.
