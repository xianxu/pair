# Boundary Review — pair#250 (whole-issue close)

| field | value |
|-------|-------|
| issue | 250 — Recover stale Couch threads without losing live sessions or checkpoints |
| repo | pair |
| issue file | workshop/issues/000250-couch-stale-thread-recovery.md |
| boundary | whole-issue close |
| milestone | — |
| window | 2e4b2df75c3c89d912e8ee91e4999990f8f031d3..75962c63effc1e928799a90988ba32d5ddc3b78e |
| command | sdlc close --issue 250 |
| reviewer | codex |
| timestamp | 2026-09-14T18:16:09-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

Recovery reuses the existing launcher, preserves checkpoint bytes, and adds meaningful ownership tests. The reviewed package suites pass. Two gaps block shipping: archive now rejects never-bound threads, and some recovery observations discard caller deadlines.

## 1. Strengths

- Warm recovery preserves the surviving session without requiring native transcript binding.
- Source absence and target-generation witnesses preserve truthful retry authority without fabricating a park receipt.
- Archive retains embedded checkpoints and removes derived copies transactionally.
- README, atlas, portable acceptance fixtures, and scheduled Zellij conformance cover the new surface.

## 2. Critical findings

**Archive loses the escape for threads without a session binding.**
[detach.go:251](/Users/xianxu/workspace/pair/cmd/internal/couchcore/detach.go:251)

Every readable record now passes through `reconcileRecoveryHelper`. Its session observer rejects missing bindings—even for an empty thread whose launch failed before session publication. Previously, [QuiesceThreadSession](/Users/xianxu/workspace/pair/cmd/internal/launcher/thread_claim.go:257) explicitly allowed this case without signalling anything. Such rows now remain unarchivable.

Preserve a separately proved, never-bound archive path while continuing to reject malformed or ambiguous observations. Add a regression exercising `Couch.ArchiveThread` with no incarnations, transactions, or session-index entry. **ARCH-PURPOSE.**

## 3. Important findings

**Retry and generation observations discard caller deadlines.**
[continuation_recovery.go:269](/Users/xianxu/workspace/pair/cmd/internal/couchcore/continuation_recovery.go:269), [switchcontext.go:325](/Users/xianxu/workspace/pair/cmd/internal/couchcore/switchcontext.go:325)

Both reach non-context session observation, which uses `context.Background()`. A short-deadline recovery can wait for Zellij’s independent five-second timeout. This violates the plan’s caller-bounded observation contract.

Use context-aware observation throughout these paths and test cancellation during failed-request retry and generation lookup. Current deadline coverage exercises initial recovery observation. **ARCH-CONSTRAINTS.**

## 4. Minor findings

None.

## 5. Test coverage notes

- Full tests passed for checkpoint, readiness, threadrecord, couchcore, couchtty, couchcmd, launcher, and wrapcmd.
- Pinned diff whitespace checks passed.
- Live Zellij and interactive smoke were inspected, not rerun.
- Add the two regressions above; existing archive fixtures provide named session bindings.

## 6. Architectural notes

| Principle | Assessment |
|---|---|
| ARCH-DRY | Pass: shared reconciliation and continuation execution. |
| ARCH-PURE | Pass: decisions and request transitions separated from effects. |
| ARCH-PURPOSE | **Flag:** missing-binding archive escape regresses. |
| ARCH-MOCK | Pass: stateful seams and recurring live conformance exist. |
| ARCH-CONSTRAINTS | **Flag:** caller deadlines bypassed. |
| ARCH-SECURE | Pass: exact identities, bounded checkpoint parsing, conservative unknown handling. |
| ARCH-ORDER | Pass: queue, revision fences, attempt correlation, canceled-park coverage. |
| ARCH-FUNERAL | Pass: retained snapshots and derived-file cleanup reuse existing lifecycle. |

## 7. Plan revision recommendations

Append `## Revisions` entries defining never-bound archive eligibility and mapping the added cancellation regressions. Reconcile the pinned plan’s pending-test statements with committed evidence; keep operator smoke explicitly pending.

```findings
findings:
  - id: new
    severity: Critical
    family: archive-preserves-unbound-escape
    title: |
      Archive rejects empty threads that never published a session binding
    detail: |
      detach.go:251 unconditionally invokes recovery observation, whose missing-binding error prevents archiving a readable, unoccupied thread after pre-session launch failure. Preserve the existing never-bound, non-signalling archive contract and add a Couch.ArchiveThread regression without a session-index entry. ARCH-PURPOSE.
  - id: new
    severity: Important
    family: observations-preserve-caller-deadlines
    title: |
      Recovery retry and generation lookup discard caller deadlines
    detail: |
      continuation_recovery.go:269 and switchcontext.go:325 reach non-context session observation using context.Background, allowing a short-deadline operation to wait for Zellij's independent five-second timeout. Thread the caller context through both paths and test cancellation at these boundaries. ARCH-CONSTRAINTS.
```

---

## Re-review — 2026-09-14T18:24:45-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 250 — Recover stale Couch threads without losing live sessions or checkpoints |
| repo | pair |
| issue file | workshop/issues/000250-couch-stale-thread-recovery.md |
| boundary | whole-issue close |
| milestone | — |
| window | 2e4b2df75c3c89d912e8ee91e4999990f8f031d3..526ee2dfa4444ee9faa91cff9b2d4d54739c5c0a |
| command | sdlc close --issue 250 |
| reviewer | codex |
| timestamp | 2026-09-14T18:24:45-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

Both prior findings are addressed, and all eight affected package suites pass. The recovery flow preserves checkpoint bytes, checks exact ownership, and reuses existing launch machinery. One new architectural finding remains: a durable incarnation transition bypasses the state model.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      detach.go now permits revision-fenced, non-signalling archive only for an empty record with explicitly absent binding. TestCouchArchiveNeverBoundThreadDoesNotSignal exercises the previously rejected case; companion tests reject unreadable indexes, retained requests and concurrent changes.
  - id: BR-2
    disposition: addressed
    note: |
      Retry and final archive observation now preserve caller context; production readiness composition installs SessionContext. Cancellation regressions exercise retry, generation, registration and status reads, and passed.
findings:
  - id: new
    severity: Important
    family: lifecycle-transitions-through-owned-model
    title: |
      Settled unknown-target recovery bypasses the incarnation transition model
    detail: |
      cmd/internal/couchcore/continuation_recovery.go:87 directly assigns IncarnationLive inside UpdateExistingThread, committing an intermediate state before reconcileRecoveryHelper retires it. ARCH-ORDER requires this transition to belong to the pure model. Express exact-receipt reconciliation as a named transition with a revision-checked store operation; test interruption between reconciliation and attachment.
```

### 1. Strengths

- Absence authority remains distinct from verified park authority; request validation rejects conflicting witnesses.
- Generation admission checks exact request-owned targets before and after launch claims.
- Archive preserves embedded checkpoints and uses a final revision fence.
- README and atlas explain warm recovery versus a new checkpoint-seeded conversation.

### 2. Critical findings

None.

### 3. Important findings

**`continuation_recovery.go:87` — ARCH-ORDER:** The IO coordinator directly promotes durable `Unknown` state to `Live` so the shared reconciler accepts it. Unlike the adjacent `AdvanceStart` call, this transition has no named model event. Move its eligibility and resulting state into the pure lifecycle model, preserving exact receipt, identity and revision checks.

### 4. Minor findings

None.

### 5. Test coverage notes

Passed: `checkpoint`, `couchcore`, `couchtty`, `couchcmd`, `readiness`, `threadrecord`, `launcher`, and `wrapcmd`. Pinned `git diff --check` passed.

Prior regressions reach the corrected branches; mutation tests were not run. Live Zellij conformance and operator smoke were not rerun.

### 6. Architectural notes

- **ARCH-DRY — pass:** Shared reconciliation and continuation machinery.
- **ARCH-PURE — pass:** Recovery decisions and request validation are independently testable.
- **ARCH-PURPOSE — pass:** Warm recovery, checkpoint import and archive escape are delivered.
- **ARCH-MOCK — pass:** Stateful fixtures share production seams; recurring live conformance is configured.
- **ARCH-CONSTRAINTS — pass:** Bounded retries, observation deadlines and no additional inventory IO.
- **ARCH-SECURE — pass:** Exact identities and validated checkpoint snapshots govern effects.
- **ARCH-ORDER — flag:** Direct durable promotion described above.
- **ARCH-FUNERAL — pass:** Existing archive journal removes derived checkpoint files while retaining authoritative bytes.

### 7. Plan revision recommendations

Append a `## Revisions` entry naming the modeled unknown-target reconciliation transition, its store boundary, and interruption test. Keep operator acceptance explicitly pending.

---

## Re-review — 2026-09-14T18:35:46-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 250 — Recover stale Couch threads without losing live sessions or checkpoints |
| repo | pair |
| issue file | workshop/issues/000250-couch-stale-thread-recovery.md |
| boundary | whole-issue close |
| milestone | — |
| window | 2e4b2df75c3c89d912e8ee91e4999990f8f031d3..0e702a9d12781772075879d462bb766982b0aad6 |
| command | sdlc close --issue 250 |
| reviewer | codex |
| timestamp | 2026-09-14T18:35:46-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: medium
```

BR-3 is addressed: registered-target recovery now retires the unknown helper atomically through a named pure transition and revision-checked store operation. The affected tests pass, and no new blocking findings emerged. Operator smoke remains explicitly pending; this review does not establish that acceptance.

```findings
dispose:
  - id: BR-3
    disposition: addressed
    note: |
      ReconcileRegisteredTarget in starttransaction.go:249 replaces the synthetic Live intermediate state through ThreadStore.ReconcileRegisteredTarget. continuation_recovery_test.go:329 exercises interruption after retirement, same-attempt reattachment, lost receipt, revived helper, and revision conflict. Both new regression tests passed independently. The interruption assertion rejects the previous implementation's persisted Live intermediate state.
  - id: BR-1
    disposition: addressed
    note: |
      The narrow missing-binding archive escape remains guarded against unreadable indexes, retained requests, and concurrent replacement; the affected archive tests passed.
  - id: BR-2
    disposition: addressed
    note: |
      Production session and generation observations retain caller context; cancellation regressions and affected package tests passed.
```

### 1. Strengths

- Recovery distinguishes helper death from session survival and preserves warm attachment without native-binding requirements.
- Checkpoint recovery preserves exact bytes and rejects unrelated generation advancement before launching.
- BR-3 tests assert durable state after interruption and successful same-attempt recovery.
- README, atlas, and plan revisions describe the implemented behavior and fixture limitations.

### 2. Critical findings

None.

### 3. Important findings

None.

### 4. Minor findings

None.

### 5. Test coverage

Passed all eight affected package suites: `couchcore`, `couchcmd`, `couchtty`, `checkpoint`, `readiness`, `threadrecord`, `launcher`, and `wrapcmd`.

The two BR-3 regression tests also passed independently. `git diff --check` passed. Live Zellij conformance, operator smoke, and mutation testing were not rerun during this review.

### 6. Architectural notes

- **ARCH-DRY — Pass:** shared observation, retirement, and continuation execution paths.
- **ARCH-PURE — Pass:** value-only decisions/request transitions; BR-3’s transition tests require no IO.
- **ARCH-PURPOSE — Pass:** warm recovery, checkpoint recovery, and archive escape are implemented.
- **ARCH-MOCK — Pass:** stateful fixtures exercise production seams; recurring live conformance is wired.
- **ARCH-CONSTRAINTS — Pass:** bounded retries, checkpoint limits, cancellable observations, and no added inventory-refresh IO.
- **ARCH-SECURE — Pass:** exact identity/generation checks and visible refusal on uncertain evidence.
- **ARCH-ORDER — Pass:** BR-3 eliminates the intermediate state; interruption and conflicting revisions are tested.
- **ARCH-FUNERAL — Pass:** recovery reuses existing snapshot retention and materialized-file cleanup.

### 7. Plan revisions

None required. Preserve the unchecked operator-acceptance item until confirmation is obtained.
