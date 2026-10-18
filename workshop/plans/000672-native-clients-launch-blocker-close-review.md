# Boundary Review — pair#672 (whole-issue close)

| field | value |
|-------|-------|
| issue | 672 — Accept native Zellij client replies without blocking launches |
| repo | pair |
| issue file | workshop/issues/000672-native-clients-launch-blocker.md |
| boundary | whole-issue close |
| milestone | — |
| window | b115dc79533f5949b765ef9b31cf02a579727c4c..5e299f51362df4c72591502cca46971ec7df5a80 |
| command | sdlc close --issue 672 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-18T11:36:51-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The compatibility repair is narrow and its regression tests demonstrably fail without it. No code defect was found, but required qualification remains blocked by two review-environment prerequisites.

```findings
findings:
  - id: new
    severity: Important
    kind: unverifiable
    family: required-qualification-evidence
    title: |
      Required consumer and repository checks cannot pass in this review environment
    detail: |
      All observed instances: normal and race runs of launcher/zellijreply/zellijsession fail at workbench_attach_invariant_test.go:155 for callers=1,3,21 with Unix socket bind: invalid argument under the review's long TMPDIR; go test ./internal/repolint -count=1 fails at catalog_test.go:447 because no configured main tracking authority or remote main ref exists. Provide passing results for these commands at pinned head 5e299f5 using a socket-compatible temporary root and authoritative main history, or repair those review-environment prerequisites and rerun.
```

1. **Strengths**

   - The captured reply remains rejected by `ParseClients`; acceptance is confined to the name probe.
   - `zellijparse.go:70` reuses the existing strict listing validator, including duplicate and identity checks.
   - `native_name_absence_test.go:41` covers the captured reply alongside malformed, contradictory, failed and cancelled outcomes.

2. **Critical findings:** None.

3. **Important findings:** Required qualification is unverifiable as detailed above. No implementation change is indicated by these environment failures.

4. **Minor findings:** None.

5. **Test coverage notes**

   - New regression tests pass normally and under race detection.
   - With base implementation files supplied through a temporary Go overlay, the tests fail on captured native absence and contradictory clients-table acceptance.
   - Fixture contract tests pass.
   - Full affected normal/race runs fail only at the three socket cases listed above; reply and session parser packages pass.
   - `git diff --check` reports trailing spaces in the verbatim native fixture; preserving captured bytes is appropriate.

6. **Architectural notes**

   - **ARCH-DRY: pass** — existing full-listing validation is reused.
   - **ARCH-PURE: pass** — normalization/classification remains pure; process execution uses the existing injected seam.
   - **ARCH-PURPOSE: pass** — creation, name-budget calibration and Couch name assignment all consume the repaired probe. Atlas documentation is updated; no new user-facing command surface requires a README change.

7. **Plan revision recommendations:** None. Record passing required qualification before crossing the boundary.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"64afc8bc-5a7b-4a5f-b61c-0d8e64cb5df5","repository":"github.com/xianxu/pair","issue_id":"000672","recipe":"small-diff-review","round":1,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/ddc2a13fccafa615","reviewed_head":"5e299f51362df4c72591502cca46971ec7df5a80","slot":"pair:1","requested_at":"2026-10-18T18:33:25.846016Z"},"phase":"finished","pid":88577,"started_at":"2026-10-18T18:33:25.982291Z","finished_at":"2026-10-18T18:36:51.000532Z","session_id":"01a1504a-420a-7ce3-b63f-ce8932a4ebba","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.15241325s
head: "5e299f51362df4c72591502cca46971ec7df5a80"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000411

---

## Re-review — 2026-10-18T11:47:09-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 672 — Accept native Zellij client replies without blocking launches |
| repo | pair |
| issue file | workshop/issues/000672-native-clients-launch-blocker.md |
| boundary | whole-issue close |
| milestone | — |
| window | b115dc79533f5949b765ef9b31cf02a579727c4c..dd8355c685a473acd7ba76a1d803cf36a6c7898b |
| command | sdlc close --issue 672 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-18T11:47:09-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The narrow compatibility repair satisfies the revised spec without relaxing the clients parser. Normal and race checks pass, and the regressions fail against the base implementation. BR-1’s host evidence is verified. No additional findings; the separately recorded full suite remains required before landing.

```findings
dispose:
  - id: BR-1
    disposition: withdrawn
    note: |
      Verified all three retained log hashes and their passing results. The evidence-file hash matches the issue’s corrected 8a2add92… value. Executable source is unchanged from qualified head 5e299f5. Independent full consumer normal and race reruns also pass with TMPDIR=/tmp; the retained authoritative repolint result answers the missing-main-authority restriction.
```

1. **Strengths**

   - Acceptance is confined to the exact requested-name diagnostic and a validated complete listing ([osruntime.go:195](cmd/internal/launcher/osruntime.go#L195)).
   - The repair reuses strict identity, metadata, and duplicate validation ([zellijparse.go:70](cmd/internal/launcher/zellijparse.go#L70)).
   - Regression controls cover malformed, contradictory, failed, and cancelled replies ([native_name_absence_test.go:41](cmd/internal/launcher/native_name_absence_test.go#L41)).

2. **Critical findings:** None.

3. **Important findings:** None remaining.

4. **Minor findings:** None.

5. **Test coverage notes**

   - Full launcher/reply/session packages pass normally and under race detection.
   - Base-code overlay makes the captured-absence and contradictory-clients regressions fail; temporary overlay files were removed.
   - Committed fixtures match retained raw captures byte for byte. Fixture trailing whitespace preserves native output.

6. **Architectural notes**

   - **ARCH-DRY: pass** — existing listing validation is reused.
   - **ARCH-PURE: pass** — pure classification remains separate from injected process execution.
   - **ARCH-PURPOSE: pass** — creation, Couch assignment, and name calibration share the repaired probe.
   - Atlas is updated. No new command, flag, or configuration surface requires a README change.

7. **Plan revision recommendations:** None. The recorded full-suite and landing/reporting steps remain outstanding workflow requirements.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"d6e1c965-d23c-438d-acf9-94acdfe24252","repository":"github.com/xianxu/pair","issue_id":"000672","recipe":"small-diff-review","round":2,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/84d3ad2caffd39f1","reviewed_head":"dd8355c685a473acd7ba76a1d803cf36a6c7898b","slot":"pair:1","requested_at":"2026-10-18T18:42:46.232683Z"},"phase":"finished","pid":42980,"started_at":"2026-10-18T18:42:46.43068Z","finished_at":"2026-10-18T18:46:40.653192Z","session_id":"01a15052-cfcf-79c3-8ed4-5446ed4dbd06","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.199583541s
head: "dd8355c685a473acd7ba76a1d803cf36a6c7898b"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
