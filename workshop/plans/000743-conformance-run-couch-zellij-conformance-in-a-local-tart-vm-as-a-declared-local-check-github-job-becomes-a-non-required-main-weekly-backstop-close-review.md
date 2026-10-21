# Boundary Review — pair#743 (whole-issue close)

| field | value |
|-------|-------|
| issue | 743 — conformance: run couch-zellij-conformance in a local Tart VM as a declared local check; GitHub job becomes a non-required main/weekly backstop |
| repo | pair |
| issue file | workshop/issues/000743-conformance-run-couch-zellij-conformance-in-a-local-tart-vm-as-a-declared-local-check-github-job-becomes-a-non-required-main-weekly-backstop.md |
| boundary | whole-issue close |
| milestone | — |
| window | e30ea629818a733f65741bfc95c11de8e1868f34..d1d00fb43615671550b93f6a247881facc5de252 |
| command | sdlc close --issue 743 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-21T03:23:08-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The revised rollout is substantially delivered, and all six prior corrections hold. One reproduced compatibility defect blocks shipping: image refresh silently deletes unknown metadata nested inside existing tool records.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Cancellation regressions pass. Removing success authority and interruption guards makes all six cancellation stages activate the image incorrectly.
  - id: BR-2
    disposition: addressed
    note: |
      Exact-limit and overflow regressions pass. Removing the publication bound installs an unreadable 65,537-byte manifest and changes the prior active bytes.
  - id: BR-3
    disposition: addressed
    note: |
      The plan's integration table and 22:07 Revisions passage now identify the delivered runtime, host, driver, positions, snapshot, process and embedded guest files.
  - id: BR-4
    disposition: addressed
    note: |
      Hosted package/case/exclusion equality is enforced in standard testing. Disabling the equality assertion produces seven regression failures.
  - id: BR-5
    disposition: addressed
    note: |
      Every retained night reserves its full possible bundle. Running the surviving-writer regression against the previous implementation fails with only 218 bytes charged.
  - id: BR-6
    disposition: addressed
    note: |
      Checked lstat controls accounting and recursion, and traversal errors propagate. The directory-classification regression passes at head and fails against the previous implementation.
findings:
  - id: new
    severity: Critical
    kind: defect
    family: persisted-unknown-field-preservation
    title: |
      Image refresh drops persisted unknown tool metadata
    detail: |
      cmd/internal/vmcmd/provision.go:215-231 replaces Tools with the guest report and preserves only old.Unknown. An existing tools.go.future_metadata member disappears even when the tool version and checksum remain unchanged; a temporary Go test overlay reproduced this loss. This violates ARCH-EVOLVE and the plan's nested unknown-field preservation contract. Preserve existing nested unknown members through publication, sweep persisted read-modify-write paths for the same rule, and add a refresh regression that fails on this head. Apply the existing final size bound after merging.
```

1. **Strengths**

   - Cleanup decisions use the production reducer and injected runtime, with controlled cancellation tests.
   - Result export has structural byte limits and exposes no guest-writable host share.
   - Hosted/local selection drift guards, README updates and atlas documentation are present.
   - Revalidating retained native streams confirmed 413 events and all 28 required cases; the known-bad stream was rejected.

2. **Critical findings**

   - `provision.go:215-231`: nested unknown metadata is lost during refresh, as detailed above.

3. **Important findings**

   None.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Five targeted Go packages and all 68 Python tests passed. Prior-fix mutations failed as expected. The additional refresh-preservation regression failed against pinned head. No live VM was started during this review.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared selection and subprocess helpers.
   - **ARCH-PURE — pass:** pure parsing/reduction separated from IO.
   - **ARCH-PURPOSE — pass:** approved phased delivery is implemented.
   - **ARCH-MOCK — pass:** stateful fakes use production seams.
   - **ARCH-CONSTRAINTS — pass:** execution, export and retention bounds are enforced.
   - **ARCH-SECURE — pass:** isolated roots, allowlisted environment and bounded parsing.
   - **ARCH-ORDER — pass:** cancellation preserves cleanup obligations and uncertainty.
   - **ARCH-FUNERAL — pass:** bounded retention and explicit archive/recovery paths.
   - **ARCH-EVOLVE — flag:** publication loses nested unknown fields.

7. **Plan revision recommendations**

   Add a `## Revisions` entry specifying preservation precedence for existing nested tool metadata during refresh, the persisted-writer sweep, and the regression proving preservation before final size validation.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"f7634c13-b6a9-409e-b6c3-e41d9866e81b","repository":"github.com/xianxu/pair","issue_id":"000743","recipe":"milestone-review","round":8,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/83ec12ea2b078202","reviewed_head":"d1d00fb43615671550b93f6a247881facc5de252","slot":"pair:3","requested_at":"2026-10-21T10:18:57.489225Z"},"phase":"finished","pid":96509,"started_at":"2026-10-21T10:18:57.643609Z","finished_at":"2026-10-21T10:23:08.227653Z","session_id":"01a15df8-a289-7841-9ef5-1f69441c3ef9","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.679399375s
head: "d1d00fb43615671550b93f6a247881facc5de252"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-21T03:32:34-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 743 — conformance: run couch-zellij-conformance in a local Tart VM as a declared local check; GitHub job becomes a non-required main/weekly backstop |
| repo | pair |
| issue file | workshop/issues/000743-conformance-run-couch-zellij-conformance-in-a-local-tart-vm-as-a-declared-local-check-github-job-becomes-a-non-required-main-weekly-backstop.md |
| boundary | whole-issue close |
| milestone | — |
| window | e30ea629818a733f65741bfc95c11de8e1868f34..b267e07012e29425ca77f8ba158b125111a64c4b |
| command | sdlc close --issue 743 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-21T03:32:34-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned range matches the revised infrastructure and burn-in scope. BR-7 is addressed with a reachable regression that fails when the fix is removed. No new blocking findings emerged. Hosted PR conformance remains enabled; production named-check bootstrap and later promotion remain explicitly assigned to post-landing work.

```findings
dispose:
  - id: BR-7
    disposition: addressed
    note: |
      provision.go:227-230 preserves existing top-level and nested tool metadata through strictjson.MergeRaw; final size validation follows at lines 240-245. TestPublishProvisionPreservesNestedMetadataAcrossRefresh passes on head and fails in both version cases when the nested merge is removed through a temporary Go overlay. Exact-65536/65537 publication tests also pass. Swept claim rewrites, image encoding, result publication and raw result-manifest export for the same preservation rule.
```

1. **Strengths**

   - Image publication preserves additive metadata and checks the fully merged manifest before replacement (`cmd/internal/vmcmd/provision.go:227`).
   - Cancellation and cleanup use the production reducer and controlled barriers; uncertain cleanup retains durable occupancy.
   - Hosted and VM case selections have executable drift checks, including package identity and exclusions.
   - README and atlas document the new commands, isolation, resource budgets and recovery paths.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Passed:

   - Full `vmcmd` tests, both ordinary and race-enabled.
   - Dispatcher, public CLI, artifact-path and repository-lint packages.
   - All 68 Python conformance, provisioning and nightly checks.
   - BR-7 mutation check: removing the nested merge makes both regression cases fail.

   Independently replayed the retained native stdout through the current validator: clean evidence validates 413 events and 28 cases; the known-bad evidence rejects the required skipped case. Live Tart execution was not repeated during this read-only review. The runner and conformance execution code is unchanged since that native qualification; subsequent relevant changes concern provisioning metadata preservation.

6. **Architectural notes for upcoming work**

   | Principle | Result | Evidence |
   |---|---|---|
   | ARCH-DRY | Pass | Shared strict JSON helpers and enforced conformance catalogs. |
   | ARCH-PURE | Pass | Request, codec, transition and validation logic have direct tests; effects use runtime/driver seams. |
   | ARCH-PURPOSE | Pass | Revised runner/burn-in scope delivered; activation and promotion remain explicit follow-up boundaries. |
   | ARCH-MOCK | Pass | Stateful runtime and native-driver fixtures exercise production seams; retained native evidence checks real behavior. |
   | ARCH-CONSTRAINTS | Pass | Snapshot, export, log, retention and cleanup limits are enforced and exercised. |
   | ARCH-SECURE | Pass | Clean guest environment, private snapshots, read-only sharing and bounded result admission. |
   | ARCH-ORDER | Pass | Explicit reducer, controlled cancellation ordering and retained uncertainty govern release. |
   | ARCH-FUNERAL | Pass | Scratch/VM cleanup, bounded retained evidence and explicit archive/disposal paths are documented. |
   | ARCH-EVOLVE | Pass | Unknown metadata survives refresh and rewrites; final reader/writer bounds remain enforced. |

7. **Plan revision recommendations:** None. Core-concept locations match the implementation; existing revisions explain the scope and post-landing acceptance sequence.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"c63aa7d1-ea91-4fef-85ed-a3277d13b827","repository":"github.com/xianxu/pair","issue_id":"000743","recipe":"milestone-review","round":9,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/569ad53622da47a7","reviewed_head":"b267e07012e29425ca77f8ba158b125111a64c4b","slot":"pair:3","requested_at":"2026-10-21T10:29:29.865954Z"},"phase":"finished","pid":68731,"started_at":"2026-10-21T10:29:30.039186Z","finished_at":"2026-10-21T10:32:33.984385Z","session_id":"01a15e02-488b-7893-9735-e70380d46a44","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.705438834s
head: "b267e07012e29425ca77f8ba158b125111a64c4b"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-21T16:34:05-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 743 — conformance: run couch-zellij-conformance in a local Tart VM as a declared local check; GitHub job becomes a non-required main/weekly backstop |
| repo | pair |
| issue file | workshop/issues/000743-conformance-run-couch-zellij-conformance-in-a-local-tart-vm-as-a-declared-local-check-github-job-becomes-a-non-required-main-weekly-backstop.md |
| boundary | whole-issue close |
| milestone | — |
| window | 68288c87ac1c362de219be0617f52b6a625dcb2c..d457e072ad0e3d07dd098a13fe8774a6433cbe02 |
| command | sdlc close --issue 743 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-21T16:34:05-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The interdiff preserves both sides of all three conflict resolutions. Certificate migration retains the audited reader classifications, and the revision additions record integration and verification without changing scope. No new defects found. BR-1 through BR-7 remain disposed; none required reopening.

```findings
{}
```

1. **Strengths**

   - Both atlas links survive and resolve (`atlas/index.md:7`).
   - VM and mock-server source classifications are retained (`cmd/internal/artifactpath/manifest.go:672`).
   - All 13 migrated certificates exactly match their prior audited dispositions and hashes.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Passed the inventory check, all 21 reader regression tests, the full artifact-path suite, and targeted repository-lint and Couch resource guards. The reader tests include real Git merging, unknown-field preservation and seeded rejection through the actual affected runner. Live Tart execution was not repeated.

6. **Architectural notes**

   - **ARCH-DRY — Pass:** Uses main’s certificate mechanism; removes the retired aggregate.
   - **ARCH-PURE — Pass:** No new mixing of business logic and IO.
   - **ARCH-PURPOSE — Pass:** Resolutions preserve both features and all migrated reader records.
   - **ARCH-EVOLVE — Pass:** Existing certificate metadata remains intact; strict validation and unknown-field regression tests pass.

   No new user-facing surface requires a README update.

7. **Plan revision recommendations:** None.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"075a4dde-6b0b-4a24-8d65-9872561fff07","repository":"github.com/xianxu/pair","issue_id":"000743","recipe":"fix-review","round":10,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/cf717fe7e728a997","reviewed_head":"d457e072ad0e3d07dd098a13fe8774a6433cbe02","slot":"pair:3","requested_at":"2026-10-21T23:31:47.574175Z"},"phase":"finished","pid":2764,"started_at":"2026-10-21T23:31:47.80256Z","finished_at":"2026-10-21T23:34:04.290435Z","session_id":"01a160ce-7fd0-7d01-949b-1431413fbc43","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.866809375s
head: "d457e072ad0e3d07dd098a13fe8774a6433cbe02"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
