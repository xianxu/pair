---
id: 000022
status: working
deps: []
created: 2026-05-25
updated: 2026-05-25
related: [bin/pair, bin/pair-restart.sh, bin/pair-quit.sh, nvim/init.lua, zellij/config.kdl]
---

# Rename a pair tag without losing the agent session

## Problem

A pair tag is the durable identity of a coding session (zellij session
name `pair-<tag>`, saved agent session in `config-<tag>-<agent>.json`,
draft `draft-<tag>.md`, etc.). Today the tag is chosen once at create
time and frozen.

Common workflow: start with a generic tag (`brain-2`) for exploratory
work, then narrow into a specific train of thought (`gstack-deep-dive`).
The user wants the tag to follow the work, not the other way around —
without dropping the agent's conversation, draft buffer, scrollback
history, or queued/quoted items that have accumulated.

There is no live-rename for a zellij session — the session name is
baked into the running session. So "rename" is necessarily a *quit,
swap tag on disk, re-exec* operation. That's structurally identical
to `pair-restart.sh` plus a tag-swap step, and the natural surface
is to fold the rename gesture into the existing restart confirm
(Ctrl+Alt+n / Alt+n).

## Spec

### Two-layer split

**Layer (a) — primitive: `pair rename <old> <new>`** (offline CLI).
Renames every tag-scoped data file on disk for a stopped session.
Refuses if anything is unsafe (live session, name collision, missing
old tag). No re-exec. Composable from anywhere.

**Layer (b) — gesture: Ctrl+Alt+n confirm grows a (R)ename option.**
Inside-session UX. The existing Y/N confirm becomes Y/N/R; choosing
R prompts for a new tag, validates it, then runs the quit → rename →
re-exec choreography. Built on top of (a) so the file-renaming logic
isn't duplicated.

Same R option added to Shift+Alt+N (rename + fresh agent session).

### Tag-scoped file families

Enumerated by grepping `DATA_DIR` paths across `bin/`, `nvim/`, `cmd/`
(matches as of M0). All under `$PAIR_DATA_DIR/`:

```
agent-<tag>                              # current agent name for the tag
agent-output-<tag>                       # latest output capture
agent-picks-<tag>                        # selection-clipboard picks history
agent-pid-<tag>                          # agent process pidfile
cmux-title-pid-<tag>                     # title-poller pidfile
config-<tag>-<agent>.json                # per-(tag,agent) saved config
draft-<tag>.md                           # default draft buffer
draft-<tag>-<agent>.md                   # per-agent draft variants (optional)
image-capture-<tag>                      # image capture state file
image-capture-<tag>.done                 # image capture sentinel
layout-mode-<tag>                        # layout ladder rung state
log-<tag>.md                             # session log (if present)
nvim-pid-<tag>-{draft,scrollback}        # nvim pidfiles per pane kind
outer-tty-<tag>                          # outer PTY path
pair-wrap-pid-<tag>                      # pair-wrap pidfile
queue-<tag>                              # paste-queue staging
quote-<tag>                              # paste-quote staging
scrollback-<tag>-<agent>.{ansi,raw,      # scrollback artifacts (per agent)
  viewport,events.jsonl}
```

The pidfiles and `outer-tty-<tag>` are inherently stale once the
session is killed, but renaming them is still cheap and keeps the
post-rename state consistent for any tooling that scans by tag.

### Matching discipline (substring safety)

Tag patterns must be matched on word boundaries, not substrings.
`brain` is a prefix of `brain-2`; a naive `*brain*` glob would
sweep both. Match rules:

- `<prefix>-<tag>` exact (no further `-<chars>`)              → e.g. `agent-brain-2`
- `<prefix>-<tag>-<rest>` (the `<rest>` is structured)        → e.g. `config-brain-2-claude.json`
- `<prefix>-<tag>.<ext>` exact                                → e.g. `draft-brain-2.md`

The renamer enumerates the families above explicitly rather than
globbing, so each family encodes its exact `(prefix, has-rest,
has-ext)` shape.

### File family registry

Today the tag-suffixed paths are hard-coded across many scripts.
Introduce one source of truth — a small helper that enumerates
`(family-id, src-path, dst-path)` tuples for a given `(old, new)`
pair. Two options:

- **bash helper**, `bin/pair-tag-files`: `pair-tag-files list <tag>`
  prints all paths for a tag (one per line). `pair-tag-files plan
  <old> <new>` prints `src\tdst` pairs. Used by `pair rename` and
  by any future tag-scoped op (delete, archive, export). Keeps the
  registry in shell, matching the rest of the family.
- **Go cmd**, similar surface, if the bash registry feels fragile
  (path-with-spaces handling, etc.).

Default to the bash helper. Promote to Go only if a real friction
appears.

### Safety checks

`pair rename <old> <new>` must verify *all* of these before
touching disk:

1. `<new>` non-empty, charset `[A-Za-z0-9_-]+`, length cap (256).
2. `<old> != <new>` (no-op rejected with a hint).
3. `pair-<old>` is not in `zellij list-sessions` (live or
   resurrectable). The live-rename refusal is a hard fail with a
   pointer to the inside-flow restart path.
4. `pair-<new>` is not in `zellij list-sessions` (live, dead, or
   resurrectable).
5. No file matching any `<new>` family exists in `$PAIR_DATA_DIR`.
6. At least one file matching any `<old>` family exists (otherwise
   the tag has nothing to rename; user typo).
7. None of the planned `dst` paths already exist (redundant with
   §5 but cheap and explicit at the per-file level).

Failure mode for each → exit non-zero with a one-line `error:` that
names what's blocking. The inside flow surfaces these as a re-prompt
in nvim rather than dropping back to the viewer.

### Atomicity

Build the full `(src, dst)` plan in memory (or in a journal file),
validate every entry, then execute the renames in one pass. If any
single `mv` fails mid-execution, abort and replay-undo the renames
done so far. The plan file (`$PAIR_DATA_DIR/.rename-<old>-to-<new>.plan`)
also gives crash-recovery: on next `pair rename` or `pair`
invocation, if a stale plan file is found, finish or roll back
based on what's on disk vs. the plan.

For the inside flow, also take a flock on `$PAIR_DATA_DIR/.rename.lock`
across the kill → rename → re-exec sequence, to prevent another `pair
<new>` launching concurrently and stealing the new name in the
window between zellij kill and the file rename.

### Inside-flow UX (b)

The current `PairConfirmRestart()` / `PairConfirmRestartNewSession()`
in `nvim/init.lua` use `vim.fn.confirm` (Y/N). Replace with a small
custom prompt accepting `y`/`n`/`r`:

```
Restart this pair session? (Y)es / (N)o / (R)ename
```

`r` → second prompt via `vim.fn.input("New tag: ", current_tag)`
(pre-filled with the live tag for edit-in-place). Empty or
unchanged → fall back to plain restart. Validation echoes the (a)
CLI's checks; on failure, re-prompt with an inline error line
rather than dropping back to the viewer.

On valid input, set `PAIR_RENAME_TO=<new>` in the env passed to
`pair-restart.sh`. `pair-restart.sh` gains a `~5-line` block:
after the zellij kill, before the re-exec, if `PAIR_RENAME_TO`
is set, run `pair rename "$tag" "$PAIR_RENAME_TO"`. Re-exec uses
`PAIR_FORCE_TAG=$PAIR_RENAME_TO` instead of the original tag.

### Atlas update

`atlas/architecture.md` already has a "Tag-restart" section from
issue #16. Add a short subsection: "Tag rename — quit/rename/re-exec
choreography; primitive is `pair rename`, gesture is the (R)ename
option on the restart confirm." Plus the file-family registry as the
canonical place to look up "what files are scoped to a tag" — that
list will get re-used by any future tag-scoped op.

## Plan

- [ ] **M1 — registry + primitive.** Add `bin/pair-tag-files` (list +
      plan subcommands). Add `bin/pair-rename.sh` (or `pair rename`
      subcommand in `bin/pair`) implementing the offline CLI per the
      spec's safety + atomicity rules. Unit-test against a fixtured
      `$PAIR_DATA_DIR` covering: clean rename, substring-collision
      tag (`brain` vs `brain-2`), partial-failure rollback, stale
      plan-file recovery, live-session refusal, name-collision
      refusal. No nvim/zellij touch at this milestone.

- [ ] **M2 — restart confirm gains (R).** `nvim/init.lua`:
      `PairConfirmRestart()` and `PairConfirmRestartNewSession()`
      switch from `vim.fn.confirm` to a custom y/n/r prompt; on `r`,
      `vim.fn.input` for new tag, then validation, then exec
      `pair-restart.sh` with `PAIR_RENAME_TO` set. `pair-restart.sh`
      gains the rename glue (kill → rename → re-exec).

- [ ] **M3 — flock + crash recovery.** Wire the flock on
      `$PAIR_DATA_DIR/.rename.lock` around the inside-flow choreography.
      Add the stale-plan-file detection on `pair` startup that finishes
      or rolls back a half-done rename. Test by interrupting the
      rename mid-flight (kill -9 on the renamer between the zellij
      kill and the file moves).

- [ ] **M4 — atlas + README.** Atlas tag-rename subsection, registry
      pointer. README adds (R) to the Ctrl+Alt+n keybind row and a
      "Rename a tag" subsection walking the flow.

## Log

- 2026-05-25: issue filed. Design discussed with operator: (a) offline
  CLI primitive first, (b) inside-flow gesture folded into
  Ctrl+Alt+n's restart confirm. File-family list grounded by
  grepping `DATA_DIR` paths across `bin/`, `nvim/`, `cmd/` —
  enumerated in the Spec.
