---
id: 000630
status: working
deps: []
github_issue:
created: 2026-10-16
updated: 2026-10-17
estimate_hours: 1.57
card_mirror: '434dcbec590334f39b16fabaaa525692b3399f22' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-17T05:43:25-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
---

# couch session-sync: carry tl-lead's stale-section list to the TL when no event would show it

## Problem

## Spec

Drafted by ariadne:4 (from ariadne#414), filed by the TL.

ariadne#414 makes tl-lead stamp each status-file section and adds `tl-lead status check FILE`, which lists stale sections: Threads and Slots older than 2 h or than the newest landing; Quota older than 12 h; Needs you when unstamped. The TL sees that list only when it runs a tl-lead status verb, which happens on an event. In a quiet stretch, or while the operator holds the lock and writes are queued, nothing prompts it. That is how Threads went 20 h stale on 10-16.

For a registered session-sync file whose owner is a hub, Couch runs `tl-lead status check FILE` at most every 30 min while the file has stale lines and the owner is idle. It sends the output to the owner as one message, and none when the output is fresh. It skips this while the operator holds the lock, then sends once after their submit. It never writes the file.

## Done when

- A fixture file with a stale Threads produces one message to its owner per interval, and none when fresh.
- A held lock suppresses it.
- tl-lead missing or failing produces no message and logs the reason once.

## Plan

- [x] Add baseline-neutral registered session inspection and a bounded serialized stale-section reminder worker, with tests for cooldown, idle/hub identity, operator locks, command failures and cancellation.
- [x] Verify real isolated Couch/tl-lead conformance, update atlas/help-facing documentation, and pass the recorded suite.

Detailed design: [stale-session reminder plan](../plans/000630-stale-session-reminder-plan.md). Close review follows verified acceptance through sdlc close; landing follows the normal SDLC gates.

## Log

### 2026-10-16

### 2026-10-17
- 2026-10-17: closed — BR-1 corrected with one shared loader route selected under registry lock: missing registration refuses before legacy IO; Read/Status/Diff/Write retain legacy compatibility. Unknown and forgotten held-flock tests plus controlled forget-during-check shutdown regression were RED before fix and pass now; complete affected packages and focused race checks pass. Rebuilt real Couch/tl-lead isolated conformance passed. Full suite on 0892495573252d491f2e4efe82ee9a0b0fcf9f83 passed in 10m13s, log SHA256 0a6514e13874f487374aa6ded7e0ccd3e33b780155f9d15c8d6ebc7be41ab7ed, no failure markers; only issue Log changed afterward. Prior suite/failure/cancellation evidence retained. TL approved isolated acceptance and owns installed smoke after restart; steps recorded, no pair:0 staging.; review verdict: SHIP

- Claimed and started planning in pair:6 after #643 landed. `tl-lead status check --help` confirms that the command owns section budgets and reports without writing. Existing wait-watcher ticks every 15 seconds; external IO must run in a separate serialized worker so it cannot delay wait notices or supervision.
- ARCH-DRY/ARCH-PURE: keep freshness rules in tl-lead; use existing registered-document transaction for an inspection that never acknowledges edits, and existing hub identity, wrapper liveness and broker self-delivery seams. No timer-based inference of idleness.
- Verified TL approval receipt 7792da29-cd2b-4de6-becc-7aeda30b307d: in-memory cooldown may reset on restart, costing at most one extra reminder; isolated real Couch acceptance here is sufficient, installed-hub smoke deferred to restart, no pair:0 staging.
- Verified TL approval receipt 5b8479be-4ca3-41b8-9745-586c7b791da1: full flow approved for the separate serialized worker and safe IO envelope, same feature scope. No code written yet.

- Plan gate PQ-1 requested an authoritative pure transition model; the durable plan now defines advanceStaleReminder and event/effect transitions for every retained state mutation. PQ-2 recurring conformance triggers and the real Git-backed fixture are specified. No implementation has started.

- Implementation checkpoint: observed missing inspection/model/worker/command APIs in red tests, then passed focused SessionStale/InspectRegistered tests. Pure transitions own allowance and diagnostics; a coalesced serialized worker is wired to the existing wait tick. Command fixtures cover missing/nonzero/overflow/cancel and owned descendant collection; worker barrier proves wait ticks remain responsive and service cancellation collects work without sending. The overflow test caught bytes.Buffer ReaderFrom promotion bypassing the writer bound; the private buffer now exposes only bounded Write. Full package/race and real CLI conformance are next.

- Verification: both complete affected packages passed (couchcmd 61.748s, sessionsync 12.854s); focused race tests passed (2.278s/1.732s). The env-gated real CLI fixture passed (1.880s package) using this slot's once-built Couch and installed tl-lead, a disposable adopted namespace, a Git-backed status file, admitted wrapper and real RemoteEndpoint transport. It verified repeated stale delivery, fresh silence, lock suppression/release and exact document/shadow/state byte preservation. Installed-hub smoke remains TL-owned after restart.

- First full-suite attempt on ce41c869 found TestProductionArtifactReferencesAreExactlyClassified: the three new production files were absent from the exhaustive source inventory. A focused run reproduced exactly those three omissions. They reference operator documents and existing session-sync/broker APIs, with no Pair artifact constructor/filename vocabulary, so they are explicitly added to NonArtifactSources alongside existing session-sync/wait code; the known-violations quarantine is unchanged. The focused inventory test now passes (2.553s). The ongoing suite is retained to collect any other failures.

- Plan/implementation comparison found that TrimSpace removed trailing control bytes before validation. Added red output-contract cases (vertical tab on stale output and carriage return on the fresh sentinel), then validated raw CLI output before normalization; focused three-package tests and reminder race tests pass. This restores the existing approved malformed-output contract without changing freshness policy. The queued, not-started suite on 9518a526 was explicitly interrupted because this code changed; it is retained as canceled, not a flake retry. Main #637 integrated cleanly before the corrected candidate.

- Retained first suite result: ce41c869 completed in 11m59s after 676s queue; exhaustive inventory was the only failing test (all other Go packages and preceding shell/editor targets passed). Log: /Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/d1590f795f30b01aedb88e3766e89b37.log, SHA256 af71cf23bec3cced9688760d00e0b6fb2bf5cab889ed6eb66bec3c486c6bca9c. Corrected raw-output tests pass; focused reminder race passed 1.737s, 33,108 fuzz cases passed, and rebuilt real CLI conformance passed 1.254s. Full suite queued on ff9b15c8; no flake waiver used.

- Command extent check added a red fixture whose successful shell leaves a child with closed output pipes. Cancellation/WaitDelay alone did not collect it. The same already-owned process group is now ended after every command completion, including success; focused reminder tests passed (0.840s). The queued suite on ff9b15c8 was interrupted before acquiring capacity so it cannot be mistaken for verification of this corrected code. No new process mechanism or product scope was introduced.

- Final command-group correction reverified: reminder race tests passed (1.980s), rebuilt real CLI conformance passed (1.626s), and the replacement full suite is queued on 238a534c. The successful-parent/closed-pipe descendant lesson is recorded in workshop/lessons.md.

- Corrected full suite PASSED on 238a534c614f040f003d1f3916ddb88cd6148719: 10m57s after 705s queue, log SHA256 6ba744447b56018065b6071f061ea9c92dc0ca4008ee9cca4d3cce36e77d9146. The retained complete log has no FAIL/panic/make-error markers. Changes since that tested commit are documentation only. All behavior acceptance items are verified; fresh close review is next.

- Close round 1 returned REWORK/high with one Critical BR-1 (registered-inspection-fails-closed): rejecting registration absence in the callback still entered generic legacy IO first. Direct unknown/forgotten held-flock tests reproduced it, and a controlled worker forget-during-check test proved shutdown could block. Enumeration found InspectRegistered is the sole registered-only caller; Read/Status/Diff/Write intentionally retain legacy compatibility. The shared loader now selects explicit route policy under the registry lock, refusing missing authority before legacy file IO/flock. Regression verification follows.

- BR-1 fix verification: direct held-flock cases were red before the route correction; the worker regression also failed with blocked shutdown before it. Focused combined tests now pass (sessionsync 0.654s, couchcmd 1.602s), race checks pass (1.817s/2.279s), and complete packages pass (11.201s/53.253s). Existing legacy operations remain exercised by the complete sessionsync package. The shared route enforces registration presence once under the registry lock, without a racy separate Lookup.

- BR-1 candidate full suite PASSED on 0892495573252d491f2e4efe82ee9a0b0fcf9f83 in 10m13s after 1783s queue; retained complete log SHA256 0a6514e13874f487374aa6ded7e0ccd3e33b780155f9d15c8d6ebc7be41ab7ed, with no failure/panic/make-error markers. Rebuilt real CLI conformance also passed (1.324s). Fresh review will verify disposition of BR-1.
- TL-owned installed smoke after restart: on an existing registered hub session file whose Threads are stale, observe a real composer self-reminder while the admitted owner is settled and picker-free; compare document/unread baseline before and after. When the allowance is due, an operator lock must suppress it and release must permit the next eligible idle tick. The isolated fixture already proves half-hour repeats/fresh silence without waiting on or altering the installed hub. No pair:0 staging was performed.

## Revisions

- 2026-10-17: quick-flow sizing exceeded the shell once baseline-neutral inspection and bounded asynchronous command execution were included. TL approved full flow before implementation; product behavior is unchanged.

- 2026-10-17: put the close-review action after the acceptance checklist, so checking verified behavior does not assert a review verdict before the gate runs. Product behavior and scope are unchanged.

- 2026-10-17: BR-1 refines the shared loader route to enforce the approved registered-only inspection contract before any legacy IO. Behavior scope is unchanged; bounded refusal and shutdown now have held-flock regressions.

## Estimate

Produced via brain/data/life/42shots/velocity/estimate-logic-v3.1.md against baseline-v3.1.md, Method A only. Calibration is marked stale by estimate-source; units are provisional. The new pure transition/serialized worker is one greenfield-go-module (base design 0.5 × 0.2 settled-spec discount; implementation 0.8 × 0.4). The bounded tl-lead command, real Couch transport conformance and stateful command/peer fixtures form one api-integration (design 1.0 × 0.2; implementation 1.0 × 0.4). Registered inspection/service wiring is one smaller-go-module (design 0.2 × 0.2; implementation 0.5 × 0.4). Existing sessionsync, broker, worker lifetime and standard exec/context APIs are available; no domain-specific library replaces the new reminder rules. Docs use design 0.1 × 0.2, implementation 0.1 × 0.4. One review uses zero design and implementation 0.5 × 0.4. Familiarity is 1.0 and thorough-plan design buffer is 15%; total 0.36 × 1.15 + 1.16 = 1.574, rounded 1.57 hours.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: greenfield-go-module design=0.1 impl=0.32
item: api-integration design=0.2 impl=0.4
item: smaller-go-module design=0.04 impl=0.2
item: atlas-docs design=0.02 impl=0.04
item: milestone-review design=0 impl=0.2
design-buffer: 0.15
total: 1.57
```
