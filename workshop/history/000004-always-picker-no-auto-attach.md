---
id: 000004
status: done
deps: [000001, 000002]
created: 2026-05-02
updated: 2026-05-02
---

# always show picker, drop the pick subcommand

## Problem

The current `bin/pair` family-walk silently attaches to a detached session when there's exactly one in the family. Discovered in use that this is a bad UX:

- pair sessions are typically long-lived (a coding session might span hours/days).
- A user typing `pair claude` doesn't necessarily remember which detached session they have lying around. Silent attach drops them into a context they may not recognize.
- The "do something explicit" path was `pair pick`, but that's an opt-in command, not the default. The good default should be the explicit one.

Once the picker fires for every interaction-with-existing-sessions, `pair pick` becomes redundant — it duplicates what plain `pair` now does. So drop it as a subcommand.

## Spec

**Unified flow for `pair [agent] [variant]`:**

1. Find detached sessions matching the agent family. Use the picker's looser regex `^pair-${BASE_TAG}(-|$)` (so custom-named sessions like `pair-claude-blogging` are included), not the strict `^pair-${BASE_TAG}(-[0-9]+)?$`.
2. Branch on count:
   - **0 detached** → skip picker, go directly to create flow (validate agent, prompt for name, create).
   - **≥1 detached** → fzf picker over the detached set + `+ new <BASE_TAG> session` sentinel. Picking a session attaches; picking the sentinel falls through to create flow.

The "1 detached → silent attach" branch is removed. The "2+ detached → picker" branch generalizes to "≥1 detached → picker."

**Drop `pick` subcommand:** remove the block at the top of `bin/pair` that special-cases `pair pick` and `pair --pick` and `pair pick <agent>`. The default flow now covers all those cases:
- `pair pick` was equivalent to "show me my sessions"; now `pair` (defaulting to claude family) does that.
- `pair pick codex` was equivalent to "show me my codex sessions"; now `pair codex` does that.

**Defer agent validation past the picker.** Currently `command -v "$AGENT"` fires before any session work. Move it inside the create branch so attaching works even when the agent name isn't a real command on PATH (e.g., `pair blogging` to attach to `pair-blogging`, where `blogging` isn't an installed binary).

**Help text:** remove the `pair pick` line.

## Plan

- [x] File this issue.
- [x] Remove the `pick` / `--pick` block at the top of `bin/pair`.
- [x] Restructure the family-walk decision: collect detached, branch on count, skip picker only when 0.
- [x] Switch family regex to looser form (`^pair-${BASE_TAG}(-|$)`) for the detached-search.
- [x] Move agent validation (`command -v`) into the create branch.
- [x] Update `--help` output to drop the `pick` line and reflect the new "always picker" behavior.
- [x] Update `atlas/architecture.md` to drop the picker-as-separate-subcommand description.
- [x] `bash -n` passes; `pair --help` renders the new help.
- [x] Manual smoke test by user.

## Log

### 2026-05-02

Created. Triggered by user observation that the auto-attach branch is surprising in a long-lived-session world. Combined with the recognition that `pick` was solving the same problem as the default flow should — so unify and drop the subcommand.

Implemented. Cleanest part of the change: deleting the entire `pick`/`--pick` block at the top of bin/pair (was ~50 lines) — its functionality folds entirely into the default flow. Net diff is *smaller* than the file was before. Agent validation moved into the create branch so attaching to a custom-named session like `pair blogging` (where `blogging` isn't a binary) works correctly.

**Spec evolved during smoke test.** The original spec said the picker should use the looser family regex `^pair-${BASE_TAG}(-|$)`. User testing surfaced that even this was too restrictive — custom-named sessions without an agent prefix (e.g. `pair-blogging`) wouldn't show up under `pair claude` because they don't match anything claude-shaped. Final decision: drop the agent filter from the picker entirely; show *all* detached `pair-*` sessions regardless of agent. The agent argument is now only meaningful for the create path (sentinel label, default name, binary to exec). Picker prompt changed from `${BASE_TAG}>` to `pair>` to reflect the new scope.

**A subtle bug worth noting** for future reference: at one point a `Write` call to `bin/pair` reported success but the file content didn't actually update on disk. Subsequent `Edit` calls then patched against text that no longer matched, all reporting success silently. Caught only when user re-tested and reported "I see pair-claude* still." Lesson: after a Write to overwrite an existing file, run `grep` to verify the change landed before chaining further Edits.

Closed.
