# Boundary Review — pair#556 (milestone M2)

| field | value |
|-------|-------|
| issue | 556 — Couch confirmed disk recovery and always-reachable fresh startup |
| repo | pair |
| issue file | workshop/issues/000556-confirmed-slot-recovery.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | c0ccda6133c4420b84e306d415f86ebb10ce9680..fe5275365b340df9a3f6d85b1b500625d30497fb |
| command | sdlc milestone-close --issue 556 --milestone M2 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-15T10:51:13-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The admission protocol correctly serializes canonical-ID resumes and preserves ownership through uncertain publication. Focused and full affected-package tests passed. Two gaps remain: existing-session selectors can bypass admission, and interrupted reserved grants lack the promised fenced recovery path.

1. **Strengths**

   - The pure reducer enforces single consumption and exact attempt matching; the store applies transitions under a shared lock.
   - The wrapper claims before PTY spawn and publishes the child generation before readiness.
   - Stateful tests cover surviving children, PID reuse, replay, publication uncertainty and competing slots.
   - README and atlas document the new surface and the approved Codex compatibility exception.

2. **Critical findings**

   **Existing-session selectors bypass admission.** In `cmd/internal/launcher/native_resume_selector.go:72`, `--teleport` merely sets `Remote`; `--from-pr` receives no classification. Both remain `NativeResumeNone`. Consequently, `cmd/internal/wrapcmd/native_admission.go:28–33` starts them without a grant. Installed `claude --help` confirms both can resume existing sessions.

   Classify every existing-session selector explicitly. Unsupported mappings must refuse before spawn; the approved Codex exception must remain narrowly scoped. Sweep the existing Fresh selector authority for sibling omissions and test both wrapper entry forms.

3. **Important findings**

   **Reserved grants cannot recover after confirmed fencing.** `cmd/internal/launcher/native_admission_os.go:257` skips every non-running entry during collection. The only reserved-entry removal requires the original helper identity (`native_admission.go:167–171`); no retirement/fencing integration provides another transition.

   If the helper dies after reservation but before wrapper claim, the native ID remains blocked even after the pending launch is fenced and its tree proved absent. Repetition consumes the global capacity. Implement exact-attempt fencing and reclamation under the store lock, rejecting any late grant afterward.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Passed focused tests and full tests for `launcher`, `wrapcmd`, `sessioninventory`, `artifactpath` and `termcmd`; affected-package `go vet` and range `git diff --check` also passed. The checkout remained unchanged. I did not run the repository-wide suite or live lifecycle acceptance.

   Missing regressions are selector-bypass refusal and the sequence: reserve → helper death → confirmed fencing/absence → replacement admission → rejection of the old grant.

6. **Architectural notes for upcoming work**

   - **ARCH-DRY — flag:** selector classification omits context selectors already identified by Fresh validation.
   - **ARCH-PURE — pass:** reservation decisions are pure; filesystem/process effects use injected seams.
   - **ARCH-PURPOSE — flag:** admission does not cover every existing-session launch outside the approved exception.
   - **ARCH-MOCK — pass:** stateful doubles exercise production boundaries.
   - **ARCH-CONSTRAINTS — pass:** explicit count, byte and operation bounds fail visibly.
   - **ARCH-SECURE — flag:** unclassified external argv gains permission to spawn.
   - **ARCH-ORDER — flag:** the promised fenced-reservation transition is absent.
   - **ARCH-FUNERAL — flag:** interrupted reserved grants have no recovery lifecycle after their helper disappears.

   The M2 pure reservation entity exists at its declared path and is tested without IO; store and wrapper integration tests use portable stateful seams.

7. **Plan revision recommendations**

   Add `## Revisions` entries enumerating all existing-session selector families and specifying the exact reserved-attempt fencing/reclamation transition, including its retirement caller and late-grant invariant. Preserve the deliberate retention policy for genuinely unresolved spawning outcomes.

```findings
findings:
  - id: new
    severity: Critical
    kind: defect
    family: existing-session-selectors-require-admission
    title: |
      Classify every existing-session selector before permitting native spawn
    detail: |
      cmd/internal/launcher/native_resume_selector.go:72 marks Claude --teleport as Remote without selecting a resume kind, and --from-pr is unclassified; both remain NativeResumeNone and reach start(cmd) without admission at cmd/internal/wrapcmd/native_admission.go:28–33. Installed claude --help confirms both resume existing sessions. Enumerate existing-session selectors using the existing Fresh authority, refuse unsupported mappings before spawn, and add regressions for both wrapper entry forms. ARCH-DRY, ARCH-PURPOSE, ARCH-SECURE.
  - id: new
    severity: Important
    kind: defect
    family: fenced-admissions-must-be-reclaimable
    title: |
      Provide the promised fenced recovery path for interrupted reserved grants
    detail: |
      cmd/internal/launcher/native_admission_os.go:257 excludes reserved entries from collection, while native_admission.go:167–171 permits cancellation only by the original helper; no retirement integration supplies a fencing/removal transition. A helper dying after reservation permanently blocks that native ID even after its pending launch is fenced and its tree proved absent, contrary to the plan at lines 90 and 223–235, and repeated failures consume global capacity. Add exact-attempt fencing and reclamation under the store lock with a sequence test proving replacement admission succeeds and the old grant cannot execute. ARCH-ORDER, ARCH-FUNERAL.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"c262336d-f2ad-4642-99cd-a177748fab4a","repository":"github.com/xianxu/pair","issue_id":"000556","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/25a87b5a1d4049fb","reviewed_head":"fe5275365b340df9a3f6d85b1b500625d30497fb","slot":"pair:2","requested_at":"2026-10-15T17:46:59.430908Z"},"phase":"finished","pid":82130,"started_at":"2026-10-15T17:46:59.761281Z","finished_at":"2026-10-15T17:51:12.801204Z","session_id":"01a140ac-ab14-73e3-8b9d-b70aa6eb0f2d","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.795785125s
head: "fe5275365b340df9a3f6d85b1b500625d30497fb"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-15T11:36:39-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 556 — Couch confirmed disk recovery and always-reachable fresh startup |
| repo | pair |
| issue file | workshop/issues/000556-confirmed-slot-recovery.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | c0ccda6133c4420b84e306d415f86ebb10ce9680..451da7b2d3a7c062ead312ee8ef32d3a31e991e9 |
| command | sdlc milestone-close --issue 556 --milestone M2 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-15T11:36:39-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned M2 implementation addresses both prior findings, with regression tests that fail when each fix is removed. Admission, selector normalization, ownership retention, and documentation match the revised plan, including the approved Codex compatibility exception. No new blocking findings.

```findings
dispose:
  - id: BR-6
    disposition: addressed
    note: |
      native_resume_selector.go:163 reuses Fresh's forbidden-selector authority. Removing this guard makes selector tests and real-wrapper tests for --teleport and --from-pr fail in both argv and encoded-command forms.
  - id: BR-7
    disposition: addressed
    note: |
      native_admission_os.go:258 fences exact dead-helper reserved grants under the Claim lock. Removing reclamation makes replacement-admission and controlled lock-order tests fail; consumed spawning entries remain retained.
```

1. **Strengths**

   - `native_admission.go:123` centralizes authoritative state transitions and preserves uncertain spawning outcomes.
   - `native_admission_os_test.go:389` deterministically exercises both Claim/reclamation lock orderings.
   - `native_selector_boundary_test.go:29` verifies both actual wrapper entry forms.
   - README and Atlas document guarded launches, reservation lifetime, unsupported selectors, and the Codex exception.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage**

   All five affected packages passed: launcher, wrapcmd, sessioninventory, termcmd, and artifactpath. Focused launcher admission/selector tests passed under race detection; affected-package `go vet` and range `git diff --check` passed. Both fix-removal experiments produced meaningful regression failures. Disposable checkouts were removed; the reviewed checkout remains clean. I did not rerun the repository-wide suite or operator acceptance.

6. **Architecture**

   - **ARCH-DRY — pass:** selector policy reuses existing authorities.
   - **ARCH-PURE — pass:** deterministic transitions are separated from store/process effects.
   - **ARCH-PURPOSE — pass:** shared admission reaches launcher and wrapper execution boundaries.
   - **ARCH-MOCK — pass:** stateful doubles exercise production seams.
   - **ARCH-CONSTRAINTS — pass:** store size, entry count, scans, and operation deadlines are bounded.
   - **ARCH-SECURE — pass:** strict decoding, process generations, command digests, and capability scrubbing guard boundaries.
   - **ARCH-ORDER — pass:** explicit phases and controlled interleavings preserve uncertainty.
   - **ARCH-FUNERAL — pass:** exact reclamation handles reserved/running entries; uncertain spawning retention has an explicit capacity bound.

7. **Plan revision recommendations:** None.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"8993afa2-605d-436e-ac2a-a58154e6ba97","repository":"github.com/xianxu/pair","issue_id":"000556","recipe":"milestone-review","round":2,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/cafcc7ec02f27c00","reviewed_head":"451da7b2d3a7c062ead312ee8ef32d3a31e991e9","slot":"pair:2","requested_at":"2026-10-15T18:33:24.122137Z"},"phase":"finished","pid":75006,"started_at":"2026-10-15T18:33:24.377152Z","finished_at":"2026-10-15T18:36:38.189937Z","session_id":"01a140d7-285a-7170-b3d7-88696ec61a73","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.399469916s
head: "451da7b2d3a7c062ead312ee8ef32d3a31e991e9"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
