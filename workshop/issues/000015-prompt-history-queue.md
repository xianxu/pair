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
H < pos > Q
```

- `H` = total history count (entries in `log-<tag>.md`).
- `pos` = `*` | `-N` | `+N`.
- `Q` = total queue count.

Examples: `42 < * > 4`, `42 < -2 > 4`, `43 < +1 > 4`.

### Keybindings

| Key | Action |
|---|---|
| `Alt+←` | Move toward history (`* → -1 → -2 → ...`); clamps at oldest. |
| `Alt+→` | Move toward queue (`-2 → -1 → * → +1 → ...`); clamps at newest queue item. |
| `Alt+q` | Append current buffer to **end** of queue, clear `*`, stay on `*`. (Distinct from `Alt+Return` to avoid send-typos.) |
| `Alt+Return` | Send buffer to agent (existing behavior). See Send rules below. |

### Edit-flow rules (the hard part)

The buffer is the single editing surface. "Baseline" is the canonical content for the current slot. A "dirty" buffer = `buffer ≠ baseline`.

**Baselines per slot:**
- `*`: contents of `draft-<tag>.md`
- `-N`: the Nth-from-end entry parsed out of `log-<tag>.md`
- `+N`: contents of `$DATA_DIR/queue-<tag>/NNN.md`

**Navigation rule (single-rule formulation):**
> When navigating away from a slot with a dirty buffer, the buffer's content must land somewhere recoverable, then the new baseline loads.

Concretely:

| From | Dirty content lands... |
|---|---|
| `*` | Push to **front** of queue (`queue-<tag>/` gets a new file with the lowest sort key). Clear `*`. |
| `-N` | Flow to `*`. **If `*` already has content, auto-evacuate `*` to queue front first**, then write the flowed content to `*`. |
| `+N` | Save back in place to `queue-<tag>/NNN.md`. |

Pure browsing (buffer == baseline at every navigation) never mutates anything.

**Auto-evacuate corollary:** the `*` slot self-protects. Anything that would overwrite a non-empty `*` first pushes the existing `*` content to queue front. This means a chain of N history-slot edits leaves N-1 items at the queue front and the latest at `*`, in reverse-chronological order.

### Send rules (`Alt+Return`)

Send and navigate-away are distinct flows because editing `+N` carries no intent to send (it's an in-place update of a queued item). Unification only applies on send.

**Send invariant:** after every send, `*` is empty and no content was silently destroyed (displaced `*` content always lands in the queue front).

| From | Effect |
|---|---|
| `*` | Append buffer to `log-<tag>.md`. Clear `*`. (Existing behavior.) |
| `-N` | Auto-evacuate `*` to queue front if non-empty; flow buffer into `*`; append to log; clear `*`. Move to `*`. The original `-N` log entry is unchanged (fork). |
| `+N` | Auto-evacuate `*` to queue front if non-empty; **remove the `+N` queue file** (consume); flow buffer into `*`; append to log; clear `*`. Move to `*`. |

The `+N` row's "consume" step is what differentiates send from navigate-away: navigating away from a dirty `+N` would have saved-in-place to the queue file; sending instead removes the file and ships its content.

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
- **M2 done.** Queue store (`§H2`) implemented as `$DATA_DIR/queue-<tag>/NNNNNN.md` files with 6-digit zero-padded keys, starting at 500000 to leave room for both `push_front` (M3) and `push_back`. Helpers: `queue_dir`, `queue_keys_sorted`, `queue_count`, `queue_read/write/remove`, `queue_push_back`, `queue_push_front` (used in M3), `queue_key_for_n`. `Alt+q` (`queue_current`) appends current `*` buffer to the back, clears `*`. Restricted to `*` only (warn-and-noop from `-N`/`+N` to avoid duplicate-vs-save ambiguity). `Alt+→` extends past `*` into queue; `Alt+←` walks queue back toward `*`. Editing a `+N` slot then navigating away saves the buffer to that queue file (mutable storage). Sending from `+N` removes the queue file (consume) and appends to log. Statusline now shows real `Q`. The `*`/`-N` discard-on-leave path is unchanged (M3 work). Verified via headless integration test: queue lifecycle (push_back, navigate, edit-in-place, persist across nav, send-consume), Alt+q from `-N` warns and no-ops, autosave gate still protects draft when navigating into queue.
