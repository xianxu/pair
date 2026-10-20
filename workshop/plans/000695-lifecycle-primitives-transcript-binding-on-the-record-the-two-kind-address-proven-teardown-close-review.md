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

---

## Re-review — 2026-10-19T19:25:33-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 695 — Lifecycle primitives: transcript binding on the record, the two-kind address, proven teardown |
| repo | pair |
| issue file | workshop/issues/000695-lifecycle-primitives-transcript-binding-on-the-record-the-two-kind-address-proven-teardown.md |
| boundary | whole-issue close |
| milestone | — |
| window | 6086689d0e6944b26007dc35ec3a7f88f2ab9c0b..cd3cf1ac2a3d89da9471e2e426f8a57ecf86dc3b |
| command | sdlc close --issue 695 |
| reviewer | claude (anthropic; author anthropic) |
| isolation | fenced |
| timestamp | 2026-10-19T19:25:33-07:00 |
| verdict | SHIP |

## Review

Still reviewing: the conflict resolutions build and are gofmt-clean. Now I'm checking whether the test failures come from the sandbox (git init is refused) or from the change.

```verdict
verdict: SHIP
confidence: high
```

I recommend shipping. This window contains no new behavior: it resolves conflicts from merging main into couch.go and warm_attach.go, moves four lessons, and adds gate records. Both conflict resolutions keep what each side intended. In `Couch`, main's `SlotEnvironment`, `ScratchStartup` and `ScratchMaintenance` fields sit next to #695's `LeaseHolders`. In warm attach, the env slice gets the lifetime-lease entry and then has `SlotEnvironment` appended. At head there are no conflict markers in the Go files, `gofmt -l` is clean, and `go build ./...` and `go vet` pass. All 22 earlier findings were already disposed, and this window gives no reason to reopen any of them.

**Strengths**
- `cmd/internal/couchcore/warm_attach.go:88-95`: the merge keeps the lease-preparation error path (`prepare lifetime lease: %w`) and adds main's `SlotEnvironment` after it. Neither env entry is lost.
- One test covers each side of that merge, and both pass at head:
  - `TestScratchEnvironmentReachesFreshAndWarmLaunches` covers main's side.
  - `TestManagedTerminalLifetimesThroughColdAndWarmResume` covers the lease side.
- `cmd/internal/couch.go:38-46`: both sides' fields are kept, and the comment that says where `LeaseHolders` comes from in production is still there.
- No scope change. The issue file only changes its status, `actual_hours` and the Log. Spec, Revisions and Done-when are untouched, and the lessons change only reorders the four #695 entries without changing their text.

**Critical findings:** none.

**Important findings:** none.

**Minor findings:** none.

**Test coverage notes**
- A full `go test ./cmd/internal/couchcore/` run here fails about 190 tests. Every failure I checked is the review sandbox refusing `git init` ("outside the reviewer's directories (sdlc #339)"), and none of them is about the change.
- Three warm-attach tests (`TestWarmAttachmentRepairsOnlyDeadManagedClient` and two siblings) hit that same refusal, so they could not run here.
- The tests that don't need git all pass, including the lease, teardown, scratch and warm-resume tests.
- The issue Log says the full suite passed on the merge head when run on APFS scratch. I did not re-run that here.

**Architectural notes**
- **ARCH-DRY:** pass. The resolution adds no duplicated code.
- **ARCH-PURE:** pass. No logic moved into IO.
- **ARCH-PURPOSE:** pass. Both sides' consumers stay wired.
- **ARCH-EVOLVE:** pass. No persisted format changes in this window.

**Plan revision recommendations:** none.

```findings
findings: []
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"be05aaf8-1a97-4576-a4bb-d6b88b2877fe","repository":"github.com/xianxu/pair","issue_id":"000695","recipe":"fix-review","round":8,"attempt":1,"agent":"claude","root":"/Volumes/TestScratch/review-checkouts/3a358c9e6d57df36","reviewed_head":"cd3cf1ac2a3d89da9471e2e426f8a57ecf86dc3b","slot":"pair:1","requested_at":"2026-10-20T02:10:26.698876Z"},"phase":"finished","pid":82054,"started_at":"2026-10-20T02:10:26.890295Z","finished_at":"2026-10-20T02:18:10.438264Z","session_id":"8f63c762-61e8-4e14-8011-9d2724e7fea7","outcome":"returned"},{"version":1,"intent":{"disposable_root":true,"id":"c8bcda53-d77e-4a1e-b5c5-289287af1150","repository":"github.com/xianxu/pair","issue_id":"000695","recipe":"fix-review","round":8,"attempt":2,"agent":"claude","root":"/Volumes/TestScratch/review-checkouts/3a358c9e6d57df36","reviewed_head":"cd3cf1ac2a3d89da9471e2e426f8a57ecf86dc3b","slot":"pair:1","requested_at":"2026-10-20T02:18:16.517387Z"},"phase":"finished","pid":62311,"started_at":"2026-10-20T02:18:16.732709Z","finished_at":"2026-10-20T02:25:33.271074Z","session_id":"969a7df4-c309-47cf-949f-53a7fea690d3","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.207438209s
head: "cd3cf1ac2a3d89da9471e2e426f8a57ecf86dc3b"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"
Preparation attempt 2; duration 2.992933917s
head: "cd3cf1ac2a3d89da9471e2e426f8a57ecf86dc3b"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"
Protocol attempt 1
reason: "missing valid findings block"
output: "Both conflict resolutions keep both sides and match `launch_existing.go`. Now checking whether the other test failures are the same sandbox git-init refusal.\n\n```verdict\nverdict: SHIP\nconfidence: high\n```\n\nThis window has no new code beyond fixing the two merge conflicts. `couch.go` and `warm_attach.go` reached the base with raw `<<<<<<<`/`>>>>>>>` markers, and the head resolves both by keeping each side's change. In `couch.go:38-53`, the `Couch` struct keeps main's `SlotEnvironment`, `ScratchStartup` and `ScratchMaintenance` fields and #695's `LeaseHolders` field, with no duplicate fields. In `warm_attach.go:88-95`, warm attach still calls `c.lifetimeLeaseEnv(thread)` and puts the lease into `env`, then adds `c.SlotEnvironment(ctx, thread)`. That is the same order the fresh-launch path uses in `launch_existing.go:106-113`, so the two launch paths still build their environment the same way. No conflict markers remain anywhere under `cmd/`. `go build ./...` and `go vet ./cmd/internal/couchcore/` pass. `workshop/lessons.md` only moves four existing #695 lessons, unchanged, to sit after another block. The close-gate and close-review files record the review process and don't change `## Spec`, `## Revisions` or `## Done when`, so the scope hasn't changed. Every prior finding (BR-1 to BR-22) was already disposed, and this window doesn't reopen any of them.\n\n1. **Strengths**\n   - `warm_attach.go:88-95`: the fix keeps both the lease (#695 P1) and main's per-slot scratch environment, in the same order as `launch_existing.go:106-113`.\n   - `couch.go:38-53`: the struct merge keeps both sides' fields and comments, with gofmt alignment and no duplicates.\n   - `lessons.md`: the lines were moved, not rewritten, so no lesson text changed.\n\n2. **Critical:** none.\n\n3. **Important:** none.\n\n4. **Minor:** none.\n\n5. **Test coverage notes:** `go test ./cmd/internal/couchcore/` fails here, but every failure I checked comes from the review sandbox: `git init refused: ... outside the reviewer's directories (sdlc #339)`, 274 times. I traced every failure in the full run, and in a narrower run of the lease, teardown and attach tests, to that same refusal. That includes the warm-attach tests that would run the merged code (`TestWarmAttachmentRepairsOnlyDeadManagedClient`, `TestWarmAttachmentRetriesManagedSlotWithDeadIncarnationAndListedTerminal`). So the merged warm-attach path has compiled but not run in this sandbox. I still judged the fix as safe because it only combines both sides, the code builds, and it matches the fresh-launch path. The implementer should confirm the full couchcore suite passes on a host where `git init` is allowed.\n\n6. **Architectural notes**\n   - ARCH-DRY: pass. The fix adds no logic. Warm attach and fresh launch still both build their environment as lease first, then `SlotEnvironment`. That shared step could later become one helper, but this window didn't create the duplication.\n   - ARCH-PURE: pass. The fix moves no logic into IO code.\n   - ARCH-PURPOSE: pass. Both launch paths that need the lease still get it after the merge.\n   - ARCH-EVOLVE: pass. No saved file format changes in this window.\n\n7. **Plan revision recommendations:** none.\n\n```findings\n```"

truncated: false
retained-bytes: 0000003987
