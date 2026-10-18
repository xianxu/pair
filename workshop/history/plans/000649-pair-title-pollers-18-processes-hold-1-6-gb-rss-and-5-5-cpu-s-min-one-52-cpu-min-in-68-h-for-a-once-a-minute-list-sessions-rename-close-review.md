# Boundary Review — pair#649 (whole-issue close)

| field | value |
|-------|-------|
| issue | 649 — pair title pollers: 18 processes hold 1.6 GB RSS and ~5.5 CPU-s/min (one 52 CPU-min in 68 h) for a once-a-minute list-sessions + rename |
| repo | pair |
| issue file | workshop/issues/000649-pair-title-pollers-18-processes-hold-1-6-gb-rss-and-5-5-cpu-s-min-one-52-cpu-min-in-68-h-for-a-once-a-minute-list-sessions-rename.md |
| boundary | whole-issue close |
| milestone | — |
| window | 3851b7338ff107d9b676d1db4eda7c7f21a2aa62..52da7b8eb248aeecafb3035f62958a5549c76529 |
| command | sdlc close --issue 649 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-18T03:34:57-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The hosted-process removal, shared rendering rules, cancellation, and confirmed-write caching are well implemented. Focused tests and race checks pass. Two gaps remain: titles omit live wrapper activity, and the required retained-memory regression bypasses the production compact projection.

```findings
findings:
  - id: new
    severity: Critical
    kind: defect
    family: shared-activity-consumer-consistency
    title: |
      Title decisions omit wrapper activity merged into idle fading
    detail: |
      cmd/internal/couchcmd/titles.go:168-172 computes frame eligibility and cmux heat from facts.Activity before cmd/internal/couchtty/console_activity.go:177-185 merges wrapper work. During silent active/background work, fading stays current while frame updates stop after two minutes and cmux heat can age incorrectly. The plan explicitly requires the final activity timestamp for both consumers. Inject admitted wrapper activity before title decisions and add a regression asserting both emitted titles and fading with stale persisted activity (ARCH-PURPOSE).
  - id: new
    severity: Important
    kind: defect
    family: production-path-resource-verification
    title: |
      The 1,000-tick heap regression never constructs a compact projection
    detail: |
      cmd/internal/couchcmd/titles_test.go:211-213 replaces production IO with fakeCouchTitles, whose source returns constant facts without constructing TitleProjection or reading native history. The heap assertion therefore cannot detect retained projection/cache growth, and this workload has no history-work counters. Exercise real compact projections over disposable file-backed fixtures for 1,000 ticks, including growth and generation turnover, while asserting retained heap, cache retirement, warm history work, and command counts (ARCH-CONSTRAINTS, ARCH-PURPOSE).
```

1. **Strengths**

   - Hosted admission exits before pipe/process allocation; fixtures cover either hosted marker.
   - `TitleUpdateState` confirms only matching successful effects and rejects stale completions.
   - Inventory replacement cancels obsolete activity work; snapshot cloning protects private target metadata.
   - README and atlas updates describe hosted ownership, standalone compatibility, and rollout limitations.

2. **Critical findings**

   - `titles.go:168`: merge wrapper activity before frame/heat decisions. The existing Console regression verifies fading only, so it misses this divergence.

3. **Important findings**

   - `titles_test.go:211`: extend the heap workload through real projections. Existing projection retained-state tests and the 20-pass OS fixture provide useful complementary coverage, but do not establish the specified combined 1,000-tick invariant.

4. **Minor findings**

   - None.

5. **Test coverage notes**

   - Passed focused title/current-launch/session-presence tests across `couchcmd`, `couchtty`, `couchcore`, `titlepoller`, `sessioninventory`, and `wrapcmd`.
   - Passed title/current-launch race tests across `couchcmd`, `couchtty`, and `sessioninventory`.
   - `git diff --check` passed; checkout remained unchanged.
   - Live rollout verification remains explicitly deferred by the documented operator decision.

6. **Architectural notes**

   - **ARCH-DRY — pass:** rendering and ownership rules are shared.
   - **ARCH-PURE — pass:** title decisions and confirmation transitions are tested directly without IO; external effects use injected seams.
   - **ARCH-PURPOSE — flag:** shared activity semantics and resource acceptance remain incomplete as described above.
   - **ARCH-MOCK — pass:** injected owner/failure behavior and disposable OS fixtures cover external boundaries, subject to the resource-test gap.
   - **ARCH-CONSTRAINTS — flag:** bounded passes and rotation are present; production-path retained-memory evidence is incomplete.
   - **ARCH-SECURE — pass:** private authority stays out of JSON; metadata reads are bounded and commands use argv.
   - **ARCH-ORDER — pass:** generation fencing, cancellation, uncertain outcomes, and matching confirmations have explicit handling.
   - **ARCH-FUNERAL — pass:** replaced entries retire, workers join shutdown, and metadata publication reuses existing cleanup ownership.

7. **Plan revision recommendations**

   Add `## Revisions` entries recording the wrapper-activity integration correction and the replacement of the fake-facts heap workload with production compact-projection coverage. Reconcile completion claims after those regressions pass.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"37005df4-831d-4c83-808b-90c0b85a747b","repository":"github.com/xianxu/pair","issue_id":"000649","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/153fcd4112efa65a","reviewed_head":"52da7b8eb248aeecafb3035f62958a5549c76529","slot":"pair:6","requested_at":"2026-10-18T10:32:08.751237Z"},"phase":"finished","pid":89695,"started_at":"2026-10-18T10:32:08.987902Z","finished_at":"2026-10-18T10:34:56.303812Z","session_id":"01a14e91-a1ed-77d0-ad7a-f6b19888a995","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.947108875s
head: "52da7b8eb248aeecafb3035f62958a5549c76529"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-18T04:17:37-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 649 — pair title pollers: 18 processes hold 1.6 GB RSS and ~5.5 CPU-s/min (one 52 CPU-min in 68 h) for a once-a-minute list-sessions + rename |
| repo | pair |
| issue file | workshop/issues/000649-pair-title-pollers-18-processes-hold-1-6-gb-rss-and-5-5-cpu-s-min-one-52-cpu-min-in-68-h-for-a-once-a-minute-list-sessions-rename.md |
| boundary | whole-issue close |
| milestone | — |
| window | 3851b7338ff107d9b676d1db4eda7c7f21a2aa62..0160140172a8a79674f9d14d3c595906ccbfca7e |
| command | sdlc close --issue 649 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-18T04:17:37-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned range implements the hosted-poller replacement and addresses both prior findings. Targeted tests, race checks, artifact checks, and regression fault injections passed verification. No blocking findings remain. Live rollout and fleet measurements remain explicitly deferred to the hub at the next operator restart.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Console snapshots wrapper activity before the combined pass; titles.go:167-181 merges it before frame, heat, and fading outputs. The regression passes at head and fails with that merge removed in a disposable checkout.
  - id: BR-2
    disposition: addressed
    note: |
      The 1,000-tick test now exercises two file-backed compact projections, transcript growth, and ten launch turnovers. It verifies heap, projection retirement, warm history work, and command counts. Injecting projection recreation makes it fail on tick 1.
```

1. **Strengths**

   - Shared activity reaches both title and fade consumers before decisions (`cmd/internal/couchcmd/titles.go:167`).
   - Exact launch authority is rechecked before effects; obsolete completions cannot confirm replacement state (`cmd/internal/couchcore/title_target.go:84`).
   - The resource regression exercises real projections and command execution over disposable fixtures (`cmd/internal/couchcmd/titles_test.go:200`).
   - README and atlas explain hosted ownership, standalone compatibility, cancellation, and rollout limitations.

2. **Critical findings**

   None.

3. **Important findings**

   None.

4. **Minor findings**

   None.

5. **Test coverage notes**

   - Fresh targeted tests passed across couchcmd, couchtty, couchcore, sessioninventory, titlepoller, and wrapcmd.
   - Targeted race tests passed for couchcmd, couchtty, and sessioninventory.
   - Artifact classification, managed IO anchors, and blocked SQLite cancellation checks passed.
   - The isolated resource workload recorded exactly 62 commands, two retained entries, and 940 stable ticks without history work. Final retained heap was 1,183,392 bytes versus a 1,331,960-byte warm baseline.
   - BR-1 failed without its production fix. BR-2 detected deliberately broken projection reuse.
   - The full suite was not rerun during this review. The review checkout remained unchanged; disposable mutation copies were removed.

6. **Architectural notes for upcoming work**

   - **ARCH-DRY — pass:** shared rendering and owner parsing serve both consumers.
   - **ARCH-PURE — pass:** title transitions remain IO-free; integration uses injected boundaries.
   - **ARCH-PURPOSE — pass:** hosted child suppression and the combined updater deliver the replacement.
   - **ARCH-MOCK — pass:** stateful doubles and disposable OS fixtures exercise the production seams; live conformance follows the documented deferral.
   - **ARCH-CONSTRAINTS — pass:** deadline, serial effects, fair rotation, and resource assertions enforce the declared envelope.
   - **ARCH-SECURE — pass:** generation checks, bounded metadata reads, private authority, and argv execution preserve boundaries.
   - **ARCH-ORDER — pass:** reducer tests cover failed, missing, late, and superseded outcomes; Console rejects obsolete activity results.
   - **ARCH-FUNERAL — pass:** generations are pruned, workers are cancelled and joined, and existing storage cleanup owns metadata retention.

7. **Plan revision recommendations**

   None. Existing revisions accurately describe the corrections and deferred live acceptance.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"e29ff72a-a518-4374-9980-6b1d79716dd4","repository":"github.com/xianxu/pair","issue_id":"000649","recipe":"milestone-review","round":2,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/a62b8366e3469799","reviewed_head":"0160140172a8a79674f9d14d3c595906ccbfca7e","slot":"pair:6","requested_at":"2026-10-18T11:15:33.153288Z"},"phase":"finished","pid":63599,"started_at":"2026-10-18T11:15:33.344258Z","finished_at":"2026-10-18T11:17:36.347902Z","session_id":"01a14eb9-5f0d-7f50-9876-3c00305bfd65","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.983467625s
head: "0160140172a8a79674f9d14d3c595906ccbfca7e"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
