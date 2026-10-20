# Boundary Review — pair#720 (whole-issue close)

| field | value |
|-------|-------|
| issue | 720 — Slot scratch: export the RAM-disk temp variables only when the agent's sandbox can write there (the probe runs outside the sandbox) |
| repo | pair |
| issue file | workshop/issues/000720-slot-scratch-export-the-ram-disk-temp-variables-only-when-the-agent-s-sandbox-can-write-there-the-probe-runs-outside-the-sandbox.md |
| boundary | whole-issue close |
| milestone | — |
| window | 7d9ce7ca7988743e3ef201221a19c985c8cecec7..7d772f18a1e923e9ced69d630e8c142f52d43cc2 |
| command | sdlc close --issue 720 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-19T21:26:38-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The shared admission predicate, bounded settings reads, selected-profile plumbing, and documentation updates are sound foundations. Two gaps still permit RAM scratch exports when Claude’s effective sandbox denies access, so the issue’s central contract is not yet satisfied.

```findings
findings:
  - id: new
    severity: Critical
    kind: defect
    family: effective-policy-denials
    title: |
      Claude admission ignores permission rules that deny sandbox writes
    detail: |
      cmd/internal/couchscratch/sandbox_settings.go:29-55 projects only sandbox.filesystem and ignores permissions.deny. With sandbox.enabled=true, allowWrite=["/Volumes/TestScratch"], and permissions.deny=["Edit(//Volumes/TestScratch/**)"], ScratchAllowed returns true although Claude applies Edit denies to sandbox access. Project these restrictions, or return unknown when unsupported write-denial rules occur; add a regression proving that this configuration produces no scratch exports.
  - id: new
    severity: Critical
    kind: defect
    family: effective-policy-source-discovery
    title: |
      Claude local settings discovery misses repository and main-checkout policy
    detail: |
      cmd/internal/couchscratch/sandbox_read.go:59-62 reads settings.local.json only beneath in.Project, supplied from record.StartingPath at cmd/internal/couchcmd/scratch.go:175. Claude can load local settings from the repository root when launched in a subdirectory, and from the main checkout for a worktree. A denying local policy there is skipped and absence is treated as sandbox disabled, allowing all three scratch exports. Resolve the native local-settings location or conservatively decline unresolved layouts; cover subdirectory and linked-worktree fixtures through the production reader.
```

1. **Strengths**

   - `ScratchAllowed` has direct, IO-free tests for ancestor grants, sibling prefixes, overlapping denies, and malformed paths.
   - `readSandboxFile` bounds reads and rejects special files; its tests include FIFOs and oversized settings.
   - Fresh and switched launches receive the selected profile without rewriting the durable profile.
   - README and atlas changes cover the new admission behavior and current-selection diagnostics.

2. **Critical findings**

   - **Ignored permission denies:** [sandbox_settings.go:29](/Volumes/TestScratch/review-checkouts/e0b2d34759bed559/cmd/internal/couchscratch/sandbox_settings.go:29). Claude’s documented sandbox policy includes `Edit` deny rules, so `allowWrite` alone cannot establish admission. Preserve the conservative projection by treating unsupported restrictions as unknown. See [Claude sandbox permission rules](https://code.claude.com/docs/en/sandboxing#permission-rules).
   - **Incorrect local-settings location:** [sandbox_read.go:59](/Volumes/TestScratch/review-checkouts/e0b2d34759bed559/cmd/internal/couchscratch/sandbox_read.go:59). The reader assumes both project files share one directory. Claude documents different local-file discovery for subdirectories and worktrees. See [Claude local settings locations](https://code.claude.com/docs/en/settings#keep-personal-settings-out-of-a-repository).

3. **Important findings**

   None separately raised.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Full `couchscratch`, `couchcmd`, and `artifactpath` packages passed. Focused fresh/warm launch, switch, show, and host-probe regression tests passed. The broader package run was stopped after roughly two minutes while `couchcore` remained running; its full result is unverified.

   Current fixtures omit both blocking cases. Corrections need tests that fail on this pinned head. There are no prior findings to dispose of.

6. **Architectural notes**

   - **ARCH-DRY — pass:** launch and diagnostics share admission logic.
   - **ARCH-PURE — pass:** predicate and projections remain IO-free; core-concept entities exist at their documented locations.
   - **ARCH-PURPOSE — flag:** the two findings leave the original host-success/consumer-denial failure class reachable.
   - **ARCH-MOCK — pass within the declared envelope:** portable settings fixtures exercise the production reader seam; native conformance remains deferred.
   - **ARCH-CONSTRAINTS — pass:** fixed-count, bounded reads; no background work added.
   - **ARCH-SECURE — flag:** incomplete external-policy interpretation turns missing evidence into permission.
   - **ARCH-ORDER — pass:** no new state retained between events.
   - **ARCH-FUNERAL — pass:** no new durable artifact family.
   - **ARCH-EVOLVE — pass:** settings are read without rewriting unknown fields.

7. **Plan revision recommendations**

   Add `## Revisions` entries specifying:

   - All write-restricting Claude policy inputs are projected or cause unknown admission, including permission denies.
   - Local-settings discovery accounts for launch directory, repository root, and linked-worktree main checkout, with explicit fallback and regression fixtures.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"8a945867-1589-45dc-b29c-29780582f2d2","repository":"github.com/xianxu/pair","issue_id":"000720","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/e0b2d34759bed559","reviewed_head":"7d772f18a1e923e9ced69d630e8c142f52d43cc2","slot":"pair:7","requested_at":"2026-10-20T04:23:39.668926Z"},"phase":"finished","pid":77636,"started_at":"2026-10-20T04:23:39.761125Z","finished_at":"2026-10-20T04:26:36.442624Z","session_id":"01a1578c-fcf1-7a12-af5d-793d2abf31bc","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.403145s
head: "7d772f18a1e923e9ced69d630e8c142f52d43cc2"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000409

---

## Re-review — 2026-10-19T21:41:02-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 720 — Slot scratch: export the RAM-disk temp variables only when the agent's sandbox can write there (the probe runs outside the sandbox) |
| repo | pair |
| issue file | workshop/issues/000720-slot-scratch-export-the-ram-disk-temp-variables-only-when-the-agent-s-sandbox-can-write-there-the-probe-runs-outside-the-sandbox.md |
| boundary | whole-issue close |
| milestone | — |
| window | 7d9ce7ca7988743e3ef201221a19c985c8cecec7..ba3584f4f35642e621411abf34f4cc59badeeee6 |
| command | sdlc close --issue 720 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-19T21:41:02-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The V2 declaration contract removes the native-policy resolver responsible for both prior findings. Launch admission and marker lifecycle checks pass, and removing the admission guard makes the regression fail. One diagnostic gap remains: showing a slot without a thread omits scratch status.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      The authoritative V2 revision replaces policy inference with an explicit declaration. Native-policy readers are absent; the production composition regression rejects exports without the marker and fails when the admission guard is removed.
  - id: BR-2
    disposition: addressed
    note: |
      V2 removes native local-settings discovery entirely. Admission uses the injected XDG/HOME declaration path; lifecycle, routing, and isolation tests pass through the production admission seam.
findings:
  - id: new
    severity: Important
    kind: defect
    family: scratch-diagnostics-cover-show-targets
    title: |
      Slot-only show omits the promised scratch diagnostic
    detail: |
      cmd/internal/couchcore/operationdispatch.go:251-255 projects scratch status only over matched threads. A valid slot with no thread therefore has no scratch status, contrary to the V2 show contract (ARCH-PURPOSE). Extending TestShowReportsASlotWithoutAThread with an injected ScratchDiagnostic and a nonempty Scratch assertion fails. Project admission for slot-only targets too, and retain that regression.
```

1. **Strengths**

   - Launch and diagnostics share `scratchAdmission`; show does not invoke scratch preparation.
   - Pure path/admission tables require no IO. Integration tests exercise real private files and marker creation/removal.
   - Existing host, ownership, preparation, and cleanup protections remain intact.
   - README and atlas document the new declaration and its accepted limitation.

2. **Critical findings:** None.

3. **Important findings:** Slot-only diagnostics, detailed above. Fix by supporting admission projection without a matched `ThreadRecord`.

4. **Minor findings:** None.

5. **Test coverage notes**

   - Focused scratch/show checks passed in `couchscratch`, `couchcmd`, and `couchcore`.
   - Complete `couchscratch` and `artifactpath` packages passed.
   - In a disposable head copy, removing the refusal return caused the marker regression to fail with all three unwanted exports.
   - The added slot-only diagnostic assertion failed against the pinned implementation.
   - Disposable files were removed. The repository-wide suite was not run during this review.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared admission and existing preparation machinery.
   - **ARCH-PURE — pass:** pure decisions separated from filesystem observation.
   - **ARCH-PURPOSE — flag:** slot-only show misses the documented diagnostic.
   - **ARCH-MOCK — pass:** private filesystem fixtures and injected probe use production seams; no new external service.
   - **ARCH-CONSTRAINTS — pass:** fixed-cost marker observation; no content reads or scans.
   - **ARCH-SECURE — pass:** invalid paths, nonregular markers, and isolated-path escapes decline admission.
   - **ARCH-ORDER — pass:** admission retains no state between calls; marker changes are observed on subsequent calls.
   - **ARCH-FUNERAL — pass:** Couch creates no new durable artifact; operator removal ends declaration adoption.
   - **ARCH-EVOLVE — pass:** no existing persisted format is rewritten; marker contents are ignored.

7. **Plan revision recommendation**

   Add a `## Revisions` entry specifying that current-selection diagnostics cover both thread targets and valid slots without threads, with the corresponding regression test.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"46562d83-f3fa-4e4b-8971-7013743b2b50","repository":"github.com/xianxu/pair","issue_id":"000720","recipe":"milestone-review","round":2,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/0adfd7a456d01ded","reviewed_head":"ba3584f4f35642e621411abf34f4cc59badeeee6","slot":"pair:7","requested_at":"2026-10-20T04:39:02.17052Z"},"phase":"finished","pid":72773,"started_at":"2026-10-20T04:39:02.465877Z","finished_at":"2026-10-20T04:41:01.645412Z","session_id":"01a1579b-1219-72b2-9457-364cefd2f4c5","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.907557083s
head: "ba3584f4f35642e621411abf34f4cc59badeeee6"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
