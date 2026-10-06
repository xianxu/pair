# Boundary Review — pair#387 (milestone M2)

| field | value |
|-------|-------|
| issue | 387 — Slot reconciler: reconcile a Couch slot's dispersed state |
| repo | pair |
| issue file | workshop/issues/000387-add-slot-repairs-a-slot-whose-directory-was-deleted.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | c4637555ac43030a646bc155f7e6b4dc8f97562c..e1fceaf7eb1a73ce1759a2de3748f17d202bbf00 |
| command | sdlc milestone-close --issue 387 --milestone M2 |
| reviewer | claude |
| timestamp | 2026-10-05T20:09:13-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

M2 delivers what it set out to do. `Ensure`'s body is now the level-triggered `reconcileLoop`. `NextHostAction`, `CreationIntent` and the "open again to retry" path are gone. Every failure flows through one function, `ReconcileAdvice`. `selectedSlot` reconciles every open, resume, reboot and fresh start. `Discover` collects slot numbers from directories, registrations and resting branches, and `SelectStartSlot` no longer refuses the whole repository. `couch --reconcile` exists, and the real-git acceptance tests pass. I ran the slot, reconcile, converge, provision, outcome, discover, reboot and select tests in a scratch worktree at e1fceaf7: all passed. The other couchcore failures there were pty starts blocked by the sandbox, and couchcmd did not build because the runtime assets were never generated in that worktree.

Two findings should be fixed before shipping:
- **Reboot can't repair a broken host when an agent is live.** The refusal tells the operator to reboot, and reboot refuses with the same text. I reproduced this in a scratch test.
- **Each refused set-aside leaves an empty saved-work entry, and those count toward the 16-entry cap.** Nothing removes them until M3's collection lands, and then only after a year.

### 1. Strengths
- **No-progress detection is exact.** `slotreconcile.go` / `slotplan.go` report a step that was already tried and is still needed as `Retried`, instead of comparing observations. Comparing would have misfired on `RepairHost`, whose effect only the next pass can judge. `SlotPlan.Empty` counts `Retried`, so a step that reports success without effect can't look converged.
- **One positive-evidence check.** `checkoutEvidence` (`slotsave.go:41`) is shared by the observer and the re-check just before the rename (ARCH-DRY). Any git error other than "not a git repository" counts as unknown, never as broken.
- **`TestReconcileOutcomeTable` checks an independent rule.** It states "blocking iff no agent could work" directly from the final observation instead of restating `OutcomeSeverity`, and that is how the per-resource severity rule was found to be wrong (Revision s).
- **The memo keeps the `tools:1` case cheap.** The R5 memo (`slotmemo.go`) is keyed by HEAD plus the `construct/deps` digests, so repeated add-slot attempts refuse at once. `--reconcile` sets `IgnoreMemo` to compile again.
- **`--show` no longer turns a failed lookup into "not a slot".** `slotOfShowReference` now surfaces the error (`slotreport.go:76`, `operationdispatch.go:198`), and the rule was written into `lessons.md`.

### 2. Critical findings
None.

### 3. Important findings

**A. Reboot can't repair a broken host when an agent is live; the advice is a dead end** (`slotfailure.go` `OutcomeSeverity`, `reboot.go:208`).
- **Reproduction:** in `SlotWorld`, set `Agent=AgentLive`, `RepairFixes=false` and host `broken/unreadable`. The result is blocking with the text "host needs repair, but an agent may be working in the slot; reboot the slot to repair it".
- **Why it dead-ends:** `rebootSlot` calls `selectedSlot` first, gets that same blocking error and returns before stopping the agent, so reboot can never clear the hold.
- **What changed:** plan rule 3 said `agent-live` is degraded "even on a host" so that reboot's first pass can go ahead. Revision (s) made severity depend on the outcome and lost that case. The atlas ("Reboot's first pass holds under the live agent, and its post-stop pass repairs") and the SKILL.md step are wrong for the host.
- **Fix:** the rule is that advice must name an action that can succeed from the state that produced it. Either let reboot's first `selectedSlot` treat an agent hold as passable (stop the agent, then let the post-stop pass converge or refuse), or give the host hold advice that doesn't name reboot.
- **Test:** add the host row to `TestOutcomeTableNamedCells`, plus a caller-level test where a reboot with a live agent and a broken host stops the agent and sets the host aside.

**B. A refused set-aside leaves an empty saved-work entry that counts toward the cap** (`slotsave.go:91`, refusals at 104/111/115; ARCH-FUNERAL).
- **Cause:** the entry directory and its pending manifest are written before three checks: the setup lock, the agent re-check and the evidence re-check. Each refusal leaves an entry with no tree.
- **Consequence:** `savedWorkEntries` counts those entries, so 16 refused attempts (for example "setup running" while the operator keeps re-running) leave the slot at `saved-work-full` for good. Collection is M3 (Task 3.3), and even then only after a year.
- **Fix:** do the refusable checks before creating the entry, and remove the entry on any refusal before the rename. Only a crash should leave a pending entry. Alternatively, count only entries that hold a tree.
- **Test:** a refusal leaves no entry.

### 4. Minor findings
- The refusal errors inside `setAside` ("live-agent: an agent appeared…", the saved-work limit) are plain `fmt.Errorf` values, so `ClassifyConvergeError` reports them as a hand-off to the `:0` agent. They should be holds.
- The `KeepOnFailure` branch in `reconcileLoop` observes again after `unlock()` and re-plans without `SavedWorkFull`, so a saved-work-full hold drops out of the final report.
- When `couch --reconcile` hits a blocking failure, it prints only the advice text. The plan's outcome table says "report, exit 1", and `SlotReconcileError.Result` already carries the report it could render.
- If `lock()` fails on pass 2 or later, the error is returned bare, and `SlotOutcome` treats the run as never having observed the slot, even though steps already ran. The result is still retryable, so the impact is low.

### 5. Test coverage notes
- The domain tests are strong: every single and pair perturbation × 7 agent states, run twice; crash after every step; and fake-versus-real-git agreement on 12 scenarios.
- Missing: a broken host under a live agent at the caller level (finding A), and a set-aside refused in the middle of the step (finding B).
- I couldn't run `couchcmd` in a fresh worktree because the runtime assets are absent, so `renderReconcile` went unchecked here.

### 6. Architectural notes
- **ARCH-DRY: pass.** There is one provisioner, `slotConverger` reuses its helpers, and every surface calls the one advice function.
- **ARCH-PURE: pass.** `PlanSlot`, `ClassifyConvergeError`, `ReconcileAdvice`, `OutcomeSeverity` and `SelectStartSlot` are pure. The loop runs against a `slotWorld` seam.
- **ARCH-PURPOSE: flag (A).** Done-when says a failure is never a futile retry, and the reboot advice for a broken host is one.
- **ARCH-MOCK: pass.** `SlotWorld` must agree with real git, the fixture's weave is stateful, and `TestWeaveConformance` checks the installed weave.
- **ARCH-CONSTRAINTS: pass.** The healthy-slot observe was measured at 68 ms against a 500 ms budget, and the loop is bounded.
- **ARCH-SECURE: pass.** Set-aside paths must be direct children of the env and not symlinks, and the locks are opened with `O_NOFOLLOW`.
- **ARCH-ORDER: pass, with minor notes.** Each run is level-triggered, the evidence is re-checked before the rename, and crash injection covers the steps. The minor gaps are the unlocked re-observe and the bare-error return on a later pass (§4).
- **ARCH-FUNERAL: flag (B).** Refusals leave residue that counts toward the cap.
- `--reconcile` runs in the calling process rather than through the operation queue, a documented departure from the Spec (Revision s). Serialization rests on the host creation lease alone, so M3's recovery report should keep that assumption visible.

### 7. Plan revision recommendations
- Add a Revisions entry for finding A: rule 3's "agent-live is degraded even on a host" no longer holds under outcome-based severity, so say how reboot handles a broken host with a live agent.
- Add a Revisions entry for finding B: create the set-aside entry only after every refusable check passes.

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      Task 2.5 still enumerates its test cases as prose bullets (as do 1.4 and 3.1); Minor, non-blocking.
findings:
  - id: new
    severity: Important
    family: advice-names-reachable-action
    title: |
      Broken host under a live agent: the advice says reboot, and reboot refuses with the same advice
    detail: |
      Outcome-based severity makes host-not-present blocking, so the first selectedSlot in rebootSlot (reboot.go:208) refuses before stopping the agent; reproduced in SlotWorld (AgentLive, RepairFixes=false, host broken/unreadable). Rule: advice must name an action that can succeed from the state that produced it. Let reboot's first pass get past agent holds, or change the host hold text; fix the atlas and SKILL.md claim.
  - id: new
    severity: Important
    family: failed-step-leaves-no-residue
    title: |
      A refused SetAside leaves an empty pending saved-work entry that counts toward the 16-entry cap
    detail: |
      slotsave.go creates the entry and pending manifest (line 91) before the setup-lock, agent and evidence checks (104/111/115). Each refusal leaves an entry, and 16 of them leave the slot saved-work-full until a one-year GC that is not yet built (M3). Run the refusable checks first, or remove the entry on refusal.
  - id: new
    severity: Minor
    family: stop-reason-precision
    title: |
      setAside's agent-appeared and limit refusals are classified as a hand-off, not a hold
    detail: |
      They are plain fmt.Errorf values that ClassifyConvergeError does not recognize, so the operator is told to ask the :0 agent instead of to reboot or clean up. This is the 2nd finding in this family: the rule is that every stop reason a converge step can return must be a typed error that ClassifyConvergeError maps to the same class PlanSlot would give it.
  - id: new
    severity: Minor
    family: error-surface-preserved
    title: |
      couch --reconcile prints only the advice on a blocking failure, not the resources and plan
    detail: |
      The outcome table promises "report, exit 1", and SlotReconcileError.Result carries the report. This is the 3rd finding in this family: the rule is that every failure path of a slot operation renders the observation it already holds next to the advice. Render Result.Observation/Plan on the error path.
```

---

## Re-review — 2026-10-05T20:24:52-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 387 — Slot reconciler: reconcile a Couch slot's dispersed state |
| repo | pair |
| issue file | workshop/issues/000387-add-slot-repairs-a-slot-whose-directory-was-deleted.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | c4637555ac43030a646bc155f7e6b4dc8f97562c..9340dc0686c04f2eea591cc4fe9058434722aee9 |
| command | sdlc milestone-close --issue 387 --milestone M2 |
| reviewer | claude |
| timestamp | 2026-10-05T20:24:52-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

Three of the five open findings are fixed: BR-10, BR-11 and BR-13. Each fix has a reachable test that would fail if the fix were reverted.
- **BR-10:** reboot now gets past the live-agent hold its own advice names. An unknown-agent hold now says to look again instead of telling the operator to reboot.
- **BR-11:** `setAside` now runs every check that can refuse before it writes anything.
- **BR-13:** a failed `couch --reconcile` now shows the resources and plan beside the advice.

The targeted tests pass in both packages. The full `couchcore` package run did not finish within the timeout, so I am not claiming a full-package pass.

BR-12 is only partly fixed. `setAside` still reports an agent it could not observe as a live agent. It also returns an untyped error, classified as a hand-off, when the checkout is no longer broken. The first case now matters more, because of the BR-10 fix: reboot's first pass accepts exactly the "agent is live" hold, so an agent that merely could not be observed gets the reboot advice that the unknown-agent case was just changed to avoid. This is cheap to fix and does not block the gate.

1. **Strengths**
   - `slotrecovery.go:232`: `selectSlot(…, toleratesHolds)` is narrow. It accepts only a hold whose cause is a live agent. Any other blocking outcome still refuses before anything is stopped. The two-case `TestRebootGetsPastAHoldItsAdviceNames` (proceeds on the hold, refuses on a hand-off) checks both directions.
   - `slotsave.go:93-111`: the checks that can refuse now run before the entry is created. The pending manifest is still written before the rename, so crash safety is unchanged. `assertNoSavedWork` checks this on three refusal paths.
   - `run.go:475`: the failed-operation report comes from `SlotReconcileError.Result`, which the error already carries. Nothing is observed a second time.
   - `ReconcileAdvice` splits the live-agent and unknown-agent holds, and the atlas and SKILL.md were updated in the same commit.

2. **Critical**
   - None.

3. **Important**
   - None new. BR-12 is re-raised as not-addressed below.

4. **Minor**
   - BR-12 remains open (details in the findings block).
   - BR-1 remains open. Tasks 1.4, 2.5 and 3.1 still list their test cases in prose.
   - Task 3.1's step order still reads "mkdir, then manifest, then lock". The appended Revisions entry corrects it, which follows the append-only rule.

5. **Test coverage**
   - The BR-10 test fakes the readiness function (`slotReadinessFunc`), so it never takes the `held` branch that skips host verification with an unverified candidate. That is the broken-host case BR-10 reproduced in SlotWorld.
   - A SlotWorld test (AgentLive, host unreadable, record kept in env/.couch) would close that gap. The slot record lives in `<env>/.couch`, so it survives host damage and `prepareRetirement` can still find the agent to stop.

6. **Architecture**
   - **ARCH-DRY: flag.** `PlanSlot`'s `setAside` closure and `slotConverger.setAside` each decide the hold reason from agent evidence and saved-work state separately. This is the root cause of the remaining BR-12 gap.
   - **ARCH-PURE: pass.**
   - **ARCH-PURPOSE: pass.**
   - **ARCH-MOCK: pass.** SlotWorld and the fake weave cover the seams. The BR-10 test bypasses them, as noted above.
   - **ARCH-CONSTRAINTS: pass.**
   - **ARCH-SECURE: pass.** `provisionSafePath` and the no-symlink check are kept.
   - **ARCH-ORDER: flag (same as BR-12).** The converge step folds an unknown agent into "live", so a state the plan keeps separate is merged at the transition.
   - **ARCH-FUNERAL: pass.** Refusals no longer leave entries behind. A rename failure after the manifest write still leaves a pending entry, which is the documented crash-safety state.

7. **Plan revisions**
   - When BR-12 is fixed, the Revisions entry should name one shared pure hold-reason function, used by both `PlanSlot` and `setAside`, as the enforcement of the rule. A list of the typed error values is not enough.

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      Tasks 1.4, 2.5 and 3.1 still list test cases in prose. Minor, and does not block the gate.
  - id: BR-10
    disposition: addressed
    note: |
      selectSlot(toleratesHolds) accepts only the live-agent hold, and the unknown-agent hold now advises looking again. TestRebootGetsPastAHoldItsAdviceNames fails when the tolerance is removed. Atlas and SKILL.md are updated.
  - id: BR-11
    disposition: addressed
    note: |
      The limit, lock, agent and evidence checks now run before the entry and manifest are written (slotsave.go:93-111). assertNoSavedWork and the 16-entry count assertion pin it.
  - id: BR-12
    disposition: not-addressed
    note: |
      setAside still returns errAgentAppeared when the agent is unknown (running || !known), so ClassifyConvergeError reports the live-agent hold where PlanSlot gives the unknown-agent one. After BR-10, reboot's first pass accepts that hold and the operator is told to reboot. The "no longer broken" refusal is still untyped and classified as a hand-off, though a rerun would converge. Rule: derive the converge-time hold reason from the same pure function PlanSlot's setAside closure uses, and give the evidence-changed refusal a typed retryable error.
  - id: BR-13
    disposition: addressed
    note: |
      run.go renders SlotReconcileError.Result's observation and plan on the error path. TestReconcileCLIShowsTheReportOnABlockingFailure pins it.
```
