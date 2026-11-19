---
description: "Add a one-line user story to the storyline backlog"
argument-hint: "\u003cwhat the user needs, in free text\u003e"
---

Turn the request below into user stories and add them to the backlog.

Request: $ARGUMENTS

1. Rewrite it as **As** <role> / **I want** <goal> / **So that** <benefit>.
   - `as` is a real user role, never "developer" or "system".
   - `want` is observable behavior, not an implementation detail.
   - `so_that` is the value the user gets.
2. If the request hides several needs, propose one story per need and ask which ones to add.
3. If anything is ambiguous, ask before adding. Do not invent business rules.
4. Add each story with the CLI, one call per story. Never edit `storyline/backlog.yaml` by hand for this:
   `storyline add --as "..." --want "..." --so-that "..."`
5. Show the result of `storyline list`.

Do not refine, plan or write code.

If a `storyline` command fails, show its error to the user and stop. Never work around it by editing `storyline/backlog.yaml` or moving story folders by hand.
