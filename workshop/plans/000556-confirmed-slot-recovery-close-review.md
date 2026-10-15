# Boundary Review — pair#556 (whole-issue close)

| field | value |
|-------|-------|
| issue | 556 — Couch confirmed disk recovery and always-reachable fresh startup |
| repo | pair |
| issue file | workshop/issues/000556-confirmed-slot-recovery.md |
| boundary | whole-issue close |
| milestone | — |
| window | f3cd4848a3fc5924e81635da9ddd0f835a83ea72..6bba5141ed4966662bd2d966188acb14bae0cf3f |
| command | sdlc close --issue 556 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-15T11:43:12-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The recovery executor, confirmation flow, and admission lifecycle have substantial coverage, and the existing checks pass. BR-6 remains incomplete: a full canonical Codex UUID after `--` bypasses admission through the indirect-selector compatibility exception. Both wrapper entry forms reproduced the bypass.

1. **Strengths**

   - Recovery preserves evidence before retirement and retains uncertain launches without destructive rollback.
   - Confirmed retirement independently checks that every accepted process disappeared.
   - Admission fencing and claiming share one lock, with deterministic tests for both lock orderings.
   - README and atlas document the confirmation surface, Fresh behavior, and approved Codex exception.

2. **Critical findings**

   **BR-6 — not-addressed:** [native_resume_selector.go:47](/Users/xianxu/Library/Caches/sdlc/review-checkouts/78f82bba5b2e9485/cmd/internal/launcher/native_resume_selector.go:47) truncates parsing at `--`. Consequently, `codex resume -- 019d1111-1111-7111-8111-111111111111` becomes `NativeResumePicker`, although Codex’s grammar assigns that positional argument to `SESSION_ID`. The compatibility policy then permits an unreserved launch at [native_admission.go:35](/Users/xianxu/Library/Caches/sdlc/review-checkouts/78f82bba5b2e9485/cmd/internal/wrapcmd/native_admission.go:35).

   Temporary overlay regressions confirmed incorrect classification for `resume` and `exec resume`. Production wrapper tests confirmed synthetic children spawned without admission through both argv and `--from-launch-env`.

   **This is the 2nd finding in family `existing-session-selectors-require-admission`.** Earlier rounds fixed instances. State and enforce the rule across native argument grammars: ending option parsing does not end positional identity parsing. Sweep command, option, delimiter, and positional forms across supported agents, then test the shared launcher and both wrapper transports. **ARCH-PURPOSE, ARCH-SECURE.**

3. **Important findings**

   None additional.

4. **Minor findings**

   None.

5. **Test coverage notes**

   - Passed full tests for launcher, wrapcmd, sessioninventory, threadrecord, and artifactpath.
   - Passed selected recovery tests across couchcore, couchcmd, couchtty, and couchmessage.
   - Passed `go vet` for couchcore, launcher, and wrapcmd; range whitespace check also passed.
   - New temporary reproductions failed as described above. Overlay files were removed; the checkout remains unchanged.
   - I did not rerun the repository-wide suite or live operator acceptance.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared recovery executor and selector policy serve their callers.
   - **ARCH-PURE — pass:** decisions and transitions have injected IO boundaries.
   - **ARCH-PURPOSE — flag:** BR-6 leaves canonical-ID admission incomplete.
   - **ARCH-MOCK — pass:** stateful process/store doubles exercise production seams; scratch conformance is documented.
   - **ARCH-CONSTRAINTS — pass:** observation budgets and admission count/byte limits are explicit.
   - **ARCH-SECURE — flag:** BR-6 misclassifies external argv and grants the compatibility bypass.
   - **ARCH-ORDER — pass:** reservation transitions serialize under one lock and preserve uncertain outcomes.
   - **ARCH-FUNERAL — pass:** confirmations expire, admission storage is bounded, and reclamation requires appropriate evidence.

7. **Plan revision recommendations**

   Add a `## Revisions` entry defining native positional-selector handling around `--`, enumerating the affected command forms and transports, and requiring regression coverage that distinguishes session IDs from prompts.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Fresh selects read-only workspace readiness while owner absence is unproved; survivor and absent-source control tests pass.
  - id: BR-2
    disposition: addressed
    note: |
      Confirmed retirement signals leaf-first and independently observes every accepted owner absent; survivor, replacement, false-success, and final-listing tests pass.
  - id: BR-3
    disposition: addressed
    note: |
      README recovery passages document standalone confirmation, token lifetime, bounded Starting, and survivor-safe Fresh, matching the executor and confirmation owner; atlas changes cover the same flow.
  - id: BR-4
    disposition: addressed
    note: |
      Profile resolution passes preview stores through the original Couch owner rather than copying Couch; profile tests and go vet pass.
  - id: BR-5
    disposition: addressed
    note: |
      Keyed Fresh resolves before native-only profile and lifecycle guards; the explicit, candidate, and operation regression matrix passes.
  - id: BR-6
    disposition: not-addressed
    note: |
      ARCH-PURPOSE and ARCH-SECURE: native_resume_selector.go:47 drops positional identity parsing after --. Codex resume -- <full UUID> becomes a picker and enters the unguarded compatibility exception. Temporary regressions fail for resume and exec resume; both production wrapper transports spawn synthetic children without admission. This is the 2nd finding in family existing-session-selectors-require-admission: enforce native positional grammar across the complete selector enumeration rather than patching only this spelling.
  - id: BR-7
    disposition: addressed
    note: |
      Reserve fences exact dead-helper reserved grants under the Claim lock; both lock-order tests, stale-grant rejection, and live/unknown/spawning retention controls pass.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"4baea88d-7fad-4c12-af47-4ab9d680d06b","repository":"github.com/xianxu/pair","issue_id":"000556","recipe":"milestone-review","round":7,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/78f82bba5b2e9485","reviewed_head":"6bba5141ed4966662bd2d966188acb14bae0cf3f","slot":"pair:2","requested_at":"2026-10-15T18:38:59.003709Z"},"phase":"finished","pid":9743,"started_at":"2026-10-15T18:38:59.200567Z","finished_at":"2026-10-15T18:43:10.739264Z","session_id":"01a140dc-43d7-7461-ba55-4e00fdedd1e4","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.894350292s
head: "6bba5141ed4966662bd2d966188acb14bae0cf3f"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
