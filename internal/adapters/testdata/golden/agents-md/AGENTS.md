<!-- storyline:start -->
## Storyline

This project is driven by a backlog of user stories managed with the `storyline` CLI.
The backlog is `storyline/backlog.yaml`: never edit it or move story folders by hand.
Story status flow: `todo → refining → ready → in-progress → done | discarded`.
Change status with `storyline status <id> <status>` (`ready` needs `--story <path>`), close a story with `storyline done <id>`, and check which tests prove each acceptance criterion with `storyline trace <id>`.
Related repos are listed in `storyline/config.yaml` (`spec.dependsOn`): the AI only suggests stories there and never blocks waiting for them.
Stories can be imported from an issue tracker with the import command: the assistant reads the tracker through its own MCP (read-only), and storyline never writes to the tracker.
If a `storyline` command fails, show its error to the user and stop; never work around it by editing files.

When the user asks for one of these steps, do only that step and stop:

- **context**: Describe the project in AGENTS.md: stack, commands, rules, Do / Don't. Input: [anything the AI should know about the project]
- **add**: Add a one-line user story to the storyline backlog. Input: <what the user needs, in free text>
- **import**: Import stories from an issue tracker into the backlog (read-only). Input: <what to import, in free text, e.g. 'connect to Jira and bring the tickets of project ABCD'>
- **refine**: Refine a backlog story into acceptance criteria. Input: [story-id] (defaults to the next open story)
- **slice**: Plan the thinnest vertical slice that satisfies a ready story. Input: [story-id] (defaults to the first ready story)
- **apply**: Implement the next task of a story, one step at a time. Input: [story-id] (defaults to the story in progress)
- **verify**: Check that every acceptance criterion of a story is covered and passing. Input: [story-id] (defaults to the story in progress)
- **done**: Close a verified story and record its behavior in the living docs. Input: [story-id] (defaults to the story in progress)

Run `storyline list` to see the backlog and `storyline validate` after changing it.
<!-- storyline:end -->
