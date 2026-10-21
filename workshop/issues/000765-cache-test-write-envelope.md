---
id: 000765
status: codecomplete
deps: []
github_issue:
created: 2026-10-21
updated: 2026-10-21
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'faf2f11581d3f155a90f222f4f5ecc8b42f6bad6' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-21T03:09:03-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "6341887b", done: "80fbe5c1"}
actual_hours: 0.75
---

# Cache-enabled Go tests exceed physical write envelopes

## Problem

#739 close review observes different physical-write envelopes for plain cache-enabled go test and go test -count=1. At branch 1b948b3c040f6979eaafa77b1ad2da127f524da4, the reviewer measured couchcore 157257728 bytes against its 134217728 cap and couchtty 88002560 against its 67108864 cap using plain go test across five packages. The assertions pass; TestMain rejects the write counts. Both packages pass -count=1, as used by the required make suite. Earlier tty controls measured exactly 39800832 bytes with and without #739-added tests; new tests alone wrote zero.

The reviewer traced Go's cache-enabled filesystem-access logging as an invocation-dependent source of writes. Main-versus-branch cold/warm measurements are requested by the hub and pending; attribution is not yet proven. This issue is independent of #739's startup behavior and does not authorize any cap increase.

## Spec

Establish the supported invocation and physical-write accounting contract for ordinary package tests. Measure main and candidate in equivalent private cold and warm caches, distinguish test workload from cache instrumentation, and enforce the chosen contract consistently in test callers or accounting. Keep real workload writes bounded; do not silently waive a cache-enabled run or raise caps to conceal the difference. Coordinate the test-policy choice with the operator.

## Done when

- A controlled regression reproduces the invocation-dependent contribution and names whether it is pre-existing on main.
- Supported cache-enabled and uncached test modes have an explicit, enforced accounting contract.
- Real fixture-write regressions still fail the existing caps; no global cache clearing or cap increase is used as a fix.

## Plan

- [x] Red native trace-heavy fixture and real go-test cache conformance, then suppress tracing only at the shared ordinary supported-budget boundary.
- [x] Keep caps/repetition/failure semantics; qualify fresh cache-enabled and uncached core/tty runs and real excess refusal.
- [ ] Document result-cache policy, run recorded affected check, close and land.


## Log



- 2026-10-21: closed — Merge gate required manual integration and narrow review after tests/affected-go-readers.json conflict. Resolved only adjacent additions by preserving both original exact hashes: own cache-conformance entry and main rightpane-rungs live entry. No hash refresh, source policy choice or new feature code; inventory/whitespace pass. Integrated full suite passed10m31s per full-suite receipt; prior branch suite passed10m34s independently. Root policy, README/atlas and native/actual-Go conformance remain unchanged from reviewed SHIP code. Only remaining Plan checkbox includes landing, so remains honestly unchecked before merge receipt. Hub owns project. Re-enter normal merge after this required narrow review; no waiver of integration verification. (fixed-then-ship; full suite: Integrated recorded attempt4b7476afad2ec562731e223ce7e4f83b on415001017306f4fae8871e738a4198f68f8c556f PASSED exit0 in10m31s; log SHA prefixeb120773dc2a at .git/sdlc/local-checks/415001017306f4fae8871e738a4198f68f8c556f/suite.log. All scripts and Go packages passed, core189.558s tty38.182s testdisk8.296s.) (done-when: contract moved since change-code; judged by the close review); review verdict: SHIP
- 2026-10-21: closed — Native read-only cache-trace regression RED5353472>1MiB, GREEN; actual256KiB excess/status7 and pure accounting tests pass0.386s. Real cmd/go cache+coverage RED10.891s then GREEN10.738s. Plain cache-enabled core100298752/134217728 bytes passed248.720s and tty39800832/67108864 passed62.400s, full log SHA cd071fb6568ee9a4893021fcd4d8e5e785f269bed2b7a6894b944e8bdf20ebc7. BR-1 fixed: audited repo/module inputs, registered real conformance in fixed SOURCE_GUARDS and exact reader inventory; tracked inventory now passes. Removed unmatched lint suppression; corrected actual affected preflight passes49.776s incl conformance12.074s/catalog5.090s, selected testdisk11.345s; separate lint3.862s. No cap, counter, repetition or production behavior change. README/atlas updated. Superseded queued suite cancelled before test acquisition, new recorded affected check submitted on corrected final head7999245e2. Full uncached coverage and landing still pending; review overlap allowed, final recorded pass required before merge. Unchecked Plan items reflect pending recorded coverage and landing only. Hub owns project. (done-when: contract moved since change-code; judged by the close review); review verdict: SHIP
Filed unclaimed per ops:0's 91039d1a-2f91-4fef-94cb-461c59be997b. Evidence: #739 close-review round4 (2026-10-21 02:21 PDT), artifact workshop/plans/000739-couch-start-up-safe-mode-no-automatic-injection-into-the-hub-restart-kind-reported-not-acted-on-one-generic-start-notice-per-worker-close-review.md. The required recorded #739 suite is still completing and uses -count=1; its only reported behavioral failure so far is the separately in-scope installed-command startup assertion.


## Revisions

### 2026-10-21 — invocation envelope owner (ARCH-DRY)

Measured main cold/warm core143224832/143204352 and tty105287680/104529920
already exceed134217728/67108864 caps; assertions pass. Retained full-core
control has137310208 physical bytes,130916787 of Go cache trace. Branch control
is lower, so #739 attribution was withdrawn. Full logs/hashes remain under
/Volumes/TestScratch/tmp/pair-slot7/pair739-cache.QFbUSM.

Choose no test-result caching for ordinary native-budgeted runs: the shared
RunWithWriteBudget clears registered test.testlogfile before m.Run and says so.
Go's testing.go2813 opens/traces only when nonempty; cmd/go test.go2174 returns
before cache.Put on a missing trace. Build caching is unaffected. Existing
-count/-cpu repetitions, real fixture accounting/caps and failure statuses stay.
Benchmarks/fuzz and unsupported native counters retain their current policy.
A real tiny-module go-test twice fixture proves fresh execution, not a forged
empty dependency log. Controlled trace-heavy reads must fit without their cache
instrumentation; a real256KiB payload still exceeds a4KiB cap. No production
surface or format changes. O(1) flag setup, no scan or per-operation wrapper.

Alternative subtraction would mix logical file length with physical IO and
could hide fixture writes. A new subprocess owner would add lifecycle and
cache identity machinery for an existing shared TestMain seam. Neither needed.
README/atlas describe the intentional result-cache tradeoff. Quick flow;
headless acceptance, no staging or machine/cache clearing.


### 2026-10-21 — implementation and red/green evidence

The read-only native workload reproduced5353472 bytes against1MiB, exit1
in0.417s with all workload assertions passing. After the shared flag change,
trace exclusion plus existing real256KiB-under/over/failure-status and pure
accounting controls pass0.386s. Caps/repetitions and all workload counters are
unchanged. The policy is enforced once at the existing shared wrapper, not a
caller convention or new exemption list (ARCH-DRY).

Actual cmd/go conformance red10.891s: the unguarded control cached as expected,
but both guarded normal and coverage runs incorrectly returned cached results.
Green10.738s: control still caches (one marker byte), guarded plain and coverage
run twice (two marker bytes), real coverage profile contains the production
value.go function. Logs /tmp/pair765-cache-red.log and
/tmp/pair765-cache-green.log. No network or shared-cache clearing; isolated
private build/result cache uses the existing module cache read-only.

Fast make test-affected-inventory passes; catalog lint passes3.839s; whitespace
check passes. Plain cache-enabled full core/tty run is now running, log
/Volumes/TestScratch/tmp/pair-slot7/pair765-checks.BL7YLd/plain.log. No pass yet.
Native Go-cache conformance defends the Go-owned flag contract on upgrades;
no production API or persisted format changes. README and performance atlas
state the supported-result-cache tradeoff and unchanged coverage boundaries.


### 2026-10-21 — BR-1 inventory registration and plain-run proof

Plain cache-enabled go test core/tty passed: core100298752/134217728 bytes
in248.720s; tty39800832/67108864 in62.400s. Both diagnostics say result-cache
disabled, build cache unchanged. Log SHA256
cd071fb6568ee9a4893021fcd4d8e5e785f269bed2b7a6894b944e8bdf20ebc7
at /Volumes/TestScratch/tmp/pair-slot7/pair765-checks.BL7YLd/plain.log.

Close BR-1 found the new tracked conformance reader missing from inventory.
The earlier fast audit ran before git add and thus did not inspect the new
file. Audited inputs: repository go.sum, local module replacement and the Go
source reached by the real guarded import; generated fixture outputs/coverage
and markers are private. Register TestGoCacheWriteBudgetConformance in the
fixed scanner preflight and record precisely this file as fixed_scanner.
Indent the generated fixture's literal TestValue declaration so the lexical
inventory does not mistake it for a real test in the enclosing package; it
remains executed through actual Go in the private module. No scanner waiver.
Fast inventory now passes on the tracked file. Cancelled obsolete queued
attempt37fb730c9fd3c77230a42eba0028629c through local-check cancel before any
test capacity was acquired. It is not a passing/failing suite attempt.

Fixed-preflight execution then found the new conformance test had an unused
test-go-build suppression: its dynamic native test argv does not match that
rule. Removed the stale suppression and re-recorded the same reader scope.
The failed preflight is not counted as passing evidence.

Corrected fixed preflight passed exit0 (49.776s), including the registered
actual Go conformance12.074s and catalog5.090s; selected testdisk then passed
11.345s. Separate catalog repeat passed3.862s. No failed check is represented
as a pass. All remaining broad uncached coverage is owned by the recorded
full suite on the corrected head.


### 2026-10-21 — close round2 SHIP, recorded suite waiting

BR-1 addressed: independent review checked reader inventory and its mutation
control, testdisk regression/mutation and real Go cache conformance. It also
ran uncached core/tty successfully270.586s/53.470s and verified retained plain
cache-enabled logs/hashes. No remaining findings; SHIP, evidence895240ab88c2.
Recorded suite attempt565ab9c18cb876ea6bb5b1f1ce3d832b on final code
7999245e29ebddadb321e849b30dba9a2bba8dfe remains queued normally; no pass
or landing claimed. Third Plan checkbox stays unchecked until it passes and
landing completes. No new code after reviewed head.


### 2026-10-21 — final recorded suite PASS; enter landing

Attempt565ab9c18cb876ea6bb5b1f1ce3d832b on final code
7999245e29ebddadb321e849b30dba9a2bba8dfe passed exit0 in10m34s. Log SHA
prefix6750613a6adc saved at .git/sdlc/local-checks/<code-head>/suite.log.
Core196.063s, tty38.332s; every package/script completed without a failure.
Later changes are evidence-only. Review SHIP is already recorded. The last
Plan checkbox still includes landing and stays unchecked until the merge
provides its receipt; no landing is claimed before that command returns.
Enter normal sdlc merge --yes now; no manual integration or bypass.


### 2026-10-21 — landing integration conflict

The merge gate integrated main77e3f916635177b73cf0666bd6310b49c0300c14
and found only an adjacent last-entry conflict in tests/affected-go-readers.json.
Its isolated attempt preserved the branch. Repeated the requested integration
manually and retained BOTH original audited hash entries: cache conformance
and main rightpane_rungs_live_test.go. No hash refresh or code-side choice.
Fast inventory and whitespace checks pass after resolution. Main brings
#762/#764 code; the earlier suite is not claimed to cover that integration.
Run the required new recorded suite, then narrow fixed-then-ship close with
its full-suite receipt before re-entering merge, as the gate requires.


### 2026-10-21 — integrated full suite PASS

Attempt4b7476afad2ec562731e223ce7e4f83b on integrated head
415001017306f4fae8871e738a4198f68f8c556f passed exit0 in10m31s. Log SHA
prefixeb120773dc2a saved under .git/sdlc/local-checks/<integrated-head>/suite.log.
Core189.558s, tty38.182s, testdisk8.296s; all scripts/packages completed with
no failure. This is the required integration evidence, separately from the
earlier branch suite. Enter narrow fixed-then-ship close for the JSON union,
then re-enter merge. No code or audited hash changed after this pass.
