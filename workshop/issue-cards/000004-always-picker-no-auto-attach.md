---
id: '000004'
status: done
created: 2026-05-02
updated: 2026-05-02
actual_hours: N/A
---

# always show picker, drop the pick subcommand

## Problem

The current `bin/pair` family-walk silently attaches to a detached session when there's exactly one in the family. Discovered in use that this is a bad UX:

- pair sessions are typically long-lived (a coding session might span hours/days).
- A user typing `pair claude` doesn't necessarily remember which detached session they have lying around. Silent attach drops them into a context they may not recognize.
- The "do something explicit" path was `pair pick`, but that's an opt-in command, not the default. The good default should be the explicit one.

Once the picker fires for every interaction-with-existing-sessions, `pair pick` becomes redundant — it duplicates what plain `pair` now does. So drop it as a subcommand.
