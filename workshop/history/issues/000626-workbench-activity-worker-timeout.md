---
id: 000626
status: done
deps: []
github_issue:
created: 2026-10-16
updated: 2026-10-17
estimate_hours:
card_mirror: 'a6a19541340707bf05d08432e81eaf2bd1812061' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-17T08:52:29-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "e50e865d", done: "6e40b30e"}
actual_hours: 0.32
---

# Workbench activity invariant intermittently times out waiting for cold worker

## Problem

During #622 M1 affected-package verification at head aaf17569ea4005aa59502de3a3e47ecacb01dada, `go test ./cmd/internal/couchcmd ./cmd/internal/couchtty ./cmd/internal/ptychild ./cmd/internal/couchcore ./cmd/internal/artifactpath` failed `TestWorkbenchPerformanceIdleUITick` at workbench_activity_invariant_test.go:77: `activity worker did not finish`. Test duration 50.43s; couchtty package 101.425s. Couchcmd passed 132.400s and ptychild passed 1.078s; the command continued through the remaining packages.

The unchanged targeted repeat `go test ./cmd/internal/couchtty -run '^TestWorkbenchPerformanceIdleUITick$' -count=1 -v` passed the read/decode accounting with test duration 30.16s. Retained repeat log: /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair622-activity-investigation.bQRoPX/current.log. Host load during investigation was 15.35/20.35/32.69.

This fixture creates no Child and never requests resources, so #622's new ingest/snapshot paths do not execute. Its activity/runtime/fixture source is unchanged against pre-counter head 242bd40216171712db036069ca5aae4f0c381366. The worker sequentially processes twenty 30MB ledgers, with per-row 2s probe deadlines; the test independently uses a fixed 30s completion guard. Contention is plausible but the exact timeout cause is unproven.

## Spec

Diagnose worker progress and deadline attribution in this fixture under normal package/suite load. Preserve its cold-read and unchanged-tick invariants; do not solve the report by merely increasing the timeout or adding retries. Keep failure evidence and distinguish lost work from expensive cold reads and scheduling delay.

## Done when

- Root cause is demonstrated with worker/probe progress evidence.
- A regression guards that cause while preserving the twenty-owner ledger accounting and unchanged UI-tick zero-read assertions.
- Package and recorded full-suite verification pass without blind retries or a larger arbitrary deadline.

## Plan

- [x] Measure cold probe progress and CPU attribution; retain historical failure as unproven beyond available evidence.
- [x] Add deterministic search-work regressions through the shared framer's delimiter-search seam, with long records, chunk boundaries, tails, offsets, early stops and limits.
- [x] Make the existing shared framer search only appended bytes after a newline-free prefix (ARCH-DRY/ARCH-PURE); preserve all consumers and the existing 30s, twenty-owner workload and cold/warm accounting.
- [x] Verify targeted packages and record a full suite on the pushed head.


## Log

2026-10-16: Filed unassigned from #622 after one package-run failure and one unchanged targeted pass. No related production or fixture change is included in #622; its recorded full suite remains queued.

### 2026-10-17 — Recovery and measured baseline
- 2026-10-17: closed — Counted delimiter regression RED at 6,888,539,021 searched bytes for 30,000,013 input, GREEN at exactly input size; boundaries/limits/tails and race pass. Full sessioninventory/threadactivity/workbenchperftest and couchtty pass. Recorded suite on pushed 31bc02234c023efaa3eaea609cb7bf1fb4429c24 passed first attempt in 11m40, log 658ad9099da7. The existing 20x30MB UI workload, 30s guard, 2s probe deadline and zero-read warm assertions are unchanged. Before/after CPU/probe traces retained; exact historical stall attribution remains unavailable as documented. Atlas and lesson updated. (done-when: contract moved since change-code; judged by the close review); review verdict: SHIP

Authenticated TL dispatch 4bbef5d9 and claimed/start-planned in pair:6. The previous TTY is unavailable; branch/issue evidence is authoritative. Initial start-plan catch-up advanced only the slot-local ariadne dependency and automatically invoked Homebrew; interrupted the owned runner before dependency installation (Homebrew/tap metadata had auto-updated), reported to TL. Ordinary start-plan repeat succeeded.

Unchanged targeted invariant passed 9.33s. A second diagnostic profile passed 8.69s (profile /tmp/pair626-cold.cpu; executable /tmp/pair626-baseline-couchtty.test). Of 6.34 sampled CPU seconds, 4.37s (68.9%) was under frameJSONLArtifact, including 0.87s delimiter search and 3.47s ReadAt/metadata work. The fixture places almost30MB before the first newline: the shared framer searches the entire growing pending prefix every64KiB, approximately6.87GB searched per owner,137GB total. The two-second row context is checked only after the synchronous framing stage. These demonstrate unnecessary cold work; the historical failure has no per-row trace, so its exact scheduler/IO contribution remains unknown.

Quick plan sent to hub (b8bbd00e), proceeding unless objected. Alternatives: increasing a deadline violates the acceptance contract; changing cache observation authority would widen this fix and needs separate evidence/approval. Keep the metadata validator, probe deadlines and full workload unchanged; correct quadratic delimiter work in the shared reader and measure again. No new durable artifact family or external dependency is introduced.

### 2026-10-17 — Red/green framing cost and progress

TL approved the exact quick plan (authenticated receipt793e6b6f); automatic package installation is tracked separately as ariadne#427. The deterministic regression ran RED with the unchanged search algorithm behind an injected delimiter function:65549 input bytes caused131085 examined bytes;524301 caused2883597;30000013 caused6888539021. GREEN after remembering the already-searched prefix: examined bytes equal input bytes for every size. Production still uses bytes.IndexByte through the same framer; limits apply to the entire record, not merely its new suffix. Explicit boundary, exact-chunk EOF/tail, multi-chunk tail, CR, offsets, finite/exact limit and early-stop cases pass. Existing ledger/fuzz-seed tests pass, complete sessioninventory/threadactivity/workbenchperftest packages pass (6.171/0.417/0.270s), and new regressions pass under race (1.292s).

Added per-probe elapsed/availability/context and generation-completion diagnostics to the existing UI invariant, preserving20 owners,30MB each,30s guard,2s production probe deadline and all read/decode assertions. A Go overlay restored only the original repeated-prefix search for the BEFORE measurement, without mutating the checkout. Both before/after completed120 probe calls across six generations with no context failures. Cold generation5.558s before vs2.969s after; mutation264ms vs139ms; unchanged generations approximately2ms. CPU samples at delimiter search450ms before vs40ms after. Total test6.07s vs3.40s. These wall/CPU samples include host and filesystem variability; deterministic searched-byte counts are the performance regression oracle, not a promised elapsed-time ratio. Metadata validation remains unchanged.

Retained diagnostics: /tmp/pair626-progress-before.log (SHA25650b582e105199fadb7b112b9083cefe1029ba3b99da29d85d3d05ef739d670d0), /tmp/pair626-progress-after.log (SHA2560699efa4aca354453f327bcc107f9fc76436a1d8cf436304c128f5806d20f35d), corresponding .cpu files and .test executables. The original historical timeout cannot be replayed exactly from retained evidence: no original per-row trace or failed-run profile exists here. We demonstrate and remove quadratic cold framing work rather than claim a proven historical scheduler stall. Shared-reader consumers use the same corrected function; FrameJSONLSuffix is a separate pure suffix API and has no per-read chunk loop. No timeouts, retry policies or cache authority changed.

### 2026-10-17 — Required suite

Recorded full suite on pushed31bc02234c023efaa3eaea609cb7bf1fb4429c24 passed first attempt,11m40s, sdlc-local/suite. Log: /Users/xianxu/workspace/pair/.git/sdlc/local-checks/31bc02234c023efaa3eaea609cb7bf1fb4429c24/suite.log; SHA256 658ad9099da7d0e52c07bdb7022d4dfbf5d93ea895385a49ee97cec5b9695afb. Full couchtty43.308s, sessioninventory7.377s, couchcore275.874s, wrapcmd49.507s; shell/editor integration passed. Separately, direct full couchtty passed35.403s. No retry, timeout increase, exclusion or production cache-authority change. Close review and landing are the next workflow checkpoints, not assertions already completed.

## Revisions

- 2026-10-17: Clarified the final Plan checkbox to cover verification only; close review and landing remain subsequent gated workflow actions. No behavior, scope or Done-when change.
