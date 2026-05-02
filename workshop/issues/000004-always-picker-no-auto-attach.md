---
id: 000004
status: working
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
- [ ] Remove the `pick` / `--pick` block at the top of `bin/pair`.
- [ ] Restructure the family-walk decision: collect detached, branch on count, skip picker only when 0.
- [ ] Switch family regex to looser form (`^pair-${BASE_TAG}(-|$)`) for the detached-search.
- [ ] Move agent validation (`command -v`) into the create branch.
- [ ] Update `--help` output to drop the `pick` line and reflect the new "always picker" behavior.
- [ ] Update `atlas/architecture.md` to drop the picker-as-separate-subcommand description.
- [ ] `bash -n` and manual smoke test.

## Log

### 2026-05-02

Created. Triggered by user observation that the auto-attach branch is surprising in a long-lived-session world. Combined with the recognition that `pick` was solving the same problem as the default flow should — so unify and drop the subcommand.
