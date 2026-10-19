# Boundary Review — pair#702 (whole-issue close)

| field | value |
|-------|-------|
| issue | 702 — Compress large database snapshots faster with measured levels and threads |
| repo | pair |
| issue file | workshop/issues/000702-large-database-compression.md |
| boundary | whole-issue close |
| milestone | — |
| window | d30ee09da710898bfaa8cc5b680d882c1c74e4e9..aae7b22965fa393aeceabe5add04d2fe16f877ec |
| command | sdlc close --issue 702 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-19T05:14:14-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The implementation consistently carries the chosen compression descriptor through planning, encoding and published metadata. Targeted compatibility, cancellation, retry and integrity tests pass. Remaining gaps are the missing large-database single-thread comparison, stale operator documentation, and native CLI qualification that this review environment could not complete.

Done-when: current — its measurement, codec qualification, integrity and documentation clauses still cover the revised kind-based policy and descriptor compatibility.

```findings
findings:
  - id: new
    severity: Important
    family: required-comparison-coverage
    title: |
      The large-database benchmark omits the required single-thread baseline
    detail: |
      workshop/issues/000702-large-database-compression.md:30 requires size/time/CPU comparison with the present single-thread path. The table at lines 52–60 and retained results.json contain only -T0 database runs; --single-thread appears only for small text. This is the sole missing baseline in this comparison family. ARCH-PURPOSE: add a coordinated, bounded database -19 --single-thread measurement, recording a censored result explicitly if its bound expires; the earlier five-minute encrypted-pipeline timeout is not a comparable zstd-only measurement.
  - id: new
    severity: Important
    family: operator-contract-consistency
    title: |
      Operator documentation still describes the superseded compression policy
    detail: |
      All affected current-contract restatements are README.md:1986 (creation always uses -19), atlas/history-backup.md:147–148 (one compression thread), and atlas/storage-retention-io.md:154 (one compression thread). These contradict plan.go:96–99 and codec.go:191. Update the README to describe database -9 versus other-kind -19, and both atlas resource descriptions to explain -T0 while retaining the serial-bundle distinction.
  - id: new
    severity: Important
    kind: unverifiable
    family: native-qualification-evidence
    title: |
      Native encrypted round-trip qualification could not run in the review environment
    detail: |
      PAIR_ARCHIVE_LIVE=1 go test ./cmd/internal/historyarchive -run '^TestCodecLiveRoundTrip$' -count=1 -v failed at codec_live_test.go:41 before encoding: gpg reported “can't connect to the gpg-agent: File name too long” and key generation failed. This blocks verification of the sole native qualification entrypoint, including levels 9, 19, 20 and 22 and coordinator recovery checks. Provide measured passing output for the pinned code from an environment with a sufficiently short GPG socket path, or make that environment available for rerun.
```

1. **Strengths**

   - Compression selection remains pure in `plan.go`; subprocess work stays behind the injected `Codec`.
   - `archive.go:242` passes the manifest descriptor directly to encoding, with tests checking manifest/label agreement.
   - Shared parsing separates reader compatibility from encoder support; tests cover legacy, current, future and malformed descriptors, plus refusal before spawning tools.

2. **Critical findings:** None.

3. **Important findings:** The three findings above require disposition before close.

4. **Minor findings:** None.

5. **Test coverage notes:** Targeted codec, policy, mixed-catalog and stateful-fake tests passed. Additional archive, cancellation, retry, corruption and deduplication tests passed. Native qualification failed during environment setup; no full-suite pass was independently established.

6. **Architectural notes**

   - **ARCH-DRY: pass** — descriptor parsing is shared across readers and encoder.
   - **ARCH-PURE: pass** — deterministic selection and validation remain separate from IO.
   - **ARCH-PURPOSE: flag** — the required database baseline comparison remains incomplete.
   - **ARCH-EVOLVE: pass for this diff** — no JSON shape or version changes; legacy descriptors remain accepted, and reader acceptance is independent of writer policy. Existing unknown-field handling is unchanged.

7. **Plan revision recommendations:** Add a `## Revisions` entry scheduling the missing bounded baseline comparison and its evidence. Record native qualification as pending until measured passing evidence answers the environment failure. No Core concepts table was present to cross-check.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"fb2aff4a-5c82-4610-abc8-8b3be6db54d7","repository":"github.com/xianxu/pair","issue_id":"000702","recipe":"small-diff-review","round":1,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/655ecc4c133aea9e","reviewed_head":"aae7b22965fa393aeceabe5add04d2fe16f877ec","slot":"pair:7","requested_at":"2026-10-19T12:11:13.216796Z"},"phase":"finished","pid":2023,"started_at":"2026-10-19T12:11:13.316107Z","finished_at":"2026-10-19T12:12:59.740007Z","session_id":"01a15412-b130-71d2-83dc-5fa7740aa253","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.739148s
head: "aae7b22965fa393aeceabe5add04d2fe16f877ec"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000409

---

## Re-review — 2026-10-19T05:17:23-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 702 — Compress large database snapshots faster with measured levels and threads |
| repo | pair |
| issue file | workshop/issues/000702-large-database-compression.md |
| boundary | whole-issue close |
| milestone | — |
| window | d30ee09da710898bfaa8cc5b680d882c1c74e4e9..ebb6962cd0cbbd7e50572a79409281119b3fcd6c |
| command | sdlc close --issue 702 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-19T05:17:23-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The implementation carries the selected compression level consistently through planning, encoding and published metadata. Focused tests and native encrypted round trips pass. All prior findings are resolved under the documented scope revision; no new findings remain.

Done-when: current — its measurement, codec qualification, integrity and documentation clauses cover the revised kind-based policy.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      The explicit scope revision at workshop/issues/000702-large-database-compression.md:84–85 limits comparison to the measured -T0 cohort and acknowledges that the prior encrypted pipeline is not a matched baseline. Inspected results.json confirms database levels 3/9/19 and the text comparisons. This resolves the finding by revising scope; a matched database single-thread measurement remains unperformed.
  - id: BR-2
    disposition: addressed
    note: |
      Inspected before/after passages at README.md:1986, atlas/history-backup.md:147 and atlas/storage-retention-io.md:154. All three now describe the kind-specific levels and -T0 threading consistently with plan.go:96–99 and codec.go:192–198, retaining serial bundle compression.
  - id: BR-3
    disposition: addressed
    note: |
      Independently reran TMPDIR=/tmp PAIR_ARCHIVE_LIVE=1 go test ./cmd/internal/historyarchive -run '^TestCodecLiveRoundTrip$' -count=1 -v on pinned HEAD: passed in 6.163s. Levels 9/19/20/22 recovered identical plaintext; coordinator recovery, corruption rejection and source preservation passed. The retained host log also matches its documented SHA256.
```

1. **Strengths**

   - Pure policy selection stays in `plan.go:96`; subprocess work remains behind the injected codec.
   - `archive.go:242` passes the manifest descriptor directly into encoding; tests verify manifest/label agreement.
   - Shared descriptor parsing separates reader compatibility from encoder support, including refusal before spawning and `--ultra` handling.

2. **Critical findings:** None.

3. **Important findings:** None remaining.

4. **Minor findings:** None.

5. **Test coverage notes:** Focused policy, codec, archive, mixed-catalog, cancellation, retry and corruption tests passed for both packages. Native qualification passed separately. No repository-wide suite pass was independently established.

6. **Architectural notes**

   - **ARCH-DRY: pass** — descriptor validation is shared; production policy has one selection point.
   - **ARCH-PURE: pass** — deterministic policy and validation remain separate from IO.
   - **ARCH-PURPOSE: pass** — implementation and retained measurements fulfill the explicitly revised scope without claiming thread-scaling evidence.
   - **ARCH-EVOLVE: pass for this diff** — no fields or versions change; legacy descriptors remain readable. Existing unknown-field handling is unchanged.

7. **Plan revision recommendations:** None required. The scope limitation is explicit under `## Revisions`; no Core concepts table is present.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"cca4f5a1-a757-492a-aaf8-63ca2e89bb07","repository":"github.com/xianxu/pair","issue_id":"000702","recipe":"small-diff-review","round":2,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/3ef1d0f18fc05a81","reviewed_head":"ebb6962cd0cbbd7e50572a79409281119b3fcd6c","slot":"pair:7","requested_at":"2026-10-19T12:16:21.293458Z"},"phase":"finished","pid":86782,"started_at":"2026-10-19T12:16:21.658748Z","finished_at":"2026-10-19T12:17:22.340233Z","session_id":"01a15417-6653-7620-8254-fbfcc5b976dd","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.308142166s
head: "ebb6962cd0cbbd7e50572a79409281119b3fcd6c"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
