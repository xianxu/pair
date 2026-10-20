# Boundary Review — pair#695 (whole-issue close)

| field | value |
|-------|-------|
| issue | 695 — Lifecycle primitives: transcript binding on the record, the two-kind address, proven teardown |
| repo | pair |
| issue file | workshop/issues/000695-lifecycle-primitives-transcript-binding-on-the-record-the-two-kind-address-proven-teardown.md |
| boundary | whole-issue close |
| milestone | — |
| window | fffbf9c0d921bfc1df6ffb1fc0fe18628e53d812..f9d4f031fe85f8fb6d525c1b8a5b5a5a13b6d17e |
| command | sdlc close --issue 695 |
| reviewer | claude (anthropic; author anthropic) |
| isolation | fenced |
| timestamp | 2026-10-19T17:35:29-07:00 |
| verdict | SHIP |

## Review

All four fixes hold up. Last, I'm running the full couchcore and couchcmd suites at head.

```verdict
verdict: SHIP
confidence: high
```

All four open findings (BR-9, BR-10, BR-21, BR-22) are fixed at head, and I'm raising nothing new. I checked the two behaviour fixes by breaking them on purpose. When I took the exclusive-lock check out of `lifetimelease.RemoveFree`, `TestRemoveFreeLeavesAHeldLease` failed at `lease_test.go:242` ("RemoveFree of a held lease = true"). When I planted a production file under `cmd/` that both calls `TestOnlyTeardownProof(` and imports `couchcoretest`, the module-wide guard flagged both lines (`retire_test.go:326` and `:329`). Both changes were reverted afterwards, and the tree is clean. The suites this issue owns all pass at head: `lifetimelease`, `threadrecord`, `strictjson`, `readiness`, `procutil`, `sessioninventory`, `artifactpath`, `wrapcmd` and `sessionwatch`, plus the couchcore teardown, lease, transcript, orphan, retire and address tests. The full `couchcore`, `couchcmd` and `launcher` suites fail here, but every assertion failure in them is a test fixture's `git init` exiting 128, which this review sandbox refuses ("outside the reviewer's directories (sdlc #339)"); the only other lines in that output are host-load diagnostics. None of these failures touch the issue's code.

1. **Strengths**
   - `lifetimelease.RemoveFree` (`cmd/internal/lifetimelease/lease.go:142`) is now the only place that unlinks a lease. Both `SweepFree` and teardown (`teardown.go:245`) call it, so the rule "unlink only while holding the exclusive lock, and only if the path still names the same file" lives in one function. It pairs with `Hold`'s reopen-if-not-current loop.
   - Teardown now records the case where the lease is held again before removal ("left in place") instead of failing silently.
   - The proof-minting guard covers the whole module, not just `cmd/internal`. It also blocks the indirect route: production code importing `couchcoretest`.
   - `Record.Unknown`'s comment (`threadrecord/record.go:113-118`) now states the round-2 rule: older nested types stay strict, and a nested type born additive keeps its own unknown members.

2. **Critical:** none.

3. **Important:** none.

4. **Minor:** `RemoveFree` returns `true` when the path already names a different file (`!current`). That is correct for the caller's old lease, but "removed" slightly overstates it if a relaunch has already recreated the file. Optional comment tweak; not raised as a finding.

5. **Test coverage notes:** Both behaviour fixes have regression tests, and both tests fail when their fix is removed. The `couchcmd`/`launcher` fixture failures come from the sandbox, not the code.

6. **Architectural notes**
   - **ARCH-DRY:** pass. Lease removal is one helper used by both callers.
   - **ARCH-PURE:** pass. The lease IO stays in the thin `lifetimelease` package.
   - **ARCH-PURPOSE:** pass. The guard now covers the class (every production package), not just the instance first reported.
   - **ARCH-MOCK:** pass. Real-flock tests run on temp dirs.
   - **ARCH-CONSTRAINTS:** pass. Locks are non-blocking, and `Hold` has a time bound.
   - **ARCH-SECURE:** pass. The minting authority can't be reached from any production package.
   - **ARCH-ORDER:** pass. The race between `Hold` and removal is closed: whoever locks first wins, and the loser reopens.
   - **ARCH-FUNERAL:** pass. Lease files are removed on proof or by the sweep.
   - **ARCH-EVOLVE:** pass. The nested-type rule is now stated where the type is declared.

7. **Plan revision recommendations:** none.

```findings
dispose:
  - id: BR-9
    disposition: addressed
    note: |
      The #684 issue's M2 checklist now owns both items from pair#695 M1 (BR-4), the stored-binding row note and the legacy-ledger row detail, and its log records receiving them. The transcript_bind.go:134 message now reads "its transcript binding is invalid", which is neutral about where the binding came from.
  - id: BR-10
    disposition: addressed
    note: |
      record.go:113-118 now says nested types older than pair#695 stay strict and a nested type born additive (TranscriptBinding) keeps its own unknown members, matching the round-2 Revision.
  - id: BR-21
    disposition: addressed
    note: |
      teardown.go:245 now calls lifetimelease.RemoveFree, the same helper SweepFree uses: it takes LOCK_EX|LOCK_NB and checks the path still names the same file. TestRemoveFreeLeavesAHeldLease fails at lease_test.go:242 when the lock check is removed.
  - id: BR-22
    disposition: addressed
    note: |
      retire_test.go now walks the module root and also refuses non-test imports of couchcoretest. A planted cmd/zzprobe/p.go was flagged at retire_test.go:326 and :329.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"b8c7408c-8a96-423f-9fef-ff9b5c780d65","repository":"github.com/xianxu/pair","issue_id":"000695","recipe":"milestone-review","round":7,"attempt":1,"agent":"claude","root":"/Volumes/TestScratch/review-checkouts/bf781e2264670ba0","reviewed_head":"f9d4f031fe85f8fb6d525c1b8a5b5a5a13b6d17e","slot":"pair:1","requested_at":"2026-10-20T00:26:13.755908Z"},"phase":"finished","pid":4682,"started_at":"2026-10-20T00:26:13.932818Z","finished_at":"2026-10-20T00:35:29.038101Z","session_id":"a53bc65f-e1da-4ce1-844c-b4beefbb2c8d","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.186401625s
head: "f9d4f031fe85f8fb6d525c1b8a5b5a5a13b6d17e"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
