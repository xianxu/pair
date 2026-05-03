---
id: 000002
status: done
deps: [000001]
created: 2026-05-02
updated: 2026-05-02
---

# each pair session may have a name

User should be able to give each pair session a meaningful name (e.g. `bugfix`, `exploration`, `customer-x`) instead of accepting the auto-generated `pair-claude-5` style suffix. Named sessions show up in the picker, so the user can reattach by recognizing the name later.

## Spec

**Trigger:** any time `bin/pair` decides a new session needs to be created — i.e. the create branch of the launcher's family-walk decision tree. This covers:
- 0 detached sessions in the family (auto-rename to next-free slot).
- "+ new session" sentinel picked from the multi-detached fzf picker.
- The first-ever launch of an agent (BASE_TAG slot is missing).

**Prompt:** before `exec zellij`, show the user the default session name and accept input. Default = the full session name we'd use otherwise (e.g. `pair-claude-5`).

```
Session name [pair-claude-5]: <user input or Enter>
```

- Empty input (Enter) → use default unchanged.
- Non-empty input → use as the session name, stripping any leading `pair-` prefix the user typed (so both `pair-bugfix` and `bugfix` work).
- Invalid characters or collision with an existing session → error and exit 1 (user re-runs).

**Validation:** name must match `[A-Za-z0-9_-]+`. Collision check via `zellij list-sessions --short`.

**Effects of a custom name:**
- `SESSION` becomes `pair-<name>`.
- `PAIR_TAG` is the name itself — drives draft and log file paths (`pair-draft-<name>.md`, `pair-log-<name>.md`).
- The session is still created with the same agent (`PAIR_AGENT` stays whatever the positional arg said), so e.g. naming a claude session "scratch" gives you `pair-scratch` running claude.
- Custom-named sessions are *not* part of the auto-rename family — the family-walk regex `^pair-<base>(-[0-9]+)?$` only matches the base name and numeric suffixes. So `pair claude` won't auto-attach to `pair-bugfix`. Reattach to custom names is via `pair pick claude` (which uses the looser `^pair-claude(-|$)` filter).

**Out of scope:**
- `-n NAME` flag for non-interactive naming. Could be added later; for v2 scope keep it interactive only.
- Re-using draft/log files of a custom-named session that was previously deleted. The files persist; user gets the old draft back if they re-create with the same name.

## Plan

- [x] In the create branch of `bin/pair`, prompt for session name with default = current `pair-${chosen_tag}`.
- [x] Strip leading `pair-` from user input if present.
- [x] Validate: regex `[A-Za-z0-9_-]+`, error on bad input.
- [x] Collision check against `zellij list-sessions --short`, error if exists.
- [x] Update `chosen_tag` from the prompt response, so SESSION / PAIR_TAG / DRAFT all derive correctly.
- [x] Read from `/dev/tty` so the prompt works even if stdin is redirected.
- [x] Verify with `bash -n`.
- [x] Manual test: run `pair claude` from a fresh state, verify prompt shows, accept default works, custom name works.
- [x] Update atlas/architecture.md to note the naming behavior.
- [x] Mark done after user smoke-tests.

## Log

### 2026-05-02

Created. Spec evolved in conversation: original idea was `pair -n "name"` flag; settled on interactive prompt with default pre-shown so the common path (just hit Enter) stays one keystroke, while the rename path is discoverable without flags.

Implemented in `bin/pair`. Restructured the create vs. attach branches so the prompt only fires on create. The PAIR_TAG export was moved per-branch (was previously hoisted above the if-attach, which would have made the prompt's tag-rewrite ineffective). All non-interactive checks pass (`bash -n`). Atlas architecture doc updated to note the family-walk + naming + picker behaviors.

Status `working` until user does the manual smoke test (interactive `read` can't be exercised by automated checks).

User confirmed it works in real use, including a `pair-blogging` session created via the prompt. Closed.
