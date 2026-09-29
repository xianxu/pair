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
