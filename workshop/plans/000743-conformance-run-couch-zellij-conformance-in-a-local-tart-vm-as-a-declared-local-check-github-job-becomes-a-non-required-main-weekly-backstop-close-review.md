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
