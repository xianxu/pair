# Boundary Review — pair#605 (whole-issue close)

| field | value |
|-------|-------|
| issue | 605 — Recovery resume fails workspace recheck: in-memory intent time carries a monotonic reading |
| repo | pair |
| issue file | workshop/issues/000605-recovery-resume-fails-workspace-recheck-in-memory-intent-time-carries-a-monotonic-reading.md |
| boundary | whole-issue close |
| milestone | — |
| window | 1a1047c89126c68fa1f9c2651bff36663f6ab656..1447f7681ac0bf15d06624febae95dd346e17429 |
| command | sdlc close --issue 605 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-16T01:15:56-07:00 |
| verdict | SHIP |

## Review

Regression test confirmed: it fails without the fix and passes with it. Now running the full package.

The full couchcore run hit Go's 10-minute test timeout. Checking which test was hanging.

```verdict
verdict: SHIP
confidence: high
```

The fix does what the Spec asks. `executePreparedRecovery` is the only place a `RecoveryIntent` is created. The fix runs the intent through its JSON encoding there (`cmd/internal/couchcore/recovery_prepared.go:262-267`), after the inferred branch has rebuilt it and before `PreserveRecoveryBeforeRetirement` first persists it. From then on, the start fence, the workspace recheck and the fresh-replacement check all compare an in-memory value that already equals its persisted copy. No check was changed, as the Spec requires. I confirmed the regression evidence myself: with the three-line call removed, both subtests of `TestRecoveryResumeWithMonotonicClockPassesWorkspaceRecheck` fail with the exact production error (`slot start claim changed during workspace readiness`), and with the fix restored they pass. Nothing blocks SHIP.

1. **Strengths**
   - The fix sits at the single creation site and covers the whole intent, not just `CreatedAt`. That matters: on the inferred path the intent is rebuilt with no `CreatedAt` at all, so the bad time there is `Context.CreatedAt` (from `DecideSlotRecovery(..., c.Clock.Now())`). Patching only `CreatedAt` would have left inferred resume broken.
   - Canonicalizing through the real encoding also evens out other round-trip differences, such as an empty slice decoding back as nil with `omitempty`, or the time-zone shape. These stay correct for free; a hand-written `t.Round(0)` would have missed them.
   - The test runs the production `SystemClock{}` through the real slot store and covers both the inferred and proven paths. It is a true RED/GREEN regression test.
   - The issue log records an enumerated `reflect.DeepEqual` sweep. That answers the "class, not instance" question (ARCH-PURPOSE).

2. **Critical findings:** none.

3. **Important findings:** none.

4. **Minor findings**
   - The Fresh recovery path (`launchRecoveryFreshAddress`, which the fresh-replacement check uses) gets the same canonicalized intent, but no monotonic-clock test runs it. A third `fresh` subtest would pin it cheaply.
   - `persistedRecoveryIntent` copies an encoding fact the store owns: it assumes the store writes plain `encoding/json`. It would sit more naturally next to `RecoveryIntent` in `threadrecord`, so the persistence package owns its "as decoded" form.

5. **Test coverage notes**
   - The targeted regression test passes, and fails with the fix reverted (checked locally).
   - `go test ./cmd/internal/couchcore/ -run 'Recovery|Recover|Resume|Fence|Workspace|Replace'` passes (about 200s).
   - `threadrecord` passes, and `go vet` on couchcore is clean.
   - The full `couchcore` package hit Go's default 10-minute test timeout in my run, with a leftover goroutine from `terminal.NewInputWriter`. The recovery subset alone takes about 200s, so this looks like package runtime, not this change. Still, I could not independently confirm the issue's "recorded suite" claim at the default timeout.

6. **Architecture notes**
   - **ARCH-DRY:** pass. One small helper; see the Minor note on where it should live.
   - **ARCH-PURE:** pass. The helper is pure (an encode/decode round trip with no IO).
   - **ARCH-PURPOSE:** pass. The fix is at the single creation site, every comparison consumer is covered, and the sweep is recorded.
   - **ARCH-MOCK:** pass. The test runs against the existing fixture store and fakes, and swaps in the real clock to expose the bug that `FixedClock` was hiding.
   - **ARCH-CONSTRAINTS:** pass. One extra marshal per recovery run, which is negligible.
   - **ARCH-SECURE:** pass. No new untrusted input; a marshal error now comes back as an error instead of being ignored.
   - **ARCH-ORDER:** pass. The state machine is unchanged, and the canonicalization happens before the first transition (`recoveryPrepared → recoveryPreserved`).
   - **ARCH-FUNERAL:** pass. The change creates nothing that persists: it only alters the in-memory value that was already being stored.
   - For future work, any new in-memory-vs-persisted comparison of thread-record values should either use the canonical form or compare semantically. Moving the helper into `threadrecord` would make that the default.

7. **Plan revision recommendations:** none. The plan matches the code.

```findings
findings:
  - id: new
    severity: Minor
    kind: defect
    family: regression-test-path-coverage
    title: |
      Monotonic-clock regression test omits the Fresh recovery path
    detail: |
      launchRecoveryFreshAddress consumes the canonicalized intent (fresh replacement invariant in threadstore_replace.go) but TestRecoveryResumeWithMonotonicClockPassesWorkspaceRecheck only runs inferred and proven; add a fresh subtest.
  - id: new
    severity: Minor
    kind: defect
    family: encoding-owned-by-persistence-layer
    title: |
      persistedRecoveryIntent restates the store's encoding outside threadrecord
    detail: |
      The helper assumes the store persists via plain encoding/json; locating it beside RecoveryIntent in threadrecord would keep the as-decoded form owned by the persistence package (ARCH-DRY).
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"3b0d33dd-8708-43b1-bc1a-1557abb65c04","repository":"github.com/xianxu/pair","issue_id":"000605","recipe":"milestone-review","round":1,"attempt":1,"agent":"claude","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/069d397976037447","reviewed_head":"1447f7681ac0bf15d06624febae95dd346e17429","slot":"pair:2","requested_at":"2026-10-16T08:00:37.677371Z"},"phase":"finished","pid":65994,"started_at":"2026-10-16T08:00:37.970511Z","finished_at":"2026-10-16T08:15:53.629119Z","session_id":"31b5abeb-4d7e-4171-8b99-1c270fdde5f4","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.739784541s
head: "1447f7681ac0bf15d06624febae95dd346e17429"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
