---
id: 000735
status: working
deps: []
github_issue:
created: 2026-10-20
updated: 2026-10-20
estimate_hours:
collaboration-mode: collaboration
card_mirror: '880f8588d212ac0b006af6a94f22413404e8e547' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-20T11:54:21-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
---

# couch: messages to an idle Claude pane are held while a background shell runs; held messages don't say why

## Problem

On 10-20, messages to ops:0 (a Claude hub, idle with an empty composer) were accepted but not typed in for 12
minutes, then all arrived at once when the operator typed into the pane ("hmm, are you stuck?"):
- `couch --message-status`: e60561fc accepted 11:09:59, 39e08f54 accepted 11:11:05; both `submitted` only at about
  11:21:42 (RetainUntil 12:21:42 minus the 1 h retention); about ten messages in the batch.
- ops:0's recording (`scrollback-1-ops-71-claude.events.jsonl`): the turn finished at 11:08 ("done 11:08AM"), the
  composer was empty, and **only 28 bytes of output** were written between 11:09:58 and 11:21:33. The footer read
  `✻ Brewed for 6s · done 11:08AM · 1 shell still running` and `⏵⏵ bypass permissions on · 1 shell · ↓ to manage`:
  a background Bash (a `gh` watch loop) was running.
- Senders saw `waiting for recognized composer` on every send to ops:0 all day.

**Hypothesis (unconfirmed):** Couch's Claude composer recognition doesn't accept the idle state while Claude shows a
running background shell in its footer, so it holds delivery until something changes the screen. The operator's
typing then started a turn and the queue flushed.

Also: Couch's supervision log (`~/.local/share/pair/couch/supervision.jsonl`) hasn't been written since 10-19 15:56
and its pair-diagnostics segment is empty, so there is no record of readiness decisions to confirm this from.

## Spec

- Reproduce: an idle Claude pane with a background shell running; send a message; observe whether it's held.
- If confirmed, recognize Claude's idle composer with the background-shell footer variants (`· N shell(s) still
  running`, `· N shell · ↓ to manage`) as deliverable; a held message names its reason in `--message-status` Detail
  ("composer not recognized: …") instead of a bare queued.
- Restore the supervision log (or say where readiness decisions are recorded now).

## Done when

- A regression with the recorded 189x39 empty Claude composer and shell footer remains deliverable after a same-size resize (and fails on the old implementation).
- `--message-status` of a held message shows why it's held.
- Hidden cursors, incomplete frames, actual geometry changes, drafts and pickers retain their conservative admission behavior.
- The operator verifies delivery at the next Couch restart; ops:0 waived pre-landing staging on 2026-10-20. Supervision-log recovery is filed separately per the same approval.

## Plan

- [ ] Regress the recorded empty shell-footer shape through native terminal paint, same-size resize, and actual peer delivery; keep real-resize, hidden, incomplete, occupied and aborted-resize cases conservative.
- [ ] Treat unchanged terminal dimensions as a no-op within the existing serialized resize owner. Keep actual geometry changes invalidating cursor evidence (ARCH-DRY, ARCH-PURE).
- [ ] Name incomplete/hidden/unrecognized composer evidence in the existing receipt Detail, without copying screen or draft text. Verify projection through broker status, including queued followers.
- [ ] Update tests, atlas/README diagnostics and lessons, run focused checks and recorded full suite, then close and land through SDLC.

## Log

### 2026-10-20

- Claimed and started in pair:3. Reproduction replayed the operator's raw recording plus resize events through the production terminal model to 11:10: complete 189x39 viewport, empty composer at (2,35), but cursor visibility false, so `peerComposerState` returned Unknown. Changing only snapshot visibility to true returns Empty. Existing background-footer delivery regressions pass on unchanged code.
- The recording's last show-cursor before the stall is raw offset 2166899. A duplicate 189x39 resize at 2166913 follows it. `terminalResize.resolve` clears visibility on every committed resize, including unchanged geometry. No subsequent repaint restores it during the quiet interval. Hypothesis: a no-op resize invalidates valid composer evidence; the shell footer itself is already supported. Preserve the fail-closed behavior for actual geometry changes and hidden/incomplete preexisting frames (ARCH-PURPOSE, ARCH-DRY).
- Logging investigation: `supervision.jsonl` remains the production destination; archived diagnostic segments exist. The current payload is frozen since 10-19 15:56; scratch-copy Open/Write succeeds. Fact logging currently discards Open errors and remains unavailable for the entire supervisor lifetime. The lost historical error prevents attributing this incident yet. Existing facts record slot snapshots and completed hub deliveries, not per-message hold reasons. Investigation ongoing; no live log/state mutated.

## Revisions

### 2026-10-20 11:59 — approved root-cause correction and scope split

- ops:0 approved the reproduced no-op resize fix, specific hold reasons and exact recording-shape regression as #735. Background-shell footer matching already works; allowing hidden Claude cursors broadly would weaken admission and would not address the invalidation source. Preserve unchanged geometry instead.
- Supervision logging moves to a separate issue: expose health/last error first, observe the actual failure, then fix it; no speculative retries. No pre-landing staging is required for #735; operator qualification happens at the next Couch restart.
- The original Spec's footer hypothesis and logging work are superseded by this revision and the revised Done when. No new persisted fields, statuses or message lifecycle changes are needed; bounded receipt Detail carries the existing wait state. Unknown screen evidence continues waiting without input effects; the operator can finish a draft, close a picker, or allow a repaint.
