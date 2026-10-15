# Boundary Review — pair#562 (whole-issue close)

| field | value |
|-------|-------|
| issue | 562 — Zellij CLI queries can finish before their reply |
| repo | pair |
| issue file | workshop/issues/000562-zellij-empty-reply-race.md |
| boundary | whole-issue close |
| milestone | — |
| window | 965fcb8147cbb4f63a43d867a571cff7afa140e0..77145de9091256e8390296f2fe10cf127bc2679c |
| command | sdlc close --issue 562 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-15T01:52:06-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The empty-reply boundary and atomic fixture publication are well covered, including counterfactual failures. Shipping is blocked by a remaining name-probe path that converts failed observations into acceptance, plus acceptance criteria that still promise a protocol regression beyond the delivered tests.

Done-when: stale — the first criterion requires deterministic query and ConnStatus protocol regressions; the revision delivers a Pair boundary fake and proposed upstream scenarios.

```findings
findings:
  - id: new
    severity: Critical
    kind: defect
    family: observation-errors-must-preserve-uncertainty
    title: |
      Failed name probes with diagnostics still teach positive acceptance
    detail: |
      cmd/internal/launcher/osruntime.go:145-151 returns nil when cmd.Run fails with nonempty output other than the recognized length refusal. A temporary regression using stderr "failed to connect to server" and exit 2 fails because ProbeSessionName accepts the name; sessionNameAcceptor can then generalize that false evidence to shorter names. Propagate unrecognized failures, including cancellation with partial output, and accept only established validator evidence. Add subprocess-level regressions covering these outcomes and their monotone-cache consumers. This leaves the revised uncertainty sweep incomplete (ARCH-PURPOSE, ARCH-ORDER, ARCH-SECURE).
  - id: new
    severity: Important
    kind: defect
    family: acceptance-criteria-must-match-delivery
    title: |
      Done-when promises protocol regressions absent from the delivered scope
    detail: |
      workshop/issues/000562-zellij-empty-reply-race.md:52 requires deterministic protocol evidence for both early query completion and ConnStatus exclusion, while line 59 marks reproduction complete. The committed helper at cmd/internal/launcher/zellij_empty_reply_test.go:116-124 implements its own Log/UnblockInput behavior and contains no ConnStatus/Connected model; the upstream draft lists receiver tests as requested scenarios. Reconcile Done-when and the completed Plan checkbox with the revised boundary-testing scope and explicit upstream dependency, or deliver the promised protocol regressions (ARCH-PURPOSE).
```

1. **Strengths**

   - The shared `ErrZellijNoReply` guard preserves the explicit observed-empty inventory exception.
   - Sequence tests verify no automatic retry and recovery through a separately requested observation.
   - The live PID/pane fixture uses the same publisher exercised by the deterministic FIFO-barrier test.
   - The atlas documents uncertainty and the limitation of nonempty incomplete inventories.

2. **Critical findings**

   - [osruntime.go:145](/Users/xianxu/Library/Caches/sdlc/review-checkouts/8503ec25161e4365/cmd/internal/launcher/osruntime.go:145): preserve failed probe outcomes instead of returning acceptance. Retain any necessary recognized no-session validator response explicitly.

3. **Important findings**

   - [Issue:52](/Users/xianxu/Library/Caches/sdlc/review-checkouts/8503ec25161e4365/workshop/issues/000562-zellij-empty-reply-race.md:52): align acceptance criteria and completed checkboxes with the actual evidence.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Focused tests passed across launcher, couchcore, and pairlifecycletest. The broader selected tests passed twice under the race detector.

   Temporary overlays confirmed that removing the central guard breaks snapshot regressions and restoring direct publication breaks both fixture-value cases. The added failed-name-probe regression fails on pinned HEAD. Temporary files were removed; repository files remain unchanged. Full-suite and live CI results were not independently rerun.

6. **Architectural notes**

   - **ARCH-DRY: pass** — shared query boundary and client parser.
   - **ARCH-PURE: pass** — small IO guards; ownership IO remains injected.
   - **ARCH-PURPOSE: flag** — incomplete uncertainty sweep and acceptance mismatch.
   - **ARCH-MOCK: pass** — stateful executable fake uses the production boundary; live conformance lane exists.
   - **ARCH-CONSTRAINTS: pass** — bounded queries; no added retries or fan-out.
   - **ARCH-SECURE: flag** — failed external evidence still becomes name acceptance.
   - **ARCH-ORDER: flag** — that false acceptance contaminates subsequent monotone decisions.
   - **ARCH-FUNERAL: pass** — no new production durable artifacts; fixtures use temporary-directory cleanup.

   Atlas coverage is present. No new command, flag, or configuration surface requires a README addition. No Core concepts table is present.

7. **Plan revision recommendations**

   Add a `## Revisions` entry distinguishing delivered Pair boundary regressions from outstanding upstream receiver regressions, and update Done-when accordingly. Extend the probe outcome enumeration to cover nonzero diagnostic failures and cancellation with partial output.

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.190809833s
head: "77145de9091256e8390296f2fe10cf127bc2679c"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
