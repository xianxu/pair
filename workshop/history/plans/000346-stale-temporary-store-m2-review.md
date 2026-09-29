# Boundary Review — pair#346 (milestone M2)

| field | value |
|-------|-------|
| issue | 346 — Stale temporary store blocks Couch startup |
| repo | pair |
| issue file | workshop/issues/000346-stale-temporary-store.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 73203fc0d103a8c5a5d9490211a09d6055db9a00..0a2a6c1168289d93389b4604fa53d6d927000068 |
| command | sdlc milestone-close --issue 346 --milestone M2 |
| reviewer | claude |
| timestamp | 2026-09-29T11:11:01-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

**Verdict: FIX-THEN-SHIP (medium confidence).** M2 does what the Spec centres on. It separates the durable resume target from parsed transcript evidence: `QueryResumeTarget` reads only the ledger. Requested A and observed D are stored as distinct v3 facts. `ConfirmIfCurrent` returns the existing record when the same root is confirmed again, refuses a competing root, and refuses when a newer launch exists. `RoundsAfterLaunch` now filters confirming events by per-root byte boundaries, so post-launch activity in an existing transcript counts and historical content does not. Codex `source` is open-world. The logic packages pass at HEAD: `sessionledger`, `sessioninventory`, `sessionwatch`, `reviewcmd` and `opener` are green. The `couchcore`, `launcher` and `wrapcmd` failures I saw came from the sandbox (PTY/`/tmp` "operation not permitted"), not from assertions.

Three things should be fixed before crossing the boundary, all cheap:
1. A Pair-chosen ID that was never materialized is admitted as a resume target.
2. `session-repair` re-implements the binding decision rule instead of reusing it.
3. `session-repair` is missing from README.

**Strengths**
- `sessionledger/store.go` `ConfirmIfCurrent`: the stale-launch check, the idempotent same-root sentinel and the competing-root refusal all run under the append lock. `WithCurrentConfirmation` also makes config publication unable to affect a newer launch. The tests in `sessionledger/probation_test.go` pin all three behaviours.
- `sessioninventory/offline.go` `RoundsAfterLaunch`: byte-offset filtering reuses the existing position encoding (`events.go:79`, reversed by `nativeEventRecordOffset`). `TestResumeExistingRootConfirmsOnlyAppendedExchange` shows an old exchange cannot confirm the new launch.
- An incomplete baseline is kept separate from an empty complete one (`BaselineComplete`). The watcher's later observation epoch is covered by `TestIncompleteBaselineAcquiresLaterEpochBeforeConfirming`.
- `codexRole` excludes only recognized child evidence. Unknown and absent sources are accepted, and `TestScanCodexRootSourceIsOpenWorldAcrossJSONShapes` covers several JSON shapes.
- The consumer sweep is complete (ARCH-PURPOSE shadow-sweep). Couch, the launcher, review, the changelog opener, the `session-inventory` owner CLI and the wrapper restart all derive from `QueryResumeTarget`. I found no remaining `QuerySession` admission caller.

**Critical findings**
None.

**Important findings**
1. **A chosen ID with no transcript is admitted as a resume target** (`cmd/internal/sessioninventory/query.go:449`, `cmd/internal/launcher/createflow.go:960`).
   - For a fresh Claude/Qoder launch, the ledger records the minted `--session-id X` with `request_origin: chosen-id`, and `QueryResumeTarget` immediately returns X with status Provisional.
   - The picker's `AgentSessionExists` check and its "not available; starting fresh" fallback were removed. `wrapcmd/agent_restart_test.go` now asserts that the unmaterialized X is the target.
   - Scenario: a thread is launched and quit before the first message. The native file for X was never created. Couch relaunch or Alt+n then passes `--resume X` for a conversation that does not exist. The old atlas text called this "the state relaunch meets most".
   - The Spec makes the *recorded* UUID the probation target on resume. For a chosen ID it says only that creating the file "can provide the handshake".
   - Fix: treat an unconfirmed `chosen-id` request as resumable only once a metadata-only check sees the named root file. That is a filename lookup, not a body parse. Otherwise fall back to a fresh start with the message. Add a production-boundary test for "chosen ID, no file ⇒ fresh".
   - I have not verified exactly when Claude creates the file (this is where the medium confidence comes from). The design gap exists either way.
2. **`session-repair` re-implements the binding rule (ARCH-DRY)** (`cmd/internal/sessionwatch/recover.go:87`).
   - `Recover` takes the union of round roots and requires exactly one. The live watcher goes through `ResolveBindings`, which filters authorized roots and intersects per-send groups.
   - The two rules disagree. Group {D} plus group {D,E} binds D live, but repair reports it as ambiguous.
   - The plan says "factor one observation pass", and a test is named `...UsesSameCorrelation`.
   - Fix: call `sessioninventory.ResolveBindings` with the rounds and accept only a unique Provisional root, so the thresholds cannot drift apart.
3. **README gap for `pair session-repair`.** This is the 2nd finding in family `readme-surface-gap`.
   - Rule: every dispatcher family an operator is expected to invoke must appear in README, or be explicitly marked internal.
   - Enforce it with a test that walks `dispatcher.Families()` against README, with an internal allowlist, rather than adding this one line by hand.
   - The atlas does document the command (`atlas/session-identity.md:75`).

**Minor findings**
- `sessionwatch/run.go:101`: the legacy proof-migration guard does not exclude v3 bindings without a proof, while `QuerySession` does (`Version < 3`). When the watcher restarts on a v3 chosen-id or correlation binding, it parses the native body, appends a v2 row that hides `ConfirmationReason`, and returns before following Codex lifecycle events.
- The Codex quarantine in `decideAutomaticResumeConfig` can no longer trigger, because `hasResumable` is always true when an ID is set. It is dead code.
- `ConfirmationReason` uses untyped string literals (`"correlation"`, `"chosen-id"`), while `RequestOrigin` is a typed enum.
- `requested_native_id` is not shape-validated before it reaches argv as `--resume <id>`. A leading `-` would be parsed as a flag (ARCH-SECURE, low risk because the ledger is Pair-owned).
- The watcher's in-memory phases are held as independent fields (`observationEpoch`, `boundRootNodeID`, `trackedTargets`). There is no tagged state enum (ARCH-ORDER).
- The chosen-id handshake adds a second full metadata `Observe` on every poll until the file appears (ARCH-CONSTRAINTS, metadata only).

**Test coverage notes**
Covered: ledger v3 round trip, idempotent and competing confirmation, stale effects, byte-boundary confirmation, incomplete baseline, the open-world Codex scanner, repair preview/apply/ambiguity/supersession, and the Couch resolver without a transcript.

Missing:
- A resume with an unmaterialized chosen ID.
- A watcher restart on a v3 binding with no proof.
- A repair-versus-live parity test run through the shared resolver.

**Architecture pass**

| Principle | Result |
| --- | --- |
| ARCH-DRY | Flag (Important 2) |
| ARCH-PURE | Pass: handshake, target projection and the ledger fold are pure |
| ARCH-PURPOSE | Pass on the consumer sweep; flag on the chosen-ID admission (Important 1) |
| ARCH-MOCK | Pass: stateful fake runtime; the sqlite flag order is covered by the existing integration test |
| ARCH-CONSTRAINTS | Pass, with the minor repeated `Observe` |
| ARCH-SECURE | Pass, with the minor unvalidated ID |
| ARCH-ORDER | Pass for durable state (a ledger fold under lock); minor on the watcher's in-memory flags |
| ARCH-FUNERAL | Pass: no new artifact families; ledger rows grow slightly per launch; dead quarantine code noted |

**Plan revision recommendations**
- Add a `## Revisions` entry stating how an unconfirmed `chosen-id` request is admitted: file existence is required before it counts as a resume target.
- Record that `session-repair` shares `ResolveBindings` with the live watcher.

```findings
findings:
  - id: new
    severity: Important
    family: unconfirmed-identity-admitted-as-resumable
    title: |
      Unmaterialized Pair-chosen session ID is returned as a resume target
    detail: |
      QueryResumeTarget (query.go:449) returns a chosen-id RequestedNativeID as Provisional before any native file exists, and createflow.go:960 dropped the AgentSessionExists check and fresh fallback. A fresh Claude/Qoder thread quit before its first message gets resumed with --resume X for a conversation that does not exist. Admit an unconfirmed chosen-id only after a metadata-only filename check; otherwise start fresh. Add a production-boundary test.
  - id: new
    severity: Important
    family: correlation-rule-single-source
    title: |
      session-repair re-implements binding resolution instead of reusing ResolveBindings
    detail: |
      recover.go:87 requires exactly one root across the union of all rounds, while the live watcher uses ResolveBindings (authorized filter plus per-group intersection). The two rules diverge, e.g. groups {D} and {D,E}. Call ResolveBindings and accept a unique Provisional root.
  - id: new
    severity: Important
    family: readme-surface-gap
    title: |
      pair session-repair operator CLI is missing from README
    detail: |
      2nd finding in this family. Rule: every dispatcher family an operator is expected to run must appear in README or be explicitly marked internal. Enforce it with a test walking dispatcher.Families() against README with an internal allowlist, not a one-off line.
  - id: new
    severity: Minor
    family: legacy-path-version-guard
    title: |
      Watcher legacy proof migration also fires for v3 bindings that have no proof
    detail: |
      run.go:101 lacks the Version < 3 guard that QuerySession has. On watcher restart it parses the native body, appends a v2 row that hides ConfirmationReason, and skips Codex lifecycle following.
  - id: new
    severity: Minor
    family: dead-code-after-policy-change
    title: |
      decideAutomaticResumeConfig quarantine can no longer trigger
    detail: |
      hasResumable is now always true when a session ID is set, so the Codex quarantine branch is dead.
  - id: new
    severity: Minor
    family: typed-enum-for-persisted-vocabulary
    title: |
      ConfirmationReason is untyped string literals, unlike RequestOrigin
  - id: new
    severity: Minor
    family: untrusted-id-to-argv
    title: |
      requested_native_id is not shape-validated before becoming a --resume argv value
  - id: new
    severity: Minor
    family: implicit-watcher-state
    title: |
      Watcher in-memory phases are separate fields rather than a tagged state enum
    detail: |
      observationEpoch, boundRootNodeID and trackedTargets together encode unknown-baseline, epoch, handshake, correlation, confirmed and lifecycle phases without a written transition set (ARCH-ORDER).
```

---

## Re-review — 2026-09-29T11:26:47-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 346 — Stale temporary store blocks Couch startup |
| repo | pair |
| issue file | workshop/issues/000346-stale-temporary-store.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 73203fc0d103a8c5a5d9490211a09d6055db9a00..e4d1035b969a874c8d8e2a2c4e83122a37684ec9 |
| command | sdlc milestone-close --issue 346 --milestone M2 |
| reviewer | claude |
| timestamp | 2026-09-29T11:26:47-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The fix round handled most of round 3 well. Repair now goes through `ResolveBindings` and has a divergence test. README coverage is enforced by a test that walks `dispatcher.Families()`, so it no longer depends on hand-added lines. The v3 proofless watcher restart is guarded in both places and tested. Confirmation reasons are typed, the dead Codex quarantine code is gone, and requested IDs are checked for argv safety at decode. Two things block SHIP. First, **HEAD fails its own suite**: the README rewrite for BR-7 removed the `provisional`/`established` vocabulary that `TestREADMEDocumentsSessionInventoryContract` requires, so `go test ./cmd/internal/sessioninventory` is red. It is a pure string test, not a sandbox artifact. Second, BR-5 is still open: it is disputed pending an operator decision, and the code still admits an unmaterialized chosen ID. Launcher, wrapcmd and couchcore failures I saw were all sandbox errors (`operation not permitted` on PTY or `/tmp`). The sessionledger, sessionwatch, dispatcher, reviewcmd and opener suites are green.

**1. Strengths**
- `sessionwatch/recover.go` `recoveryRoot`: repair now delegates to `sessioninventory.ResolveBindings`. `TestRepairUsesSameRoundIntersectionAsLiveWatcher` pins the {D},{D,E} case the previous rule got wrong (ARCH-DRY resolved).
- `dispatcher/readme_test.go`: it enforces the rule, not the instance. Every family needs README usage or a written internal reason, and stale allowlist entries fail too. This is the right answer to the `readme-surface-gap` escalation.
- `sessionwatch/run.go`: the `Version < 3` guard sits both at the call site and inside `migrateProoflessBinding`. `validateCurrentLifecycleTarget` lets v3 filename-confirmed bindings still follow Codex lifecycle. `TestRestartedV3ProoflessWatcherPreservesConfirmationAndFollowsLifecycle` asserts zero ledger writes and 2 lifecycle records.
- `run.go` takes one metadata snapshot per poll, shared by the epoch, handshake and correlation steps. This fixes the round-3 ARCH-CONSTRAINTS minor, and `TestChosenProbationUsesOneMetadataSnapshotPerPoll` pins it.
- `launcher/ledger.go` `ParseLedger` now uses the shared pure `ResumeTargetForLaunch`. `TestRunRestartPreservesRequestedProbationFromRealLedger` covers the requested-A restart gap the implementor found.

**2. Critical findings**
None.

**3. Important findings**
- **The committed HEAD fails `TestREADMEDocumentsSessionInventoryContract`** (`cmd/internal/sessioninventory/runcli_test.go:208-209`, `README.md` around line 728).
  - The BR-7 rewrite replaced the "reports `provisional` until … `established`" paragraph with probation prose. The status values `pair session-inventory` actually emits are no longer documented.
  - Fix: restore a sentence naming the `provisional`/`established`/`ambiguous` statuses next to the probation paragraph, then run the full non-sandboxed `make test` before re-closing.
  - This is not another `readme-surface-gap` instance: the rule is already enforced by a test, and that test caught this. What failed is that the boundary was committed without the suite being run. See the family note below.
- **BR-5 is still open** (disposition below).

**4. Minor findings**
- BR-11 was only partly swept. Binding `RootNativeID` also reaches argv through `ResumeTargetForLaunch` → `--resume`, but `validateRecord` checks only `RequestedNativeID`. A hand-edited or older binding row bypasses the check. Apply the same argv-safety predicate to every ledger ID that can become argv, i.e. binding `RootNativeID` too.
- BR-12: only a comment was added in `run.go:88`. The phases are still implicit fields. Deferring this is acceptable at Minor.
- The `readme_test.go` substring match `"pair "+name` would accept a prefix collision (e.g. `pair gc` inside `pair gcx`). This is low risk.

**5. Test coverage notes**
- The new tests assert independently stated outcomes: ledger write counts, lifecycle record counts, argv-safe and unsafe ID sets, and repair/live parity.
- Missing: a production-boundary test for "chosen ID, no native file → ?". It should pin whichever policy the operator picks for BR-5. Today `wrapcmd/agent_restart_test.go` pins admission implicitly.

**6. Architecture pass**

| Principle | Result |
| --- | --- |
| ARCH-DRY | Pass: repair and live share one resolver; restart and query share one target projection |
| ARCH-PURE | Pass: `ResumeTargetForLaunch` and `recoveryRoot` are pure over their inputs |
| ARCH-PURPOSE | Pass on the consumer sweep. Flag: BR-11 sibling, and BR-5 is pending |
| ARCH-MOCK | Pass: the stateful fake native runtime drives the watcher tests |
| ARCH-CONSTRAINTS | Pass: one snapshot per poll, now tested |
| ARCH-SECURE | Minor: binding `RootNativeID` is not argv-validated |
| ARCH-ORDER | Durable state passes (ledger fold under lock). The in-memory watcher phases remain implicit (BR-12, Minor) |
| ARCH-FUNERAL | Pass: no new artifact families; the v3 fields add a few bytes per row |

For upcoming work, `AgentSessionExists` is still wired (`createflow.go:595`, collision check). The metadata-only gate BR-5 proposes is therefore a one-line reuse whichever policy is chosen.

**7. Plan revision recommendations**
- Add a `## Revisions` entry recording the operator's decision on unmaterialized chosen-ID admission: either "probation target regardless of file existence" or "requires a named-file metadata check, else fresh". Whichever way it goes, the decision closes BR-5.
- Record that `session-repair` shares `ResolveBindings` with the live watcher.

```findings
dispose:
  - id: BR-5
    disposition: not-addressed
    note: |
      Intentionally unchanged pending operator decision (issue Log 2026-09-29); QueryResumeTarget still admits an unmaterialized chosen-id and runConfigPicker has no existence check. Resolve by recorded operator Revision (then withdraw) or add the metadata-only gate plus a boundary test.
  - id: BR-6
    disposition: addressed
    note: |
      recover.go recoveryRoot now calls ResolveBindings; TestRepairUsesSameRoundIntersectionAsLiveWatcher pins the {D},{D,E} divergence the union rule got wrong.
  - id: BR-7
    disposition: addressed
    note: |
      README documents session-repair and dispatcher/readme_test.go enforces the family-level rule with an internal allowlist and stale-entry check.
  - id: BR-8
    disposition: addressed
    note: |
      Version<3 guard at run.go call site and in migrateProoflessBinding; TestRestartedV3ProoflessWatcherPreservesConfirmationAndFollowsLifecycle asserts no ledger writes and lifecycle followed.
  - id: BR-9
    disposition: addressed
    note: |
      decideAutomaticResumeConfig and its dead quarantine branch removed along with its test.
  - id: BR-10
    disposition: addressed
    note: |
      ConfirmationReason is a typed enum validated in validateRecord; all call sites use the constants.
  - id: BR-11
    disposition: not-addressed
    note: |
      RequestedNativeID is validated, but the sibling binding RootNativeID that also reaches --resume via ResumeTargetForLaunch is not; apply the same argv-safety predicate to every ledger ID that can become argv.
  - id: BR-12
    disposition: not-addressed
    note: |
      Only a descriptive comment was added at run.go:88; phases remain independent fields. Acceptable to defer as Minor.
findings:
  - id: new
    severity: Important
    family: full-suite-before-boundary
    title: |
      HEAD fails TestREADMEDocumentsSessionInventoryContract after the BR-7 README rewrite
    detail: |
      The fix commit replaced the README paragraph naming the provisional/established statuses with probation prose, so go test ./cmd/internal/sessioninventory is red (a pure string check, not the sandbox). Restore the status vocabulary next to the probation paragraph. The rule: every boundary commit runs the full non-sandboxed make test before sdlc milestone-close, because a docs-only-looking edit can break an enforced contract test.
```

---

## Re-review — 2026-09-29T13:31:56-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 346 — Stale temporary store blocks Couch startup |
| repo | pair |
| issue file | workshop/issues/000346-stale-temporary-store.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 73203fc0d103a8c5a5d9490211a09d6055db9a00..c1a98adbbceef2b3174482d24d9f4114bb40ac6f |
| command | sdlc milestone-close --issue 346 --milestone M2 |
| reviewer | claude |
| timestamp | 2026-09-29T13:31:56-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

Three of the four open findings are fixed. BR-5 is resolved at the production boundary: `ResumeTargetForRuntimeLaunch` now checks for the root filename, using metadata only, before it reuses an unconfirmed Pair-chosen ID. If the file is missing, it sets `FreshRequired`, and every consumer honours that: standalone Alt+n, createflow, and Couch relaunch, which also re-checks before it spawns a child. BR-11 now also validates the binding's `RootNativeID` before it can reach argv. BR-13 is fixed: every affected package passes when run outside the sandbox (sessioninventory including the README contract test, plus sessionledger, sessionwatch, launcher, couchcore, dispatcher, opener, reviewcmd, wrapcmd and couchcmd). BR-12 is still open, but it is Minor.

The new gate brings one new Important defect. It cannot tell "the probe failed" apart from "the file is missing". When listing the native root fails with anything other than `ErrStorageAbsent`, or returns only part of the listing, the gate decides "unmaterialized" and restarts fresh with a new UUID. That breaks the README promise that filesystem failures don't block a recorded target, and it breaks ARCH-ORDER's rule that a failed probe does not prove absence.

**Strengths**
- `query.go:465` `ResumeTargetForRuntimeLaunch` is one shared projection. The query, `OSRuntime.ReadLedger` (`osruntime.go:653`) and Couch all use it, so ARCH-DRY holds.
- `osruntime_test.go:771` runs the whole chain end to end: real OS ledger, then the metadata adapter, then the Alt+n marker, then a fresh launch that mints `new-Y`. It covers both Claude and Qoder, and both the absent and present cases.
- `resume_target_test.go:84` checks that no native body is read (`OperationReadAt == 0`). This proves the metadata-only claim instead of just stating it.
- `relaunch_test.go:444` covers two things changing between the check and the use of the ID: a new request appears, or the transcript materializes. In both cases Couch must refuse and spawn nothing.
- `planRestart` now routes through `FreshAgentArgs`, which strips stale `--session-id` and `--resume` flags. Without that, the stale ID could come back.

**Critical**
- None.

**Important**
- **The metadata probe treats a failed probe as absence** (`query.go:467-475`, `incremental_inventory.go:72-76`, `scan_helpers.go:149`). `ObserveAgentMetadata` skips a root whose `ListFiles` fails, or returns a partial list. Its diagnostics are appended to the result, but the decision ignores them, so `SelectTargetWork` reports Unavailable and the gate sets `FreshRequired=true`.
  - **Scenario:** an EACCES or EIO error, or a partial listing, on `~/.claude/projects` while chosen ID X has a materialized transcript.
  - **Result:** Alt+n, Couch relaunch or `pair <tag>` starts a fresh conversation under a new UUID. Createflow also deletes the config. The tag is now bound to the new conversation and X is abandoned.
  - **Fix:** make the probe three-valued: present, confirmed absent, or unknown. Only a complete listing, or `ErrStorageAbsent`, counts as absent. If the result is unknown, set neither `NativeID` nor `FreshRequired`, and let the existing provisional refusal handle it (with a diagnostic in Couch, a refusal in standalone).
  - **Test to add:** a fake-runtime `ListFiles` error with X present must produce neither fresh nor admit.

**Minor**
- BR-12 is still open: `run.go:84-92` keeps the watcher phases in separate fields and only added a comment (ARCH-ORDER).
- ARCH-CONSTRAINTS: `ResumeTargetForRuntimeLaunch` walks every native root for the agent, and Couch relaunch calls it at least three times in one relaunch. Only provisional chosen IDs pay this cost, but a targeted filename lookup, or one observation reused across the relaunch, would bound it.
- `history.go:165` still uses the pure `ParseLedger`, so a provisional chosen ID shows an empty session there, while the runtime projection may admit it. The behaviour is conservative, but the two projections are not documented as differing.

**Test coverage notes**
- Present, absent and confirmed-without-file are covered. The failed-listing case and the partial-listing case are not, and that is exactly the gap behind the Important finding.

**Architecture pass**
- **ARCH-DRY:** pass.
- **ARCH-PURE:** pass. The IO lives in the runtime adapter, and the selection is the pure `SelectTargetWork`.
- **ARCH-PURPOSE:** pass. The shadow sweep of `QueryResumeTarget` consumers (runcli, reviewcmd, opener, launcher, couchcore) found that all of them derive from the runtime projection.
- **ARCH-MOCK:** pass. There is a stateful `FakeRuntime` at the same seam, plus a real-OS test.
- **ARCH-CONSTRAINTS:** minor note above.
- **ARCH-SECURE:** pass. Both requested and root IDs are argv-safe.
- **ARCH-ORDER:** flag. See the Important finding (failed probe collapsed into absence) and BR-12. The check-then-use race is handled well.
- **ARCH-FUNERAL:** pass. The change creates nothing durable beyond the existing ledger rows.

**Plan revision recommendations**
- Add one line to the BR-5 Revision stating that "unmaterialized" requires a complete listing, and that a probe failure keeps the provisional refusal rather than restarting fresh.

```findings
dispose:
  - id: BR-5
    disposition: addressed
    note: |
      ResumeTargetForRuntimeLaunch gates chosen-id on root filename via metadata; FreshRequired honoured by Alt+n, createflow, Couch; tests in resume_target_test.go:84, osruntime_test.go:771, relaunch_test.go:413 fail without it.
  - id: BR-11
    disposition: addressed
    note: |
      record.go:437 applies safeNativeArg to binding RootNativeID too; probation_test.go:92 covers v1-v3 unsafe binding IDs.
  - id: BR-12
    disposition: not-addressed
    note: |
      run.go:84-92 still separate fields with a descriptive comment only; acceptable to defer as Minor.
  - id: BR-13
    disposition: addressed
    note: |
      go test ./cmd/internal/sessioninventory (incl. TestREADMEDocumentsSessionInventoryContract) passes at HEAD, as do the other touched packages unsandboxed.
findings:
  - id: new
    severity: Important
    family: failed-probe-treated-as-absence
    title: |
      Chosen-id materialization probe collapses a native listing failure into FreshRequired
    detail: |
      ObserveAgentMetadata skips roots whose ListFiles fails (non-ErrStorageAbsent) or returns partial listings, and ResumeTargetForRuntimeLaunch (query.go:467) ignores the diagnostics and sets FreshRequired, so a transient EACCES/EIO abandons a materialized conversation X for a new UUID (createflow also removes the config). Make the probe tri-state (present / confirmed-absent / unknown); on unknown set neither NativeID nor FreshRequired and keep the provisional refusal; add a fake ListFiles-error test.
```

---

## Re-review — 2026-09-29T13:47:01-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 346 — Stale temporary store blocks Couch startup |
| repo | pair |
| issue file | workshop/issues/000346-stale-temporary-store.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 73203fc0d103a8c5a5d9490211a09d6055db9a00..81106f6ad724e35dc84eeb6ac5739fa9f0fe81b6 |
| command | sdlc milestone-close --issue 346 --milestone M2 |
| reviewer | claude |
| timestamp | 2026-09-29T13:47:01-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: medium
```

**VERDICT: SHIP.** The BR-14 fix works. A chosen ID is now accepted only when a listing positively shows its root filename. Absence is concluded only after a complete listing or when the storage root is absent. Any other failure leaves the identity unknown, and each consumer refuses before doing anything destructive: standalone restart refuses before writing markers or killing the session, the config picker refuses before removing the saved config, and Couch refuses before parking. The new tests cover a failed listing, a partial listing, a present file alongside an unrelated error, an unreadable ledger and conflicting bindings. I ran them and they pass. BR-12 is unchanged, but it is Minor and was already accepted for deferral. Nothing blocks the boundary. The new findings are two Minor notes about how the "unknown" state is represented and how its refusal is worded.

1. **Strengths**
   - `cmd/internal/sessioninventory/query.go:470-483`: "a matching filename proves presence even in a partial listing" is the right order of checks. A present file wins over unrelated diagnostics, and absence needs every diagnostic to be `DiagnosticStorageAbsent`. `TestChosenTargetListingFailureIsUnknown` checks both directions. Restoring the old `else { FreshRequired = true }` would make the `!got.FreshRequired` assertion fail, so the test would catch the regression.
   - `cmd/internal/launcher/createflow.go:988-1030`: the picker reuses the typed projection it already read instead of probing a second time. That avoids a race where a second probe could turn an observed root into "unknown", and `TestConfigPickerReusesTypedResumeProjection` pins it.
   - `restart.go` and `createflow.go` now treat a non-ENOENT ledger read error as a refusal instead of silently continuing. This covers more than the site BR-14 named, and `TestUnreadableLedgerRefusesRestartAndCreate` covers both paths.
   - `TestCouchChosenUnknownMetadataRefusesBeforePark` checks that no runner ops happen and the process stays Live, so it checks what actually happens, not just the error text.
   - The atlas and README were updated in the same commit for the new refusal behaviour.

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - **"Unknown" has no field of its own.** BR-14 asked for a tri-state; it was delivered as "neither `NativeID` nor `FreshRequired` is set", and three places work it out separately:
     - `ResumeTargetForRuntimeLaunch` (query.go:477)
     - `couchcore/resume.go:377`, which loops over the diagnostics again; that loop is redundant, since the outer condition already implies unknown
     - `launcher/ledger.go:75`, the `ResumeBlocked` predicate
     
     Also, the pure `ParseLedger` sets `ResumeBlocked=true` for every unconfirmed chosen entry, because the pure projection can't see filenames. So the field's meaning depends on which projection built it. See the family note after the findings block.
   - **The refusal is permanent in some cases but tells the operator to retry.** A symlinked `~/.claude/projects/<dir>` is rejected as a non-regular entry (`ListingIssuesError`). That makes every unmaterialized chosen ID "unknown" permanently, yet the message says "retry when its storage can be listed". Refusing is correct here, because the file could sit behind the symlink. But the message should name the rejected entry and the fresh-start escape hatch (`pair restart --new-session` / fresh launch).

5. **Test coverage**
   - The fault seams (`incompleteNativeListing`, `SetError`, `unreadableLedgerRuntime`, a real symlink under the OS runtime) are all injectable, and each one forces a single, reproducible failure. That gives the ARCH-ORDER failed-probe cases proper coverage.
   - **My runs:**
     - `sessioninventory`, `sessionwatch` and `sessionledger` all pass.
     - The named `launcher` regressions pass and the `couchcore` `Chosen|Resume` subset passes.
     - The failures I saw are sandbox errors, not regressions: `TestCreateLayoutWrapperPreservesAgentCommand` failed with "operation not permitted", and several `couchcore` pty tests failed with "ptychild ... operation not permitted", which is the known sandbox limit on starting pty children. Re-run the full suite outside the sandbox before `sdlc close`.

6. **Architecture**
   - **ARCH-DRY:** pass on the picker (it reuses the projection). Minor flag on "unknown" being derived in three places.
   - **ARCH-PURE:** pass. The decision sits in `ResumeTargetForRuntimeLaunch`, and the probe is injected through `Runtime`.
   - **ARCH-PURPOSE:** pass. Every consumer (restart, picker, Couch) was covered in the same round, not only the site that was named.
   - **ARCH-MOCK:** pass. The stateful fake runtime is used, and an OS-level symlink test checks it against the real filesystem.
   - **ARCH-CONSTRAINTS:** pass. The probe reads filenames only, never transcript bodies.
   - **ARCH-SECURE:** pass. An unreadable or partial listing now shows up as a refusal instead of being read as absence.
   - **ARCH-ORDER:** pass on behaviour, since an uncertain result is no longer treated as absence. Minor flag: the tri-state is implicit (same issue as BR-12).
   - **ARCH-FUNERAL:** pass. The fix creates nothing durable, and a refusal now keeps the saved config instead of removing it.

7. **Plan revisions:** none needed. The issue Log entry for BR-14 matches the code.

```findings
dispose:
  - id: BR-12
    disposition: not-addressed
    note: |
      sessionwatch/run.go:84-92 still carries trackedTargets/boundRootNodeID/observationEpoch as independent fields; Minor, deferral previously accepted.
  - id: BR-14
    disposition: addressed
    note: |
      query.go:470-483 now returns neither NativeID nor FreshRequired when any non-absent diagnostic exists; TestChosenTargetListingFailureIsUnknown (failed + partial listing) goes red against the old else-branch; restart/picker/couch refuse before effects (restart_test, osruntime_test, relaunch_test).
findings:
  - id: new
    severity: Minor
    family: implicit-watcher-state
    title: |
      Chosen-ID "materialization unknown" is an implicit field combination re-derived in three places
    detail: |
      ResumeTarget encodes unknown as NativeID=="" and !FreshRequired; couchcore/resume.go:377 (redundant diagnostics loop) and launcher/ledger.go:75 (ResumeBlocked) re-derive it, and pure ParseLedger sets ResumeBlocked for every unconfirmed chosen entry. Rule: a state with more than two legal values goes on the producing type as a tagged enum (e.g. Materialization present/absent/unknown) that consumers switch on; apply it to ResumeTarget and to the watcher phases from BR-12.
  - id: new
    severity: Minor
    family: outage-diagnostic-actionability
    title: |
      A persistent incomplete listing (such as a symlinked project dir) refuses with a "retry" message that never succeeds
    detail: |
      A ListingIssuesError from a non-regular entry is permanent, not transient. Rule: every unknown-identity refusal names the failing root or entry and the explicit fresh-start escape hatch (pair restart --new-session or a fresh launch), rather than only telling the operator to retry.
```

**Family note.** This is the 2nd finding in `implicit-watcher-state` and the 2nd in `outage-diagnostic-actionability`. In both cases the findings block above states the rule to fix rather than the single instance: one typed enum for any state with more than two legal values, and refusal messages that name the failing entry and the escape hatch.
