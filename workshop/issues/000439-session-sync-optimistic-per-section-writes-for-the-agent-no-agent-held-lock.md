---
id: 000439
status: working
deps: []
github_issue:
created: 2026-10-11
updated: 2026-10-11
estimate_hours:
card_mirror: 'fcbfbcf04060b0416a839a49f52ffacfd2f3f9fb' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-11T09:11:08-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: inferred}
---

# session-sync: optimistic per-section writes for the agent, no agent-held lock

## Problem

A session-sync file (the TL status file, ops `teams/*/tl-status-*.md`) has a lock that two kinds of holder share: the operator's `.lock` while they edit, and an agent-held lock that the operator's submit hands to the agent. `couch --session-sync write` (pair#437) keeps the agent-held lock; only `release` returns it. An agent that forgets `release` at the end of its turn locks the operator out of their own file: on 10-11 the TL held it from 08:03 to about 09:00, and the operator couldn't add a comment (ops learnings, ariadne-robustness-1 kink 65). Relying on the agent remembering a final step is the failure.

The conflict check is also coarser than the write. `write --section` replaces one section, but the guard refuses (exit 4, `unread-edits`) when the file differs from the shadow anywhere, and `--accept-edits` accepts every difference at once (`cmd/internal/sessionsync/sync.go:157`). The check and the atomic rename aren't under one exclusive lock, so a change landing between them is not caught.

## Spec

Design principle, the operator's (10-11): **pessimistic locking for the human, optimistic for the agent.** A person holding the lock has the file's state in their head and needs it to hold still; an agent can re-read and recompute cheaply, so a lost race costs it one retry.

- **The operator's lock stays pessimistic.** parley writes `<file>.lock` (holder `operator`) while the operator edits; every agent write refuses while it exists (exit 3, unchanged).
- **No agent-held lock.** Nothing in Couch writes or expects holder `agent`. A submit records an owed event (pair#437's ledger) and that is the whole hand-off. `release` retires (kept as an accepted no-op for one release, so older skills don't break, then removed). A legacy `holder: agent` lock left on disk counts as no lock.
- **Optimistic per-section compare-and-replace:**
  - `couch --session-sync read --section HEADING FILE [--json]` prints the section body and its hash (`read --all FILE` prints the file and a whole-file hash).
  - `write --section HEADING BODYFILE FILE --if-hash H` replaces the section only if its current hash is H; otherwise it refuses with a distinct exit code (`stale`, e.g. 6) and prints the current hash, so the agent re-reads, recomputes and retries. `write --all ... --if-hash H` does the same for the whole file.
  - Edits elsewhere in the file don't block a section write. The shadow copy and `diff` stay, for the agent to see what the operator changed; `--accept-edits` goes away (the owed ledger, not a whole-file diff gate, makes sure a submit gets read).
  - Without `--if-hash`, write refuses (an unconditional write is the bug this removes). Decide in the design whether `--force` is warranted; default no.
- **Each command is atomic:** take an exclusive `flock` on the sidecar for the whole read-check-write (and for `read`), so a concurrent writer can't land between the check and the rename.
- Sections are matched by whole heading lines (unchanged); the hash covers the section's body as stored, so whitespace-only operator edits count as changes.
- `status --json` drops `Holder: agent`; `Holder` is `operator` or empty. parley.nvim reads only `Code` and the oldest `Owed[].At` (parley.nvim#314), so its contract holds; parley.nvim#315 is the parley side.

## Done when

- A section write with a matching `--if-hash` succeeds while the operator has an unread edit in another section; with a stale hash it refuses `stale` and reports the current hash. Pinned by tests.
- Two concurrent writers to the same section: exactly one wins, the other gets `stale` (a test with two processes).
- No Couch path creates or requires an agent-held lock; a TL turn that never runs `release` leaves the operator able to edit (test: submit, fold, write; the lock file is absent).
- Any agent write refuses while the operator holds `.lock` (unchanged, still pinned).
- The xx-tl draft in ops (`skills-draft/xx-tl/SKILL.md`) gets the new command shapes (the TL updates it on landing; note it in the report).

## Plan

Design (2026-10-11):
- **Hash:** sha256 hex, first 16 chars, of the section body as stored (the lines after the `## HEADING` line up to the next `## ` line or EOF; `--all`: the whole file). Pure function `Hash`, `SectionBody` beside `ReplaceSection` (ARCH-DRY: one section scanner shared by read and replace).
- **Exit code:** `stale` = 6 (a new `Refusal.Current` carries the current hash; JSON envelope gains `Hash`). 4 (`unread-edits`) is retired, not reused.
- **`read`:** `read --section HEADING FILE [--json]` / `read --all FILE [--json]`; text = `hash: H` line, then the body; JSON = `{Code,Error,Read:{File,Section,Hash,Holder,Body}}`. Read is pure (no shadow/state change).
- **`write`:** requires `--if-hash H` (no `--force`); `--accept-edits` removed. The shadow still takes the whole written file, so `diff` shows operator edits since the agent's last write (informational only, no gate). A successful write prints the new hash.
- **Atomicity:** every verb holds `flock(LOCK_EX)` on `.<file>.flock` from load to the last write. ARCH-FUNERAL: one empty file per session-sync file, same lifetime as `.shadow`/`.sync` (dies with the file's directory); creates nothing per event.
- **Agent lock:** `HolderAgent` is gone; `ParseHolder` returns `operator` for any lock whose holder isn't `agent`, and `""` (no lock) for none or a legacy `holder: agent`. Couch never writes or deletes `.lock` (parley owns it; deleting a lock parley may have just rewritten as `operator` would break the human's pessimistic lock).
- **`release`:** accepted no-op for one release: takes its old flags, changes nothing, prints a deprecation note (folds are `write --fold`). Removal is a follow-up.
- **status:** `Holder` is `operator` or `""` (text: `lock: none`); Code/Owed unchanged for parley.nvim#314/#315.

- [x] Sidecar `flock` around every command; `read --section/--all` with hashes.
- [x] `write --if-hash` compare-and-replace; drop the whole-file unread-edits gate and `--accept-edits`; refuse a write without `--if-hash`.
- [x] Remove the agent-held lock: submits only record owed events; `status` reports holder operator or none; `release` a no-op.
- [x] Tests for the Done-when rows (incl. a two-process race); update atlas, SKILL.md, README, usage.

## Log

### 2026-10-11
- 2026-10-11: closed — make -k test green (240 ok, 0 FAIL); new tests pin every Done-when row: section write over an edit elsewhere + stale refusal with current hash, two-process race (one wins, one stale), flock-under-check (mutation-checked), no-release turn leaves no lock, operator lock still refuses; CLI read/stale/release-no-op tests.; review verdict: SHIP
- 2026-10-11: flow upgraded quick → full — 262 added lines in code files (limit 100)

- Claimed from the TL dispatch (ops:0, receipt 563f892e verified). Design in `## Plan`.
- Shadow stays "the file as the agent last wrote it" (whole file), so `diff` shows operator edits since the last write; a per-section shadow merge was considered and dropped (Simplicity First: the owed submit, not the diff, makes the agent read the file).
- Couch never deletes a legacy `holder: agent` lock: parley may rewrite `.lock` as `operator` between couch's read and its delete, and deleting that would break the human's pessimistic lock. Legacy agent locks are parsed as none; parley.nvim#315 owns its file.
- Tests: `TestASectionWriteSucceedsOverAnEditElsewhereAndRefusesStale`, `TestWriteWithoutIfHashRefuses`, `TestAWriteChecksItsHashUnderTheFlock`, `TestTwoProcessWritersExactlyOneWins` (two helper processes), `TestATurnWithoutReleaseLeavesNoLock`, operator-lock test kept; CLI: read text/JSON, stale exit 6 + hash, release no-op. Mutation check: replacing the flock with LOCK_UN fails the flock test 5/5 and the two-process test 1/5.
- Diff left the quick-flow shell (~260 added code lines), so close runs the full review.
- Follow-up: remove the `release` no-op after one release (ARCH-FUNERAL), and the TL updates ops `skills-draft/xx-tl/SKILL.md` to `read` + `write --if-hash`.
- Verification: `make -k test` green (240 packages ok, no FAIL).
