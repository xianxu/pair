# Boundary Review — pair#517 (milestone M1)

| field | value |
|-------|-------|
| issue | 517 — couch: an agent switches a slot's coding agent (couch --switch-agent repo:N --agent NAME), with a handoff built from durable state |
| repo | pair |
| issue file | workshop/issues/000517-couch-an-agent-switches-a-slot-s-coding-agent-couch-switch-agent-repo-n-agent-name-with-a-handoff-built-from-durable-state.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | e8d839f8d5012bf1560787284615a6f3f4dabbfc..e337c825c5d8e1cf7f2f6fa454f3e3063c84959b |
| command | sdlc milestone-close --issue 517 --milestone M1 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-12T20:49:39-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

M1 adds bounded previous-session persistence, exact capture selection, and documented CLI access. However, switching an already-parked source can retain evidence from an older incarnation. The core-concepts table also contradicts the implemented M1 locations. Relevant core tests and the full scrollback suite passed; CLI verification was blocked by missing generated runtime assets.

1. **Strengths**

- Descriptor validation and defensive cloning cover malformed persisted data, size limits, legacy absence, and pointer ownership.
- `RecordPreviousSession` checks the verified park, capture, binding, and record revision before persisting evidence.
- Previous/full peeks bypass current live-tail selection; tests explicitly prohibit current-source fallback.
- README and atlas document the new flags, costs, limits, and terminal-history semantics.

2. **Critical findings**

- **Incorrect previous incarnation after a separate park — ARCH-ORDER / ARCH-PURPOSE.** At `cmd/internal/couchcore/switchagent.go:361`, an existing descriptor is retained whenever the source is already parked. Sequence: switch A→B, park B separately, switch B→C. `FinalizePark` updates `VerifiedPark` but retains A’s descriptor; the switch consequently skips recording B. Preserve a descriptor only when its source identity matches the current verified park. Add this sequence alongside the existing failed-launch retry test.

- **Core-concepts locations contradict M1 implementation.** At `workshop/plans/000517-agent-switch-plan.md:114`, `PeekSelection` is assigned to `peek.go`, but exists in `peek_selection.go`. Lines 139–143 similarly claim modifications to `threadstore.go`, `retention.go`, and `peek_fast.go` that are absent; the implementation instead introduces `previous_session.go`, `bounded.go`, and runtime wiring in `run.go`. Update the table and add a `## Revisions` entry. This is Critical under the explicit core-concepts cross-check rule, not a runtime defect. M2/M3 entities are appropriately deferred.

3. **Important findings**

- **Promised capture-lease ordering coverage is absent — ARCH-MOCK / ARCH-ORDER.** `cmd/internal/scrollbackcmd/bounded_test.go:98` verifies token selection but never observes an active lease, interleaves collection, or checks release after failure. The older retention test exercises another entry point. Add controlled tests through `RenderOwnedCaptureLines` proving collection cannot remove the selected capture during replay and that success/error paths release its lease.

4. **Minor findings**

None.

5. **Test coverage notes**

- Passed: selected threadrecord/core tests covering previous sessions, switching, peeking, and operations; full `scrollbackcmd` suite; pinned-range `git diff --check`.
- Blocked: `couchcmd` failed compilation at `cmd/internal/runtimebundle/embed.go:8`: `assets/runtime/files` was missing. CLI integration remains unverified.
- Existing successive-switch coverage performs both parks inside switching; it misses the separate-park sequence above.
- No prior findings required disposition. No repository files were changed.

6. **Architectural notes for upcoming work**

- **ARCH-DRY — pass:** reuses replay, path validation, retention leases, and lifecycle operations.
- **ARCH-PURE — pass:** descriptor validation/cloning and source selection are pure; recording IO is injected.
- **ARCH-PURPOSE — flag:** stale evidence defeats exact previous-incarnation access.
- **ARCH-MOCK — flag:** the new capture reader lacks the planned controlled retention interaction test.
- **ARCH-CONSTRAINTS — pass:** byte, dimension, row, and single-reference limits are enforced and tested.
- **ARCH-SECURE — pass:** strict descriptor validation and bounded regular-file reads protect the new persistence/read boundaries.
- **ARCH-ORDER — flag:** descriptor reuse relies on parked status rather than matching source identity.
- **ARCH-FUNERAL — pass:** one bounded descriptor replaces its predecessor; captures reuse existing retention and readers close leases.

7. **Plan revision recommendations**

Add entries recording the actual M1 file locations and the invariant: “Reuse previous-session evidence only for the same verified source park; a different park supersedes it.” Retain the promised lease-interleaving tests as unfinished until demonstrated.

```findings
findings:
  - id: new
    severity: Critical
    family: evidence-reuse-requires-source-identity
    title: |
      Switching an already-parked source retains an older incarnation's evidence
    detail: |
      cmd/internal/couchcore/switchagent.go:361 preserves any existing descriptor for a parked source. After A→B, a separate park of B, and B→C, previous-session access still selects A. ARCH-ORDER and ARCH-PURPOSE require comparing the descriptor's source with the current verified park, retaining it only for the same source. Add a regression sequence covering separate parking while preserving failed-launch retry behavior.
  - id: new
    severity: Critical
    family: core-concepts-match-implementation
    title: |
      M1 core-concepts locations and modification claims contradict the diff
    detail: |
      workshop/plans/000517-agent-switch-plan.md:114 and :139–143 name incorrect M1 locations or unchanged files. PeekSelection lives in peek_selection.go; persistence and capture reading introduce previous_session.go and bounded.go, with runtime wiring in run.go. Correct the table and add a Revisions entry, as required by the core-concepts cross-check.
  - id: new
    severity: Important
    family: retention-ordering-regression-coverage
    title: |
      Exact capture reader lacks the promised lease and collection ordering tests
    detail: |
      cmd/internal/scrollbackcmd/bounded_test.go:98 tests selection only. ARCH-MOCK and ARCH-ORDER require controlled coverage through RenderOwnedCaptureLines proving the exact capture is protected during replay and leases release after both success and failure. Existing retention tests exercise a different entry point.
```

---

## Re-review — 2026-10-12T20:58:18-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 517 — couch: an agent switches a slot's coding agent (couch --switch-agent repo:N --agent NAME), with a handoff built from durable state |
| repo | pair |
| issue file | workshop/issues/000517-couch-an-agent-switches-a-slot-s-coding-agent-couch-switch-agent-repo-n-agent-name-with-a-handoff-built-from-durable-state.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | e8d839f8d5012bf1560787284615a6f3f4dabbfc..71a96de7c9fe217f641b22201463d3efb5333832 |
| command | sdlc milestone-close --issue 517 --milestone M1 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-12T20:58:18-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

All three prior findings are addressed, with mutation tests confirming BR-1 and BR-3. Previous-session persistence, exact selection, and documentation are in place. Two reproduced replay bugs block M1: full reads silently lose history, and accepted resize dimensions can crash the CLI.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      switchagent.go:361 compares the descriptor source with the verified park. Reverting that condition makes TestPreviousSessionSuccessiveSourceParksChooseImmediatePrevious/separate_park_before_switch fail; failed-launch retry coverage also passes.
  - id: BR-2
    disposition: addressed
    note: |
      The pinned correction updates plan lines 114 and 139–143 to peek_selection.go, previous_session.go, bounded.go, and run.go, matching their actual definitions and wiring. The Revisions entry at line 326 records the correction and identifies unchanged dependencies.
  - id: BR-3
    disposition: addressed
    note: |
      TestOwnedCaptureLeaseProtectsReplayAndReleases interleaves real collection with the production reader and replay. Removing either lease acquisition or release makes both success and failure cases fail.
findings:
  - id: new
    severity: Critical
    family: full-replay-requires-loss-detection
    title: |
      Full replay silently evicts history and reports the remaining tail as complete
    detail: |
      cmd/internal/scrollbackcmd/scrollbackcmd.go:382–409 changes only the emulator history row limit, leaving its 65,536-cell and 4 MiB history limits active. A 2,000-line, 80-column recording returns only 464 lines, starting at line 1536, with no error; PeekThreadSelection then sets Complete=true. ARCH-PURPOSE and ARCH-CONSTRAINTS: configure explicit replay limits and detect every capacity-driven eviction, returning a visible error instead of declaring truncated output complete. Add regression coverage exceeding the cell and byte limits independently.
  - id: new
    severity: Critical
    family: replay-admission-matches-emulator-limits
    title: |
      Accepted recording dimensions can panic during emulator construction
    detail: |
      cmd/internal/scrollbackcmd/bounded.go:38–43 accepts 1000×1000, but replayData calls vt.NewEmulator, whose default maximum is 262,144 cells. An initial 1000×1000 resize reproducibly panics with "vt: invalid geometry"; later oversized-area resizes are silently rejected by Emulator.Resize. ARCH-SECURE and ARCH-CONSTRAINTS: align admission with explicit emulator geometry limits, use error-returning construction and resize operations, and test both initial and subsequent resize failures.
```

1. **Strengths**
   - Previous evidence is persisted before launch, retained through failed-launch retries, and superseded by a different verified source.
   - Descriptor validation, strict decoding, and defensive-copy tests cover malformed and legacy records.
   - Exact-capture tests prove protection during collection and lease release after successful and failed replay.
   - README and atlas updates cover the new flags, costs, evidence selection, and limits.

2. **Critical findings**
   - The two replay findings above require fixes and regression tests before M1 closes.

3. **Important findings**
   - None additional.

4. **Minor findings**
   - None.

5. **Test coverage notes**
   - Targeted threadrecord, couchcore, couchcmd, scrollbackcmd, and artifactpath tests passed; the complete scrollbackcmd suite passed.
   - CLI tests required runtime asset generation in the disposable copy.
   - Both new adversarial replay probes failed on pinned HEAD.
   - `git diff --check` passed. Full repository suite was not run.
   - Disposable test copy removed; tracked checkout unchanged.

6. **Architectural notes**
   - **ARCH-DRY — pass:** shared artifact resolution, retention, and replay mechanisms reused.
   - **ARCH-PURE — pass:** descriptor validation and source selection are pure; recording IO is injected.
   - **ARCH-PURPOSE — flag:** silently truncated output defeats full previous-session access.
   - **ARCH-MOCK — pass:** stateful lifecycle fixtures and portable retention storage exercise production seams.
   - **ARCH-CONSTRAINTS — flag:** declared replay bounds do not account for emulator limits.
   - **ARCH-SECURE — flag:** persisted resize input can trigger a panic.
   - **ARCH-ORDER — pass:** source replacement, retry preservation, and collection ordering have meaningful regression coverage.
   - **ARCH-FUNERAL — pass:** one descriptor per thread; existing capture retention and tested lease cleanup govern residue.

7. **Plan revision recommendations**
   - Add a `## Revisions` entry specifying the complete replay envelope, including history cells/bytes and screen area. Require visible failure for capacity loss or rejected geometry, distinguish these from intentional terminal history erasure, and update atlas limits if the accepted envelope changes.

---

## Re-review — 2026-10-12T21:10:47-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 517 — couch: an agent switches a slot's coding agent (couch --switch-agent repo:N --agent NAME), with a handoff built from durable state |
| repo | pair |
| issue file | workshop/issues/000517-couch-an-agent-switches-a-slot-s-coding-agent-couch-switch-agent-repo-n-agent-name-with-a-handoff-built-from-durable-state.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | e8d839f8d5012bf1560787284615a6f3f4dabbfc..0f3a7543cb999a96c70c2d80b64a9c71ecd73691 |
| command | sdlc milestone-close --issue 517 --milestone M1 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-12T21:10:47-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

M1 delivers durable previous-session evidence and bounded full recording reads. Both open correctness findings are addressed, with regression tests confirmed to fail when enforcement is removed. No new blocking findings. This verdict covers M1 only; asynchronous switching and live acceptance remain M2/M3 work.

```findings
dispose:
  - id: BR-4
    disposition: addressed
    note: |
      Explicit history limits and monotonic capacity-loss accounting prevent silent truncation. Removing accounting fails independent cell/byte, rejected-row, erase-after-loss, and resize-loss regressions.
  - id: BR-5
    disposition: addressed
    note: |
      Screen-area admission and checked emulator construction/resizing reject invalid geometry. Removing enforcement reproduces the initial panic and silently ignored later resize; regression tests fail.
  - id: BR-1
    disposition: addressed
    note: |
      Exact verified-park identity controls descriptor reuse; successive-source and failed-launch retry tests pass.
  - id: BR-2
    disposition: addressed
    note: |
      The plan's M1 file map and Revisions identify previous_session.go, peek_selection.go, bounded.go, and run.go as implemented owners and distinguish unchanged dependencies.
  - id: BR-3
    disposition: addressed
    note: |
      Controlled collection during exact-capture replay verifies protection and subsequent lease release on success and failure.
```

**Strengths**

- Replay refuses capacity loss even when a subsequent terminal command clears history (`third_party/vt/scrollback.go:200`).
- Geometry admission matches emulator limits and uses error-returning APIs (`cmd/internal/scrollbackcmd/bounded.go:65`).
- Previous-session persistence checks source identity, capture, binding, and revision before writing (`cmd/internal/couchcore/previous_session.go:13`).
- README and atlas document selection semantics, costs, limits, and unavailable evidence.

**Critical findings:** None.

**Important findings:** None.

**Minor findings:** None.

**Test coverage notes**

Passed threadrecord and scrollback package tests, the nested VT suite, targeted core/CLI tests, artifact inventory checks, and diff whitespace checks. Both open fixes passed mutation verification in a disposable copy, subsequently removed.

Missing generated runtime assets initially prevented CLI compilation and affected inventory checks; generating the required assets resolved those checks. The broader multi-package run was stopped before completion; no full-suite pass is claimed. Tracked files remain unchanged.

**Architectural notes**

- **ARCH-DRY — Pass:** shared replay, artifact-path authority, and retention machinery are reused.
- **ARCH-PURE — Pass:** descriptor validation and selection remain deterministic; IO uses injected boundaries.
- **ARCH-PURPOSE — Pass:** M1 provides exact previous evidence without current-session fallback.
- **ARCH-MOCK — Pass:** lifecycle fakes and portable stores exercise production seams.
- **ARCH-CONSTRAINTS — Pass:** explicit replay limits reject overflow visibly.
- **ARCH-SECURE — Pass:** persisted descriptors, geometry, offsets, and file types receive boundary validation.
- **ARCH-ORDER — Pass:** revision/source checks preserve evidence through retries; collection ordering is controlled in tests.
- **ARCH-FUNERAL — Pass:** one bounded descriptor supersedes its predecessor; leases and emulators close, and captures retain existing collection policy.

**Plan revision recommendations:** None.
