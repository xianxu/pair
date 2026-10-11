---
id: 000439
status: open
deps: []
github_issue:
created: 2026-10-11
updated: 2026-10-11
estimate_hours:
card_mirror: 'a86807074f21953ae29606f1471cf27e6f1d6a8b' # card fields mirrored from issue-cards; edit via sdlc
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

- [ ] Design: hash function and exit code; `read` output and `--json` shape; the `release` deprecation path; legacy `holder: agent` handling.
- [ ] Sidecar `flock` around every command; `read --section/--all` with hashes.
- [ ] `write --if-hash` compare-and-replace; drop the whole-file unread-edits gate and `--accept-edits`; refuse a write without `--if-hash`.
- [ ] Remove the agent-held lock: submits only record owed events; `status` reports holder operator or none.
- [ ] Tests for the Done-when rows; update the session-sync atlas page and `couch --help`.

## Log

### 2026-10-11
