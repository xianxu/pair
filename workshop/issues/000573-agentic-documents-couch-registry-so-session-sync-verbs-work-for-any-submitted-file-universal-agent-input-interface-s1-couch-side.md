---
id: 000573
status: working
deps: []
github_issue:
created: 2026-10-15
updated: 2026-10-15
estimate_hours: 1.704
card_mirror: '3a06369e2a8eaecda421654c9bc1535050e6be49' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-15T10:56:32-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
---

# Agentic documents: Couch registry so session-sync verbs work for any submitted file (universal-agent-input-interface S1, Couch side)

## Problem

Project universal-agent-input-interface (ops `teams/devinfra/projects/universal-agent-input-interface/design.md`,
slice S1): Couch's session-sync verbs (`read`, `diff`, `write`, `status`, the owed-event ledger, the CAS write and the
operator-lock refusal) work only for files with `type: session-sync` frontmatter. The operator wants every markdown
file submitted from Couch's right-pane nvim to become an **agentic document** with the same guarantees, without
writing anything into the file.

## Spec

- A Couch-side **agentic registry** keyed by canonical path: `couch --session-sync register FILE --owner SLOT` (called
  by parley on the first submit), `couch --session-sync forget FILE`, and listing. Agentic status lasts until forgotten.
- All `--session-sync` verbs accept a registered file without `type: session-sync`; the owner comes from the registry
  (or an explicit `couch-owner:`, which wins).
- The owed-event ledger works for registered files; the submit delivery names the file.
- **Forgotten lock:** if the agent needs to write and the operator has held the lock for more than 30 minutes, the
  write refusal says so (lock age), so the agent can ping the operator (or raise a Needs you item for the TL's file).
- A path move or rename updates the registry entry.

## Done when

- Tests: register, then read/diff/write/status on a file with no frontmatter; owed events; lock-age in the refusal
  after 30 minutes (fake clock); forget; rename. Existing session-sync files are unaffected.
- `couch --skill` documents the registry and the 30-minute rule.

## Plan

Detailed implementation plan: [agentic document registry](../plans/000573-agentic-document-registry-plan.md).

- [x] Build registry model/store and stable document metadata, with idempotent register/forget/rename and read-only membership discovery.
- [x] Route every sync verb through registration-aware ownership, CAS, owed matching and lock-age refusals, preserving legacy behavior.
- [x] Expose the agreed Parley CLI/JSON contract and update help, skill, README and atlas.
- [x] Verify targeted/race/full suites and prepare the candidate for the close review.

After verification: the mandatory close review owns SHIP, then land through sdlc. The TL owns the combined operator live check after parley.nvim#332 staging, per the approved revision.

## Log

### 2026-10-15

Claimed on TL dispatch; read ops project's project.md and design.md before design. Scope is S1 Couch only; TL owns the project file. Sent proposed contract to parley.nvim:2, confirmed submitted, and received its integration requirements (read-only membership, canonical missing-old rename paths, explicit partial-effect outcomes). Existing ledger and CAS are reused (ARCH-DRY); stable document identity keeps rename from losing shadow/folds or old-path submits (ARCH-ORDER). Read-only design exploration independently confirmed these integration points. Concrete plan pending hub approval; implementation has not started. start-plan's attempted weave refresh refused due to the preserved unrelated untracked pensive; it explicitly admitted planning on the existing substrate, with ariadne #385/#386 in flight.

TL approved the concrete plan, with 4096 registrations / 16 MiB registry and explicit forget advice on overflow. No :0 staging for #573: land independently, TL rebuilds, and the combined plain-note live check follows parley.nvim#332 staging and operator restarts. Await Parley contract agreement before implementing the public verbs. Full flow is selected for the planned code volume; no estimate derived before plan-quality.



### 2026-10-15 — implementation checkpoint

Plan-quality round 1 found stale legacy metadata on forget, an incorrect assumption of unconditional 14-day ledger expiration, and verbose test inventories; all three were addressed. Rounds 2 and 3 returned CLEAN/high with no open findings. change-code admitted full flow after derived 1.704h estimate (calibration flagged stale). Estimate-quality INFO cautioned that durability/handoff and verification allowances are optimistic; own verification, coordination and landing are included in the module/review allocations, while the TL's combined postlanding editor acceptance is separate. Actual remains measured at close.

Parley accepted register/lookup/rename semantics and filesystem ownership; frozen caller contract committed at 432fab15c7f655d2392d823053e37b575638ee7b in atlas/couch-agentic-documents.md and sent to parley.nvim:2. The producer model was delegated as a bounded pure task; metadata/CLI integration stayed here. Core implementation supports all registered verbs, stable central state, durable legacy handoff and old-path submits. Tests captured plain-file refusal/stale legacy handoff/missing aged-lock advice before integration, then passed. A private-publication fault keeps central authority through partial legacy export. Same-identity register after uncertain publication retains registration horizon. Real symlink-parent and two-process registered CAS controls pass.

Queued-submit loss was independently confirmed: delivery recorder timestamps completion, potentially long after a rename. Requested and received TL approval for the additive AcceptedAt/AddressedAt delta; real broker queue/rename/path-reuse/late-completion test first lost the original submit, then passed after using address-time evidence. Broker focused/full/race tests passed independently (0.162s/7.043s/1.191s). Focused races passed sessionsync/couchmessage/couchcmd (4.768s/1.488s/2.816s); earlier complete sessionsync race passed 10.061s. CLI contract/status/read tests passed 2.055s; make build passed. Latest full sessionsync/artifactpath package checks remain in flight at this checkpoint; required pushed-head full suite, close review and landing remain outstanding. No :0 staging or operator acceptance claimed.

Updated embedded Couch reference passed a fresh-reader retrieval/application check: old reference lacked registration/rename/age/unsupported-binary facts; current reference supplied the safe protocol, including null-only absence and never expiring the lock. No user files edited by reference probes.

Final integration checks caught legacy leaf-symlink baselines and operator locks: import the caller's actual legacy companions, export known live aliases before removing central authority, and honor their locks even when a write names the canonical target. A leaf-alias rename retains its addressing interval and retries safely. The regression first observed baseline loss and an allowed write under an alias's lock, then passed; full sessionsync race passed 57.334s. A controlled real-flock test also proves a writer waiting behind rename/forget reloads authority before writing OLD (including saveas where OLD remains). Both operations pass under race. make build passed after main integration. Candidate PR is #348. The initial suite on 1c1bb7468ef3a1739c2c1ca0e5d51bf28a278584 was cancelled with SIGINT while still fifth in the machine queue (no suite execution) so the final concurrency-test candidate receives its own required run; that cancellation is not a test pass.

CI conformance run 38786915144 attempt 1 on ed386c4abe27d4ae805950c56362a418e78e1815 failed TestParkLifecycleLiveIntentOnlyMutation during helper-attachment setup (before its scenario and 750ms intent-only deadline). PTY explicitly reported Zellij losing its server connection; helper subsequently found no quit intent, as none had been published yet. Launcher live controls and recovery/warm passed. The fixture continuously drains the PTY; this evidence does not prove backpressure/load or a missing drain. Previous candidate 1c1bb746 conformance run 38786331015 passed. The intervening production delta is only the injectable registry flock boundary with unchanged real default, and the failing path does not exercise it. Full Actions log, including final Zellij dumps, retained locally at /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair573-ci.b54TFC/full.log (no uploaded artifacts exist). Logs establish a native disconnect, not its upstream cause. Requested one same-head failed-job rerun after preserving evidence; no source/timeout/fixture change and no CI bypass. Required local suite remains queued on ed386c4a.

Same-head conformance attempt 2 passed (2m33s) at https://github.com/xianxu/pair/actions/runs/38786915144/job/116603740452; all four server checks are green on ed386c4a. Attempt-1 full log SHA256 is 52bd5911b8eed33819b10826cb5661a6229f11ab615ca8e42dfea0f227f5723f; its failure remains evidence, not erased by this pass. Required suite is the still-running sdlc local-check invocation for ed386c4a, inflight record 9426472ddcea969cf8f776eec2f26b01, detached checkout sdlc-local-435314917; last observed fourth in machine queue. Only README/issue Markdown changed after that code head. Actual preview is measurable; close will adopt its own fresh measurement, never the typed estimate.


Required suite on ed386c4a ran after 1841s queued and failed after 8m54s with exactly TestPeerRuntimeBrokerToWrapperSubmitsOnce/{claude,qoder}: CLI changed canonical envelope. All other Go packages passed, including sessionsync (13.240s), couchmessage (6.988s), couchcmd (52.151s) and couchcore (270.529s); preceding shell/editor checks passed. Retained full failure log: /Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/9426472ddcea969cf8f776eec2f26b01.log, SHA256 ece6dc2e8cf26b01e58ca857a2266fceff742afef9bd860a6afb2e53e5169cc2. Focused test reproduced both failures. Diagnostic comparison showed identical wire instants; the fixture normalized Deadline before whole-struct comparison but omitted the newly added AcceptedAt monotonic-clock normalization. Added a nonzero/Equal assertion for AcceptedAt, then normalized it only for the final full-envelope comparison, preserving the wire-integrity assertion. Focused race test passed (1.872s). No production change; a fresh pushed-head required suite follows.

Replacement required suite passed on 5aea67ee68e0b2103d40305fd28934f6002fde3d: exit 0 in 9m52s after 1695s queued. Complete shell/editor checks and go test ./... passed, including wrapcmd 46.874s, sessionsync 9.657s, couchmessage 7.322s, couchcmd 61.237s and couchcore 290.175s. Log: /Users/xianxu/workspace/pair/.git/sdlc/local-checks/5aea67ee68e0b2103d40305fd28934f6002fde3d/suite.log; recorded as sdlc-local/suite success on that head. All four server checks also passed on this head, including native conformance 2m39s without retry. Earlier failed attempts remain above. Parley received the focused real-namespace/CLI/broker conformance invocation (receipt f6ef855b-85e3-4300-b315-1342477bf42a submitted). Ready for the sole close review; TL owns project updates and the approved postlanding operator check, so close uses only the precise no-project acknowledgment.

Close round 1 returned REWORK with two Critical findings, BR-1 canonical-document-lock-authority and BR-2 persisted-writer-reader-envelope. Reproduced repeat alias registration bypassing its lock and omitting its legacy export, plus successful import of oversized shadow/state and a section write publishing an oversized preserved shadow. Fixed the classes: bounded additional spellings live inside historical intervals and feed locks/export/owed/retry matching; every registered metadata producer preflights shadow/state, and writes preflight restamped document/shadow/post-fold state before document mutation. Added a fold-state overflow test whose state-bound mutation first allowed the write, then restored the guard. Tests for import, preserved shadow and restamp growth observed refusal failures before fixes; corrected a test helper to canonicalize its temporary path before finding central state. Model/alias integration race passed 3.253s; complete session-sync race and focused downstream checks in progress. Plan Revisions and atlas describe these bounds. No close has finalized; a new pushed-head suite and full close review remain required. Complete sessionsync race now passes (24.626s), focused couchmessage/couchcmd/artifactpath checks pass (0.202s/3.153s/3.422s), and make build passes. The first complete race run exposed the old symlink test expecting a newly admitted spelling to be a no-op; it now asserts Changed true for the new spelling and false for its identical retry, with stable ID. The historical alias rename/reuse/late-submit integration also passes.

Integrated newly landed #572 trimpath code from main without conflicts before final retesting. The required suite request on 349d80ccbc05399430f93fda1f37e50849674ef7 was cancelled with SIGINT while still first in queue (b975d2a33dffbbfb4c7592dc3ff67b8c; no suite execution), to replace it with an integrated-head run rather than consume a slot for a candidate known to require another suite. Caller contract and review fixes are unchanged; source inventory merged cleanly.

Required integrated suite on 729680c86961a7e7c2ebd299e69ab7984fba4a88 failed in upstream tests/make-test-env-test.sh after 4m11s (430s queued), before go test: FAIL knob PAIR_GO_TRIMPATH did not reach the test recipe. The #572 declaration invokes make PAIR_GO_TRIMPATH=-trimpath test, and inherited command-line MAKEFLAGS overrides the environment marker used by that nested-make fixture. Focused make PAIR_GO_TRIMPATH=-trimpath test-make-env reproduces the same failure. Preserved log /Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/ae50fbe5f599298fae1a463a5f01e6fb.log, SHA256 03865444e9147c5bde7f5287f916fd7f9e9a6f8c64f4789657939dfbd09bdd3f. Hub blocker/scope question receipt ac0ea343-c3c6-4057-803e-b5d90a6e7e07 submitted; awaiting upstream repair or explicit narrow scope authorization. All server checks passed on this integrated head, including native conformance 3m23s. A diagnostic Go remainder uses a temporary Makefile with the real Makefile.workflow full-suite guard and the unchanged Makefile.local test-environment header; it runs go test ./... under the machine gate, does not edit tracked gates or claim the required suite passed. Its queued dry-run inspection was cancelled before admission when make -n still acquired the parse-time guard; no test result from that inspection.

TL confirmed #577 owns the upstream fixture repair; do not fold it into #573, await its landing then merge main and rerun the recorded suite. Parley requested legacy-adoption and isolation details. Rebuilt candidate, inspected vcs.revision=49c3285c847133f6f10005c12fee584bac30989a (modified only for the unrelated untracked pensive), and ran the real binary against a private singleton initialized by public adoption. Register/lookup/repeat, filesystem rename, metadata rename and old/new lookup passed. A bare empty isolated namespace first refused UNMIGRATED, so bootstrap instructions are recorded in the caller contract and the tested helper /tmp/pair573-isolated-cli.py was sent to Parley (a6560099-ad36-400f-83b2-1b8a32bf7ffb). Existing typed legacy state is adopted before native rename; metadata rename never imports vanished legacy companions. No supervisor or live operator namespace was touched.

#577 landed as PR #350; merged its main repair without conflicts. Focused make PAIR_GO_TRIMPATH=-trimpath test-make-env now exits 0, including marker/session checks and real recursive command-line override checks. The diagnostic Go remainder was still second in queue after 1445s and was cancelled before admission to replace it with the now-unblocked required suite; it produced no Go-test result, and nothing is called passing on its account. All remaining checks will run in the required recorded invocation on this integrated candidate.

Required suite passed after BR-1/BR-2 fixes and #572/#577 integration on 0f939be63be6fdb783206d53380b9741aacc0733: exit 0 in 11m35s after 854s queued. Full shell/editor/environment checks and go test ./... pass, including sessionsync 17.847s, couchmessage 7.494s, couchcmd 90.677s, couchcore 436.862s and wrapcmd 70.169s. Log /Users/xianxu/workspace/pair/.git/sdlc/local-checks/0f939be63be6fdb783206d53380b9741aacc0733/suite.log, SHA256 23a49bb8f01b7d3b1706d0f05b6af63fb331d3cffbed4baea3e45a39656e1fa9; recorded sdlc-local/suite success. All server checks also green on this head, native conformance 2m55s. Fresh origin/main has no commits absent from this candidate. Second full close review now due. TL permits no-actual only if the known ariadne#391 incomplete-qoder-metadata defect actually refuses measurement; no project edits, stage or operator acceptance.

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only. Calibration source is flagged stale by sdlc, so the result is provisional.

Registry model/store is one greenfield Go concern: base design 2h, halved for existing durablefile/flock/ledger libraries then ×0.2 for the approved concrete plan = 0.2h; implementation 0.7h ×0.4 = 0.28h. Three existing-module extensions cover resolution/CAS, historical owed matching/handoff, and CLI: each base design 0.2h ×0.2 = 0.04h, impl 0.4h ×0.4 = 0.16h. Issue/spec work uses 1h ×0.2 design and 0.2h ×0.4 implementation. Docs use 0.1h ×0.2 design and 0.15h ×0.4 implementation; one review uses 0.1h ×0.2 design and 0.4h ×0.4 implementation. Familiar filesystem/Go stack = 1.0. Thorough-plan design buffer = 15%; Parley implementation and operator live-check time are outside this issue's build scope. Approximate primitive range 1.0–2.1h, selected 1.704h (0.56×1.15 + 1.06).

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.20 impl=0.08
item: greenfield-go-module design=0.20 impl=0.28
item: smaller-go-module design=0.04 impl=0.16
item: smaller-go-module design=0.04 impl=0.16
item: smaller-go-module design=0.04 impl=0.16
item: atlas-docs design=0.02 impl=0.06
item: milestone-review design=0.02 impl=0.16
design-buffer: 0.15
total: 1.704
```

## Revisions

### 2026-10-15 — candidate checklist clarification

The implementation checklist now separates completed code/docs and pending automated verification from future close/landing actions. The original prelanding operator-live step was superseded by the TL's explicit independent-landing decision; it remains owed by the project after Parley staging, not marked completed here. README's public command inventory now includes the registry verbs.

### 2026-10-15 11:21 PDT — queue-safe rename attribution

TL approved immutable broker AcceptedAt and additive ledger AddressedAt for historical pathname matching only. Completion time retains its existing delivery/status/retention meaning. This fixes queued pre-rename submits completed after rename, without changing the public caller contract. The durable plan records the exact state ordering and cross-service regression.
