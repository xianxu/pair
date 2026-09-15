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
