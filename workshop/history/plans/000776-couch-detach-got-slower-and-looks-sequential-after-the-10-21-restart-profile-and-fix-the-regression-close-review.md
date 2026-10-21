# Boundary Review — pair#776 (whole-issue close)

| field | value |
|-------|-------|
| issue | 776 — couch detach got slower and looks sequential after the 10-21 restart: profile and fix the regression |
| repo | pair |
| issue file | workshop/issues/000776-couch-detach-got-slower-and-looks-sequential-after-the-10-21-restart-profile-and-fix-the-regression.md |
| boundary | whole-issue close |
| milestone | — |
| window | a9522724d4d84586afe222d201029d0713a89fc8..f41bac24797392cca7187a74d645e00072c8020e |
| command | sdlc close --issue 776 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-21T13:07:25-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The cancellation fix is narrow, preserves descendant cleanup, and passes targeted and race tests. Restoring the old unconditional-wait behavior makes the new regression fail. No production defect was found; the remaining issue is acceptance wording that does not match the revised cancellation contract.

Done-when: stale — the new regressions test group-cancellation completion and lease ownership, while the second acceptance clause still requires a detach-time bound or parallel-retirement regression.

```findings
findings:
  - id: new
    severity: Important
    kind: defect
    family: acceptance-contract-traceability
    title: |
      Align Done when with the approved cancellation regression contract
    detail: |
      workshop/issues/000776-couch-detach-got-slower-and-looks-sequential-after-the-10-21-restart-profile-and-fix-the-regression.md:47-48 still requires a regression that bounds detach time or asserts parallel retirement. The revised implementation instead supplies deterministic group-completion tests (provision_cancel_test.go:49,83) and a descendant-lease control (provision_subprocess_test.go:145). The existing parallel-Leave test also passes without this fix, so it cannot establish that regression requirement. Family enumeration: this second Done-when bullet is the sole stale acceptance clause in this window. Revise it explicitly to accept completion before grace on confirmed group absence, while preserving descendant cleanup and error handling.
```

1. **Strengths:** Only ESRCH permits early completion; permission and inspection failures retain escalation ([provision_io.go:109](/Volumes/TestScratch/review-checkouts/9764e9b79d971783/cmd/internal/couchcore/provision_io.go:109)). The native test verifies inherited lease ownership before cancellation and release after the owned join. Atlas documentation accurately describes the implementation. Retained profiles distinguish observed timing from unproved changed-path execution.

2. **Critical findings:** None.

3. **Important findings:** The acceptance-contract mismatch above; a documentation correction is sufficient.

4. **Minor findings:** None.

5. **Test coverage:** `TestProvision(Process|Group)` passed in 4.263s. The race run covering those tests plus `TestLeave` passed in 6.291s. A temporary Go overlay restoring baseline cancellation semantics made `whole_group_gone` fail with `[TERM,KILL]` instead of `[TERM]`; temporary files were removed. The full recorded suite remains a landing prerequisite.

6. **Architecture:** **ARCH-DRY: pass** — one shared cancellation helper serves the existing runner. **ARCH-PURE: pass** — signal and clock dependencies are injected into the cancellation seam. **ARCH-PURPOSE: pass** — the code addresses the measured shutdown wait without changing fan-out. **ARCH-EVOLVE: pass** — no persisted format changes. Atlas was updated; no new user-facing command or configuration requires a README change.

7. **Plan revision recommendation:** Add a `## Revisions` entry aligning acceptance with deterministic cancellation completion, preserved grace/escalation, and native descendant-lease protection, while retaining the measured four-slot before/after requirement.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"95b66f25-f3eb-4d86-b6b4-0cacf6679991","repository":"github.com/xianxu/pair","issue_id":"000776","recipe":"small-diff-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/9764e9b79d971783","reviewed_head":"f41bac24797392cca7187a74d645e00072c8020e","slot":"pair:6","requested_at":"2026-10-21T20:05:16.896795Z"},"phase":"finished","pid":32838,"started_at":"2026-10-21T20:05:17.005238Z","finished_at":"2026-10-21T20:07:16.720138Z","session_id":"01a16011-6d5b-77e2-9120-835c348f7095","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.800233041s
head: "f41bac24797392cca7187a74d645e00072c8020e"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-21T13:09:45-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 776 — couch detach got slower and looks sequential after the 10-21 restart: profile and fix the regression |
| repo | pair |
| issue file | workshop/issues/000776-couch-detach-got-slower-and-looks-sequential-after-the-10-21-restart-profile-and-fix-the-regression.md |
| boundary | whole-issue close |
| milestone | — |
| window | a9522724d4d84586afe222d201029d0713a89fc8..1af622b6443e765282dc49d2e1d7209d338c49cd |
| command | sdlc close --issue 776 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-21T13:09:45-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The cancellation change matches the revised acceptance contract, preserves descendant cleanup and error handling, and passes targeted and race tests. BR-1 is addressed by a documentation correction; no new defects were found.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      The pinned before/after diff replaces the stale timing-or-parallelism requirement at issue lines 46–47 with completion before grace on confirmed group absence, conservative unavailable-probe handling, and native descendant lease cleanup. These requirements match provision_cancel_test.go:49,83 and provision_subprocess_test.go:145. The Revisions section preserves the approval and original clause.
```

1. **Strengths**
   - Only ESRCH permits early completion; other probe results preserve escalation (`provision_io.go:109`).
   - Deterministic tests cover group disappearance, surviving descendants, unavailable probes and signal errors.
   - The native test verifies inherited lease ownership before cancellation and release after `Run` joins.
   - Retained profiling evidence clearly states attribution and sampling limits.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage:** Provision process/group and Leave controls passed in 4.691s; the same set passed with race detection in 6.092s. A temporary overlay restoring unconditional grace waiting made `whole_group_gone` fail with TERM+KILL instead of TERM. Temporary files were removed; the checkout remains unchanged. The separately recorded full suite remains a landing prerequisite.

6. **Architecture**
   - **ARCH-DRY: pass** — one shared cancellation helper.
   - **ARCH-PURE: pass** — signal and clock dependencies are injected; production IO stays in the runner.
   - **ARCH-PURPOSE: pass** — addresses the measured post-Leave wait while retaining cleanup guarantees.
   - **ARCH-EVOLVE: pass** — no persisted format changes.

   Atlas documents the cancellation behavior. No new command, flag or configuration requires a README update.

7. **Plan revision recommendations:** None; the existing Revisions section records the corrected acceptance contract.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"145d4c4f-c047-43d0-a2bf-b38ea31df120","repository":"github.com/xianxu/pair","issue_id":"000776","recipe":"small-diff-review","round":2,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/4f8f627c22e4a810","reviewed_head":"1af622b6443e765282dc49d2e1d7209d338c49cd","slot":"pair:6","requested_at":"2026-10-21T20:08:40.713486Z"},"phase":"finished","pid":74128,"started_at":"2026-10-21T20:08:40.818485Z","finished_at":"2026-10-21T20:09:44.411139Z","session_id":"01a16014-8999-7e71-b2a8-1bf4f7e029c8","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.818690042s
head: "1af622b6443e765282dc49d2e1d7209d338c49cd"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-21T13:44:19-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 776 — couch detach got slower and looks sequential after the 10-21 restart: profile and fix the regression |
| repo | pair |
| issue file | workshop/issues/000776-couch-detach-got-slower-and-looks-sequential-after-the-10-21-restart-profile-and-fix-the-regression.md |
| boundary | whole-issue close |
| milestone | — |
| window | 9d6cf03065a778e9e2c4b26fc5536515d4b83228..072ca58490ea0949c6b1111344d48e5e7ed99324 |
| command | sdlc close --issue 776 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-21T13:44:19-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The interdiff correctly restores post-TERM grace ordering and provides a regression test that fails without the correction. BR-1 remains addressed. The tracker revision records the fix without changing approved scope; no new defects were found.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      The corrected Done-when clause at issue lines 46–47 remains unchanged across this window and matches the deterministic completion, unavailable-probe, and native descendant-lease controls. Revisions retains the original clause and correction rationale.
```

1. **Strengths**

   - Clock construction follows TERM, with cleanup deferred on every constructed-clock path (`provision_io.go:115–120`).
   - Tests assert ordering, exactly-once cleanup, and no clock construction for initial ESRCH (`provision_cancel_test.go:73–91`).
   - Atlas accurately documents the corrected sequence (`atlas/workspace-provisioning.md:245`).

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage:** Race-enabled provision/process/group and Leave controls passed in 12.436s. Moving clock construction before TERM through a temporary overlay made the regression fail at the ordering assertion. Inventory audit and diff whitespace checks passed. Temporary artifacts were removed; checkout remains clean. The full recorded suite was not rerun in this review.

6. **Architecture**

   - **ARCH-DRY: pass** — shared cancellation helper remains authoritative.
   - **ARCH-PURE: pass** — signal and clock dependencies are injected; production IO remains at the runner boundary.
   - **ARCH-PURPOSE: pass** — preserves the approved grace and descendant-cleanup contract.
   - **ARCH-EVOLVE: pass** — no persisted-format changes.

   Atlas is updated; no new user-facing surface requires README changes.

7. **Plan revision recommendations:** None.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"26ffba91-606c-4d1f-b892-08d3dbe353cc","repository":"github.com/xianxu/pair","issue_id":"000776","recipe":"fix-review","round":3,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/6e47b74298199faa","reviewed_head":"072ca58490ea0949c6b1111344d48e5e7ed99324","slot":"pair:6","requested_at":"2026-10-21T20:43:05.554961Z"},"phase":"finished","pid":34402,"started_at":"2026-10-21T20:43:05.730585Z","finished_at":"2026-10-21T20:44:18.161262Z","session_id":"01a16034-0c1d-71a3-a276-9348c2a8f9e7","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.949962291s
head: "072ca58490ea0949c6b1111344d48e5e7ed99324"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
