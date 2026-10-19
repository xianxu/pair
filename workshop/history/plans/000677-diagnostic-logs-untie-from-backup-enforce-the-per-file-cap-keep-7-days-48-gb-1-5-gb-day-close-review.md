# Boundary Review — pair#677 (whole-issue close)

| field | value |
|-------|-------|
| issue | 677 — diagnostic logs: untie from backup, enforce the per-file cap, keep ~7 days (48 GB, +1.5 GB/day) |
| repo | pair |
| issue file | workshop/issues/000677-diagnostic-logs-untie-from-backup-enforce-the-per-file-cap-keep-7-days-48-gb-1-5-gb-day.md |
| boundary | whole-issue close |
| milestone | — |
| window | 9950fa540423ab382e682437fc0564fadc104f29..687e59af511ceca9516f0755171c2876277d5de2 |
| command | sdlc close --issue 677 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-18T19:11:07-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The shared retention policy and diagnostic-only scope are well covered, and the saved live evidence supports removal of 18 payloads totaling 12,734,567,104 bytes. Shipping is blocked by incomplete reader protection during deletion recovery, scheduled traversal stopping at stale declarations, inaccurate apply reporting, and a failing artifact-classification guard.

```findings
findings:
  - id: new
    severity: Critical
    family: exact-target-deletion-proof
    title: |
      Writer-side deletion replay does not inspect the segment being deleted
    detail: |
      cmd/internal/diagnosticlog/collect.go:192 removes the backup fence from recoverDeletion, but writer.go:231-236 still authorizes replay using proof of the current path only; Write and maintain have the same pattern. After interruption at delete-intent, a reader can open the closed segment and a subsequent writer reopen can unlink it without inspecting that holder. Enforce exact-target proof across every recovery caller, including ordinary writers without ReaderProof, and add held-segment replay regressions. ARCH-ORDER, ARCH-SECURE.
  - id: new
    severity: Critical
    family: diagnostic-traversal-blocker-isolation
    title: |
      A missing-parent declaration terminates scheduled diagnostic collection
    detail: |
      registry.go:150 now permits structurally valid missing-parent declarations, but cmd/internal/gcruntime/schedule.go:90-97 propagates their canonicalization error. storagegc.ScheduleWorker terminates on that error and preserves its prior cursor, so subsequent eligible paths remain unreachable across restarts. Isolate expected per-path blockers, preserve them visibly, and advance the bounded cursor without recreating or deleting the stale declaration. Test a stale declaration followed by an eligible path through the scheduler. ARCH-PURPOSE, ARCH-FUNERAL.
  - id: new
    severity: Critical
    family: core-concept-contract-accuracy
    title: |
      The Core concepts tables contradict the implemented boundaries and locations
    detail: |
      workshop/plans/000677-diagnostic-log-retention-plan.md:34 classifies diagnostic-only CLI scope as PURE, although runWithService performs service/filesystem IO and its tests require temporary directories and injected services. Line 43 claims modified measurement tests in wrap_events_test.go and diagnostic_buffer_test.go, neither changed in this range; the new implementation is diagnostic_turn_budget_test.go. Promote CLI scope to INTEGRATION and correct the measurement location/status in a Revisions entry. ARCH-PURE.
  - id: new
    severity: Important
    family: diagnostic-result-completeness
    title: |
      Apply omits managed blockers and drops completed work when a page errors
    detail: |
      cmd/internal/diagnosticlog/collect.go:145-187 skips noneligible managed rows and returns only removals, so ApplyDiagnostics can report no blockers while managed payloads remain blocked. Separately, gcruntime/runtime.go:314-317 discards rows returned alongside an error, losing already-completed deletions from byte accounting. Preserve retained/blocked results and completed effects independently of errors; test mixed managed rows and failure after one completed deletion. ARCH-PURPOSE, ARCH-ORDER.
  - id: new
    severity: Important
    family: artifact-source-classification
    title: |
      The required artifact ownership guard fails on both changed source surfaces
    detail: |
      go test ./cmd/internal/artifactpath fails TestProductionArtifactReferencesAreExactlyClassified: diagnosticlog/proof.go remains a NonArtifactSource despite its new artifactpath import, and diagnosticlog/sweep.go is absent from the exhaustive inventory. Update cmd/internal/artifactpath/manifest.go with accurate source classifications and rerun the guard; do not expand the known-violations allowance. ARCH-PURPOSE.
```

**1. Strengths**

- One pure age/size selector serves managed and legacy generations; boundary tests cover seven-day expiry and young oversized files.
- Stateful diagnostic-only tests verify fresh preview/apply observations and zero session-clock effects.
- Live evidence hashes match their manifest. All 18 removed payload identities are absent from the after-inventory, supporting the recorded byte total.

**2. Critical findings**

The three Critical findings above require exact-target recovery protection, scheduled progress past stale declarations, and corrected Core concepts tables.

**3. Important findings**

Apply reporting needs complete blocker and partial-progress accounting. The artifact source inventory also needs updating before its existing guard passes.

**4. Minor findings**

None.

**5. Test coverage notes**

Passed: diagnosticlog, gcruntime, gccmd and wrapcmd suites. Artifactpath failed with the two classification violations above. `git diff --check` passed.

Missing regressions cover held-segment recovery through writer entry points, scheduled traversal past stale declarations, managed blocker reporting, and partial-page accounting.

**6. Architectural notes**

- **ARCH-DRY — pass:** shared eligibility policy and existing deletion journal reused.
- **ARCH-PURE — flag:** CLI classification contradicts its IO boundary.
- **ARCH-PURPOSE — flag:** scheduled progress and complete reporting remain incomplete.
- **ARCH-MOCK — pass:** portable storage, stateful inspection doubles and native conformance coverage exist.
- **ARCH-CONSTRAINTS — pass:** bounded pages/output and counted workload tests are present.
- **ARCH-SECURE — flag:** replay checks the wrong payload’s holders.
- **ARCH-ORDER — flag:** recovery authorization and partial-progress reporting have gaps.
- **ARCH-FUNERAL — flag:** stale declarations can prevent automatic cleanup of later logs.

README and atlas updates cover the new configuration, CLI scope and retention behavior.

**7. Plan revision recommendations**

Add `## Revisions` entries correcting the concept classifications and measurement path, enumerating every deletion-recovery caller, defining scheduled blocker progression, and requiring complete apply results across partial failures.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"98845410-e886-4819-8666-cf53bf2967ba","repository":"github.com/xianxu/pair","issue_id":"000677","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/1326dc8662ea9d21","reviewed_head":"687e59af511ceca9516f0755171c2876277d5de2","slot":"pair:3","requested_at":"2026-10-19T02:07:23.288681Z"},"phase":"finished","pid":59117,"started_at":"2026-10-19T02:07:23.585476Z","finished_at":"2026-10-19T02:11:05.675623Z","session_id":"01a151e9-df70-7791-b337-aa02e2d8a952","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.326571917s
head: "687e59af511ceca9516f0755171c2876277d5de2"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-18T19:23:36-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 677 — diagnostic logs: untie from backup, enforce the per-file cap, keep ~7 days (48 GB, +1.5 GB/day) |
| repo | pair |
| issue file | workshop/issues/000677-diagnostic-logs-untie-from-backup-enforce-the-per-file-cap-keep-7-days-48-gb-1-5-gb-day.md |
| boundary | whole-issue close |
| milestone | — |
| window | 9950fa540423ab382e682437fc0564fadc104f29..8238fd8577e5a9244600bdbaa9913e90173bf74a |
| command | sdlc close --issue 677 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-18T19:23:36-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The retention policy, diagnostic-only scope, exact-target replay protection, and reporting fixes are supported by passing tests. All five prior corrections are verified. One related blocking defect remains: other per-path diagnostic failures still terminate scheduled collection and strand later eligible logs.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      collect.go:209 now proves the exact deletion target in shared recoverDeletion. The regression covers Open, Write, maintain and direct recovery; all four fail when the correction is removed through a temporary overlay.
  - id: BR-2
    disposition: addressed
    note: |
      The real-scheduler missing-parent regression passes and fails without the schedule.go correction. It verifies later collection, unchanged declaration bytes, no parent recreation and visible preview blocking. A remaining sibling in the broader isolation family is reported below.
  - id: BR-3
    disposition: addressed
    note: |
      The plan's Core concepts table now classifies CLI scope under Integration point at line 41 and identifies diagnostic_turn_budget_test.go as a new test at line 42. Both match the implementation; the Revisions passage records the correction.
  - id: BR-4
    disposition: addressed
    note: |
      ObserveCollection preserves managed retained/blocked decisions, and runtime.go:323 appends completed rows before propagating errors. Both mixed-result and second-deletion-fault regressions pass at head and fail with the prior implementations overlaid.
  - id: BR-5
    disposition: addressed
    note: |
      manifest.go classifies proof.go as a resolved consumer and sweep.go as a non-artifact source. The ownership guard passes at head and fails with the previous manifest; the known-violations allowance is unchanged.
findings:
  - id: new
    severity: Critical
    kind: defect
    family: diagnostic-traversal-blocker-isolation
    title: |
      Other per-path diagnostic blockers still terminate scheduled collection
    detail: |
      cmd/internal/gcruntime/schedule.go:96-110 isolates only missing registered paths. A temporary regression using reportingFixture, an external append to its managed current log, and a later expired wrap-events-z-good.jsonl fails on scheduler batch 2 with "diagnostic reservation identity or size changed"; the later log remains uncollected. storagegc/schedule.go:117-118 propagates that error, preserving the cursor, and its worker exits. This is the 2nd finding in family diagnostic-traversal-blocker-isolation: state and enforce the rule across all path-local certificate/recovery blockers, retaining them visibly while advancing past that path. Enumerate and test the class rather than adding another message-specific exception; preserve cancellation, busy handling and global inventory failures. ARCH-PURPOSE, ARCH-FUNERAL.
```

1. **Strengths**

   - Shared recovery now checks the actual payload before deletion effects: `cmd/internal/diagnosticlog/collect.go:209`.
   - One pure age-or-size selector governs retention, with configurable days and conservative handling of invalid evidence.
   - Diagnostic-only integrations verify zero session-clock effects, holder revalidation, and authored-data survival.
   - README and atlas document the new flag, configuration, and oversized-generation policy.

2. **Critical findings**

   - Scheduler blocker isolation remains incomplete, as detailed above. Introduce explicit classification of path-local blockers and sequence tests proving subsequent paths remain reachable.

3. **Important findings**

   - None additional.

4. **Minor findings**

   - None.

5. **Test coverage notes**

   - Passed: diagnosticlog, gcruntime, gccmd, artifactpath, and wrapcmd package suites.
   - Native current/closed-file inspection conformance passed explicitly.
   - Temporary overlays demonstrated meaningful failures without the BR-1, BR-2, BR-4, and BR-5 corrections.
   - The temporary scheduler sibling regression fails at pinned head. Temporary files were removed; tracked files remain unchanged.
   - Full recorded-suite and landing gates remain outstanding in the tracker.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared retention selection and deletion recovery.
   - **ARCH-PURE — pass:** policy logic remains separate from filesystem/process integrations.
   - **ARCH-PURPOSE — flag:** scheduled collection still strands eligible paths behind local blockers.
   - **ARCH-MOCK — pass:** injected stateful inspection and portable fixtures exercise production seams; native conformance is present.
   - **ARCH-CONSTRAINTS — pass:** bounded pages, native output/deadlines, and counted N/2N effects have coverage.
   - **ARCH-SECURE — pass:** exact identity checks and uncertain inspection retain data.
   - **ARCH-ORDER — pass:** reviewed replay fixes protect exact targets; partial-result tests preserve completed effects.
   - **ARCH-FUNERAL — flag:** scheduler termination defeats automatic expiration for later logs.

7. **Plan revision recommendations**

   Add a `## Revisions` entry defining which failures are path-local blockers versus global failures, how blockers stay observable, and scheduler tests covering changed certificates and blocked recovery before later eligible paths.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"de4f7aaf-4757-4c1a-8c19-ba3953bf2a04","repository":"github.com/xianxu/pair","issue_id":"000677","recipe":"milestone-review","round":2,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/e94da22159af4b75","reviewed_head":"8238fd8577e5a9244600bdbaa9913e90173bf74a","slot":"pair:3","requested_at":"2026-10-19T02:20:49.684649Z"},"phase":"finished","pid":8052,"started_at":"2026-10-19T02:20:49.978767Z","finished_at":"2026-10-19T02:23:35.581654Z","session_id":"01a151f6-2d4d-7572-ad08-f9a096bab008","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.17413875s
head: "8238fd8577e5a9244600bdbaa9913e90173bf74a"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000411

---

## Re-review — 2026-10-18T19:31:31-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 677 — diagnostic logs: untie from backup, enforce the per-file cap, keep ~7 days (48 GB, +1.5 GB/day) |
| repo | pair |
| issue file | workshop/issues/000677-diagnostic-logs-untie-from-backup-enforce-the-per-file-cap-keep-7-days-48-gb-1-5-gb-day.md |
| boundary | whole-issue close |
| milestone | — |
| window | 9950fa540423ab382e682437fc0564fadc104f29..4729c0d969d5995197d53c9a8063986de5bcf04e |
| command | sdlc close --issue 677 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-18T19:31:31-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned range satisfies the documented diagnostic-retention policy and resolves BR-6 across the tested blocker class. Path-local failures no longer strand later logs; incomplete inspection, cancellation, deadlines, and contention retain their existing handling. No new blocking findings.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Shared recoverDeletion proves the exact payload; the four-entry-point recovery regression passes.
  - id: BR-2
    disposition: addressed
    note: |
      The real-scheduler missing-declaration regression passes, preserving the declaration and collecting the later eligible log without recreating its parent.
  - id: BR-3
    disposition: addressed
    note: |
      The plan Core concepts tables classify CLI scope as integration and identify diagnostic_turn_budget_test.go as the measurement owner, matching the source.
  - id: BR-4
    disposition: addressed
    note: |
      Managed apply reporting preserves blocked/retained rows and completed deletion bytes across later errors; both reporting regressions pass.
  - id: BR-5
    disposition: addressed
    note: |
      proof.go is classified as a resolved artifact consumer and sweep.go as non-artifact source; artifactpath and testpair guards pass.
  - id: BR-6
    disposition: addressed
    note: |
      schedule.go:125-130 isolates local collection failures. Real-scheduler regressions cover changed certificates, malformed state, held recovery and storage faults. Removing that branch in a disposable checkout made all four cases fail; the pinned head passes. Global inspection failure controls preserve the durable cursor.
```

1. **Strengths**

   - One pure age-or-size selector governs managed retention, with conservative handling of invalid evidence (`diagnosticlog/policy.go:90`).
   - Scheduler isolation covers a failure class while preserving global evidence failures (`gcruntime/schedule.go:94`).
   - README and atlas explain configuration, diagnostic-only collection, deliberate oversized-generation loss, and remaining safety blockers.
   - Independently checked the retained live-evidence hashes: 18 preview/apply matches, **12,734,567,104 bytes** removed, and every corresponding prior inode absent afterward.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Passed full package tests for `gcruntime`, `gccmd`, `artifactpath`, and `testpair`; targeted diagnostic/retention/inspection/recovery tests for `diagnosticlog` and `wrapcmd`; and `git diff --check`. The BR-6 mutation check demonstrated meaningful regression coverage. The disposable checkout was removed. A repository-wide suite was not run.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared selector, inspection seam, and deletion journal.
   - **ARCH-PURE — pass:** policy stays pure; filesystem and process effects remain in integration owners.
   - **ARCH-PURPOSE — pass:** explicit and scheduled diagnostics operate independently of session migration.
   - **ARCH-MOCK — pass:** portable fixtures and stateful inspection doubles exercise production boundaries; native conformance coverage remains.
   - **ARCH-CONSTRAINTS — pass:** bounded pages, native-output limits, and counted N/2N tests cover the declared envelope.
   - **ARCH-SECURE — pass:** identity checks and incomplete-evidence blockers remain enforced.
   - **ARCH-ORDER — pass:** durable replay targets exact payloads; scheduler tests exercise failure and cursor preservation.
   - **ARCH-FUNERAL — pass:** age/oversize collection supplies automatic removal while retaining visibly blocked artifacts.

7. **Plan revision recommendations:** None. The existing BR-6 revision matches the implemented rule.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"a3072a49-d93f-4734-b51a-074b0056ac40","repository":"github.com/xianxu/pair","issue_id":"000677","recipe":"milestone-review","round":3,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/65fa65af44a6503f","reviewed_head":"4729c0d969d5995197d53c9a8063986de5bcf04e","slot":"pair:3","requested_at":"2026-10-19T02:28:52.605154Z"},"phase":"finished","pid":22823,"started_at":"2026-10-19T02:28:52.712444Z","finished_at":"2026-10-19T02:31:28.707501Z","session_id":"01a151fd-8a6b-7be2-8ce1-1cbb053003fc","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.751894833s
head: "4729c0d969d5995197d53c9a8063986de5bcf04e"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-18T20:21:31-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 677 — diagnostic logs: untie from backup, enforce the per-file cap, keep ~7 days (48 GB, +1.5 GB/day) |
| repo | pair |
| issue file | workshop/issues/000677-diagnostic-logs-untie-from-backup-enforce-the-per-file-cap-keep-7-days-48-gb-1-5-gb-day.md |
| boundary | whole-issue close |
| milestone | — |
| window | abf8579a70acf2ce637d9f5c4f69f4440fb8373e..163c91304af64e5d7d4f35677a870aef8aa95669 |
| command | sdlc close --issue 677 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-18T20:21:31-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The interdiff corrects comment attachment and records review history without changing executable behavior or approved scope. No new defects found. BR-1 through BR-6 remain previously disposed; none is reopened.

```findings
{}
```

1. **Strengths**
   - `cmd/internal/diagnosticlog/sweep.go:21` correctly attaches the snapshot contract to `SweepSnapshot`.
   - `workshop/lessons.md:439` identifies the actual guard and its owning package.
   - The issue’s added revision records the correction; Spec, Done-when, and the implementation plan remain unchanged.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes:** Independently passed `TestNoDeclarationCarriesTwoStackedGodocs` in `termcmd` and pinned-range `git diff --check`. Inspected the guard’s declaration and comment matching. No runtime regression test is required for this comment-only correction. Repository-wide tests were not rerun.

6. **Architectural notes**
   - **ARCH-DRY — pass:** Reuses the existing guard.
   - **ARCH-PURE — pass:** No logic or IO boundary changes.
   - **ARCH-PURPOSE — pass:** Repairs the documented defect without expanding scope. No new surface requires README or atlas updates.

7. **Plan revision recommendations:** None.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"904d7072-bdba-4cc4-ae89-6f4160ecfd46","repository":"github.com/xianxu/pair","issue_id":"000677","recipe":"fix-review","round":4,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/fc42f9c18ce2f55b","reviewed_head":"163c91304af64e5d7d4f35677a870aef8aa95669","slot":"pair:3","requested_at":"2026-10-19T03:20:47.431774Z"},"phase":"finished","pid":91713,"started_at":"2026-10-19T03:20:47.727569Z","finished_at":"2026-10-19T03:21:29.569476Z","session_id":"01a1522d-1359-7092-b616-00997aba8774","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.208643084s
head: "163c91304af64e5d7d4f35677a870aef8aa95669"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
