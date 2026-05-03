---
id: 000015
status: working
deps: []
created: 2026-05-03
updated: 2026-05-03
---

# Prompt history & future queue navigation in nvim pane

## Problem

Two related shortcomings in the pair drafting flow today:

1. **No way to recall previous prompts.** `log-<tag>.md` accumulates every send, but the user has to leave the pane and grep the file to look at past prompts. There's no in-buffer way to "go back" to what was sent.
2. **No way to queue a thought while the agent is busy.** If an idea strikes while the agent is mid-task and the user wants to send it later (after responding to the current agent reply), there's nowhere to park it. They either lose it, send too early, or maintain it in their head.

## Spec

A unified history-and-queue navigation model overlaid on the nvim draft pane:

### Position model

The nvim buffer represents a virtual cursor over a sequence of slots:

```
[ -N ... -2  -1 ]   *   [ +1  +2 ... +M ]
   history (log)    draft     queue (future)
```

- `-N`: most recent → oldest log entries (immutable).
- `*`: the persistent draft slot, backed by `draft-<tag>.md`.
- `+N`: queued prompts, ordered front-to-back; `+1` is "next to send" semantically.

### Status line

Reintroduce nvim's status line (`laststatus = 2`, custom `statusline`). Format:

```
H < pos[*] > Q
```

- `H` = total history count (entries in `log-<tag>.md`).
- `pos` = `*` | `-N` | `+N`.
- Trailing `*` appears only on `-N` slots when the buffer differs from the loaded baseline (a pending fork awaiting Send/Queue/Discard).
- `Q` = total queue count.

Examples: `42 < * > 4`, `42 < -2 > 4`, `42 < -2* > 4` (dirty fork), `43 < +1 > 4`.

### Keybindings

| Key | Action |
|---|---|
| `Alt+←` | Move toward history (`* → -1 → -2 → ...`); clamps at oldest. |
| `Alt+→` | Move toward queue (`-2 → -1 → * → +1 → ...`); clamps at newest queue item. |
| `Alt+q` | Push current buffer to **front** of queue (`+1`). Return to `*`. Source-slot specifics: from `*`, also clear `*`. From `-N`, the buffer is forked into `+1` (original log entry untouched). From `+N`, the source queue file is removed first (move-to-front). Distinct from `Alt+Return` to avoid send-typos. |
| `Alt+Return` | Send buffer to agent (existing behavior). See Send rules below. |

### Edit-flow rules

Mutability differs per slot:
- `*`: mutable. Edits autosave to `draft-<tag>.md` (existing pre-issue behavior, gated on `nav.pos == '*'`).
- `+N`: mutable. Edits autosave to `queue-<tag>/NNN.md`. Same `BufLeave/FocusLost/InsertLeave` autocmd handles it.
- `-N`: immutable (history can't be rewritten). An edit on `-N` is a *pending fork* that needs a destination. The status line marks the slot dirty (`-N*`).

Navigation rules:

| From    | Behavior on Alt+←/→ |
|---------|---|
| `*`     | Save buffer to draft file (`:w`); load destination. |
| `+N`    | Save buffer to `+N`'s queue file; load destination. |
| `-N` clean | Load destination. |
| `-N` dirty | **Prompt** the user with four options (see below). |

**Dirty-`-N` prompt (`vim.fn.confirm`):**

1. **Send** — append fork to log, return to `*`. Original navigation cancelled.
2. **Queue (+1)** — push buffer to front of queue, return to `*`. Original navigation cancelled.
3. **Discard** — drop edit, proceed with the navigation as requested.
4. **Stay** — cancel navigation, remain at `-N` with edits.

(Default = 4 / Stay, including ESC, so an accidental confirmation doesn't lose work.)

### Send rules (`Alt+Return`)

Send is always an explicit user action — no prompts, no auto-evacuation needed. After every send, the buffer is cleared and we end at `*`.

| From | Effect |
|---|---|
| `*` | Append buffer to `log-<tag>.md`. Clear `*`. (Existing behavior.) |
| `-N` | Append buffer (possibly edited) to log as a fork. The original `-N` log entry is unchanged. Clear `*`. Move to `*`. |
| `+N` | **Remove the `+N` queue file** (consume). Append buffer (possibly edited) to log. Clear `*`. Move to `*`. |

`*` is mutable storage; the persistent draft state lives in `draft-<tag>.md`. Send-from-`*` clears it as the existing semantics; send-from-`-N`/`+N` doesn't displace `*` because navigating to those slots saved `*`'s state already (mutable autosave).

### Storage

- `$DATA_DIR/queue-<tag>/` — directory of one-file-per-prompt. Filenames are zero-padded sortable keys (e.g. `0042.md`). New items at the front get a lower key (use the existing-min minus one, or renumber on insert — TBD in plan). Filenames are an implementation detail; the user only sees `+N` ordinals.
- `log-<tag>.md` — unchanged source of truth for history. Parser separates entries by the existing timestamp header pattern.

### Visual hints

- A flash message (`echo` or `vim.notify`) on auto-evacuate, e.g. `*→ +1 (stashed)`, so the user sees the move.
- Status-line position updates immediately on navigation/send/queue.

### Out of scope (v1)

- Searching/filtering the queue or history.
- Multi-pane preview of upcoming queue items.
- Editing log entries in place.
- Cross-tag history (each tag's log/queue is independent, as today).

## Plan

See `workshop/plans/000015-prompt-history-queue-plan.md`.

## Log

### 2026-05-03
- Issue filed after design discussion. Spec captures the auto-evacuate `*` rule, push-to-front-on-stash semantics, and the send-from-`-N` fork behavior. Awaiting plan + implementation approval.
- Refined send vs navigate-away rules per user feedback: editing `+N` carries no intent to send (navigate-away saves in place), but sending from `+N` consumes the queue file. The two flows share the auto-evacuate-`*` invariant only on send.
- **M1 done.** History-only navigation in `nvim/init.lua`. New helpers: `parse_log`, `read_history`, `buffer_text`, `set_buffer_text`, `pos_label`, `load_baseline_for_current_pos`, `_G.PairStatusline`. Keymaps: `<M-Left>` / `<M-Right>` for `*` ↔ `-N` traversal. Status line reintroduced (`laststatus = 2`) with custom format `H < pos > Q`. Dirty edits on `*` are discarded with a notify warning (M3 will add the proper auto-evacuate-to-queue). The `BufLeave/FocusLost/InsertLeave` autosave is now gated on `nav.pos == '*'` so navigating to `-N` doesn't clobber `draft-<tag>.md`. Verified via headless integration test: T1–T5 plus clamp behavior at oldest history and at `*`, plus send from both `*` and `-N` correctly resetting nav state.
- **M1 wiring fix:** zellij's defaults bind `Alt+Left` / `Alt+Right` to `MoveFocus`, swallowing the keys before nvim sees them. Added `unbind "Alt Left"` / `"Alt Right"` to `zellij/config.kdl` (same pattern as `Alt+i`).
- **M2 done.** Queue store (`§H2`) implemented as `$DATA_DIR/queue-<tag>/NNNNNN.md` files with 6-digit zero-padded keys, starting at 500000 to leave room for both `push_front` and `push_back`. Helpers: `queue_dir`, `queue_keys_sorted`, `queue_count`, `queue_read/write/remove`, `queue_push_back`, `queue_push_front`, `queue_key_for_n`. Initial `Alt+q` was push-to-back from `*` only, but per user feedback was changed to push-to-front from any slot (uniform rule, +N source = move-to-front). `Alt+→` extends past `*` into queue; `Alt+←` walks queue back toward `*`. Editing a `+N` slot then navigating away saves the buffer to that queue file (mutable storage). Sending from `+N` removes the queue file (consume) and appends to log. Statusline now shows real `Q`. Buffer↔file newline accumulation fixed: `set_buffer_text` strips one trailing `\n`, `nav.baseline` is recomputed from `buffer_text()` post-load.
- **Spec refinement before M3:** auto-evacuate-`*` model replaced with explicit user choice on dirty `-N` only. `*` and `+N` are mutable (autosave directly to underlying files); only `-N` has the dirty/fork concept. Status line shows `-N*` mark when dirty.
- **M3 done.** Edit-flow rules implemented:
    - `is_dirty_history_slot()` checks buffer ≠ baseline only on `-N`. `*` and `+N` autosave to their underlying files via `autosave_current_slot()`, called from both the existing `BufLeave/FocusLost/InsertLeave` autocmd and `go_to()` on every navigate.
    - `leave_dirty_history()` shows a `vim.fn.confirm` prompt with four options (Send, Queue (+1), Discard, Stay; default = Stay). Send → ship as fork via `ship_buffer_and_reset`; Queue → `queue_push_front`; Discard → drop edit and proceed; Stay → cancel navigation.
    - `send_and_clear` and `ship_buffer_and_reset` were updated to preserve `*`'s persistent draft when sending from `-N` or `+N` (clear-the-draft semantic now applies only when sending FROM `*`).
    - Status line picks up dirty mark via `is_dirty_history_slot()`.
  - Verified via headless integration test: dirty mark renders, all four prompt options behave correctly (incl. Stay cancellation), `+N` edits autosave with no prompt, draft survives both clean send-from-`-N` and prompt-Send-from-`-N`.
