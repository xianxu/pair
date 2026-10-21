# Boundary Review — pair#774 (whole-issue close)

| field | value |
|-------|-------|
| issue | 774 — diagnostic checkpoints survive device remounts |
| repo | pair |
| issue file | workshop/issues/000774-diagnostic-device-rebind.md |
| boundary | whole-issue close |
| milestone | — |
| window | 9404e4e27c3cdd6eda6d2974f2aad57439a18e2c..549f4c8a99e5a57c338ce3b7390b4cc57b5d223a |
| command | sdlc close --issue 774 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-21T09:17:27-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The current-checkpoint repair preserves payload bytes and stale reservations, and focused, race, and full diagnosticlog tests pass. One blocking gap remains: retained collection can accept a device change between observations within the same operation, contrary to the documented contract.

Done-when: current — its identity and fail-closed requirements cover the expanded retained-generation recovery; the plan makes those consumers explicit.

```findings
findings:
  - id: new
    severity: Critical
    kind: defect
    family: live-observations-require-strict-identity
    title: |
      Retained collection accepts device changes within one operation
    detail: |
      cmd/internal/diagnosticlog/collect.go:169 compares the second retained-file observation against persisted metadata, discarding the observation already made by previewPageLocked at pages.go:311. If the device changes between those observations while inode, size and mtime remain equal, collection rebinds metadata and can delete the file instead of refusing. Preserve the first observed identity and strictly compare subsequent observations before publishing metadata or deletion intent. Family enumeration: the retained preview-to-collection transition is affected; current collection already uses strict matches, and pending creation/rotation/deletion recovery remains strict. The corresponding unsupported claim is atlas/couch.md:1247. Add an integration regression exercising this transition; checkpoint_identity_test.go:279 only calls matches directly and cannot detect the faulty caller policy. ARCH-PURPOSE: the promised persisted-versus-live boundary is incomplete.
```

1. **Strengths**

   - The shared classifier checks inode, size and nanosecond timestamp before allowing device-only equivalence.
   - Current recovery durably publishes the rebound checkpoint before admission and preserves reservation bytes.
   - Tests cover payload preservation, uncertain tails, pending intents, unsafe paths and checkpoint-save failure.
   - README and atlas updates accompany the changed recovery behavior.

2. **Critical findings**

   - Retained collection loses the first observation at [collect.go:169](/Volumes/TestScratch/review-checkouts/858f812b6c5f0dab/cmd/internal/diagnosticlog/collect.go:169). Carry its observed identity through deletion preparation and require strict equality thereafter.

3. **Important findings**

   None.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Passed:

   - Focused device-rebind/classifier tests.
   - Focused tests under the race detector.
   - Full diagnosticlog package suite.
   - Range whitespace check.

   These results do not cover the retained observation transition identified above. Recorded verification and hub live acceptance remain pending in the tracker; neither is claimed verified here.

6. **Architectural notes**

   - **ARCH-DRY: pass** — consumers share the comparison policy.
   - **ARCH-PURE: pass** — the classifier is deterministic and directly tested; filesystem work stays in integration callers.
   - **ARCH-PURPOSE: flag** — retained collection incompletely enforces the promised strict boundary.
   - **ARCH-EVOLVE: pass** — no format or codec change; existing strict decoding remains unchanged.

7. **Plan revision recommendations**

   Add a `## Revisions` entry specifying that retained collection carries its first observed identity through subsequent checks, with a regression that fails when a device change between preview and deletion preparation is accepted.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"777d7973-2565-43f8-8213-d3a939e0a0b7","repository":"github.com/xianxu/pair","issue_id":"000774","recipe":"small-diff-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/858f812b6c5f0dab","reviewed_head":"549f4c8a99e5a57c338ce3b7390b4cc57b5d223a","slot":"pair:7","requested_at":"2026-10-21T16:15:31.956122Z"},"phase":"finished","pid":18292,"started_at":"2026-10-21T16:15:32.121411Z","finished_at":"2026-10-21T16:17:27.012426Z","session_id":"01a15f3f-16b8-7901-87a3-edb258ae0200","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.225030125s
head: "549f4c8a99e5a57c338ce3b7390b4cc57b5d223a"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-21T10:03:52-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 774 — diagnostic checkpoints survive device remounts |
| repo | pair |
| issue file | workshop/issues/000774-diagnostic-device-rebind.md |
| boundary | whole-issue close |
| milestone | — |
| window | 9404e4e27c3cdd6eda6d2974f2aad57439a18e2c..f05fa31829f93d977d1df4ab0a5d6f493d67dade |
| command | sdlc close --issue 774 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-21T10:03:52-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

BR-1 is addressed, and no new blocking findings emerged. Retained collection now preserves preview’s first observation and strictly revalidates it before metadata rebinding or deletion intent. The regression passes at the pinned head and fails when that guard is removed through a temporary Go overlay. Recorded verification and hub live-write acceptance remain separate, outstanding acceptance gates.

Done-when: current — its identity, byte-preservation and fail-closed requirements cover the retained-generation changes and conservative transaction recovery.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      pages.go:340 carries preview's observed generation into Segment; collect.go:165 strictly checks it before metadata rebinding or deletion intent. TestDeviceRebindCollectionKeepsFirstObservation passes at head and fails with the guard removed, reporting that collection deleted the replacement. The existing synthetic device-change test separately verifies strict device comparison.
```

1. **Strengths**

   - The shared classifier requires matching nonzero inode, size and nanosecond mtime before accepting device-only differences.
   - Current recovery durably publishes the rebound identity while preserving payload and stale reservation bytes.
   - The BR-1 regression exercises the production preview-to-deletion path through a controllable reader-proof callback.
   - README and atlas changes accurately describe the recovery boundary and conservative refusals.

2. **Critical findings**

   None.

3. **Important findings**

   None.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Passed:

   - Full `diagnosticlog` package suite.
   - Focused device-rebind and classifier tests.
   - Focused tests under the race detector.
   - BR-1 mutation check: removing the strict observation guard makes the regression fail.
   - Range whitespace check.

   The PURE classifier tests perform no IO; integration fixtures use disposable files. No live diagnostic store was touched. Recorded suite completion and hub restart/write confirmation were not verified here.

6. **Architectural notes**

   - **ARCH-DRY: pass** — persisted comparison is shared; live checks retain strict policy.
   - **ARCH-PURE: pass** — deterministic classification is directly tested; filesystem effects remain in integration callers.
   - **ARCH-PURPOSE: pass** — current and retained consumers are covered, including the previously missing observation transition.
   - **ARCH-MOCK: pass** — no new external binary or service dependency; filesystem fixtures exercise production boundaries.
   - **ARCH-CONSTRAINTS: pass** — fixed-size comparison and existing bounded traversal remain; rebind publications match the documented cost.
   - **ARCH-SECURE: pass** — unsafe-path and identity mismatch controls remain fail-closed; tests use temporary storage.
   - **ARCH-ORDER: pass** — the first observation survives across phases, with deterministic callback ordering testing refusal before deletion authority.
   - **ARCH-FUNERAL: pass** — no new durable artifact family; the observation snapshot dies with its page.
   - **ARCH-EVOLVE: pass** — no persisted format or codec change; the new snapshot is private and excluded from JSON.

7. **Plan revision recommendations**

   None. The BR-1 revision accurately describes the implemented snapshot, strict revalidation and regression coverage.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"e5a15d10-b6f4-44a6-a8cc-a560ef7a81e9","repository":"github.com/xianxu/pair","issue_id":"000774","recipe":"milestone-review","round":2,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/c09fb9572dd3024b","reviewed_head":"f05fa31829f93d977d1df4ab0a5d6f493d67dade","slot":"pair:7","requested_at":"2026-10-21T16:41:01.586464Z"},"phase":"finished","pid":14189,"started_at":"2026-10-21T16:41:01.685375Z","finished_at":"2026-10-21T17:03:51.781726Z","session_id":"01a15f56-6d09-72a0-ab84-c634f819ac33","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.043838542s
head: "f05fa31829f93d977d1df4ab0a5d6f493d67dade"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
