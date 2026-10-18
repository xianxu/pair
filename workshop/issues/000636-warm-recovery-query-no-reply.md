---
id: 000636
status: working
deps: []
github_issue:
created: 2026-10-16
updated: 2026-10-18
estimate_hours:
card_mirror: '536944828337201ce358ad5a1a35004bec92aab9' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-17T10:14:59-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "fb20d267", done: "9f6648a3"}
actual_hours: N/A
---

# Warm recovery live conformance intermittently receives no Zellij query reply

## Problem

Native Zellij 0.45.1 occasionally returns no reply to a recovery fixture query while the exact owned session remains alive. The pre-existing warm live recovery test fails before fixture cleanup. This is separate from #627's daemon/cache cleanup join.

## Spec

Diagnose empty native list-clients/list-panes replies during warm recovery and make the fixture or product respect the observed readiness contract. Establish whether Zellij, the fixture's detach transition, or the query adapter causes the gap. Preserve recovery and ownership assertions and keep unknown observations distinct from absence; no blind retries or longer deadlines.

## Done when

Current delivery, explicitly revised by hub approval5a7cb535 on2026-10-18:

- Production recovery through the shared native fake refuses empty/whitespace, malformed and wrong-kind ownership/client replies before durable mutation, launch or quiescence, without retry; valid detached replies remain usable.
- Both live fixtures explicitly retain disposable HOME/config/data/cache/runtime/socket roots after actual environment setup; poisoned ambient roots cannot leak into their children.
- Existing query bytes/status/timing and local lifecycle ordering are retained through bounded deferred capture without extra queries or changed deadlines/assertions. Counted8/16 and capacity/cancellation controls prove the measurement contract.
- Targeted checks, required recorded suite, close review and normal landing pass. After landing, ask the hub whether native qualification is still warranted. No native run is an acceptance prerequisite for this approved diagnostic delivery.

Original cause-level acceptance below is retained as the historical objective; its execution is deferred until the hub chooses a native window after diagnostic landing:

- Reproduce the no-reply seam deterministically, classify its cause, and add a regression at that seam.
- Original observed failure cases and repeated isolated warm recovery conformance pass with every existing assertion retained.

## Plan

Approved diagnostic delivery: [durable plan](../plans/000636-warm-recovery-query-no-reply-plan.md). First pin production recovery's unknown-reply refusal through the shared native fake and explicitly isolate the live fixture environment. Then instrument existing queries and the local detach boundary without extra queries, and request a separately approved bounded native window. A cause-level readiness/product correction is conditional on the resulting evidence and requires an approved design revision; no guessed fix, automatic retry or longer deadline.

- [x] Approve diagnostic plan; use change-code before test/helper edits.
- [x] Prove complete production recovery refuses targeted native reply faults with no durable mutation/launch/quiescence or retry, with a healthy detached control.
- [x] Prove private roots survive actual fixture environment setup; bound deferred diagnostic capture and preserve existing conformance assertions.
- [x] Commit/push diagnostics and isolation fix, pass the required recorded suite, and prepare close evidence.

Remaining lifecycle steps: sole close review, normal landing, then ask the hub whether a native window remains useful. Cause-level diagnosis/native controls remain explicitly pending that post-landing decision.

## Log

2026-10-16: Found during #627 qualification on Git-verified code head 8847a4c0501a8ea19daf142445f555c29a433ff2, Darwin, /opt/homebrew/bin/zellij 0.45.1. Private HOME/config/data/cache/runtime/socket roots from native-private-env.json in the scratch directory below, no overlapping live runs. Initial test-couch-zellij-live target passed all four launcher tests and park test, then warm recovery failed: observe live session panes: action list-panes --json --command: zellij query returned no reply.

The original helper (byte-for-byte verified against fc899cd5d40530707ffe2577a2e0d392c8e7dd31) and original shortcut body were restored with the original-fixture-overlay.json Go overlay. Warm-only count=10 PASS (34.375s), but the exact core park+warm selection count=10 FAIL (47.705s), one warm failure: recovery detached state could not be checked; retry: action list-clients: zellij query returned no reply. Immediate diagnostic list-panes succeeded and showed live wrapper/source panes. This proves the no-reply query family predates #627; it does not claim identical failing CLI actions. Same selection on candidate unchanged head count=10 PASS (48.288s). All controls use the same private environment and run sequentially.

Reproduction selection: PAIR_LIVE_COUCH=1 go test ./cmd/internal/couchcore -run '^(TestParkLifecycleLiveIntentOnlyMutation|TestRecoveryRealHelperAndSessionConformanceLive)$' -skip '^TestRecoveryRealHelperAndSessionConformanceLive$/^checkpoint$' -count=10 -v; original adds -overlay original-fixture-overlay.json. Native empty replies map to launcher.ErrZellijNoReply (cmd/internal/launcher/zellij.go). No production/fixture diagnosis claimed yet.

- /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair627.34fmyl6i/r3-live-target.log: SHA256 89d52bd636f2c6694c47f0e3e279ae05bed9efe23d2bc49504513a69e9bd19cf
- /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair627.34fmyl6i/r3-recovery-original.log: SHA256 362b4eeaed6676e8839b52930f1e72fab8bf21aac26932e91cf7925b37d09676
- /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair627.34fmyl6i/r3-core-original.log: SHA256 a4700a0f4aada38e80878549dfd9b03be429ba55509cc1194a3e665d562ec7e8
- /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair627.34fmyl6i/r3-core-candidate.log: SHA256 8c28e75af317e1a7f9064f98d35515133c65fa617252227710052a9a47cbdc46
- /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair627.34fmyl6i/r3-shortcut-race.log: SHA256 54f57c0bd67c9fbe90e48cda2164ca39859a0f5740e2f9ed871a10d2eb979ec9

### 2026-10-17 — Coordinated read-only audit with #640

Authenticated TL dispatch5aa2c335; claimed in pair:6. Before any implementation/native run, coordinated with pair:1 (submitted request06ba1299; authenticated replyfe007ec6). Read its committed #640 Log at Git-verified2c42749dc01932629eaa2f9d5f1d33f82041e0d4. Pair:1 owns the bounded native-trigger diagnostic and requested all other native runs held; TL's scope there is read-only investigation until about11:30, with no upstream publication or Pair transport/retries. No source/fixture code, native environment, timeout or assertions were changed for #636.

Read-only source audit finds no demonstrated #636-specific cause. ControlledZellij.KillClient (pairlifecycletest/live_zellij.go) joins the local client process but has no server detach/RemoveClient acknowledgement. Warm recovery then proves exact live ownership through list-panes and detached state through separate listings/list-clients (couchcore/recovery_conformance_live_test.go, recovery_execute.go, artifactcollision.go). Local calls are sequential; queued native work can overlap later CLI clients. This is compatible with #640's candidate race, not independent proof of its trigger (ARCH-ORDER).

Production launcher.ZellijSource rejects empty/whitespace successful stdout as ErrZellijNoReply before parsing. Header-only list-clients is a received zero-client response; exact no-sessions exit1 diagnostic is the separate absence proof. SessionOwnerProbe preserves unknown on empty pane responses and checks exact server identity around ownership evidence. Recovery observer errors return before mutation/launch; existing empty-reply stateful-fake and recovery refusal regressions cover those boundaries. No safe cause-level Pair change is justified by the present evidence (ARCH-PURPOSE).

Retained original failure log still exists and its SHA256 matches the filed value: /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair627.34fmyl6i/r3-core-original.log, a4700a0f4aada38e80878549dfd9b03be429ba55509cc1194a3e665d562ec7e8. Its failed warm list-clients is followed by a successful same-session pane diagnostic. That later observation proves the session subsequently answers, not what bytes the failed query received.

Pair:1's /tmp/pair640-mi9l7uyn/protocol-control.jsonl was inspected read-only (SHA256e8de734728e0dabe1e7dfd59ab1ad2cf1dd18cad4edd9135d12dd4edf348acef). It records installed0.45.1 CLI: Connected then JSON Log -> exit0/[]; Unblock before Connected -> exit1/no-active-session; query Unblock before JSON Log -> exit0/empty, with the private listener alive in all cases. These deterministically establish the client failure shapes; the native stale-ID/double-RemoveClient trigger remains unproven. Transparent proxy runs perturb scheduling, so their passes are not acceptance proof. Pair:1's raw native diagnostic is the shared blocker; do not duplicate its native work or weaken unknown handling.

Independent source-level fixture weakness filed unclaimed as #645: park_lifecycle_live_test.go zellijClientCount accepts empty stdout as0 and counts the header; baseline0 followed by header-only count1 can falsely prove helper attachment. No causal connection to #636/#640 is claimed and no fixture edits were made.

If more #636 evidence is needed after native coordination, use the existing WithZellijQueryObserver for bounded in-memory argv/exit/stdout/stderr/timing capture, retaining fixture client PID/start and PTY-close/signal/wait timestamps. Flush after the run: synchronous observer IO or extra queries/proxies can alter scheduling. Do not add retries or infer detachment/absence from silence. Implementation planning remains pending shared native-cause classification and hub scope guidance.

### 2026-10-18 — resumed after #647 / #656
- 2026-10-18: closed — Narrow post-close catalog compatibility delta: two tmp-literal exception comments explain existing private short AF_UNIX allocations; no allocation, cleanup, deadline or executable Go behavior change. New main catalog initially failed both sites, then passed with the annotations; targeted catalog and real recovery/isolation tests passed2.350s/3.820s. Existing 103-byte socket-budget proof and owned real socket remain. Main integrated automatically without code conflicts; authoritative integrated suite remains required by merge. Prior corrected branch suite d6e985ec passed11m36s log d6e007689f46 and close SHIP. Actual N/A remains per hub bff4b081 held-claim overlap with measured #649/#656; no invented hours. No local native lane/operator action. (fixed-then-ship) (done-when: contract moved since change-code; judged by the close review); review verdict: SHIP
- 2026-10-18: closed — Hub-approved diagnostics/isolation delivery, native-cause phase deferred. Fresh recorded suite d6e985ec8aa2fbc498f9b1689bf07a202ae921f8 exit0 11m36s, log d6e007689f46, one attempt; all PR431 server checks green including macOS conformance3m49s. BR-1 fixed via query-boundary caller/effective deadlines; actual-adapter shorter-child regression RED before/GREEN after; absent/shorter/longer caller cap tests and targeted normal/race observer/recovery/isolation/capture families pass. Shared fake12 faults refuse before effects or retry; poisoned parent/nested roots and six guard mutations; bounded64-record/8MiB exact capture, cancellation/LIFO and counted8/16 traces. Assertions/deadlines/query counts unchanged. Atlas/lesson and catalog follow-up recorded. No local native lane/operator stage/restart. Only actual waived per hub bff4b081: held claim overlaps already-measured #649/#656 so9.28h double-counts; N/A excludes calibration, no invented replacement; hub owns ariadne fix. (done-when: contract moved since change-code; judged by the close review); review verdict: SHIP

Hub dispatch f21bd6b6 retained pair:6 ownership after an accidental pair:1 dispatch; follow-up700ec68b directs diagnosis on strict decoding/shared fake now on main, plan before code and hub approval before any native lane. #656 landed PR427, Git-verified merge9265395740b065c03a2ff0fc317a9dbf754616a9, and this branch starts from861e4f1de857889c0d55d716b026828f88338f58. start-plan first refreshed the local ariadne tool dependency and instructed a repeat; repeat succeeded. No #636 code edits or native runs yet.

All five retained pair627 logs still exist with the original filed hashes; original overlay/native-private-env files also survive. Protocol control /tmp/pair640-mi9l7uyn/protocol-control.jsonl still hashes e8de734728e0dabe1e7dfd59ab1ad2cf1dd18cad4edd9135d12dd4edf348acef. The historical failed query's own stdout/stderr, native IPC ordering and exact detach timing were not retained; later successful pane output cannot fill that gap. Original overlay paths refer to slot1 and older sources, so it is not a current-main control without reconstruction.

Read-only source audit confirms strict client/pane decoding and unknown preservation now exist, including no automatic retry. Existing high-level recovery observer-error tests bypass the production native query adapter; the missing cross-layer regression is RecoverThread with ScopedThreadArtifactCollisionChecker plus shared zellijtest faults at owned-pane and list-clients observations. Current ControlledZellij.KillClient still closes PTY, signals SIGTERM and waits locally without a server detach acknowledgement; this is a hypothesis boundary, not the proved cause (ARCH-ORDER).

New fixture isolation evidence: testpair.ApplyEnvironment clears inherited environment via tests/lib/pair-test-env.sh. The warm fixture names only TMPDIR/SHELL/config/data overrides; park adds cache. Neither explicitly retains ZELLIJ_SOCKET_DIR/XDG_RUNTIME_DIR, so the historical outer environment file does not establish current effective socket/runtime isolation. Before native testing, explicitly private roots must cross this existing boundary and be asserted. The old failed run also shows first-run setup wizard panes; it is a fixture-state difference to retain, not a cause claim (ARCH-SECURE).

Read-only baseline: launcher/fake selected empty-reply/client/owner/diagnostic tests passed (launcher26.092s, zellijtest5.091s; zellijreply selector matched no tests). Full zellijreply/zellijtest tests then passed0.136s/13.564s. No native Zellij commands executed. Diagnostic plan pending hub approval; no production retry/deadline/parser change proposed.

### 2026-10-18 — reopened after diagnostic landing

Reopened per hub receipt3d185bf9-4735-409e-afc0-51281a306e7d, verified submitted. PR431 landed the diagnostic and fixture-isolation phase (merge1e90001c06796ba600c77d3cde3255ad6818f611); authoritative integrated suite passed13m01s, log75d0b1b32f85, all five checks green. The original native no-reply cause remains unproved: its own failed-query bytes and IPC ordering were not retained. Leave the issue available for the next recurrence, which now captures its own bounded bytes, query deadlines and local lifecycle ordering. No separate native window is approved; no further experiment or operator restart is requested. Release the claim and stay idle pending hub dispatch.

## Revisions

- 2026-10-18 — Hub approval5a7cb535-36ce-47e9-963b-4195132eb96e, receipt verified submitted, revises this delivery to diagnostics plus isolation fix first, normal gates/landing, then ask whether native work remains worthwhile. Current Done-when now names that concrete delivery; the original cause/repeated-native objectives are retained above as deferred historical scope. Durable plan tasks1-4 and final gates are executable now; proposed native/cause phases require a later hub decision. No permission to use a native lane or stage/restart was given. This is explicit scope authorization, not an inference from a quiet slot.

### 2026-10-18 — approved implementation and guard coordination

change-code passed inferred quick flow after the hub-approved scope revision. Independent bounded tasks own the production-path shared-fake tests, explicit private environment helper/poisoned-ambient regression, and deferred query recorder. Root/child propagation is one isolation class: startParkLifecycleHelper rebuilds a child environment and also must explicitly retain the selected roots. No native binary has been executed.

Hub04b60ce2 explicitly approved keeping a local guard now instead of waiting for #665's absent pkg/golint catalog. Notified its owner pair:1 (receipt10518c75, verified submitted) of the environment guard in recovery_live_environment_test.go and future catalog migration after #665 lands. Follow-up is rule placement, not deferral of current runtime/source coverage. Isolation RED was captured with actual harmless direct/nested child reports missing private cache/runtime/socket/config-dir values under the pre-helper setup. Existing strict decoding/recovery behavior remains unchanged.

### 2026-10-18 — diagnostic implementation and targeted verification

Added production RecoverThread/scoped-checker/owner-probe/strict-decoder integration using only shared zellijtest replies, real private Unix socket and fake exact process identities. Twelve faults cover empty/whitespace/wrong-kind/malformed at first/second ownership observations and client query5. Complete durable record/registry remain unchanged, exact argv/ordinal stops at failure, no launch/signal/delete/quiescence, later healthy replies unused. Valid header-only clients control completes warm reattachment. A disposable one-retry production overlay made the regression fail (unknown native reply admitted recovery); no tracked production change.

Shared test-only setup now passes7 private existing HOME/config/data/cache/runtime/socket/config-dir roots through actual parent and nested park child environment boundaries. RED direct/nested child reports exposed missing roots before the helper; GREEN tests verify poisoned ambient isolation, restoration and cleanup. Real harmless Unix socket bind proves current session names fit <=103bytes including resolved /private alias/version directory. Empty default config behavior remains unchanged. Six in-memory source-guard mutations detect omitted setup/propagation. Source guard/rule migration coordination with pair:1 acknowledged in dac99321; ariadne catalog proposal will route through the hub after landing, not expand #665.

Bounded synchronized test recorder retains original completed-query bytes/status/error/timing/caller deadline and local lifecycle markers. It defers JSON logging until after subsequent fixture cleanup, caps64 combined records/8MiB output plus per-record metadata, and explicitly records drops/truncation. TDD RED captured zero-record stub; exact byte/copy ownership, overflow, concurrent marks, canceled fake child and LIFO flush tests pass. 8/16 real fake CLI operations make exactly8/16 queries and zero effects; observer-on/off traces match. Added real recovery failure + recorder integration proving the empty client response itself is retained, with no diagnostic follow-up query. Warm fixture's old post-failure pane query was replaced by this original-request capture; all recovery assertions and query deadlines remain unchanged. Park cleanup markers explicitly say its local wait is not observed there; client wait never claims server detach acknowledgement.

GREEN: combined new family passed10.178s; its targeted race run10.260s. Existing recovery refusal/deadline and park/helper count controls passed6.334s. Existing testpair environment guard fixtures/whole-tree guard passed3.253s. Agent-focused isolation normal/race passed0.334s/3.426s and capture family normal/race passed. git diff --check passed. Remaining package checks and required recorded suite pending. No installed native Zellij, operator session/build, stage/restart or OS clipboard use.

### 2026-10-18 — fixture inventory check

Broader package check passed launcher77.499s, zellijreply0.282s, testpair13.401s and pairlifecycletest1.947s, but zellijtest failed TestEveryZellijFixtureReplyUsesProductionDecoder: the new capture cancellation wrapper was an unregistered CLI reply producer. Inspection confirms it writes only a readiness file and execs the shared stateful fake; it serializes no native reply. Classified that boundary explicitly in the inventory's existing non-producer exclusions, with its reason, rather than introduce duplicate generated replies. Targeted contract rerun pending; this is a deterministic inventory omission, not a flake.

The corrected zellijtest package then passed9.342s; all targeted package and race checks are green. Diagnostic tasks1-4 are complete; remaining work is the required recorded suite, close review and landing under the revised contract. Native/cause-level phases remain deferred to the post-landing hub decision.

### 2026-10-18 — recorded suite and server conformance

Required recorded sdlc-local/suite passed on pushed ee34617a25bdacceacaf831da92c5441fd0f20d6: make PAIR_GO_TRIMPATH=-trimpath test, exit0 in11m10s, log sha256 prefix0797bc40085b; queue wait1235s. One full-suite attempt, no failures/retries. All Go, shell/editor and diagnostic checks passed. Log: /Users/xianxu/workspace/pair/.git/sdlc/local-checks/ee34617a25bdacceacaf831da92c5441fd0f20d6/suite.log. Closing this tested branch without manually chasing newer main.

PR431 server checks are green, including conformance (3m0s, run39139067645/job117709529278). The existing macos-latest workflow installs Zellij0.45.1 and runs Makefile.local test-couch-zellij-live, whose selection includes park intent-only plus warm recovery, unchanged assertions. This is a separate remote healthy conformance sample, not a local native lane or proof of the original trigger. No local native run, operator stage/build/restart or clipboard action occurred. The post-landing native-window decision remains with the hub under approved revision5a7cb535.

- 2026-10-18 — Close preflight refused before any review because the final checkbox included later lifecycle actions. Split completed implementation/suite preparation (checked with evidence above) from the still-pending close/landing/post-landing hub question (prose instructions). Scope and acceptance are unchanged; no gate bypass.

- 2026-10-18 — Actual-hours exception approved by hub receipt bff4b081-b7b1-4d8e-9070-b59f07bd4cc4 (verified submitted). The measured 9.28h spans the held claim from 2026-10-17 10:14 to 2026-10-18 08:41, overlapping #649 and #656, which pair:6 worked and which already adopted their own measured hours. That double-counts slot time; no replacement number is fabricated. Close uses only --no-actual, records N/A and excludes this issue from velocity calibration. Hub is filing the owning ariadne measurement fix so slot time counts once toward the issue actually worked. All verification and review gates remain required.

### 2026-10-18 — close round1 BR-1

Close returned REWORK for observation-context-fidelity: the capture callback closed over its installation context and misreported later shorter query contexts. Confirmed with a new actual-adapter/shared-fake regression: a4s child was recorded as its1min parent (RED). Query observer now carries actual caller and effective capped deadlines, and capture consumes those facts. Added absent/shorter/longer caller coverage at the query adapter; no timeout, retry, subprocess or query-count change. All inherited capture consumers use the same corrected event path (ARCH-PURPOSE). Corrected the durable plan selector, added the review lesson; normal targeted observer/recovery/isolation/capture tests passed launcher0.876s/couchcore5.689s. Fresh race/suite verification follows before the next review.

The same targeted family passed with race detection: launcher1.806s, couchcore7.075s. The initial shorter-child test failed before the fix and passed after it. Fresh recorded suite is required for these source changes.

Fresh recorded suite on d6e985ec8aa2fbc498f9b1689bf07a202ae921f8 passed exit0 in11m36s, log sha256 prefix d6e007689f46, after1026s queue wait. One attempt, no failures/retries. Log: /Users/xianxu/workspace/pair/.git/sdlc/local-checks/d6e985ec8aa2fbc498f9b1689bf07a202ae921f8/suite.log. Updated PR431 server checks all pass, including macOS conformance3m49s (run39142223454/job117718804022). No local native lane used. Ready for close round2, with actual-hours exception unchanged.

### 2026-10-18 — landing integration catalog

The landing gate integrated main including #665 (fab5a7861807aa4fcc26e7826a43e3ebd2f40e3d). Targeted new catalog check found the two deliberate short /tmp socket roots; these are not ordinary temporary storage: the native session/version leaves and actual ownership-probe socket must fit AF_UNIX limits even under macOS long TMPDIR. Stopped only our own waiting merge before edits. Added the catalog's precise tmp-literal exception comments at both allocation sites, retaining existing private allocation/cleanup and actual socket-budget tests. Catalog and recovery/isolation tests then passed2.350s/3.820s; git diff --check passed. No executable Go behavior changed. The merge gate still owns authoritative integrated-suite validation; no suite was canceled and no blind retry used.
