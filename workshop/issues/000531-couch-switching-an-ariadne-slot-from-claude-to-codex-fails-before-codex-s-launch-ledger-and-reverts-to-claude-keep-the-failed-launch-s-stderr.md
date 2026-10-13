---
id: 000531
status: codecomplete
deps: []
github_issue:
created: 2026-10-13
updated: 2026-10-13
estimate_hours:
card_mirror: '0291c1356917378beb97b6c1f2eb316171e985d5' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-13T13:00:09-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: inferred}
actual_hours: 1.47
---

# couch: switching an ariadne slot from Claude to Codex fails before Codex's launch ledger and reverts to Claude; keep the failed launch's stderr

## Problem

An Ariadne Claude-to-Codex switch failed before any Codex launch ledger entry. The operator saw only `target launch failed; the source is still parked ... await Pair registration ... context canceled`; the target Pair output and exit evidence were not retained. Later inspection could not distinguish a failed helper from an interrupted registration wait. The current scope is diagnostic preservation, not an unproven Ariadne-specific fix.

## Spec

Retain bounded, attempt-scoped evidence for failed Couch launches, including the final switch-agent error, so failure remains inspectable after rollback, a Couch restart, or a later successful attempt. Capture the target helper's stderr in the ordinary runner, or combined stdout/stderr in a PTY (label it accurately), plus whether and when its exit was actually observed. A canceled wait alone must never claim that the helper exited or that startup failed at a particular stage.

Use the shared launch path and existing runner/PTY capture seams (ARCH-DRY). Keep diagnostics separate from authoritative thread lifecycle records so rollback cannot discard the evidence and older thread decoders are unaffected. Proposed bounded namespace store: newest 16 failed attempts, output tail at most 16 KiB and error at most 4 KiB per attempt, with an encoded-store bound. Exact thread address, attempt identity, target agent, stage/time and observed exit accompany the output. No environment dump or unbounded transcript. Record write failures must be reported without replacing the original launch error.

Immediate failure text and `couch --show` should identify the attempt and show readable error/exit/output evidence. Escape sequences from child output must not execute in the operator's terminal. Successful starts do not erase a previous failure; the bounded history evicts oldest failures. The diagnostic recorder must not take focus, change launch admission or retry automatically.

Only targeted tests while machine load is at least 40. No full suite until load is below 40. No further live reproduction; no changes to ariadne:1-4, ariadne:5, or pair:0 for this investigation. The independent start-slot agent-choice bug is #529 and is out of scope.

## Done when

- A target that prints an error and exits before registration leaves durable evidence containing its exact attempt identity, bounded output and observed exit code, including the switch's final error.
- Timeout and cancellation while the helper has not observably exited retain that uncertainty; cleanup-caused exit is distinguished from an exit observed before cleanup.
- Evidence survives launch rollback and re-opening the diagnostic store, and `couch --show` can inspect it without launching or focusing a slot.
- Duplicate updates enrich one attempt; concurrent or late updates cannot replace another attempt's evidence; storage and rendering remain bounded and sanitize terminal controls.
- Tests exercise real short-lived PTY and ordinary subprocesses plus the existing stateful launch fake, including pre-spawn refusal, output-before-exit, canceled wait, success, persistence failure and truncation.

- A setup failure after a helper spawned retains that helper's output and separate pre/post-cleanup exits even when StartBlocked returns no handle.
- Immediate failures include bounded attempt correlation, diagnostic location, one sanitized explanation, and exactly one recovery action; persistence errors preserve their original error chain.
- Local install creates links for the Go binaries, and an installed link follows a rebuilt checkout binary without reinstalling. No real ~/.local/bin mutation is required to verify this.

## Plan

- [x] Approve the diagnostic design and durable implementation plan.
- [x] Implement bounded capture, persistence and failure reporting through the shared launch seams.
- [x] Verify targeted regressions and documentation; attempt full suite below the load threshold, then proceed to close with the authorized #514 exception.

## Log

### 2026-10-13 — investigation
- 2026-10-13: closed — Round1 fixes: actual reader-close post-spawn subprocess retains stderr/exit23; reverting branch via overlay fails regression. Recorder partial-start and attempt/path correlation tests red before fixes, green after; errors.Is preserved. Combined targeted race tests passed core27.541s/cmd25.458s; install-link regression passed; main merged with docs-only additions. Full make test previously stopped at known #514 (11s vs6s), before Go suite; TL authorized this exception a521528a. No full-suite pass claimed or live/local-bin changes.; review verdict: SHIP
- 2026-10-13: flow upgraded quick → full — 797 added lines in code files (limit 100); an earlier round of this close already ran the full review

The switch form parses explicit argv, fingerprints the accepted preview, and passes an explicit FreshRequired launch profile through PAIR_COUCH_LAUNCH_PROFILE to Pair. Fresh launches bypass saved-config, repository-default and config-picker reads. Last successful preferences remain Claude when the replacement fails; that alone does not mean the supplied arguments were ignored.

Live Ariadne and Pair processes had identical PATH and PAIR_HOME. The supervisor resolved Pair from /Users/xianxu/workspace/pair/bin/pair and Codex from /opt/homebrew/bin/codex. Launch uses Go exec directly, so shell dev-alias functions and weave's build-only generated tool environment do not participate. Neither observed Couch process had PAIR_CODEX_ALT_SCREEN set.

Read-only inspection found seven agent-bearing numbered-Ariadne ledger rows, all Claude. This proves no Codex attempt reached AppendLedger, not that the Pair helper never ran. Archived source 1-ariadne-45 had successful Claude park at 2026-10-12T23:11:29.994936-07:00 and no retained target stderr. pair:2 reported a supervisor restart at 23:12:34 during that attempt. Cancellation is consistent with shutdown; the earlier wait remains unexplained. Today's attempt time was not captured; replacement Claude source48 started at 10:54.

Scratch tests used a copied real Pair binary (build revision 69a1e510f54ba4551df0d04015be4e408e54c0bb, modified=true), isolated HOME/data/assets and a git repo named ariadne. External zellij/agents were fakes. Both empty-history and existing-Claude/no-Codex-config cases reached the ledger. With exact operator argv `--sandbox danger-full-access`, existing Claude tag and exact Couch session-intent protocol, Pair exited 0 with empty stderr; final PAIR_AGENT_COMMAND was `codex --disable shell_snapshot --sandbox danger-full-access --no-alt-screen`. Both runtime flags survive explicit parameters. These tests establish the controlled pre-ledger path, not real-agent readiness. Raw investigation: /tmp/ariadne-switch-investigation.txt; scratch command capture: /private/var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair-switch-probe-mm8dnww1/exact-command.json.

The start/add-slot form separately has an uninitialized Agents inventory. Constructor-based diagnostic confirmed it; #529 was filed unclaimed with its own proposed fix. It is not the cause of the switch failure.

### 2026-10-13 — operator live non-reproduction

TL receipt c975c7b5-a1c6-4853-a408-f234fc4ac58a reports the operator created ariadne:5 and successfully switched Claude to Codex with exactly `--sandbox danger-full-access`. No agent-driven live test was performed. Earlier failures do not reproduce on demand; today's historical cause remains unknown.

## Revisions

### 2026-10-13 13:02 PDT — operator narrows scope after live success

Supersedes the initial two-part scope (diagnostics, then agent-driven live investigation in ariadne:5): deliver durable failed-launch output/exit and the switch's own error only. Record the operator's successful live attempt; no further live reproduction or speculative startup fix. Machine-load threshold for a full suite remains below 40.

### 2026-10-13 13:12 PDT — approved design and operator-facing failure rule

TL approved plan 2572f5a2 (receipt 2c92fd96-1536-4649-a9ed-6f5aa76f89b1). Every visible failure must state what happened in one line and name one action: retry, resume the old agent, or inspect with `couch --show <slot>`. Retained detail must not replace that concise recovery guidance. Added authorized install audit: identify the stale ~/.local/bin/couch provenance and change any repo install target from copying to linking; do not modify ~/.local/bin itself.

### 2026-10-13 — install provenance and implementation observations

The stale ~/.local/bin/couch is a regular file with birth/modification time Sep 30 18:06:50 and embedded revision 10c444b4829e9af3f2800af7396f2555903dfb49 (modified build). Makefile.local's supported install recipe copied GO_BINS using install -m755; that recipe dates to 14dc87970a (May 10). This matches an old make install but does not prove the actual invocation. The repo target now links Go binaries to the checkout; a temporary-prefix regression covers stale copies, old symlinks, spaces, repeated install, and rebuilt source replacement. ~/.local/bin was not changed.

The ordinary stderr tee changes isatty(2) to false. Its one-second drain budget bounds descendants retaining stderr after direct exit; later writes may see EPIPE. A real subprocess test measured exit visibility around 1.034 s with an inherited descriptor held for 10 s. PTY capture reuses its existing ring. While exercising direct exit, targeted tests found FakeRunner.Wait waited for process-group death even after Exited closed; the fake is being corrected to match the real direct-child contract. Existing timing-detail tests now inspect retained LaunchError, since the operator-visible error is intentionally one line.

### 2026-10-13 — targeted verification

Combined targeted race run passed for launch failures/output, switch and cold-resume paths, show resolution and rendering: GOMAXPROCS=2 go test -p 1 ./cmd/internal/couchcore ./cmd/internal/couchcmd -run 'Test(LaunchFailure|LaunchOutput|ShowLaunchFailure|RenderLaunchFailures|RenderShowIncludesLaunchFailures|SwitchAgent|UncertainSwitch|ColdResume)' -race -count=1 -timeout=180s (core 14.206 s, cmd 11.472 s). Temporary-prefix install link regression passed; git diff --check clean. An earlier broad targeted run was stopped after finding the fake direct-child Wait mismatch, now fixed and covered; timing assertions now verify retained detail instead of truncated one-line UI. Real PTY process-group conformance failed once on background-child timing and passed isolated rerun; no live slots changed.

### 2026-10-13 13:24 PDT — full-suite attempt blocked by host-sensitive performance check

After one-minute load fell to 37.17, started the machine-gated full command `GOMAXPROCS=2 GOFLAGS=-p=2 make test`, log /tmp/pair531-full-suite.log. It exited 2 at unchanged doctor/perf_test.sh: capture took 11 s against a 6 s budget. Earlier shell/Neovim checks passed; the Go suite was not reached. Load rose to 125.86 during the run and was 105.25 at completion. No second full suite started while above the operator's threshold. Targeted combined race and isolated installer checks remain passing; full validation, boundary review and landing are pending a host-load window below 40. Main merged at 6f02ec4bb; only merge conflict was this issue's previously published text, resolved to retain the newer approved record. No live slots or ~/.local/bin changed.

### 2026-10-13 13:41 PDT — close authorized with #514 exception

TL receipt a521528a-f421-4f8c-ae55-6b12485aa4ed authorizes proceeding to close, citing #514 for the only failure observed in the full-suite attempt (11 s versus 6 s). This supersedes waiting for another below-40 window. The run stopped before the Go suite; do not call it a full passing run. Targeted combined race tests and installer regression are the passing evidence. Hub sends failed because its wrapper was briefly absent; TL recovered the report by peeking. For future not-live delivery failures, retry after a few minutes rather than ending work immediately.

### 2026-10-13 — boundary round 1 corrections

BR-1: preserve evidence across every post-spawn runner setup refusal (acknowledgment-reader close, priority setup and a starter returning both handle/error), before lower-level cleanup destroys it; transfer immutable snapshots through a typed error to the shared recorder. BR-2: immediate errors now require explicit bounded attempt identity and diagnostic location, including storage-failure location while retaining errors.Is/As. BR-3: Done when now explicitly covers the already-authorized installer links and single-action recovery contract. These clarify coverage of the approved scope; no new live repro or installation action is authorized.

### 2026-10-13 — round 1 fix verification

BR-1 actual reader-close post-spawn path tested with a real subprocess: stderr and exit23 survive. Reverting that branch via a Go overlay fails the regression (missing retained setup evidence). Separate regression for the shared recorder failed with HelperStarted=false/empty output before the fix and passes with pre/post exits and both runner-cleanup/rollback errors retained. BR-2 correlation tests failed for missing attempt/path on both successful/failed storage before the fix and pass afterward, preserving errors.Is for the launch and disk errors. BR-3 completion criteria now match authorized scope. Combined targeted race run after fixes passed: core27.541s, cmd25.458s; git diff --check clean. #514 full-suite exception remains in force; no repeat full suite run.
