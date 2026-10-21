# Boundary Review — pair#733 (whole-issue close)

| field | value |
|-------|-------|
| issue | 733 — couch: notifications are ephemeral: a restart starts with an empty list, dismiss and clear-all remove items, old items age out |
| repo | pair |
| issue file | workshop/issues/000733-couch-notifications-are-ephemeral-a-restart-starts-with-an-empty-list-dismiss-and-clear-all-remove-items-old-items-age-out.md |
| boundary | whole-issue close |
| milestone | — |
| window | 80fe2ecd13f3b64c0bf57ca32afbc8e9ce31ba54..d23f89c948bc96012025b918e93dc153f9ba6cb2 |
| command | sdlc close --issue 733 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-20T17:08:09-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The implementation matches the revised Spec: notifications are ephemeral, removal suppresses current live conditions until resolution, and notification events expire after 12 hours. No blocking findings. One atlas summary still describes the former lifecycle.

**1. Strengths**

- `AttentionList` owns suppression, expiry, selection and clearing through pure transitions with private state.
- The startup regression exercises real service composition against a legacy file and verifies that its bytes remain unchanged.
- Terminal tests cover clearing an overflowing panel by keyboard and mouse, plus expiry without another source event.

**2. Critical findings**

None.

**3. Important findings**

None.

**4. Minor findings**

`atlas/index.md:16` still calls this the “durable attention list.” Change that description to ephemeral, consistent with the updated README and detailed atlas pages.

**5. Test coverage notes**

- Attention tests passed across `couchcore`, `couchtty` and `couchcmd`, normally and under the race detector.
- Full affected-package suites passed. The initial overlapping run exceeded native write budgets in `couchcore` and `couchtty`; both passed an isolated rerun. The pinned base comparison also passed.
- `git diff --check` passed; the checkout remains unchanged.
- The operator’s live restart check remains deferred until the next restart, as documented in the issue revision.

**6. Architectural notes**

| Principle | Assessment |
|---|---|
| ARCH-DRY | Pass: shared reducer transitions serve all controls. |
| ARCH-PURE | Pass: deterministic core; clock and UI stay at the boundary. |
| ARCH-PURPOSE | Pass for behavior; Minor documentation inconsistency noted above. |
| ARCH-MOCK | Pass: existing stateful host/child fakes exercise production paths. |
| ARCH-CONSTRAINTS | Pass: notification bounds remain; expiry reuses the existing tick. |
| ARCH-SECURE | Pass: legacy persisted input is neither decoded nor rewritten. |
| ARCH-ORDER | Pass: private authoritative state and sequence tests enforce suppression and recurrence. |
| ARCH-FUNERAL | Pass: expiry/removal end notification lifetimes; source resolution removes suppression; saver worker removed. |
| ARCH-EVOLVE | Pass: no persisted format rewrite; legacy bytes remain untouched. |

**7. Plan revision recommendations**

None. The implementation matches the revised plan.

```findings
findings:
  - id: new
    severity: Minor
    kind: defect
    family: documentation-contract-consistency
    title: |
      Atlas index still describes the attention list as durable
    detail: |
      atlas/index.md:16 retains “the durable attention list,” contradicting the ephemeral lifecycle implemented here and documented in README.md and atlas/couch.md. Update the index summary to match the revised contract (ARCH-PURPOSE).
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"ac3d9f1c-61f5-4a90-a8a4-decf0c5fccfe","repository":"github.com/xianxu/pair","issue_id":"000733","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/596fbfc3052b8ada","reviewed_head":"d23f89c948bc96012025b918e93dc153f9ba6cb2","slot":"pair:4","requested_at":"2026-10-20T23:56:51.632504Z"},"phase":"finished","pid":67600,"started_at":"2026-10-20T23:56:51.76425Z","finished_at":"2026-10-21T00:08:08.606496Z","session_id":"01a15bbf-1630-7e61-9b9a-ecf49ecb0966","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.7331905s
head: "d23f89c948bc96012025b918e93dc153f9ba6cb2"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000410
