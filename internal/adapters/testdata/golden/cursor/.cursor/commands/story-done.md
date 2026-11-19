# story-done

Close a verified story and record its behavior in the living docs

Story: the text the user typed after the command
If empty, use the first story with `status: in-progress`.

1. Ask the user to confirm they tried it by hand and it works. If not, stop.
2. Merge the story's criteria into `storyline/living/<capability>.md`, where `<capability>` is the user-facing area (e.g. `bookings.md`). Ask if it is unclear.
   - Living docs describe how the product behaves **now**: update criteria that changed instead of appending duplicates.
   - Keep the criterion IDs.
3. Run `storyline done <ID>`. It checks that every task is done, moves the folder to `storyline/done/`, updates the backlog and validates it. Do not move folders or edit the backlog yourself, and do not use `--force` unless the user asks.
4. Show `storyline list`.

If a `storyline` command fails, show its error to the user and stop. Never work around it by editing `storyline/backlog.yaml` or moving story folders by hand.
