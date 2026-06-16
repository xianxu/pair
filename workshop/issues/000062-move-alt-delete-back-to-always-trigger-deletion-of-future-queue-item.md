---
id: 000062
status: working
deps: []
created: 2026-06-16
updated: 2026-06-16
estimate_hours: 0.5
---

# move alt+delete back to always trigger deletion of future queue item

at some point, we moved alt+delete in future queue item to condition on mode: normal mode it would behave as deletion to beginning of line, normal mode to delete that future queued item. this, in practice, is more confusing that helping. let's move alt+delete to always delete "future queued item" if we are in the +N portion of draft buffer. 

## Done when

- On a `+N` (future-queue) slot, **Alt+BS deletes the queue item in both normal
  and insert mode** (restores the pre-`97cc1e1` always-delete behavior for the
  +N portion; the Y/n confirm stays).
- Off the queue (on `*` or `-N`), insert-mode Alt+BS still kills to line start
  (`<C-U>`); normal-mode stays a no-op. No regression to `*`-draft editing.

## Spec

`ef148e0` originally bound `{n,i} <M-BS>` → `delete_current_queue_item` (which
self-guards to `+N`, so it deleted the item in any mode on a queue slot and
no-oped elsewhere). `97cc1e1` then split it: normal-mode kept the item-delete but
insert-mode became an unconditional `<C-U>` line-kill — so on the *same* `+N`
slot Alt+BS does different things depending on mode. That mode-dependence is the
confusion.

Fix: make the **insert-mode** `<M-BS>` conditional — delete the queue item when
`nav.pos.kind == 'queue'`, else fall back to `<C-U>`. Normal-mode keymap is
unchanged (already `delete_current_queue_item`, self-guarded). Net: "always
delete the future queued item when in the +N portion," while keeping line-kill
where it's actually an editing convenience (the `*` draft).

## Plan

- [ ] Insert-mode `<M-BS>` → delete `+N` item when on a queue slot, else `<C-U>`;
      update the keymap comment. Verify `luac -p` + dogfood (normal & insert
      delete on +N; `*`/`-N` insert line-kill preserved).

## Log

### 2026-06-16

