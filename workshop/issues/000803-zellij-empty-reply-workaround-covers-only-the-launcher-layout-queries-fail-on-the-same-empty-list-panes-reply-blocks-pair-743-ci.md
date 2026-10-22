---
id: 000803
status: codecomplete
deps: []
github_issue:
created: 2026-10-22
updated: 2026-10-22
estimate_hours: 1.7935
collaboration-mode: collaboration
card_mirror: '3b93a33667dc0c39955c5fd9bce10d528a933f56' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-22T09:04:04-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
actual_hours: 1.50
---

# Zellij empty-reply workaround covers only the launcher; layout queries fail on the same empty list-panes reply (blocks pair#743 CI)

## Problem

## Spec

pair#743's hosted conformance run 39759978845 failed `TestTerminalRecoveryZellijConformance`
(terminal_recovery_live_test.go:146): native `zellij action list-panes --json --command --state --geometry` returned
exit 0 with empty stdout and stderr after an earlier valid 4996-byte answer, and the strict parser refused it. pair:3
traced production: `layoutcmd.OSRuntime.execZellijContext` runs once and `confirmWorkbenchPanesContext` uses
`zellijreply.Confirm` (exactly two valid observations, no retry); `ParsePanes` rejects empty JSON. Only
`launcher.ZellijSource.runContext` has the bounded 3-attempt empty or crossed-reply workaround (pair#748), and the
atlas explicitly excludes other consumers. So the layout path fails in production on the same Zellij behaviour
(upstream zellij-org/zellij#5751 family; pair#775 evidence).

Fix: one shared read-compatibility seam (#748's bounded retry on an empty or crossed reply, unchanged bounds) used by
every Zellij query consumer, layout first, not a per-consumer copy and not a test-only retry. Update the atlas
exclusion.

Approved boundary (2026-10-22): this issue covers every **Go** production query
consumer through an independent lower-level seam, with layout first, then clip,
focus, opener, title liveness and launcher probes. #761 false-miss handling and
per-attempt observation survive the extraction. Mutations never retry; raw
diagnostic probes remain excluded. Lua consumers are explicitly deferred to
#804, which will use a thin CLI adapter over this same seam. Full flow applies.

## Done when

- The layout consumer retries an empty successful reply within the existing bound, through the shared seam (test with
  a fake that returns empty once).
- The live terminal-recovery conformance test passes through the production seam
  without test-only retries. Per TL e9f8e0ca, pair:3 reruns #743's hosted CI after
  #803 lands; that downstream rerun is not a pre-landing gate for this issue.
- Every in-scope Go query consumer derives bounded read recovery from one
  lower-level owner without a launcher import cycle; the atlas names #804's
  remaining Lua exclusion and the raw-diagnostic exclusion explicitly.
- Counted 8/16-read tests, cancellation/collector-error cases, strict malformed
  refusal and no mutation replay pin the shared bound and authority contract.

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md`
against `baseline-v3.1.md`, Method A only. Calibration is tagged stale by SDLC;
hours are provisional. Familiar stack and existing collector/parser/fake libraries
cover the work; no greenfield library or external API is required. Design for
the extraction/module/docs uses the thorough-spec 0.2 discount; implementation
values already use the v3.1 40% scale. Native qualification uses the existing
installed Zellij and controlled fixture, with discovery at the upper bound.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.5 impl=0.08
item: cross-cutting-refactor design=0.12 impl=0.2
item: smaller-go-module design=0.03 impl=0.2
item: real-api-discovery design=0 impl=0.24
item: atlas-docs design=0.02 impl=0.08
item: milestone-review design=0.02 impl=0.2
design-buffer: 0.15
total: 1.7935
```

## Plan

- [x] Claim, enter planning, and enumerate query consumers and existing retry authority.
- [x] Settle the Go/Lua boundary with the hub, then checkpoint the executable plan before change-code.
- [x] Implement and qualify the approved shared seam, keeping strict confirmation and mutation behavior intact.
- [ ] Record verification, close review, and land through SDLC; #759 operator acceptance takes precedence.

## Log

### 2026-10-22
- 2026-10-22: closed — Post-close test-only correction: full draftroute (3.690s), termcmd (6.671s), zellijreply (0.183s) pass after reproduced integrated-suite failures; persistent crossed faults exhaust three attempts with zero effects, transient controls require three reads before scoped effects, semantic mismatch refusal unchanged; read-only Output inventory updated. Reader inventory and diff checks pass. Fresh recorded final suite queued on pushed 5c8da1498; tl-worker overlap allowed, passing final suite still mandatory before landing. --no-atlas because this delta changes tests only and existing seam atlas is current; --no-plan-check for pending workflow row; --no-project TL-owned. (fixed-then-ship); review verdict: SHIP
- 2026-10-22: closed — BR-1 reproduced then fixed with refusal intact and case-specific three-attempt false-miss expectation; full launcher passed 69.348s, reader inventory and diff check passed. Earlier full adapters, race, fuzz and installed-native recovery conformance passed as recorded in Log. Replacement recorded suite queued on pushed 632ed7c31; tl-worker permits review overlap, final suite pass still required before landing. --no-plan-check acknowledges only pending suite/review/landing row; --no-project leaves project updates to TL. #743 hosted CI rerun follows landing per hub.; review verdict: SHIP

Claimed by pair:6 and entered start-plan. The catch-up rebuilt the sibling
Ariadne tools and generated an unrelated merge-check workflow refresh (draining
grep instead of quiet grep); inspected and stashed only that workflow as
`803 planning: preserve weave-generated workflow refresh`. No CI gate changes
belong to this issue.

The read-consumer sweep found launcher inventory, legacy listing and name-probe
queries; layout panes/current-tab; clipboard panes/clients; Couch shortcut focus;
title liveness; opener panes/dump-screen; wrapper peer-viewport dump-screen; and
failure-only version diagnostics. Launcher retries currently preserve per-attempt
observers, exact absence diagnostics, and #761 false-session-miss classification.
Existing `zellijexec.Run` owns joined private process groups but has no read retry.
`zellijreply.Confirm` must still require two independently valid agreeing reads;
malformed same-kind replies must never become retryable parse failures.

ARCH-PURPOSE shadow sweep also found production Lua consumers:
`nvim/workbench_route.lua`, `nvim/dom_transport.lua`, and `nvim/pair_poke.lua`
query panes through `zellij_observation.query` under an existing admission budget;
`nvim/review/restore_client.lua` queries panes and floating-pane visibility through
its asynchronous runner. A Go-only seam cannot cover these consumers. Raw native
reproduction/latency probes intentionally observe the unmodified binary and must
remain outside retry. The shared seam must not replay native mutations.

Sent the hub the boundary decision before code (delivery
bd248d7c-4d1e-4931-bd42-1252539377aa): recommend one lower-level Go read seam with a
thin CLI adapter for Lua, requiring a short full-flow plan; alternative is an
explicit Go-only scope plus a separate Lua follow-up. Awaiting that decision;
no implementation or new refusal mechanism has been added. #759's iPhone
acceptance retains priority.

### 2026-10-22 — approved scope and implementation plan

Authenticated TL receipt a27b640f-2e13-4c62-b6d8-b2faf5b422e0 approves Go-only
scope, full flow, preserving false-miss and per-attempt evidence. Filed #804
unclaimed with dependency pair#803 for the Lua consumers and future thin CLI
adapter. The executable plan is
`workshop/plans/000803-shared-zellij-read-plan.md`. It extracts the existing
read/classification into independent zellijquery and keeps launcher compatibility
aliases. Original process/collector errors remain distinct from synthetic read
classification so the name probe keeps its exact diagnostic authority.

### 2026-10-22 — implementation and qualification checkpoint

Plan-quality finished CLEAN with PQ-1/PQ-2 disposed; estimate-quality was INFO
and change-code admitted full flow. The estimate's issue-spec item accounts for
planning since claim; cross-cutting refactor covers adapter wiring, smaller Go
module covers classifier/fuzz/lint checks, real-API discovery covers native
qualification, and review covers active suite/gate execution and remediation.
Machine queue wait is not implementation work.

Extracted Source, strict session classifiers and native socket spelling into
zellijquery. All Go consumers and the shared controlled-Zellij setup listing
derive from it; launcher public error/observer APIs retain aliases. Layout's
OS/Confirm regression first failed on empty JSON; launcher name-probe recovery
first failed after one unanswered attempt. Both are green through the shared
owner. Raw errors stay distinct from read classification, partial failure stdout
is preserved, and direct exit stderr remains available to focus diagnostics.

The title whole-group ownership test failed reproducibly when the new default
collector isolated its child group. Authenticated TL 68b76271 approved the
documented Source.Run=InheritedGroup injection, preserving title CLI EOF cleanup
with one-second WaitDelay and unchanged read policy. Full titlepoller then passed
(2.016s). Other adapter full packages passed: layout 10.507s, clip 23.593s,
Couch 101.838s, opener 0.565s, wrapper 166.399s. Persistent refusal fixtures now
inject the same transient fault through all three admitted attempts; their
semantic refusal and zero-effect assertions remain unchanged. The native-reply
capture resets each attempt so final success cannot retain an earlier error.

New regression proofs also caught and fixed implicit-session argv borrowing a
socket name, custom child environments borrowing the caller's socket namespace,
invalid public reply-kind panic, and observer-time cancellation exposing a direct
exit as diagnostic authority. The last read-context check preserves cancellation
before returning or starting another attempt. Shared deadline/count and raw-error
tests remain in the normal suite; independent confirmations count 2N healthy and
6N recovery attempts at N=8/16.

Disposable native `PAIR_LIVE_ZELLIJ=1 go test ./cmd/internal/layoutcmd -run
'^TestTerminalRecoveryZellijConformance$' -count=1 -v` passed (1.79s test,
1.938s package) through the actual production source. No live Couch, operator
session, :0 build or network setup was touched. Artifactpath, repolint and shared
Zellij reply-generator packages passed (20.137s / 17.316s / 27.345s), followed by
an explicit reader-certificate re-audit. Earlier race checks across eight changed
packages passed; final post-cancellation targeted race rerun is pending.

The post-cancellation shared-query/launcher race run passed (20.895s / 42.277s),
including native diagnostic provenance and the name-bound matrix. A final narrow
check follows the duplicate-context-error formatting cleanup before commit.

Fresh-eyes implementation check found no production correctness defect but
required stronger promised fuzz oracles. Added an independent exact argv grammar
and complete strict different-kind validation, observed RED for empty bytes
misclassified as a complete session reply, then fixed that classifier guard.
Recheck found the gap resolved. Final three-second two-worker fuzz campaigns
passed: ReadKind 33,181 executions; CrossedReply 73,757; session validation 39,236.
Recorded branch suite, SDLC close review and landing remain pending.

### 2026-10-22 — branch suite and close admission

Committed implementation 24a9162c73d0810e9bab16f9aa0753223c388c7c, pushed PR544.
Recorded suite attempt 5b7bcd82655d564773c58af65b79f7af / lease-3328783035
is retained on that exact code head. Final narrow post-format cleanup race guard
passed (1.843s); reader inventory passed. TL e9f8e0ca grants two-hour priority,
asks for close review next, and owns revocation on landing; pair:3 then reruns
#743 hosted CI. Done when now states that approved downstream ordering explicitly.

Close review overlaps the queued recorded suite under tl-worker's explicit rule:
the final code head must still record a passing suite before landing; any review
code fix gets fresh verification. The only unchecked Issue Plan row is the suite,
review and landing workflow itself, so --no-plan-check acknowledges this overlap
without claiming those gates passed. --no-project leaves the TL-owned project
updates to the hub; neither flag waives suite, CI, review or merge validation.


2026-10-22 — Close review R1 returned REWORK for BR-1, the remaining native
name-absence attempt-count fixture. Independently reproduced the exact failing
requested-identity-present subtest (three calls, refusal retained). It now pins
three calls only for that false-miss case; other cases retain one. Audited the
other launcher single-attempt assertions: native positive/negative authority,
nonmatching signatures, cancellation and collector errors remain single-attempt.
The reader certificate retains non_go_full_fallback for its committed capture.
Cancelled only obsolete queued suite lease-3328783035 on 24a9162c, joined its
cancelled runner; no suite work had started. Full launcher package passed (go test ./cmd/internal/launcher -count=1,
69.348s); reader-inventory check and git diff --check passed.

Read-only title-fixture cleanup audit: earlier RED tests positively observed an
escaped matching descendant for over one second when the new private collector
broke the CLI's inherited-group ownership. The approved InheritedGroup fix and
current passing cleanup regression address that class. Those failed runs did not
retain their PID/identity/root, and cleanup killed only the title group, so the
historical escaped child's eventual cleanup is unproved. No current process was
attributed or killed; later passes are not historical cleanup evidence.

## Revisions

2026-10-22 — TL-approved scope refinement: "every query consumer" means every
Go production consumer in #803; #804 owns the Lua integration. Expanded to full
flow because moving the existing strict classifiers and all Go adapters exceeds
the quick-flow production-line shell. No mutation or raw diagnostic retry is
added, and no caller's strict confirmation is relaxed.

2026-10-22 — TL-approved ownership preservation: title's existing whole CLI group
remains its cleanup owner via the documented inherited collector injection;
default readers retain joined private query groups. No new ownership mechanism.

2026-10-22 — TL e9f8e0ca confirms #743's hosted CI rerun follows #803 landing,
owned by pair:3; local installed-native conformance qualifies this issue first.


### 2026-10-22 — integrated suite own-defect classification and correction

Recorded integrated suite d9e69ff7be7f94089c57fb8f9b1751c0 on beb222d0
failed in 11m9s. Log SHA-256:
e6f47b7b238431b8653395f76863e2daceaa4299a392b83e539016b91d2f1246.
This is own-change test contract drift, not flake/infrastructure evidence.
The four failed top-level tests were draftroute's scoped draft admission,
termcmd's caller-tab admission and broker-notified fullscreen refusal, and
zellijreply's M2 effect inventory. One transient crossed reply is now recovered;
negative stimuli must persist for all three physical attempts, while two valid
semantic observations remain required. Kept negative names and effect/notice
assertions, pinned three/four refusal reads, added single-crossed positive
controls with exactly three reads and unchanged scoped effects. Semantic
mismatches still refuse with their original one/two reads. Updated only the
existing read-only inventory entry from execZellijContext:Run to :Output.

Reproduced routing failures independently; full packages draftroute (3.690s),
termcmd (6.671s), zellijreply (0.183s) now pass. Audited and refreshed the four
changed reader certificates without changing their dispositions. Inventory and
whitespace preflights passed. Production source unchanged.
The failed merge automatically queued a diagnostic main baseline on 5dd027f1;
cancelled only its exact lease-1172289527 before start because these defects
were already reproduced and explained. Original merge joined exit1; no retry,
flake registration, or gate bypass. Final fixed-head suite/review/landing remain
pending; previous SHIP does not certify this post-close delta.
