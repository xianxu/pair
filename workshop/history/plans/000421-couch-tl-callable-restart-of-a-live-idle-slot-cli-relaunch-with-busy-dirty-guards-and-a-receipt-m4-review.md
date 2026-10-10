# Boundary Review — pair#421 (milestone M4)

| field | value |
|-------|-------|
| issue | 421 — couch: TL-callable restart of a live idle slot (CLI relaunch with busy/dirty guards and a receipt) |
| repo | pair |
| issue file | workshop/issues/000421-couch-tl-callable-restart-of-a-live-idle-slot-cli-relaunch-with-busy-dirty-guards-and-a-receipt.md |
| boundary | milestone M4 |
| milestone | M4 |
| window | a44ef496162f06bcaae791701107e7e9673ec604..26781ac7faee91d447eef393431897d45cc46298 |
| command | sdlc milestone-close --issue 421 --milestone M4 |
| reviewer | claude |
| timestamp | 2026-10-10T01:22:59-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Reviewing the M4 diff now; checking admission, the socket's implicit-arg trust, and the README claim.

Confirmed the dispatch wiring gap; waiting on the test run, then writing the verdict.

I've finished reviewing the M4 window; writing up the verdict.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

M4's code is small and correct. Admission now records its evidence in the implicit `require-settled` argument: `known` when the wrapper said it was settled, `forced` when `--force-unknown` let an unknown through. Both verbs carry that evidence to the effect. `refuseUnlessIdle` gives one rule for both effect sites. A slot whose idle state turns unknown after a known-settled admission is now refused (`busy-unknown`) instead of being treated as idle. Implicit arguments cannot be injected by an untrusted caller (`operationdispatch.go:108`). The skill doc and atlas sections are clear. Nothing is Critical. Two Important items should be fixed before the boundary, and both are cheap:
- No test checks how dispatch turns the argument into the probe's `forced` flag. The fake records the flag, but no test ever reads it.
- The live check was deferred to the TL, but neither the Done-when nor the plan's M4 body was revised to say so.

1. **Strengths**
   - `live_restart_probe.go:153`: `refuseUnlessIdle` is one rule for both verbs. Moving it before the `p.proc == nil` check means a busy wrapper is reported as busy, not as "cannot signal".
   - `live_restart.go:238-241`: admission now always sets `require-settled` for both verbs. That fixes the old gap where reload-context was missing it.
   - `live_restart_probe_test.go:271-277`: the new case "known at admission, unknown at the effect" is pinned for both `ConfirmNotBusy` and `RestartConversation`.
   - `TestLiveRestartAdmissionRecordsItsEvidence` covers both verbs and both outcomes, including that `--force-unknown` on a known slot stays `known`.
   - The SKILL.md rollout section maps each refusal code to its fix and says plainly that overrides are not a retry tactic.

2. **Critical:** none.

3. **Important**
   - **No dispatch test reaches `forced`** (`operationdispatch.go:415,425`). The executor turns `a[require-settled]` into the `forced` bool for both verbs. Nothing tests that step: `fakeLiveRestartProbe.forced` (`slot_operation_test.go:192`) is written but never read.
     - Mutation check: change `:425` to pass a constant `false`. Every test stays green, yet `--force-unknown` on reload-context is then refused at every effect, so the override has no effect. A constant `true` would let a disconnect after admission through.
     - This is the **5th finding in family `safety-guard-wiring-untested`**. **Rule:** a value that admission writes into an implicit argument is tested from the producer to the consumer. The test is one table through `PrepareSlotOperation` → `CouchLiveOwnerExecutor` → probe, over verb × {known, forced, absent/console}, asserting whether the probe is called and the `forced` value it receives. Testing each side of the seam alone does not count. Add this rule to `lessons.md`.
   - **The deferral did not revise the Done-when or the plan body.** The issue's Done-when (line 65) still says "a live check restarts a real idle slot". The plan's M4 row (plan:134) still promises the live check with no superseded/deferred marker. Only the issue's Plan checkbox and a Revisions entry mention the deferral.
     - This is the **5th finding in family `revision-supersedes-body-unmarked`**. **Rule:** a revision that changes scope marks every line that states that scope (issue Done-when, issue Plan, plan body) in the same commit. Extend the existing "A plan revision sweeps the body" lesson to say this explicitly.
     - Either mark Done-when with the deferral and the TL's acceptance, or leave the issue open until the TL runs the checklist.

4. **Minor**
   - `live_restart.go:239` recomputes DecideLiveRestart's "unknown" condition (`!Session || !SettledKnown`, `:80`). The decision should return its own evidence (ARCH-DRY). The value is a free string, and any non-empty value other than `forced` is read as known. Family `typed-outcome-survives-boundary` (3rd). **Rule:** a decision's outcome crosses the seam as the value the decision produced, not one recomputed from the inputs.
   - SKILL.md's `busy-unknown` line names only "wrapper predates idle reporting → manual Alt+n". Since this change it also fires when the session vanishes or the wrapper's settled claim disappears between admission and effect; there the fix is to peek or retry, not Alt+n. Family `docs-surface-lags-cli` (2nd). **Rule:** each refusal code's doc lists every site that emits it.

5. **Test coverage notes**
   - The targeted couchcore tests pass.
   - The couchcmd tests fail in this sandbox on a `/tmp` mkdir. That is the environment, not this change.
   - `artifactpath` fails on many inventory entries from other areas (`couchmessage/*`, `nvim/review/*`). That failure predates this change, which only adds four entries; the main agent should confirm it also fails on main.

6. **Architecture**
   - **ARCH-DRY:** flag (Minor, above).
   - **ARCH-PURE:** pass. The decision is pure and the probe is a thin injected layer.
   - **ARCH-PURPOSE:** flag (the live check from Done-when was deferred without a revision; Important, above).
   - **ARCH-MOCK:** pass. The probe seam has a fake and procOps has a fake, though the fake's `forced` record is unused (above).
   - **ARCH-CONSTRAINTS:** pass. The effect re-check is one in-memory read.
   - **ARCH-SECURE:** pass. Implicit arguments are rejected from untrusted callers.
   - **ARCH-ORDER:** pass. Admission's evidence now carries to the effect, and an unknown is no longer collapsed into idle. The interleaving is tested by setting state, not by injecting order, which is acceptable for a single read.
   - **ARCH-FUNERAL:** pass. Nothing new is persisted.

7. **Plan revision recommendations**
   - Mark plan:134 (M4) with "*(live check deferred to the TL; Revisions 2026-10-10 M4)*".
   - Annotate the Done-when live-check bullet the same way, or keep it as a gate that holds the issue open.

```findings
findings:
  - id: new
    severity: Important
    family: safety-guard-wiring-untested
    title: |
      Dispatch translation of require-settled into the probe's forced flag is untested; the fake's forced field has no reader
    detail: |
      operationdispatch.go:415 and :425 map known/forced into a bool, and no test asserts it. Hard-coding false at :425 makes reload-context's --force-unknown always refused, and every test stays green. Fifth in this family. Rule: a value admission writes into an implicit argument is tested from producer to consumer, one table through PrepareSlotOperation, CouchLiveOwnerExecutor and the probe, over verb x {known, forced, absent}. Add the rule to lessons.md.
  - id: new
    severity: Important
    family: revision-supersedes-body-unmarked
    title: |
      M4 live-check deferral leaves the issue Done-when and plan M4 row unmarked
    detail: |
      Issue Done-when line 65 still requires a live check; plan line 134 still promises it with no deferred marker. Fifth in this family. Rule: a scope-changing revision marks every line that states the scope (Done-when, issue Plan, plan body) in the same commit. Extend the existing lesson to cover Done-when.
  - id: new
    severity: Minor
    family: typed-outcome-survives-boundary
    title: |
      Admission recomputes the unknown predicate for the forced evidence instead of taking it from DecideLiveRestart
    detail: |
      live_restart.go:239 duplicates :80, and the value is a free string. Rule: a decision's outcome crosses the seam as the value the decision produced, not one recomputed from the inputs.
  - id: new
    severity: Minor
    family: docs-surface-lags-cli
    title: |
      SKILL.md busy-unknown names only the old-wrapper cause, not the disconnect-after-admission refusal added here
    detail: |
      Rule: each refusal code's doc lists every site that emits it, and the fix for each.
```

---

## Re-review — 2026-10-10T01:26:03-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 421 — couch: TL-callable restart of a live idle slot (CLI relaunch with busy/dirty guards and a receipt) |
| repo | pair |
| issue file | workshop/issues/000421-couch-tl-callable-restart-of-a-live-idle-slot-cli-relaunch-with-busy-dirty-guards-and-a-receipt.md |
| boundary | milestone M4 |
| milestone | M4 |
| window | a44ef496162f06bcaae791701107e7e9673ec604..77b1f23cebdb4acfaeba1b258638faa4c9a9b9d7 |
| command | sdlc milestone-close --issue 421 --milestone M4 |
| reviewer | claude |
| timestamp | 2026-10-10T01:26:03-07:00 |
| verdict | SHIP |

## Review

Disposing prior findings; running the targeted tests plus one mutation check now.

```verdict
verdict: SHIP
confidence: high
```

All four open findings are fixed, and I found nothing new to raise. Admission now records what it decided (`DecideLiveRestart` sets `Forced`). It passes that decision to the effect through the implicit `require-settled` argument, which is either `known` or `forced`. One effect-time rule, `refuseUnlessIdle`, then serves both verbs. That rule closes the gap where a slot admitted as known-settled could disconnect and then pass as idle. The end-to-end table test catches the mutation BR-22 described: I hard-coded `false` at the reload-context dispatch in a scratch worktree and `TestRequireSettledProducerToConsumer` failed (`reload-context/forced: probe saw forced=[false], want [true]`). The scope-change markings, the SKILL.md refusal docs and the new lessons are all in place.

1. **Strengths**
   - `cmd/internal/couchcmd/live_restart_probe.go:52` — one effect-time rule for both verbs. It refuses a known-busy slot even when forced, and refuses an unknown slot unless admission was forced. Any value other than `known` or `forced` is treated as not forced, so an unexpected value fails safe.
   - `cmd/internal/couchcore/live_restart.go:250` — the decision's own output (`d.Forced`) crosses into the call arguments, so admission no longer recomputes it from the facts. This fixes BR-24, and it applies the rule for the `typed-outcome-survives-boundary` family, not just this one site.
   - `cmd/internal/couchcore/slot_operation_test.go:480` — one table runs from admission through dispatch to the probe, over both verbs × {known, forced, absent}. The fake records `forced`, and the test reads it.
   - `cmd/internal/couchcore/slot_operation_test.go:454` — pins that `--force-unknown` on a slot already known to be settled stays `known`.
   - `workshop/lessons.md` — four rules stated at the family level, as the rule-not-site instruction asks.

2. **Critical:** none.

3. **Important:** none.

4. **Minor:**
   - The `absent` row for reload-context hand-deletes the argument. That state can't occur in production, because remote admission always sets it. The row is harmless and documents the fail-safe default.
   - In `RestartConversation`, a mismatched process identity also returns `busy-unknown`. The SKILL.md wording ("idle state was lost … a disconnect") covers it only loosely. Not worth another round.

5. **Test coverage**
   - The targeted `couchcore` tests pass.
   - The `couchcmd` restart and arity tests pass.
   - `TestLifecycleStartDetachReattachReplaceRestart` failed in the sandbox with `mkdir /tmp/...: operation not permitted`. That is the sandbox blocking `/tmp`, not a regression in this window.

6. **Architecture**
   - **ARCH-DRY:** pass — one guard function now serves both verbs.
   - **ARCH-PURE:** pass — the decision stays in a pure function, and the probe is a thin layer over input/output.
   - **ARCH-PURPOSE:** pass — the live check is deferred, and that deferral is marked in the Done-when, the issue's M4 row, the plan's M4 row and the Revisions.
   - **ARCH-MOCK:** pass — the fake probe now records the `forced` flag it receives, and the test reads it.
   - **ARCH-CONSTRAINTS:** N/A — no new runtime work.
   - **ARCH-SECURE:** pass — `require-settled` is implicit (callers can't supply it), and unknown values fail safe.
   - **ARCH-ORDER:** pass — the window between admission and effect is modelled explicitly: known→unknown refuses, and forced→unknown passes.
   - **ARCH-FUNERAL:** N/A — this window creates nothing durable.

7. **Plan revisions:** none. The plan's Revisions already record the M4 deferral and the round-1 fixes.

```findings
dispose:
  - id: BR-22
    disposition: addressed
    note: |
      TestRequireSettledProducerToConsumer covers both verbs x known/forced/absent through PrepareSlotOperation, CouchLiveOwnerExecutor and the probe. A scratch mutation hard-coding false at the reload dispatch turns it red. The rule is in lessons.md.
  - id: BR-23
    disposition: addressed
    note: |
      The issue Done-when (lines 65-68), the issue M4 row (line 101) and the plan M4 row (line 134) all mark the deferral. The lesson extends to Done-when.
  - id: BR-24
    disposition: addressed
    note: |
      DecideLiveRestart returns Forced, and prepareLiveRestart writes d.Forced; the duplicate predicate is gone.
  - id: BR-25
    disposition: addressed
    note: |
      SKILL.md busy-unknown now names the admission causes and the effect-time disconnect, each with its fix.
```
