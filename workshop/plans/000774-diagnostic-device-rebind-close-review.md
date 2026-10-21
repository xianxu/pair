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
