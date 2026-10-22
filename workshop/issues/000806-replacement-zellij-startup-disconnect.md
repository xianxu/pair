---
id: 000806
status: working
deps: []
github_issue:
created: 2026-10-22
updated: 2026-10-22
estimate_hours:
severity: high
collaboration-mode: collaboration
card_mirror: '74ef158e26de79ec542c005e4420b3dfc9540eba' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-22T15:25:37-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "36f7a046", done: "503048df"}
---

# Replacement Zellij disconnects before first wrapper startup

## Problem

A disposable public reboot completed exact old-session retirement, but its replacement Zellij session disconnected before the first wrapper startup/agent-ready publication. Pair exited1 before Couch registration. This is distinct from #797's ownership refusal and quiescence ordering correction. Cause is unclassified; no flake is registered. It may relate to #773's missed-ready startup evidence, but no shared root cause is established.

## Spec

Retain and diagnose the new-session-before-first-wrapper disconnect through the shared disposable Couch harness. Distinguish a server initialization panic, an external termination, and failed client admission from downstream registration cancellation. Preserve exact-generation ownership and all startup/query fences, current time/frame/output/write bounds, and complete cleanup. Do not infer environmental causation from a passing rerun or weaken startup admission.

## Done when

- A retained failing startup includes the private Zellij server log and exact session/launch identities before root cleanup.
- Causal replay identifies the trigger and pins the failure with a regression through the existing startup path; unknown ownership remains refused.
- Disposable Claude and Codex startup/reboot+delivery acceptance and exact owned cleanup pass under existing bounds.

## Plan

Diagnostic proposal: [replacement startup diagnosis plan](../plans/000806-replacement-zellij-startup-disconnect-plan.md). Diagnostic phase approved by ops:0; production correction still requires the follow-up approval described in the plan.

- [ ] Retain bounded private server evidence before cleanup and qualify its reader boundaries.
- [ ] Run one bounded diagnostic per personality and, only with a captured failure, one causal contrast; preserve unknown outcomes.
- [ ] Bring the measured cause and concrete correction choice to the hub; revise the implementation plan before production changes.
- [ ] Complete the approved correction, causal regression, both-personality native acceptance and normal close/landing gates.

## Log

Filed unassigned per authenticated ops:0 disposition5a670f4e-1ef7-49da-99ba-835f86d2814b while closing #797. The ownership fix passed native acceptance on both agents; the review's separate replacement-start failure remains recorded here rather than being classified as a safety-test flake.

Failure: reviewer pinned head003e3ba8deb1dd28cedb89b58774a1295b7916e6, Zellij0.45.1, TestLiveRecoveryOwnerReboot/codex. Go57.333s/sub46.39s, writes81424384/cap100663296. Public confirmation f6a256ff-6e79-4b9a-9f38-1769f302c783 ran10:15:11.169601–10:15:20.185623 PDT, failure nonce operation-d68b74db8d679ed8aedd38304f768342. Old session 📁9223559885839400961-2 / agent65380/start1792689303.200098 was absent. Replacement session 📁9223559885839400961-3, launch nonce start-9deeaa7f8d7bb690, lost its server before any replacement STARTUP-identity. PTY output: Lost connection to the Zellij server; terminal client exited before admission was verified. Existing bounded native query evidence is absent. Registration context cancellation follows Pair exit and is not the initiating cause. Cleanup joined direct child/IO and proved every owned generation absent.

Complete9233-byte reviewer output retained from its own transcript at /Volumes/TestScratch/pair797-br2-transcript-fgmhr1gg/reviewer-codex-native.log; SHA256b86a30bb1b3d2fb642557ee91b7a511807a9ce5926fcf6a2feb95166bf07a99d. The private root pc-2732206875, reviewer checkout and binaries were already cleaned; its server panic log was not retained. The original transcript is rollout-2026-10-22T10-11-57-01a16499-1829-7561-b781-9852d0ae2bbb.jsonl, CommandExecution output line156.

Controlled diagnostic: exact reviewed003e3ba8 source in a detached disposable checkout, a Go overlay adds only a bounded8192-byte read of existing private zellij.log AFTER confirmation; no production code, pre-confirmation action, assertion, or bound change. Overlay SHA25671e6f37992708b2d6525b562fb66c9b3a06cdd8f5c259f55e9a39e1aaae03c54. After normal runtimebundle generation, Codex confirmation e320a096-1397-4f90-bb76-4131c80df245 succeeded, old61659/start1792690077.658530 absent, new63356/start1792690086.142535 accepted public message, exact cleanup complete. Go30.182s/sub23.82s, writes81977344/cap100663296. Complete log /Volumes/TestScratch/tmp/pair-slot2/pair797-br2-probe-o0bm5t4z/native2.log; SHA256bd819f74b2424feaa2cf0886e4571d5a4581732bc7b9f72afd547fa3f9155b73. It retains all7236 bytes of private server log before cleanup, with several Server not ready/retry entries but no panic. The controlled pass does not explain or erase the failure.

Hypothesis only: the documented Zellij0.45.1 birth race can panic when a global query connects before its first client initializes (launcher/session_handoff_os.go observeAdmission comment; atlas/couch.md cold-resume birth section). This matches the failure boundary, but there is no triggering query or panic evidence in the failed run. The new quiescence carries only old session -2's immutable server generation and finishes synchronously before Fresh launches distinct -3; it cannot authorize the replacement. Ordinary park also waits for the old helper to die. Investigate independent background/global queries and any unproved helper cleanup rather than assigning causation from wording alone. #773 may be related; its evidence remains independently unclassified.

Read-only follow-up found independent query paths: message_service.go constructs ListingPoller(ZellijSource{}, messageListingInterval=1s) and starts its Run goroutine; immediate/periodic Refresh calls global list-sessions --no-formatting, with no birth-lease participation. Resource inventory seeding/lifecycle refresh also reaches SessionPresence/LivenessContext and current owner probes independently of terminal admission. The resource sampler itself and Git/filesystem hub prewarm do not issue Zellij queries; COUCH_RESOURCE_CADENCE=1s is not the listing cadence. These are concrete possible overlaps, not proof of the triggering query in the failed run.


### 2026-10-22 — Claimed; evidence classification and diagnostic proposal

Dispatched by authenticated ops:0 delivery6205eb4f-b97c-4655-b177-c2706eef19c3 after #797 landed as PR545. Claimed806, entered planning on its own main-based branch; no production or harness code changed and no new native probe ran. Start-plan's catch-up generated the known ariadne#557 seeded merge-check.yml diff. Per the authenticated fleet instructionf46bd9db-0a67-4b5a-b93f-19c51a3009eb, saved that exact diff at /Volumes/TestScratch/tmp/pair806-seeded-refresh.e7PLYt/merge-check.yml.patch, restored only that tracked workflow, and reran start-plan successfully. The older unrelated #797 workflow-refresh stash remains preserved. Seeded gate changes are not part of806.

Read-only classification on main basee814b17f15f5f194119f41b209bc35f76285e1e7: original failure is pre-wrapper replacement transport loss, initiating cause still unknown. Rehashed the9233-byte failing log and10600-byte controlled-pass log; both match the SHA256 values recorded above. The successful trace contains Server not ready warnings and proceeds through first-client initialization; those warnings do not establish the original failure's cause. The failed root/server log cannot be recovered after its recorded cleanup. Private pane evidence rejects the mismatched old launch nonce; downstream registration cancellation follows Pair exit and is not the initiating event.

Current-source census (read-only helper, no edits or native runs) identifies independently scheduled global queries from the message listing worker, resource inventory seed/refresh, console inventory, existing standalone title pollers, birth watchdog, and concurrent lifecycle/handoff work. The per-socket terminal flock and exact own-readiness admission gate do not cover all global readers. Couch-hosted wrappers skip the standalone title helper; resource sampling and Git prewarm are not native query sources. #803 supplies bounded collection/recovery and #773 supplies deadline ownership, neither supplies cross-query birth exclusion. No evidence yet identifies which, if any, query overlapped the failed replacement.

The existing bare birth-race probe and upstream v0.45.1 RemoveClient optional-data unwrap support the early-connection hypothesis, not attribution. Do not run that older instrument unchanged: its shared log and name-based cleanup do not provide the private exact-generation proof required here. Proposed next step is diagnostic-only reuse of testcouch's root-confined reader: at most two8192-byte server-log captures around existing failure cleanup, one native Go invocation per personality with all existing bounds, then at most one controlled causal contrast after a retained failure. A single-poller patch, broader retry or deadline change would be speculative. See the durable plan for the full census, alternatives, state table, bounds and correction approval gate.

## Revisions

### 2026-10-22 — Diagnose before selecting a correction

Current main includes #773/#803/#797, so the earlier hypothesis must be tested against the current shared boundaries. Preserve original Spec and Done when; add a bounded diagnostic phase and an explicit return to the hub before any production correction. No new runtime state, protocol or time budget is approved or inferred.

2026-10-22 diagnostic authorization: authenticated ops:0 delivery595ae402-f90b-498f-a7ca-b254b66ebac5 approved at most2x8192-byte private server-log snapshots before teardown and after joined producers; one current-bound96MiB native run each for Claude/Codex; at most one causal contrast after a retained failure; return with correction choice before any production change. Proceeding through change-code for this diagnostic scope only.


### 2026-10-22 — Diagnostic capture ready for the approved native samples

Implemented only shared testcouch diagnostics, with no production launch/admission change. Canonicalized the durable plan filename to the issue slug. Retains one escaped 8192-byte private Zellij tail before cancellation/teardown and one after process/console/IO joins; same-descriptor size/offset, errors and truncation accompany each. Two sync.Once budgets include failed reads and repeated cleanup. The native recovery test opts into these same captures for successful controls, default off for other successful fleets. Private root confinement, final no-follow, nonblocking regular-file checks remain shared with the existing reader.

TDD: initial server-retention tests failed because diagnostics omitted the server log, then passed. Repeated-cleanup tests at16/32 calls failed with17/33 snapshots and late-file reads, then passed with exactly one snapshot per phase. Tests cover8192/16384-byte tails, escaped terminal controls, independent phases, missing files, final symlinks, escaping parent symlinks, FIFOs and directories. Focused TestStartupEvidence PASS1.251s; targeted launcher handoff/admission PASS1.239s; internal/repolint PASS9.826s and termcmd PASS4.550s. Native reader inputs remain private fixture/runtime files with existing non_go_full_fallback ownership; refreshed that reader's certificate after the option change. No new repository-input reader was detected in the new unit fixture.

Next: push this diagnostic candidate, then exactly one native invocation each for Claude and Codex under unchanged180s preparation/120s operation/15s cleanup and96MiB package write cap. Preserve complete logs and hashes; no native run has started yet. Return correction choice or explicit uncertainty to the hub before any production change. At most one causal contrast is authorized after a retained failure.
