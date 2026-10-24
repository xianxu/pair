---
id: 000861
status: working
deps: []
github_issue:
created: 2026-10-24
updated: 2026-10-24
estimate_hours:
card_mirror: 'e5904d2c6f5b195f7b39a4fc8d5d7ff946dfa712' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-24T00:02:47-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
---

# Adopt Ariadne shared sidedoc pin and qualify newly active lint rules

## Problem

Pair#838 requires the landed shared sidedoc API from ariadne#575, #576 and #581. Main still pins ariadne 81c832d5; the required pin 5a17f789619d also activates newer golint rules and changes Rule.Match to accept *CheckContext. This blocks #838 pre-close and its joint operator stage.

## Spec

Adopt the newer Ariadne substrate on its own main-based issue branch. Preserve the shared catalog and fix or individually justify every existing source finding; do not disable rules wholesale or blindly annotate suppressions. Adapt Pair's AST-only catalog matchers and direct sleep matcher to the contextual API. Refresh affected-reader certificates only after auditing each changed reader. The qualification fixture must preserve the upstream planner dependency sources and same-snapshot module metadata.

## Done when

- The normal module pin includes landed ariadne#581 (5a17f789619d3dcaafec62ae51b350833836a3e7), with no fork or replace.
- Rule.Match API integration compiles and the full Pair catalog passes with all 51 observations resolved or precisely justified by evidence.
- Affected qualification and reader inventory pass; pre-close, mandatory review and landing pass.
- #838 can merge the landed adoption without an independent broad lint cleanup.

## Plan

Durable source-qualified plan: workshop/plans/000861-ariadne-pin-lint-adoption-plan.md.

- [ ] Adopt normal pin and contextual catalog boundary; qualify real upstream planner fixture.
- [ ] Consolidate three JSON readers with preserved owner contracts; retain two authorized independent-role exceptions pending ariadne#590.
- [ ] Reuse canonical ShortRoot; preserve ordinary inherited TMPDIR and optional startup socket isolation.
- [ ] Replace virtual budget shortcuts and native expiry races with explicit observed transitions and cue-owned deadlines/timers.
- [ ] Audit readers, pass catalog/targeted/pre-close, update atlas, close and land; then return #838.

## Log

Discovered by #838 retained pre-close attempt 5966ebc327601a4b5febe2c1bdf4ed3f on pushed head 8448a507ba579038d83eadf15ab70810bc8a030f. Full suite reached all Go packages and failed in repolint compile plus a separate #838 stale deleted-file artifact manifest entry. Raw suite log SHA256 69149fe9335d1b664e172bc0a7bcd1482b04a33bb54c68f828ab00d279f0d9c0. The artifact manifest error belongs to #838, not this adoption.

After the minimal contextual callback adapter, targeted TestTrackedGoFilesPassCatalog reports exactly 51 findings: test-shrunk-deadline 40; test-socket-fixture-root 5; json-recursive-scanner 5; tmp-literal 1. Raw capture currently /tmp/pair838-catalog-pin-findings.log; #838 will retain it in workshop/evidence/000838/catalog-pin-findings.log. Findings follow verbatim for owner qualification.

```text
--- FAIL: TestTrackedGoFilesPassCatalog (6.01s)
    catalog_test.go:465: checked 3451 tracked Go sources, 33 catalog rules plus Pair native-fixture-roots; 100 exact sleep debt sites
    catalog_test.go:467: cmd/internal/broadcast/cloudflared_test.go:187: test-shrunk-deadline: a sub-second deadline races fixture startup under load; fire it on cue after readiness (testfix.DeadlineOnCue), or suppress where the test only asserts the bound (ariadne#434, ariadne#514)
    catalog_test.go:467: cmd/internal/broadcast/session_test.go:177: test-shrunk-deadline: a sub-second deadline races fixture startup under load; fire it on cue after readiness (testfix.DeadlineOnCue), or suppress where the test only asserts the bound (ariadne#434, ariadne#514)
    catalog_test.go:467: cmd/internal/couchcmd/owned_query_test.go:106: test-socket-fixture-root: an inherited temporary root can exceed the Unix socket path limit; use a short canonical private socket root with immediate cleanup, and justify any tmp-literal exception (ariadne#549, pair#792)
    catalog_test.go:467: cmd/internal/couchcore/operationdispatch_test.go:121: test-shrunk-deadline: a sub-second deadline races fixture startup under load; fire it on cue after readiness (testfix.DeadlineOnCue), or suppress where the test only asserts the bound (ariadne#434, ariadne#514)
    catalog_test.go:467: cmd/internal/couchcore/park_wait_test.go:34: test-shrunk-deadline: a sub-second deadline races fixture startup under load; fire it on cue after readiness (testfix.DeadlineOnCue), or suppress where the test only asserts the bound (ariadne#434, ariadne#514)
    catalog_test.go:467: cmd/internal/couchcore/provision_subprocess_test.go:53: test-shrunk-deadline: a sub-second deadline races fixture startup under load; fire it on cue after readiness (testfix.DeadlineOnCue), or suppress where the test only asserts the bound (ariadne#434, ariadne#514)
    catalog_test.go:467: cmd/internal/couchcore/startup_detach_timing_live_test.go:156: tmp-literal: a hardcoded temp path (/tmp, /private/tmp, /var/folders, /Volumes/AgentScratch) ignores TMPDIR, which the macOS sandbox requires and which points suites at the TestScratch RAM disk; use os.TempDir, t.TempDir or os.MkdirTemp("", …) (ariadne#342, ariadne#468)
    catalog_test.go:467: cmd/internal/historyarchive/drive_model.go:168: json-recursive-scanner: a hand-rolled recursive duplicate-key JSON scanner repeats jsonstrict's rule and drifts from it; call jsonstrict.RejectDuplicateKeys or jsonstrict.Scan (cmd/sdlc/internal/jsonstrict), which folds keys as encoding/json matches them (ariadne#489)
    catalog_test.go:467: cmd/internal/launcher/recovery_terminal_confirmed_test.go:304: test-shrunk-deadline: a sub-second deadline races fixture startup under load; fire it on cue after readiness (testfix.DeadlineOnCue), or suppress where the test only asserts the bound (ariadne#434, ariadne#514)
    catalog_test.go:467: cmd/internal/launcher/recovery_terminal_test.go:245: test-shrunk-deadline: a sub-second deadline races fixture startup under load; fire it on cue after readiness (testfix.DeadlineOnCue), or suppress where the test only asserts the bound (ariadne#434, ariadne#514)
    catalog_test.go:467: cmd/internal/layoutcmd/fullscreen_reply_live_test.go:860: test-socket-fixture-root: an inherited temporary root can exceed the Unix socket path limit; use a short canonical private socket root with immediate cleanup, and justify any tmp-literal exception (ariadne#549, pair#792)
    catalog_test.go:467: cmd/internal/sessionwatch/peer_turn_test.go:62: test-shrunk-deadline: a sub-second deadline races fixture startup under load; fire it on cue after readiness (testfix.DeadlineOnCue), or suppress where the test only asserts the bound (ariadne#434, ariadne#514)
    catalog_test.go:467: cmd/internal/sessionwatch/probation_test.go:47: test-shrunk-deadline: a sub-second deadline races fixture startup under load; fire it on cue after readiness (testfix.DeadlineOnCue), or suppress where the test only asserts the bound (ariadne#434, ariadne#514)
    catalog_test.go:467: cmd/internal/sessionwatch/probation_test.go:73: test-shrunk-deadline: a sub-second deadline races fixture startup under load; fire it on cue after readiness (testfix.DeadlineOnCue), or suppress where the test only asserts the bound (ariadne#434, ariadne#514)
    catalog_test.go:467: cmd/internal/sessionwatch/probation_test.go:110: test-shrunk-deadline: a sub-second deadline races fixture startup under load; fire it on cue after readiness (testfix.DeadlineOnCue), or suppress where the test only asserts the bound (ariadne#434, ariadne#514)
    catalog_test.go:467: cmd/internal/sessionwatch/probation_test.go:144: test-shrunk-deadline: a sub-second deadline races fixture startup under load; fire it on cue after readiness (testfix.DeadlineOnCue), or suppress where the test only asserts the bound (ariadne#434, ariadne#514)
    catalog_test.go:467: cmd/internal/sessionwatch/run_test.go:43: test-shrunk-deadline: a sub-second deadline races fixture startup under load; fire it on cue after readiness (testfix.DeadlineOnCue), or suppress where the test only asserts the bound (ariadne#434, ariadne#514)
    catalog_test.go:467: cmd/internal/sessionwatch/run_test.go:167: test-shrunk-deadline: a sub-second deadline races fixture startup under load; fire it on cue after readiness (testfix.DeadlineOnCue), or suppress where the test only asserts the bound (ariadne#434, ariadne#514)
    catalog_test.go:467: cmd/internal/sessionwatch/run_test.go:209: test-shrunk-deadline: a sub-second deadline races fixture startup under load; fire it on cue after readiness (testfix.DeadlineOnCue), or suppress where the test only asserts the bound (ariadne#434, ariadne#514)
    catalog_test.go:467: cmd/internal/sessionwatch/run_test.go:319: test-shrunk-deadline: a sub-second deadline races fixture startup under load; fire it on cue after readiness (testfix.DeadlineOnCue), or suppress where the test only asserts the bound (ariadne#434, ariadne#514)
    catalog_test.go:467: cmd/internal/sessionwatch/run_test.go:344: test-shrunk-deadline: a sub-second deadline races fixture startup under load; fire it on cue after readiness (testfix.DeadlineOnCue), or suppress where the test only asserts the bound (ariadne#434, ariadne#514)
    catalog_test.go:467: cmd/internal/sessionwatch/run_test.go:439: test-shrunk-deadline: a sub-second deadline races fixture startup under load; fire it on cue after readiness (testfix.DeadlineOnCue), or suppress where the test only asserts the bound (ariadne#434, ariadne#514)
    catalog_test.go:467: cmd/internal/sessionwatch/run_test.go:465: test-shrunk-deadline: a sub-second deadline races fixture startup under load; fire it on cue after readiness (testfix.DeadlineOnCue), or suppress where the test only asserts the bound (ariadne#434, ariadne#514)
    catalog_test.go:467: cmd/internal/sessionwatch/run_test.go:520: test-shrunk-deadline: a sub-second deadline races fixture startup under load; fire it on cue after readiness (testfix.DeadlineOnCue), or suppress where the test only asserts the bound (ariadne#434, ariadne#514)
    catalog_test.go:467: cmd/internal/strictjson/decode_equivalence_test.go:46: json-recursive-scanner: a hand-rolled recursive duplicate-key JSON scanner repeats jsonstrict's rule and drifts from it; call jsonstrict.RejectDuplicateKeys or jsonstrict.Scan (cmd/sdlc/internal/jsonstrict), which folds keys as encoding/json matches them (ariadne#489)
    catalog_test.go:467: cmd/internal/submission/privacy.go:221: json-recursive-scanner: a hand-rolled recursive duplicate-key JSON scanner repeats jsonstrict's rule and drifts from it; call jsonstrict.RejectDuplicateKeys or jsonstrict.Scan (cmd/sdlc/internal/jsonstrict), which folds keys as encoding/json matches them (ariadne#489)
    catalog_test.go:467: cmd/internal/testcouch/actor_cancel_test.go:33: test-socket-fixture-root: an inherited temporary root can exceed the Unix socket path limit; use a short canonical private socket root with immediate cleanup, and justify any tmp-literal exception (ariadne#549, pair#792)
    catalog_test.go:467: cmd/internal/testcouch/agentfixture/cli_group_test.go:24: test-shrunk-deadline: a sub-second deadline races fixture startup under load; fire it on cue after readiness (testfix.DeadlineOnCue), or suppress where the test only asserts the bound (ariadne#434, ariadne#514)
    catalog_test.go:467: cmd/internal/testcouch/agentfixture/request_lifetime_test.go:148: test-shrunk-deadline: a sub-second deadline races fixture startup under load; fire it on cue after readiness (testfix.DeadlineOnCue), or suppress where the test only asserts the bound (ariadne#434, ariadne#514)
    catalog_test.go:467: cmd/internal/testcouch/agentfixture/run_test.go:34: test-socket-fixture-root: an inherited temporary root can exceed the Unix socket path limit; use a short canonical private socket root with immediate cleanup, and justify any tmp-literal exception (ariadne#549, pair#792)
    catalog_test.go:467: cmd/internal/testcouch/agentfixture/run_test.go:436: test-shrunk-deadline: a sub-second deadline races fixture startup under load; fire it on cue after readiness (testfix.DeadlineOnCue), or suppress where the test only asserts the bound (ariadne#434, ariadne#514)
    catalog_test.go:467: cmd/internal/testcouch/drive_observation_test.go:64: test-socket-fixture-root: an inherited temporary root can exceed the Unix socket path limit; use a short canonical private socket root with immediate cleanup, and justify any tmp-literal exception (ariadne#549, pair#792)
    catalog_test.go:467: cmd/internal/vmcmd/runtime_test.go:141: test-shrunk-deadline: a sub-second deadline races fixture startup under load; fire it on cue after readiness (testfix.DeadlineOnCue), or suppress where the test only asserts the bound (ariadne#434, ariadne#514)
    catalog_test.go:467: cmd/internal/webcast/drive/drawing_test.go:422: test-shrunk-deadline: a sub-second deadline races fixture startup under load; fire it on cue after readiness (testfix.DeadlineOnCue), or suppress where the test only asserts the bound (ariadne#434, ariadne#514)
    catalog_test.go:467: cmd/internal/wrapcmd/harness_tty_live_test.go:324: test-shrunk-deadline: a sub-second deadline races fixture startup under load; fire it on cue after readiness (testfix.DeadlineOnCue), or suppress where the test only asserts the bound (ariadne#434, ariadne#514)
    catalog_test.go:467: cmd/internal/wrapcmd/harness_tty_live_test.go:325: test-shrunk-deadline: a sub-second deadline races fixture startup under load; fire it on cue after readiness (testfix.DeadlineOnCue), or suppress where the test only asserts the bound (ariadne#434, ariadne#514)
    catalog_test.go:467: cmd/internal/wrapcmd/harness_tty_live_test.go:341: test-shrunk-deadline: a sub-second deadline races fixture startup under load; fire it on cue after readiness (testfix.DeadlineOnCue), or suppress where the test only asserts the bound (ariadne#434, ariadne#514)
    catalog_test.go:467: cmd/internal/wrapcmd/harness_tty_live_test.go:342: test-shrunk-deadline: a sub-second deadline races fixture startup under load; fire it on cue after readiness (testfix.DeadlineOnCue), or suppress where the test only asserts the bound (ariadne#434, ariadne#514)
    catalog_test.go:467: cmd/internal/wrapcmd/harness_tty_live_test.go:366: test-shrunk-deadline: a sub-second deadline races fixture startup under load; fire it on cue after readiness (testfix.DeadlineOnCue), or suppress where the test only asserts the bound (ariadne#434, ariadne#514)
    catalog_test.go:467: cmd/internal/wrapcmd/harness_tty_live_test.go:367: test-shrunk-deadline: a sub-second deadline races fixture startup under load; fire it on cue after readiness (testfix.DeadlineOnCue), or suppress where the test only asserts the bound (ariadne#434, ariadne#514)
    catalog_test.go:467: cmd/internal/wrapcmd/harness_tty_live_test.go:389: test-shrunk-deadline: a sub-second deadline races fixture startup under load; fire it on cue after readiness (testfix.DeadlineOnCue), or suppress where the test only asserts the bound (ariadne#434, ariadne#514)
    catalog_test.go:467: cmd/internal/wrapcmd/harness_tty_live_test.go:390: test-shrunk-deadline: a sub-second deadline races fixture startup under load; fire it on cue after readiness (testfix.DeadlineOnCue), or suppress where the test only asserts the bound (ariadne#434, ariadne#514)
    catalog_test.go:467: cmd/internal/wrapcmd/harness_tty_live_test.go:424: test-shrunk-deadline: a sub-second deadline races fixture startup under load; fire it on cue after readiness (testfix.DeadlineOnCue), or suppress where the test only asserts the bound (ariadne#434, ariadne#514)
    catalog_test.go:467: cmd/internal/wrapcmd/harness_tty_live_test.go:452: test-shrunk-deadline: a sub-second deadline races fixture startup under load; fire it on cue after readiness (testfix.DeadlineOnCue), or suppress where the test only asserts the bound (ariadne#434, ariadne#514)
    catalog_test.go:467: cmd/internal/wrapcmd/harness_tty_live_test.go:478: test-shrunk-deadline: a sub-second deadline races fixture startup under load; fire it on cue after readiness (testfix.DeadlineOnCue), or suppress where the test only asserts the bound (ariadne#434, ariadne#514)
    catalog_test.go:467: cmd/internal/wrapcmd/harness_tty_live_test.go:504: test-shrunk-deadline: a sub-second deadline races fixture startup under load; fire it on cue after readiness (testfix.DeadlineOnCue), or suppress where the test only asserts the bound (ariadne#434, ariadne#514)
    catalog_test.go:467: cmd/internal/wrapcmd/harness_tty_live_test.go:534: test-shrunk-deadline: a sub-second deadline races fixture startup under load; fire it on cue after readiness (testfix.DeadlineOnCue), or suppress where the test only asserts the bound (ariadne#434, ariadne#514)
    catalog_test.go:467: cmd/internal/wrapcmd/harness_tty_live_test.go:535: test-shrunk-deadline: a sub-second deadline races fixture startup under load; fire it on cue after readiness (testfix.DeadlineOnCue), or suppress where the test only asserts the bound (ariadne#434, ariadne#514)
    catalog_test.go:467: cmd/internal/wrapcmd/peer_live_tall_test.go:133: test-shrunk-deadline: a sub-second deadline races fixture startup under load; fire it on cue after readiness (testfix.DeadlineOnCue), or suppress where the test only asserts the bound (ariadne#434, ariadne#514)
    catalog_test.go:467: examples/mock-to-real/internal/jsonvalue/value.go:28: json-recursive-scanner: a hand-rolled recursive duplicate-key JSON scanner repeats jsonstrict's rule and drifts from it; call jsonstrict.RejectDuplicateKeys or jsonstrict.Scan (cmd/sdlc/internal/jsonstrict), which folds keys as encoding/json matches them (ariadne#489)
    catalog_test.go:467: internal/repolint/catalog_test.go:164: json-recursive-scanner: a hand-rolled recursive duplicate-key JSON scanner repeats jsonstrict's rule and drifts from it; call jsonstrict.RejectDuplicateKeys or jsonstrict.Scan (cmd/sdlc/internal/jsonstrict), which folds keys as encoding/json matches them (ariadne#489)
FAIL
FAIL	github.com/xianxu/pair/internal/repolint	6.076s
FAIL

```

A fixture-only repair is already pushed as b3da4973ef63b6a1af8ab941721abde8f3038faf on #838: four unmodified provider packages (affected/jsonstrict/processgroup/pkg-procgroup), root module go.mod/go.sum, all sibling/exact-ref/installed-module paths PASS (778 group/file inputs each). Reuse the qualified source change via the hub; do not stack an issue branch on unlanded #838. The callback adapter is still uncommitted in pair:3, available as a patch on request.

## Revisions

### 2026-10-24 — assigned adoption and source-qualified design

Hub a942ac93 assigned this slot to land adoption first. Source qualification splits40 deadline observations into13 virtual-clock,6 selection/wait-stub and21 real-timer/native sites. Hub f9c009c1 authorizes exactly the independent fuzz oracle and standalone DOM materializer per-site allows until ariadne#590 fixes role classification. Every other observation gets its cause repaired. Design is in the durable plan; no implementation or estimate yet.

### 2026-10-24 — plan approved; PQ-1 refined

Authenticated hub d3628148 approves the durable plan. First change-code review found only PQ-1 function-scoped-test-strategy; durable plan now names each risky function, the cue API, independent adversarial oracle and mechanical assertions. Re-running plan-quality before implementation.
