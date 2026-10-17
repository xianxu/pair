---
id: 000626
status: working
deps: []
github_issue:
created: 2026-10-16
updated: 2026-10-17
estimate_hours:
card_mirror: 'fdc52484590ee531a166623d1c4ea90aaa027001' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-17T08:52:29-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "e50e865d", done: "6e40b30e"}
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

- [ ] Measure cold probe progress and CPU attribution; retain historical failure as unproven beyond available evidence.
- [ ] Add deterministic search-work regressions through the shared framer's delimiter-search seam, with long records, chunk boundaries, tails, offsets, early stops and limits.
- [ ] Make the existing shared framer search only appended bytes after a newline-free prefix (ARCH-DRY/ARCH-PURE); preserve all consumers and the existing 30s, twenty-owner workload and cold/warm accounting.
- [ ] Verify targeted packages, record a full suite on the pushed head, close review, and land.


## Log

2026-10-16: Filed unassigned from #622 after one package-run failure and one unchanged targeted pass. No related production or fixture change is included in #622; its recorded full suite remains queued.

### 2026-10-17 — Recovery and measured baseline

Authenticated TL dispatch 4bbef5d9 and claimed/start-planned in pair:6. The previous TTY is unavailable; branch/issue evidence is authoritative. Initial start-plan catch-up advanced only the slot-local ariadne dependency and automatically invoked Homebrew; interrupted the owned runner before dependency installation (Homebrew/tap metadata had auto-updated), reported to TL. Ordinary start-plan repeat succeeded.

Unchanged targeted invariant passed 9.33s. A second diagnostic profile passed 8.69s (profile /tmp/pair626-cold.cpu; executable /tmp/pair626-baseline-couchtty.test). Of 6.34 sampled CPU seconds, 4.37s (68.9%) was under frameJSONLArtifact, including 0.87s delimiter search and 3.47s ReadAt/metadata work. The fixture places almost30MB before the first newline: the shared framer searches the entire growing pending prefix every64KiB, approximately6.87GB searched per owner,137GB total. The two-second row context is checked only after the synchronous framing stage. These demonstrate unnecessary cold work; the historical failure has no per-row trace, so its exact scheduler/IO contribution remains unknown.

Quick plan sent to hub (b8bbd00e), proceeding unless objected. Alternatives: increasing a deadline violates the acceptance contract; changing cache observation authority would widen this fix and needs separate evidence/approval. Keep the metadata validator, probe deadlines and full workload unchanged; correct quadratic delimiter work in the shared reader and measure again. No new durable artifact family or external dependency is introduced.
