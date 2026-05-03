# Plan: prompt history & future queue navigation

Implementation plan for `workshop/issues/000015-prompt-history-queue.md`.

## Files touched

- `nvim/init.lua` — bulk of the work: parser, state, keymaps, statusline, autocmds.
- `bin/pair` — ensure `$DATA_DIR/queue-<tag>/` exists at session create (analogous to draft file init).
- `atlas/architecture.md` — document the new per-tag concept (queue dir, position model, statusline).
- `README.md` — keybindings table + a short paragraph on history/queue.
- `nvim/init.lua` cleanup: re-enable `laststatus` and define a custom `statusline`.

## Module layout inside `nvim/init.lua`

Group new code into one logical block (no external files needed for v1; keep the `nvim -u` single-file isolation property). Sections:

```
-- §H1: history parser            (pure)
-- §H2: queue store ops           (filesystem)
-- §H3: navigation state          (module-local table)
-- §H4: edit-flow primitives      (dirty check, evacuate, flow)
-- §H5: navigation actions        (alt-left, alt-right, alt-q)
-- §H6: send integration          (modify existing send_and_clear)
-- §H7: statusline                (format function + laststatus)
-- §H8: keymaps + autocmds        (wire up)
```

## Data model

### History parsing (`§H1`)

Pure function:
```lua
-- parse_log(text) -> { "entry1 body", "entry2 body", ... }   -- index 1 = oldest
local function parse_log(text)
  -- entries are separated by "\n\n---\n\n"; each begins with
  -- "## YYYY-MM-DD HH:MM:SS\n\n" — strip that header to recover the body.
end
```

Reads all of `log-<tag>.md` once per navigation event (cheap; logs are small markdown files). If perf matters later, cache + invalidate on file mtime.

History indexing: `-1` = most recent = last element of returned list. `-N` = `list[#list - N + 1]`.

### Queue store (`§H2`)

Directory `$DATA_DIR/queue-<tag>/`. Filenames are zero-padded keys; ordering is by sort. We only need:

```lua
queue.list()       -- returns { {key="0042", body="..."}, ... } in display order (+1 first)
queue.read(key)    -- string body
queue.write(key, body)
queue.remove(key)
queue.push_front(body)  -- generate a new key < current min, write, return key
queue.push_back(body)   -- generate a new key > current max, write, return key
```

**Key generation:** start from a sortable wide range so push_front and push_back never collide. Use a 6-digit zero-padded counter starting at `500000`. push_front decrements the current min by 1; push_back increments current max by 1. If the front hits `000000` (vanishingly unlikely), renumber. Document this in a comment; don't write a renumber routine in v1 unless we hit it.

`+1` (display) = the lowest-keyed file. Display index N maps to the Nth entry of `sort(filenames)`.

### Navigation state (`§H3`)

Module-local `nav` table:

```lua
local nav = {
  pos = '*',           -- '*' | { kind='history', n=2 } | { kind='queue', key='000499' }
  baseline = '',       -- buffer content at last load; used for dirty check
}
```

A small abstraction `position_describe()` returns `{ kind, ord, baseline }` so callers don't poke at the table directly.

### Edit-flow primitives (`§H4`)

```lua
local function buffer_text() ... end       -- nvim_buf_get_lines + concat
local function set_buffer_text(s) ... end  -- nvim_buf_set_lines
local function is_dirty()                  -- buffer_text() ~= nav.baseline
local function flush_star_to_queue_front_if_dirty(text)
  -- internal helper: only used when something is about to overwrite *
  -- read draft file; if non-empty, queue.push_front(its content); truncate draft.
end
local function evacuate_current(text)
  -- precondition: is_dirty() and nav.pos ~= '*'
  -- behavior depends on pos.kind:
  --   'history': flush * if non-empty, then write text to draft file.
  --   'queue':   queue.write(nav.pos.key, text)
end
```

Save-on-leave is invoked from inside the navigation actions (`§H5`).

### Navigation actions (`§H5`)

```lua
local function nav_left()
  -- if dirty and pos == '*': queue.push_front(buffer_text); clear draft file.
  -- elif dirty and pos.kind == 'history': evacuate_current(buffer_text).
  -- elif dirty and pos.kind == 'queue': queue.write(...).
  -- compute new pos (clamp to oldest history).
  -- load new baseline into buffer; update nav.baseline.
  -- redraw statusline; show flash if we just stashed.
end

local function nav_right() ... end   -- symmetric

local function queue_current()       -- Alt+q
  -- push_back(buffer_text); clear draft file; nav.pos = '*'; reload baseline (empty).
end
```

### Send integration (`§H6`)

Send and navigate-away are distinct flows. Navigation rules are about preserving in-flight edits in their natural slot. Send rules are about routing the buffer to the log, with `*` as the universal staging slot.

**Send invariant:** after every send, `*` is empty and no content is silently destroyed (any prior `*` content is preserved in the queue front).

Modify `send_and_clear` (currently at `nvim/init.lua:289`):

```lua
local function send_and_clear()
  local body = buffer_text()
  if body == '' then return end

  -- Stage the buffer into * before sending. * is the canonical staging slot,
  -- so that send-from-* semantics (clear *, append to log) applies uniformly.
  if nav.pos ~= '*' then
    -- Auto-evacuate * if non-empty, regardless of source slot — this is the
    -- "no silent destruction" half of the invariant.
    flush_star_to_queue_front_if_non_empty()

    if nav.pos.kind == 'queue' then
      -- Send-from-+N consumes the queue file: the buffer (possibly edited)
      -- is what ships, and the queue slot is removed. Note we remove BEFORE
      -- writing draft so the displaced content (if any) lands at queue front
      -- correctly relative to the consumed slot.
      queue.remove(nav.pos.key)
    end
    -- For both -N and +N: write buffer to draft so canonical send-from-*
    -- ships it.
    write_draft_file(body)
  end

  send_to_agent(body)
  append_log(body)
  write_draft_file('')
  set_buffer_text('')
  nav.pos = '*'
  nav.baseline = ''
  refresh_statusline()
end
```

**Resulting send rules per source slot** (derived from the two-step "stage to *, send-from-*"):

| From  | Effect |
|---|---|
| `*`   | Append `*` to log; clear `*`. (Existing behavior.) |
| `-N`  | Auto-evacuate `*` to queue front if non-empty; flow buffer to `*`; send-from-`*`. End: `*` empty, log+1, queue may have grown by 1. Original `-N` log entry untouched (fork). |
| `+N`  | Auto-evacuate `*` to queue front if non-empty; remove `+N` queue file; flow buffer to `*`; send-from-`*`. End: `*` empty, log+1, queue net change = 0 if `*` was non-empty (gained one at front, lost the consumed `+N`); -1 if `*` was empty. |

**Contrast with navigate-away rules:** navigating from `+N` saves in place (queue is mutable storage and the user's intent is to update, not ship). Sending from `+N` removes the queue file (intent is to ship and consume). This asymmetry is the whole point of having distinct rule sets.

### Statusline (`§H7`)

```lua
function _G.PairStatusline()
  local h = #history_entries_cached_or_reread
  local q = #queue.list()
  local pos = format_pos(nav.pos)   -- '*' or '-2' or '+1'
  return string.format(' %d < %s > %d ', h, pos, q)
end

vim.opt.laststatus = 2
vim.opt.statusline = '%!v:lua.PairStatusline()'
```

The H/Q values must be recomputed lazily. We re-read on TextChanged events sparingly: actually, statusline is re-evaluated by nvim on most redraws automatically, so just keep `PairStatusline()` cheap. A small in-memory cache keyed on `log_mtime + queue_dir_mtime` keeps it free.

### Keymaps + autocmds (`§H8`)

```lua
vim.keymap.set({'n','i'}, '<M-Left>',  nav_left,        { silent = true })
vim.keymap.set({'n','i'}, '<M-Right>', nav_right,       { silent = true })
vim.keymap.set({'n','i'}, '<M-q>',     queue_current,   { silent = true })
-- Alt+Return already mapped to send_and_clear; just modify the function body.
```

Autocmds — none required. nvim re-evaluates `statusline` on virtually every redraw; navigation actions explicitly invalidate cache.

## Edge cases & decisions

1. **Empty `*` on Alt+←:** no push (nothing to stash). Just load history baseline.
2. **Alt+← when at oldest history (`-N` where N == #log):** clamp; flash a brief "at oldest" hint or no-op silently. Pick: silent no-op.
3. **Alt+→ when at newest queue (`+M`):** clamp; silent no-op.
4. **Alt+→ from queue middle: `+2 → +1 → * → -1 ...`:** standard sequence. Edits at `+2` save-back to queue file before loading `+1`.
5. **Empty queue dir:** `queue.list()` returns `{}`; `+N` is unreachable; statusline shows `Q=0`.
6. **Empty log:** statusline shows `H=0`; `-N` unreachable.
7. **Concurrent writes to log:** the agent pane's send runs from the same nvim — no concurrent writer. Skip locking.
8. **Image references in drafts:** `[Image #N]` tokens are just text — they navigate fine. The image-paste counter (`PAIR_IMG_COUNTER`) is separate state and unaffected.
9. **Queue file with empty body:** treat as non-empty (display as `+N` even if blank). Sending an empty buffer is already a no-op (existing send code returns on empty).

## Testing strategy

No test framework exists in repo. Verify manually by working through the issue's walkthrough scenarios as a checklist. Each scenario specifies the expected statusline at each step:

```
T1. Empty start. Type "abc". Status: 0 < * > 0.
T2. Send. Status: 1 < * > 0. Buffer empty.
T3. Type "def". Send. Status: 2 < * > 0.
T4. Type "draft1". Alt+←. Status: 2 < -1 > 1. Buffer="def". Queue=[draft1].
T5. Alt+←. Status: 2 < -2 > 1. Buffer="abc".
T6. Edit to "abc-mod". Alt+←. Status: 2 < -2 > 1 (clamped at oldest). Buffer should still show -2 baseline; flow happened?
    → Decision: clamping = no nav happened, so no flow. Buffer keeps "abc-mod" (still dirty at -2).
T7. Alt+→. Status: 2 < -1 > 2. "abc-mod" flowed to *. Queue front reordered? No — flow puts it in *, then auto-evacuate isn't needed (* was empty). Queue still =[draft1].
T8. Alt+→. Status: 2 < * > 2. Buffer="abc-mod".
T9. Send. Status: 3 < * > 2. Buffer="".
T10. Multi-edit chain: type "x". Alt+←. Status: 3 < -1 > 3 (was 2, "x" pushed to front).
     Edit "x"→"x1". Alt+←. Status: 3 < -2 > 4 (no — wait, "x1" flows to *, * was empty so no extra push).
     → Re-walk: 3 < -2 > 3. Buffer reloaded to -2 baseline. * holds "x1".
T11. Edit -2 to "y2". Alt+←. * has "x1" → push to queue front. Then write "y2" to *. Status: 3 < -3 (or clamp) > 4.
     → If only 3 history entries, Alt+← from -2 clamps. So Alt+→ instead:
T11'. From T10 (3 < -2 > 3, *="x1"), edit -2 to "y2". Alt+→. Need to evacuate dirty -2: "y2" flows to *; * has "x1" so first push "x1" to front → queue len 4. Write "y2" to *. Move to -1. Status: 3 < -1 > 4.
T12. Verify Alt+q on draft pushes to back, not front:
     From scratch: type "future-task". Alt+q. Status: 0 < * > 1. Buffer=''. Type another. Alt+q. 0 < * > 2. The first one is at +1, second at +2 (back-of-queue ordering).
```

These scenarios become a manual-verification script in the issue's Log section once implemented.

## Milestones

Each milestone leaves the script in a working state. After each, log status in the issue's Log section.

### M1: History-only navigation, read-only

- §H1 parser + §H3 state (no queue references yet).
- Alt+← / Alt+→ navigate `*` ↔ `-N` only.
- No edit-flow yet — Alt+← discards dirty edits silently with a `vim.notify` warning.
- Statusline: `H < pos > 0`.
- **Verify:** scenarios T1–T5.

### M2: Queue store + Alt+q

- §H2 queue ops.
- Alt+q pushes buffer to back, clears draft.
- Alt+→ extends past `*` into queue.
- Editing on `+N` saves back to queue file.
- Sending from `+N` removes queue file.
- Statusline shows real `Q`.
- **Verify:** T12 plus general Alt+→ traversal.

### M3: Edit-flow with auto-evacuate

- §H4 primitives: flow-to-`*`, auto-evacuate to queue front.
- Wire into `nav_left`/`nav_right`.
- Flash on auto-evacuate (`vim.notify` with INFO level).
- **Verify:** T7, T10, T11'.

### M4: Send integration

- Update `send_and_clear` per the refined send rules.
- Send-from-`-N` and send-from-`+N` correctly preserve the persistent draft.
- **Verify:** sending from each slot type, draft survival.

### M5: Polish + docs

- Atlas update (architecture.md): document position model, queue dir, statusline.
- README keybindings table.
- Help text in `bin/pair`.

## Open implementation questions (raise during build)

- Does nvim's `<M-Left>` map cleanly across Linux+macOS terminals? If terminal forwards Alt+arrow as ESC sequences inconsistently, may need explicit `<Esc>[1;3D` etc. fallbacks. Test early.
- Persistent undo: when we `set_buffer_text` to load a baseline, it lands in the undo tree. Acceptable? Probably yes — user can `u` to recover prior buffer, which is a happy accident for the "I accidentally navigated and overwrote my draft" case (though they should rely on the queue stash, not undo).
- Image references in queued items: a queued prompt like `look at [Image #3]` — when sent later, image #3 may have been recycled. Out of scope for v1; document as a known limitation.

## Risk register

- **`laststatus = 2` reintroduces visual noise.** Status line takes one row; `cmdheight = 0` gives it back. Acceptable.
- **Queue dir grows unbounded if user never sends queued items.** No cleanup in v1. Sessions are per-tag and short-lived; if it becomes a problem, add a "drop +N" key later.
- **Performance: re-reading log on every nav.** Logs are small (megabytes at most). If it bites, cache by mtime.

## Approval gate

Pause for user review before starting M1.
