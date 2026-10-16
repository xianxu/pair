---
id: 000544
status: codecomplete
deps: []
github_issue:
created: 2026-10-13
updated: 2026-10-16
estimate_hours: 6.29
card_mirror: '5bf85a660bb03e22388415d599acd448ef14af5a' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-14T01:55:19-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
actual_hours: 16.95
---

# couch: let the hub drive a slot's agent keys (switch model/effort remotely), without message chrome

## Problem

The operator wants the hub (TL) to switch a slot's model and effort remotely. Every agent does it through an interactive picker (`/model`, then arrow keys and Enter; Codex adds an effort step; Claude and others differ). `couch --send-to` can't do it: it delivers a prompt wrapped in peer-message chrome ("[Couch peer from …]"), and can't send arrow keys. `couch --switch-agent` changes the agent, not its model.

A manual drive of idle pair:6 (Codex) on 2026-10-13 worked, from the hub, with zellij and `couch --peek`:
- Target: the slot's zellij session (`ZELLIJ_SESSION_NAME` from the wrapper process's environment; wrapper PID in `pair-wrap-pid-<tag>`) and the agent pane (`pane_id` in `pane-<tag>-<agent>.json` → `terminal_<id>`).
- `zellij --session S action write-chars --pane-id terminal_N '/model'` typed into the agent's input box.
- **Plain Enter (CR) inserts a newline in the agent pane** (Pair's deliberate mapping, pair#162); **Alt+Enter (ESC CR) submits**. Inside the agent's picker, plain Enter selects.
- Arrows `ESC [ A` / `ESC [ B` moved the picker; a lone ESC (sent separately) backs out; Ctrl+U clears the input line.
- `couch --peek` between steps showed the picker state (the highlighted row is reverse video), so each step can be verified.

## Spec

Two layers, hub-only, idle slots only (refuse when the agent is busy or the agent’s composer has unsent text):
1. **`couch --keys repo:N KEYSPEC…`**: inject named keys into the slot's agent pane, with no chrome: `text:…`, `enter` (plain CR), `submit` (Alt+Enter), `up`, `down`, `esc`, `clear-line`, `wait:MS`. Couch resolves the session and pane itself (no env scraping), writes through the slot's terminal, and returns the agent pane's tail after the last key, like `--peek`. One request, one bounded sequence; no retries.
2. **`couch --set-model repo:N --model NAME [--effort LEVEL] --confirm`**: per-agent recipes built on (1) that open the picker, find the row by its visible label in the peeked screen (not by position), select it, handle the effort step where the agent has one, and verify the result from the agent's own status line (e.g. Codex's footer `GPT-6-Astra medium`). If the label isn't found, back out with ESC and report the visible choices; never guess.

Agents' pickers differ and change across versions; recipes live in one table per agent, tested against recorded screens. The slot-continuity target's spirit applies: one action, an honest result, no limbo (a failed drive backs out to the agent's prompt).

## Done when

- `couch --keys` drives an idle scratch slot's agent pane (text, submit, arrows, esc) and returns the tail; it refuses busy slots and non-hub callers.
- `couch --set-model` switches a Codex slot's model and effort and a Claude slot's model, verified from each agent's status line, with recorded-screen tests per agent; an unknown model backs out cleanly and lists the choices.
- The couch skill documents both, and the hub's TL skill notes the recipe.
- `--set-model` defaults to session-only scope, selects shared defaults only with explicit scope, and reports the observed model/effort/scope.
- Model-control confirmation lines do not generate false “waiting on a prompt” notices; actual questions and pickers still do.
- Exact caller/slot ownership and wrapper identity are revalidated; busy/unknown or occupied agent composers refuse, and failures never automatically replay uncertain keys.

## Plan

Durable implementation plan: [agent keys and model selection](../plans/000544-agent-keys-model-plan.md).

- [x] Implement typed bounded key driving and verified per-agent model recipes through the exact-bound wrapper.
- [x] Wire hub ownership/idle guards, receipts and model-control submission accounting; prevent false prompt waits.
- [x] Verify recorded screens, stateful socket/PTY sequences and isolated native conformance; update docs. Close and land through SDLC; live fleet model/keys/delivery acceptance is post-landing, owned by TL ops:0 under the revision below.

## Log

### 2026-10-14 — implementation and conformance checkpoint

- Implementation HEAD ac8de5d4cc974bd461a4f6d40dfeefa7b610d70e built cleanly; bin/couch reported that exact revision with vcs.modified=false. Typed input uses the existing writer, exact binding/session/ownership guards, coherent current-viewport evidence and bounded retained outcomes. No credential plumbing or ops TL skill edit.
- Ad-hoc fresh-eyes review found stale cached picker actions, decision-time confirmation fences, Return-remap-disabled submission accounting and mixed terminal snapshots. Regression tests reproduced each class; fixes bind semantic picker proof to an atomic terminal generation, fence after actual emission, retain normal raw-input accounting, and separate visible viewport authority from scrollback evidence. Operator interruption is proved through the real input loop and a blocked context-aware writer.
- Real isolated Codex0.162.1 new-driver conformance passed all three cases (15.615s): session selection confirmed in five writes with disposable defaults unchanged; explicit default confirmed in five writes with only disposable defaults changed; unsupported effort recovered to a fresh empty composer in six writes with defaults unchanged. Native configuration bytes stayed unchanged in every case; model controls produced lifecycle generation0 and no submission events. Live session race check passed6.335s; screen fuzz passed37,570 executions.
- Recorded Claude2.1.299 evidence includes the combined picker, left/right effort, s=session, successful “Set model to Opus 5.5 for this session only” line and medium footer. Claude default scope refuses before input because default success evidence is absent. Actual new-driver Claude conformance remains pending with the hub/operator.
- Full affected packages passed: wrapcmd136.278s, couchcmd51.915s, couchmessage7.240s, agentdrive0.160s and artifactpath8.928s; prior unchanged couchcore214.905s, couchtty36.808s and ttyio5.773s also passed. Root focused wrapper/drive/socket races and docs/cost checks passed. Initial broad verification exposed master-pump fixtures missing the newly required transport; six direct fixture sites now initialize it and affected races pass. An initial full make -k run compiled during the viewport API change and failed only those build targets; final stable implementation full make -k test completed with exit0 after admission at the native gate. The complete shell and Go suite passed; log /tmp/pair-544-full-suite-final.log. No gate bypass or custom admission loop.
- TL receipt097a24f1-6418-4492-806e-079efcafced9 verified: operator-owned restarts and live Claude check are deferred to morning, after pair#528 M1 acceptance. Hold before close. A stage/build started before that instruction was received; it restarted no process, was cleared back to main, and main was rebuilt cleanly at b63f4bd76e3e81d1433cfeda544bb3bed8501522 (vcs.modified=false). #549 PR328 continues to await the operator’s protected-path label.

- 2026-10-14 — wrapper key-to-delivery handoff test added through translateStdinFrom: a raw key drive reaches a fresh empty composer, then normal Couch delivery pastes, submits and confirms. Temporarily omitting drive lease release reproduced the exact queued “waiting for previous automatic input to clear” failure (RED0.373s); original production code restored, focused test GREEN, ten race repetitions PASS3.825s; all focused AgentDrive/PeerDelivery/PeerDraft race checks PASS5.724s. No production change. This new test will be included in the recorded close suite on resume; prior full-suite evidence remains the earlier implementation run. Live Claude hold unchanged.

- 2026-10-14 — P0 interruption checkpoint: staged dbdde238 live smoke failed on Codex ariadne:5 (operation12f7dea1), leaving /model plus newline and refusing explicit cleanup; Claude remained disconnected before focus (#528). Stage-clear is queued behind pair:3’s SDLC merge lock. Committed design Revision5e98ce5be, service recovery guards5f18f1dc1, pure recovery15ce67b7a. Pure/DTO APIs: strict Request.RecoveryOnly; Screen.Recoverable+ComposerText; DriveObservation.Recoverable; current-screen EventTick with min3s/remaining10 reserved within absolute budget. Pure cleanup supports exactly singleline /model and owned recognized pickers; no multiline cleanup without capture evidence. Service authority/session/freshness tests pass with races; pure tests/races pass. Wrapper recovery projection/admission (including AwaitingEmpty lease), current-screen ticks and slash-content extraction remain UNIMPLEMENTED. Submit root cause remains unproved: logical KeySubmit already emits ESC CR before wrapper translation, which native isolated conformance accepted; investigation of real pane/protocol is pending. No repaired-build or smoke-success claim. TL’s P0 native-procutil reattach hotfix takes priority on a fresh main-based issue; all #544 subagent edits are stopped and committed. Return here after P0, finish recorded/live fixes, then re-smoke and recorded close suite.

- 2026-10-14 — read-only submit investigation checkpoint: target chain is one Pair wrapper → node Codex → native Codex0.162.1; no double wrapper. Live target TERM=pair-vt-256color/TERM_PROGRAM=ghostty/privatePairTERMINFO negotiates CSI>5u and CSI?u, whereas scratch conformance TERM=xterm-256color answered DSR only and captured CSI>7u. No native tui keybinding override or PAIR_WRAP_REMAP_RETURN override. Ordinary kitty Alt+Enter trace translated to native CR; drive writes lack byte tracing, so submit failure remains unresolved. Follow-up must exercise actual Run/inputloop/endpoint and Pair virtual-terminal keyboard query responses, plus isolated multiline CtrlU clearing capture. No live input or scratch PTY was started by the investigator.

- 2026-10-14 — resumed after #553 landed, authorized by verified TL receipt fa4556db-8b96-499c-8856-9fe7a23e4944. Checkpoint aa6ef7d2 restored and origin/main merged cleanly at 006843476c28241b713bb2b47c843186f7e8e312. Pair:0 remains reserved for #528; no staging. Focused agentdrive/couchmessage/couchcmd drive, key and wait tests PASS (0.190s/0.156s/1.125s). Wrapper recovery integration and real Run/socket/native input-protocol conformance proceed as bounded independent tasks. Reference retrieval RED: README, atlas and embedded skill categorically refused occupied-composer recovery; documentation updated to describe only strict ESC/clear-line recovery, preserving empty-composer guards for all other requests and the same absolute deadline.

- 2026-10-14 — wrapper recovery committed at 0fdc58f3b: typed Recoverable proof for occupied Claude/Codex composers, strict recovery-only takeover of awaiting-empty leases, exact single-line /model timeout proof and current-screen ticks. Initial tests reproduced occupied refusal, absent timeout cleanup and a subsequent ESC escaping after recovery proof changed; all are GREEN. Split display-only screen snapshots from peer-mutex-owned mailbox/lease projection to preserve concurrent reader safety. Root focused races PASS: agentdrive1.306s, wrapcmd9.450s, couchmessage1.214s, couchcmd2.356s; inventory PASS9.034s. Submit protocol remains under actual Run/socket/native-child investigation; no repaired-build or stage claim yet.

- 2026-10-14 — measured repair checkpoint: native actual Run/socket RED showed `/model` plus CR coalescing as multiline text; CSI13u did not fix it. A 150ms quiet interval opened the picker. Version-pinned Codex source documents the 120ms Enter suppression window; terminal negotiation was a useful test gap but not the proven cause. Marker-only native picker evidence added, preserving unique/complete/stale guards. Parser7f824760e and real Run/native fixture12d1ec57a/519bce237 committed; production pacing/recovery budget ed5f553c52995bde0b2a3a24d2451d3c3cb1918e. Deterministic RED/GREEN covers early Enter, cancellation, pending submit at absolute deadline, reserve preemption, and unchanged pre-text idle falsely claiming recovery. Cleanup acknowledgments retain the backout phase.
- Exact ed5f553c live evidence: actual Run scratch+Pair virtual PASS22.682s, each session model/effort confirmed plus explicit dirty `/model` clear-line recovery and final idle. Direct native session/default/unsupported-effort PASS18.972s; session defaults unchanged, only disposable explicit-default config changed. Native CR bytes unchanged. Real multiline Ctrl+U leaves `/model` behind; no whole-composer cleanup claim. Clean local build reports vcs.revision=ed5f553c52995bde0b2a3a24d2451d3c3cb1918e, vcs.modified=false. Focused combined races PASS (core1.400s/wrapper11.717s/protocol1.282s/service2.812s). No peer input or staging.
- Ad-hoc review found no further production issue; correcting conformance-fixture diagnostics to slice one owned snapshot and fencing native group cleanup by start identity. The recorded suite attempt on ed5f553c was cancelled with SIGINT/exit130 while still queued, before admission, so the final fixture commit gets its own recorded run; no passing full-suite claim. Full affected-package races running. Operator/hub live Claude and renewed smoke remain pending; pair:0 stays reserved for #528.

- 2026-10-14 — fixture hardening a9609f1e5a03adc7b3c5ec5d3a3ea240031c0647: diagnostics use one owned snapshot; group signals require matching start identity, with gone/reused candidates skipped and permission/unknown failures refused. Relay teardown joins native child and both pumps with bounded output drain and an acknowledged cleanup marker. Blind-signal mutation RED; safety tests/races GREEN. Actual Run scratch+Pair virtual GREEN18.469s after these fixture changes. Local Couch rebuild at a9609f1e reports vcs.modified=false. Recorded sdlc-local/suite running on a9609f1e, log at the Git common directory sdlc/local-checks/a9609f1e5a03adc7b3c5ec5d3a3ea240031c0647/suite.log.
- Broadened unfiltered package races: core PASS1.363s and protocol PASS9.302s, but wrapcmd FAIL566.450s due TestCodexWorkingNotificationIsSuppressedByCouch (codex_working_test.go:249 Emulator.Close versus :252 reader); couchcmd FAIL128.659s due TestHubSupervisionReplyEvidenceAndResend (supervision_test.go:428/429 fixture callback writes versus message_service.go:657 worker reads). These fixture files and emulator code are unchanged against main. Separate focused baseline reproduction is running on origin/main e1cab8eca2041604827a7bda25ed958da5eab7af. Do not claim the full race run passed. File confirmed inherited races as separate unassigned issues; no repairs in #544. Regular recorded suite continues independently; re-smoke remains after #528 acceptance, with no staging yet.

- Recorded full suite PASS exit0 in10m56s on a9609f1e5a03adc7b3c5ec5d3a3ea240031c0647, status sdlc-local/suite, log sha256 prefix387183db2c9b. Entire shell/headless/Go run green, including couchcmd114.660s, couchcore348.774s and wrapcmd40.297s. Later commits merge/publish issue prose only; no production or test source changes since the recorded head.
- Main baseline e1cab8eca2041604827a7bda25ed958da5eab7af: exact Codex-working race test failed count10 in7.305s with matching Close/Read stacks. Filed/published #554 (race in unchanged files, found by #544's full-package race run) and confirmed it is open with no owner. Exact hub-supervision race test passed count10 in2.932s; baseline status remains unconfirmed, so no second confirmed-main issue was filed. Baseline logs are supplementary at /tmp/pair-baseline-codex-working-race.log and /tmp/pair-baseline-hub-supervision-race.log; temporary checkout removed. No unrelated fixture repairs in #544. Await hub scheduling for renewed Codex/keys smoke and live Claude Opus5.5/medium/session acceptance after #528; no stage, restart, close or landing performed.

### 2026-10-15 — integrated verification and live acceptance proposal
- 2026-10-15: closed — BR-1/2/3 disposed in prior reviews. BR-4 controlled fixture RED then GREEN; qualification-loss admission/dispatch regressions assert refused Step0 and zero bytes, race count10 PASS3.918s. Exact make test-agent-drive-live PASS42.876s, all five native cases/no skips. Production safety policy unchanged; fixture never retries uncertain input. Final integrated full suite required at merge; TL owns post-landing fleet smoke per receipt35fe918a.; review verdict: SHIP

- Clean integrated commit23e4f40a96f231da94daf5e0b10a4cc0241d7578 builds both Pair and Couch with matching vcs.revision and vcs.modified=false. Combined drive/recovery/resume/operation targeted races PASS: agentdrive1.331s, wrapcmd9.169s, couchmessage1.195s, couchcmd7.549s. Protocol/endpoint focused races separately PASS2.743s/3.567s. Source inventory PASS18.126s; core operation declarations/recovery races PASS27.187s. Both formerly failing unrelated fixture cases now pass count3 on integrated main (wrapper3.950s/service1.884s).
- Exact clean commit23e4f40a native Codex0.162.1 conformance GREEN: actual Run/socket scratch+Pair virtual39.838s, each session selection and explicit dirty recovery with joined relay cleanup. Direct session/default/unsupported-effort44.984s; session defaults unchanged, only disposable explicit-default config changed, shared native configuration unchanged. Submit bytes remain CR; no peer input or staging. Recorded suite queued through the normal machine gate on that same commit.
- Live proposal for the hub after suite PASS: stage the clean branch on pair:0 with build, then batch an operator Couch restart and relaunch only the selected idle Codex/Claude wrappers onto it. Main lacks the new broker verbs and endpoint capability. Capture each slot's empty-composer/model baseline first. From the owning hub run session-only set-model with confirm: Codex GPT-6.1-Sol/medium, Claude Opus5.5/medium. Require fresh session confirmation, matching effort and empty composer; no shared-default selection. For --keys on the idle Codex slot, send text:smoke-544 plus wait:200, verify the draft text is visible, then send clear-line as a separate recovery-only request and require a fresh empty composer. Follow with one authorized ordinary Couch message and its submitted receipt/response to prove lease handoff. Inspect operation-status and peek on uncertainty; do not replay automatically. Record actual slot IDs, operation IDs, results and tested binary hash. No live actions until the hub schedules them.

- Recorded integrated suite PASS: `sdlc local-check run --name suite` on exact source commit23e4f40a96f231da94daf5e0b10a4cc0241d7578, exit0 in12m13s after727s native gate wait. All shell/editor integrations and `go test ./... -count=1` passed, including couchcmd111.960s, couchcore472.304s, couchmessage8.187s and wrapcmd76.279s. Saved log: `/Users/xianxu/workspace/pair/.git/sdlc/local-checks/23e4f40a96f231da94daf5e0b10a4cc0241d7578/suite.log`, SHA256 d33ad0dddcef7274ab9e0e9f77b0ef1e31fdceeb83f3f154e0acb5ea0cad4566. Recorded passing sdlc-local/suite status. Subsequent issue-log commits change no tested source. Ready for the hub to schedule the live proposal above; acceptance, close and landing remain pending.


### 2026-10-15 — staged for the combined operator restart

Merged main a0e3e4d27 (including #556 PR349/bcf734588) as e13111215270762629c27e860584429b7ae66a98. Focused race verification PASS: agentdrive1.266s, wrapcmd11.420s, couchmessage1.141s, couchcmd2.488s, launcher2.784s; coverage includes drive/recovery/operations and native admission/selector boundaries. Independent protocol/service recovery races PASS1.367s/3.869s. Core operation/dispatch/recovery-confirmation/candidate checks PASS3.349s. Inventory PASS12.640s; root discovery PASS0.127s. Actual Run/socket/native Codex0.162.1 boundary PASS22.591s (scratch11.16s, Pair virtual11.11s), confirming session selection, explicit dirty-composer recovery, unchanged CR/s bytes and joined relay cleanup. The fixture used the merged source before commit; no source changed between it and e1311121.

`sdlc stage --build` placed e13111215270762629c27e860584429b7ae66a98 on pair:0. Verified its HEAD, clean checkout, and Pair/Couch/launch-helper metadata: exact revision, vcs.modified=false. Staging reports an unrelated pre-existing pair-hoprtt revision warning; Pair and Couch are exact. The declared recorded suite runs on the same source commit through `sdlc local-check run --name suite` (make PAIR_GO_TRIMPATH=-trimpath test). No passing-suite claim yet for this new commit.

Sent the hub the staged hash and exact smoke commands, receipt013b70ca-a5d2-45e7-b7be-f0e3bff6df37 submitted. Proposed currently idle ariadne:5 (Codex GPT-6.1-Sol/medium) and parley.nvim:3 (Claude Opus5.5), recheck/relaunch after the operator's one Couch restart, session-only model/medium changes, raw text and separate recovery clear-line, then ordinary delivery/response. The broker requires a verified owning-hub caller for keys/model, so pair:6 cannot impersonate ops:0; hub execution is needed for those requests. Operator restart, live acceptance, close and landing are pending. No live input performed.


- Final staged-source recorded suite PASS: `sdlc local-check run --name suite` on e13111215270762629c27e860584429b7ae66a98, main-declared `make PAIR_GO_TRIMPATH=-trimpath test`, exit0 in13m14s, recorded as sdlc-local/suite. All shell/editor and Go checks pass, including couchcmd94.287s, couchcore538.155s, couchmessage7.472s and wrapcmd155.119s. Log `/Users/xianxu/workspace/pair/.git/sdlc/local-checks/e13111215270762629c27e860584429b7ae66a98/suite.log`, SHA256 e07d82adbc090d21bd7ac6ae850c305999b0ec542d40b90f953294e33428b261. Later commits are issue evidence only; pair:0 remains staged at the tested e1311121. Await operator restart and owning-hub live smoke, then close review.


### 2026-10-15 — restaged with the agentic-document registry

Verified TL receipt07bd2c8d-c68d-4b63-9452-246f3506ed2c authorizes main merge/restage after #573 landed and keeps smoke on hold for the operator's ariadne:5 UI park retry. Merge of main c88967b551a1416e0024a248ca09871848eb9f33 (including #573/55ae58b12) was conflict-free: source commit6d65c4c4ef29bcdea3ea5b45f3dfdeed41ce1154. Incoming integration adds the registry, session-sync routing and broker admission timestamps; the drive reducer/writer are unchanged. Focused races PASS sessionsync16.757s, couchmessage2.528s, couchcmd23.473s (drive/receipt/session-sync/admission timestamp/registry/rename/delivery cases); inventory PASS21.183s. Diff check clean.

`sdlc stage --build` replaced e1311121 on pair:0 with 6d65c4c4ef29bcdea3ea5b45f3dfdeed41ce1154. Verified target HEAD, clean checkout and Pair/Couch metadata at that exact revision with vcs.modified=false. The unrelated old pair-hoprtt warning remains. Started a new recorded suite on6d65c4c4 using the declared command; it is queued at the native gate, so the prior e1311121 pass is not claimed for this merge. Hub notified of the staged hash and hold. No restart, smoke, or peer input performed.


- Recorded suite on staged6d65c4c4ef29bcdea3ea5b45f3dfdeed41ce1154 completed PASS, exit0 in11m12s, sdlc-local/suite recorded; all shell/editor/Go packages passed, including couchcmd63.316s, couchcore334.554s, wrapcmd88.520s. Log at `/Users/xianxu/workspace/pair/.git/sdlc/local-checks/6d65c4c4ef29bcdea3ea5b45f3dfdeed41ce1154/suite.log`, SHA256 prefix3108442b8880. Verified TL receipt48a986a8-1509-4793-b0b0-781f3d9473a2: wait for BOTH #587 and #569 to land, then merge origin/main, targeted verification and one restage/build for the operator's combined restart. At19:46 PDT both tracker landing outcomes remain not-landed; no merge/restage or smoke yet. #573/#579/#584/#585/#594 are already on origin/main.

- Evidence retention note: the suite command reported the log path and SHA256 prefix above on completion, but the file is no longer present at19:46 PDT (local-check retention keeps only the newest20 commits). The preserved command output confirms exit0 and recorded status; no full SHA256 was recomputed or inferred. A new integrated commit will require its own suite before landing.


### 2026-10-15 — staged the requested tonight restart build

Merged main2156c014cee0af8fa1662064ebfe32d746e6568a at42b2086fb040b9cbcb527fdcc91f09d8a7bbfdd1, incorporating #573/#579/#584/#585/#594/#587 and excluding #569 per verified TL instruction. Three additive conflicts resolved without changing upstream runtime semantics. Targeted races PASS agentdrive1.276s, wrapcmd22.557s, couchmessage1.267s, launcher5.335s, procutil2.581s; service drive/receipt/operation/resource/compatibility subset PASS8.433s. Core operation/resource/native-owner/recovery-confirmation checks PASS3.578s, inventory PASS16.664s. The initial broader service run failed unchanged TestAutomaticInputStallOwningHub at automatic_input_stall_test.go:156 with recipient-not-live; focused -race count3 then PASS1.699s. No baseline reproduction or passing initial aggregate is claimed. Actual Run/socket/native Codex0.162.1 conformance PASS24.968s: scratch11.48s and Pair virtual13.14s, session confirmation, dirty recovery, original CR/s bytes and joined cleanup. No source change after that merged-worktree measurement.

`sdlc stage --build` replaced pair:0 stage with42b2086fb040b9cbcb527fdcc91f09d8a7bbfdd1. Verified exact unchanged target HEAD and clean checkout. Pair, Couch, pair-launch-helper and pair-slug all report that revision and vcs.modified=false. The obsolete pair-hoprtt, outside current GO_BINS, retains2284b41247340aec386afa83eb998502bc14aa40/modified=true; it was not rebuilt or used. Sent the complete metadata and evidence to the hub, receipt726f3453-bc6f-4bed-9d85-f9be5391ec35. Operator restart and hub-run smoke remain pending; no live input or restart performed here. This new source requires a fresh recorded suite before landing; earlier source passes are not substituted.


### 2026-10-15 — full-suite integration failure and fixture correction

Recorded suite on926fa9083 (same production source as staged42b2086f) failed exit2 in13m11s. All other packages passed; wrapcmd panicked reading nil childIO from masterPump. Failed log `/Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/8ba015a2ee32b525ddcc938c42f47306.log`, reported SHA256 prefixf4bb40558229. Focused TestMasterPumpFlushesCaptureOnInjectedTick reproduced the identical panic. Three newly merged #579 diagnostic_buffer_test fixtures start masterPump directly but had not installed #544's owned ttyio transport. Other pump fixtures already use prepareMasterPumpTransport. Added that existing helper before each of the three new pump starts; no production changes or lazy nil fallback. Focused three-test -race count3 PASS1.945s, covering tick flush, EOF/error drain and in-flight resize after owner ends. Full-suite re-verification required; do not substitute earlier passes.

Restart notices bf9e8370 and9bda8abc could not be verified or answered because couch was not on PATH (exit127). No alternate peer channel or binary path used. Stage remains42b2086fb040b9cbcb527fdcc91f09d8a7bbfdd1 with production binaries unchanged. Live model/key smoke remains pending the hub.


- After the fixture-only correction, recorded suite on12d6b46cef82aaea977dad6a5a90325f6035ca2d PASS exit0 in10m7s (make PAIR_GO_TRIMPATH=-trimpath test), recorded sdlc-local/suite. All shell/editor and Go packages passed, including wrapcmd66.797s. Reported log `/Users/xianxu/workspace/pair/.git/sdlc/local-checks/12d6b46cef82aaea977dad6a5a90325f6035ca2d/suite.log`, SHA256 prefix876e98926efc. Production code is identical to staged42b2086fb040b9cbcb527fdcc91f09d8a7bbfdd1; only tests and issue evidence changed. Restart noticef432124b could not be verified/responded to because couch is still absent from PATH (exit127). Live acceptance and close remain pending; no replay, peer input or stage change.

### 2026-10-15 — recovered state and TL landing decision

Previous recovery screen is unavailable. Clean pushed recovery HEAD was fb11e0c81; saved suite log on12d6b46ce still exists and its SHA256 is876e98926efcec6328b2a0cbcda300b1212fc5f16c13a3e9c9eb0896fb49d49e. Production is identical to42b2086f. Current pair:0 has returned to main and its clean binaries report f136934d03b953354de3eab53bcb0575700897f2, so historical staging is not current runtime proof.

Verified submitted TL receipt35fe918a-6e29-4552-a88c-b95671e9822a from ops:0 to this incarnation directs landing now, integrating current main and using the normal review/suite/merge gates. TL reports the production-equivalent42b2086f fleet Couch ran20:27–23:13 tonight without regressions attributable to it. This is operational soak evidence, not successful model/key acceptance. Live session-only Codex and Claude set-model, raw keys/recovery and ordinary delivery handoff acceptance move to main after the next operator Couch restart, owned by TL ops:0. No live-success claim or worker-driven restart is made.

### 2026-10-15 — current-main integration and native fixture reconciliation

Integrated current main at2b602b9a5; only conflict was additive artifact inventory, retaining both source sets. Targeted races PASS agentdrive1.254s, couchmessage1.249s, wrapcmd6.840s, couchcmd2.430s; inventory PASS10.754s and diff check clean. PR362 opened. Recorded integrated suite started normally on2b602b9a5.

Additional actual Run/native conformance RED twice: wrapper never registered, although Codex reached its empty prompt. New main #597 TestMain clears Couch thread scope/tag for ordinary tests; this subprocess fixture had depended on those inherited values. Restore only its disposable scope/tag inside the explicit boundary helper after TestMain (no production change), and retain bounded wrapper diagnostics for registration failures. Actual Run/socket/native conformance now GREEN18.770s: scratch9.18s and Pair virtual9.29s, model/effort/session confirmation, dirty recovery and joined cleanup. The final fixture change needs a recorded suite on its own source before landing; do not claim the2b602b9a5 run covers it.

### 2026-10-15 — first boundary review and failure-evidence correction

Close round1 returned REWORK with BR-1 (failure choices lost after backout) and BR-2 (prospective Core concepts names diverged from delivered implementation). No close finalized. Tests first reproduced lost model labels after completed recovery for both Codex and Claude, lost Codex effort choices after two-step recovery, and lost Claude observed effort on unsupported requests. Retain bounded whole labels in Result.Detail, independent of latest Tail; cap diagnostics below the existing1KiB transport/receipt bound and explicitly name omitted choices. No new wire field or parallel result store. Broker receipt/CLI tests verify choices and current empty prompt in plain and JSON results. Model failure and recovery authority remain unchanged (ARCH-DRY, ARCH-ORDER).

Targeted drive/boundary races PASS core1.271s, protocol1.343s, CLI/service3.287s, wrapper9.896s; full pure-core races after the final Claude diagnostic correction PASS1.270s. Durable plan appends the current symbol inventory and reconciles explicit per-agent parsing with the shared reducer; no nonexistent Recipe entity or RemoteEndpoint.Drive API remains in the active inventory. Prior2b602b9a5 suite was deliberately interrupted exit130 after the helper changed; it is not passing evidence. The91b007dca suite remains the earlier-source run; the production diagnostic fix needs a final recorded suite before landing.

### 2026-10-15 — second boundary disposition and executable conformance

Round2 explicitly addressed BR-1/BR-2, including reviewer mutation proof that removing choice preservation fails all four recovery regressions. Verdict FIX-THEN-SHIP did not finalize: BR-3 Important identified the absent committed native conformance entrypoint. Added Makefile.local test-agent-drive-live, enabling both native suites, and documented prerequisites/after-change-or-upgrade cadence in atlas and the plan revision. Exact make target GREEN41.648s with no SKIP: direct-native session5.55s/default8.70s/unsupported-effort-recovery5.86s, actual Run scratch11.05s/Pair virtual10.17s. Native Codex0.162.1; only disposable defaults changed in the explicit-default case. No fleet input.

Interrupted the superseded91b007dca suite with SIGINT/exit130 after review changed production; no passing suite claim. The queued66a653100 suite was cancelled before admission (exit130) when BR-3 changed the Makefile. Final recorded full-suite evidence will be obtained on the final integrated landing head through sdlc merge, per the TL's instruction, rather than retain obsolete queued candidates. No gate waived.

### 2026-10-15 — deterministic native recovery synchronization (BR-4)

Round3 addressed BR-3 and reproduced one native fixture refusal after successful model selection; two follow-up runs passed, proving an ordering-dependent fixture, not unsafe production input. Root cause: boundaryDirtyRecovery replaced its initially qualified observation without rechecking Recoverable; it also handled only admission-time stale-sequence refusal, not the dispatch-time guard. Extracted the test-only recovery endpoint seam preserving old behavior; controlled qualified→pending→zero-input dispatch-refusal regression was RED. The fixture now requalifies Recoverable/non-idle before every NEW request, polls that request to terminal, and permits at most three distinct IDs only after an exact known admission/dispatch refusal with OutcomeRefused and Step0. Unknown, cancelled, partially acknowledged or unconfirmed outcomes never retry; all observations/attempts share the outer absolute context. Production code and conservative refusal policy are unchanged.

Controlled wrapper tests independently enumerate sequence or screen changes before admission and before dispatch, assert OutcomeRefused/Step0/zero writer bytes, and prove later repaint does not replay. Fixture regression and no-uncertain-replay tests plus those four production interleavings PASS -race count10 in3.918s. Exact make test-agent-drive-live GREEN42.876s, both suites/all five cases executed. Three additional actual Run/socket virtual-suite repetitions are running as a load-ordering check; no passing repetition claim yet. Final integrated suite remains due at merge.

### 2026-10-16 — close cleared; integrated landing next

Actual Run/socket native virtual conformance count3 PASS74.287s, covering three scratch plus three Pair virtual model/effort/session selections and explicit dirty recovery. Round4 SHIP with no new findings and all BR-1–BR-4 addressed; reviewer independently mutation-tested qualification and dispatch-refusal handling and ran exact native target PASS38.050s. Codecomplete published by sdlc close, bound to evidence2ee17868c9f3, measured actual16.95h (build16.63/converge0.32). The TL-owned post-landing fleet smoke remains outstanding. Proceed through sdlc merge --yes for fresh-main integration, required recorded suite and CI; no bypass.

## Revisions

### 2026-10-15 — synchronize conformance across qualification changes (BR-4)

**Reason:** Fresh-context review observed the native fixture fail on a legitimate zero-input refusal between admission and dispatch.

**Delta:** Keep production fail-closed behavior; positively qualify each new test request, await its terminal result, and only after an exact known zero-input refusal reobserve and use a new ID within three attempts and the original outer deadline. Never replay uncertain input. Controlled endpoint and actual wrapper tests pin qualification loss before admission/dispatch and subsequent no-replay. Re-run exact native target and repeated virtual conformance (ARCH-ORDER).


### 2026-10-15 — native conformance target (BR-3)

**Reason:** The planned opt-in entrypoint was missing despite implemented and manually executed native tests.

**Delta:** Commit make test-agent-drive-live enabling both direct and actual Run/socket conformance suites; atlas and durable plan name prerequisites and require execution after drive/parser/writer changes or Codex upgrades before landing. Preserve normal suite isolation; no new CI/local-gate configuration or peer input (ARCH-MOCK).


### 2026-10-15 — preserve choices through failure recovery (BR-1/BR-2)

**Reason:** Boundary review verified loss of actionable picker evidence and drift in the prospective plan inventory.

**Delta:** Preserve model/effort observed labels in bounded failure Detail through prompt recovery, existing receipt clipping and CLI output. Long lists explicitly report omitted labels. Sweep missing-choice branches for both agents; regress completed recovery and receipt output. Reconcile active plan concepts/APIs/conformance locations by an appended inventory revision, retaining the original design as history. No change to input effects, authority, deadlines or supported scope.


### 2026-10-15 — restore isolated conformance identity after TestMain

**Reason:** Main #597 intentionally strips inherited Couch identity from test processes; the opt-in boundary helper must set up its own explicitly.

**Delta:** Set the helper's existing disposable scope/tag after TestMain, preserving ordinary test isolation and all production admission guards. Keep registration failure diagnostics local to the fixture. Red/green actual Run/native conformance proves both scratch and Pair virtual protocol paths again (ARCH-DRY).


### 2026-10-15 — move live acceptance after landing

**Reason:** Verified TL decision35fe918a accepts the recorded suite plus fleet operational soak as landing evidence and takes ownership of live feature smoke on main after the next Couch restart.

**Delta:** Retain all feature acceptance criteria; change their scheduling rather than assert unmeasured success. The final implementation checkbox covers recorded/stateful/native conformance, docs and gated landing; live set-model/keys/delivery acceptance remains explicitly outstanding under TL ownership. Integrate current main including #597/#600, preserve both additive source inventories (ARCH-DRY), run targeted checks and the integrated full suite through SDLC before landing.


### 2026-10-15 — reconcile landed recovery and stall supervision

**Reason:** TL unblocks #544 after #528 acceptance/landing and the operator's main Couch restart. Integrate main3dcdd38c361e4ceed303374099a16ab668c9553d before renewed verification. Nine conflict sites overlap additive documentation, source inventory, candidate-bound recovery receipts, endpoint validation and wrapper shutdown/state fields.

**Delta:** Preserve both typed drive intent/digest and resume CandidateKey through validation, receipt identity and dispatch; retain main's endpoint context/incarnation safeguards and once-only wrapper shutdown while cancelling active drives. Keep both inventory/docs additions. Main's automatic-input reducer now owns alerted residual waits and a monotonic stall sequence: explicit recovery can take an awaiting-empty or alerted residual lease after positive composer proof, while active owners remain exclusive. Route drive begin/finish through that reducer, preserve its sequence and distinguish confirmed final input/recovery from uncertain outcomes. New integration tests first reproduced alerted recovery refusal and sequence reset; both are now GREEN. No broadened input authority or deadline is introduced (ARCH-DRY, ARCH-ORDER).

Rerun targeted races, isolated native conformance and the recorded full suite on the integrated commit. The new broker operations and wrapper endpoint capability require this branch's Couch plus current wrappers in selected idle slots. Propose staging pair:0 for an operator-batched restart only after those checks; no staging or restart is authorized by this revision itself. Renew session-only Codex/Claude model selection and explicit keys/dirty-composer recovery, followed by ordinary delivery to prove the input lease hands off.

### 2026-10-14 — measured submit timing and native selection evidence

**Reason:** Actual Run/socket/native-child conformance reproduced the operator failure in both scratch and Pair virtual terminals: `/model` and CR can reach Codex in one rapid input burst. Changing CR to CSI13u did not fix it. A 150ms quiet gap did open the picker. The pinned [Codex 0.162.1 paste-burst implementation](https://raw.githubusercontent.com/openai/codex/rust-v0.162.1/codex-rs/tui/src/bottom_pane/paste_burst.rs) retains Enter suppression for 120ms after fast text, even after the text has rendered. Keyboard negotiation differs between Ghostty and the old scratch environment, but is not the demonstrated cause. Native raw capture also proves selected picker rows may use a unique `›` without reverse-video styling.

**Delta:** Keep submit bytes unchanged. Share the existing 150ms peer-submit delay as a pure effect NotBefore bound after acknowledged Codex text, covering raw and model Enter/submit. The single writer retains pending effects without sleeping or extending the absolute budget. Reserved model recovery preempts an un-emitted delayed key; absolute-clock checks fence input even before a context callback runs. Require fresh post-write empty evidence for automatic cleanup. Codex selection accepts exactly one explicit `›` under the recognized complete heading/footer, with duplicate/unknown/incomplete/stale evidence still refused. Preserve the recorded reverse-video cases as compatible evidence.

Native multiline Ctrl+U capture leaves `/model` behind after the empty second line is removed; it does not establish whole-composer deletion. Automatic cleanup therefore remains limited to the acknowledged exact single-line command or recognized owned picker. Explicit recovery clears the current line. Actual Run/socket conformance covers both scratch and Pair terminal negotiation plus explicit dirty-composer recovery; direct native cases retain session/default/unsupported-effort checks. No live peer input, restart or staging is authorized here; pair:0 remains reserved for #528.

- 2026-10-14 02:15 — add (TL, from the first fleet-wide remote switch): (1) `--set-model` takes an explicit scope:
  Codex offers "session only" (`s`) vs "save as default" (Enter); default to session-only, since Codex's default is
  shared by every Codex slot. (2) After a remote switch, the agent's own confirmation line ("Model changed to … for
  this session only") must not trigger Couch's 10-minute "waiting on a prompt" notice: three false notices came from
  ariadne:5, parley.nvim:1 and others on 10-14. (3) The verified recipe used (Codex 0.x): Ctrl+U, `/model`,
  Alt+Enter (plain Enter is a newline in Pair's agent pane), ↑/↓ until the reverse-video row shows the label,
  Enter, ↓ to the effort label, `s`; confirm from the footer `GPT-6.1-Sol medium`. Skip slots whose screen shows
  "esc to interrupt" (mid-turn).

- 2026-10-14 — TL clarified the idle precondition: only the agent’s composer must be empty; the separate Neovim draft is unaffected and needs no guard. Existing wrapper Settled is the required evidence. Spec wording updated accordingly.
- 2026-10-14 — planning survey: exact-bound wrapper endpoint, styled terminal tail and single input writer already exist. Peer delivery adds chrome and can run during turns, so reuse its identity/transport and input ownership seams, not its delivery policy. No versioned model/effort/scope captures exist yet; gathering those in isolated scratch PTYs is the first execution step after approval. #549 PR #328 awaits the operator label and is not the base of this independent branch.

- 2026-10-14 — durable plan ready for TL approval. Shared wrapper transport/input lease and pure screen-driven reducer provide one driver; no implementation or estimate yet. Added scope and false-wait acceptance to Done when, matching the existing Revision. All outstanding design uncertainty about actual picker shapes is handled by the first isolated capture step and a stop/revision on mismatch.

- 2026-10-14 — plan approved by TL at 44e2e723. Full-flow change-code round 1 requested precise registered-wrapper lifetime, isolated Claude model auth (existing native-HOME fallback is unsuitable), and named-function test strategies. Added an active refinement for each; retained the approved feature/bounds. TL will update ops/skills-draft/xx-tl/SKILL.md after landing; this issue updates Pair docs only.

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only. Calibration is marked stale: provisional ship wall-clock, not a deadline.

Decomposition: shaping 0.7 design / 0.2 implementation; four pure concerns (keys, recipe, screen evidence, reducer) each 0.8 / 0.6; wrapper protocol/input integration and Couch authority/receipt integration each 2.0 / 1.5; CLI, receipt extension and wait classifier each 0.3 / 0.5; docs 0.1 / 0.2; close review 0 / 0.5; Codex and Claude live discovery each 0 / 0.6. Library availability: existing PTY, terminal rendering, translator, transport and receipt seams are reused; no external library substitutes for versioned picker interpretation and domain transitions. No novel stack multiplier (1.0). Approved plan discounts implementation-phase design by 0.2; shaping is undiscounted. Implementation values below are already scaled by 0.40. Design subtotal 2.34, implementation 3.60, design buffer 15%: 2.34 × 1.15 + 3.60 = 6.291, rounded 6.29h. Tests/conformance are included in component allowances and two discovery primitives; no separate test surcharge.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.70 impl=0.08
item: greenfield-go-module design=0.16 impl=0.24
item: greenfield-go-module design=0.16 impl=0.24
item: greenfield-go-module design=0.16 impl=0.24
item: greenfield-go-module design=0.16 impl=0.24
item: api-integration design=0.40 impl=0.60
item: api-integration design=0.40 impl=0.60
item: smaller-go-module design=0.06 impl=0.20
item: smaller-go-module design=0.06 impl=0.20
item: smaller-go-module design=0.06 impl=0.20
item: atlas-docs design=0.02 impl=0.08
item: milestone-review design=0 impl=0.20
item: real-api-discovery design=0 impl=0.24
item: real-api-discovery design=0 impl=0.24
design-buffer: 0.15
total: 6.29
```

- 2026-10-14 — plan-quality round 2 CLEAN, all three findings addressed. Derived estimate after approval/gate; no implementation yet.

- Capture checkpoint: Codex0.162.1 model picker uses reverse-video selected numbered row; effort picker explicitly offers Enter=default and s=session. Session selection yields fresh model/effort footer plus “for this session only”; disposable default hash unchanged. Claude private credential copy lacks usable tokens and is expired; isolated composer says Not logged in. No native writable-HOME/config fallback used. Asked TL for an isolation-compatible credential-source Revision; raw-key/Codex core can continue independently. Estimate review INFO: auth helper lives in wrapper integration, final checks/CI/landing span integration and review allowances, which may be optimistic.

- 2026-10-14 — TL provided read-only real Claude picker/recovery captures and directed no credential plumbing/Keychain reuse. Observed version2.1.299; Claude combines model/effort and uses left/right for effort, s=session. Plan Revision replaces isolated Claude auth/conformance with recorded evidence and hub-mediated idle-slot live checks. Requested the missing successful session-apply confirmation from the hub; no Claude success claim yet.


### 2026-10-14 09:57 UTC — captured recipe and implementation-seam revision

**Reason:** Captures established the real phases and scope controls; implementation inspection identified existing bounded writer/transport cancellation seams. The durable plan's matching revision is the active delta; earlier text is retained for provenance.

**Delta:** Codex Advanced Reasoning adds a nested Max/Ultra picker; permit at most three separate recognized ESC steps for Advanced → reasoning → model → composer, while retaining the absolute budget and fresh recovery proof. Disposable captures verify both session-only unchanged defaults and explicit default persistence. Hub-supplied Claude 2.1.299 evidence now confirms session apply; Claude default scope remains unsupported before input. No new Claude auth helper or Keychain reuse.

Reuse `ttyio.Writer.WriteContext` and context-aware `couchmessage.Call`; jobs remain owned by console/wrapper lifetimes and the admission deadline, independently of short RPC exchanges (ARCH-DRY, ARCH-FUNERAL). Picker-derived writes carry the atomic Feed generation plus parsed screen proof. A changed generation permits input only when fresh complete picker evidence is identical in phase, selection, labels, model, effort and footer; changed recognized evidence refuses the cached action. Unknown/incomplete screens and initial/raw effects wait without keys within the absolute budget (ARCH-ORDER). Unknown or uncertain effects never replay, and human cancellation writes no cleanup over a new draft. README, atlas, the embedded Couch skill, verb cost labels and the ten-file artifact inventory now describe this bounded control surface; the ops TL skill remains hub-owned.

### 2026-10-14 — viewport authority correction from live conformance

Live-wrapper regressions showed that retained scrollback can contain a former picker after the current viewport changes, or duplicate headings after an identical repaint. Picker decisions and effect revalidation therefore read only the coherent current viewport; retained history is separate evidence for result tails and fresh confirmation occurrence counts. Unknown current views authorize no input. This preserves the original no-input-on-an-unknown-screen intent without refusing harmless identical repainting (ARCH-ORDER).

### 2026-10-14 — wrapper writer handoff after raw key drives

**Reason:** TL reported that a raw Zellij /status injection on parley.nvim:1 at07:40 bypassed wrapper input ownership: subsequent Couch deliveries expired with “waiting for previous automatic input to clear”, and relaunch refused busy despite an empty composer. Receipt78b7fb14-68ed-4bef-a090-f3b403c89721 verified.

**Delta:** All --keys effects remain on the wrapper’s existing single input writer; never substitute direct Zellij key injection. Add an integration test through that writer which completes a key drive, observes its final empty composer, then completes a normal Couch paste/submit/confirmation. This defends the shared automatic-input lease handoff and ordinary delivery liveness (ARCH-DRY, ARCH-ORDER). Preserve the hold before live Claude acceptance after #528; no staging, restart or close is authorized by this revision.

### 2026-10-14 — failed live smoke: submit boundary and recoverable composer

**Reason:** Verified TL receipt6d4558bb-25e2-449b-ac22-63fd1e9b8d2d reports unconfirmed operation12f7dea1 on relaunched Codex ariadne:5 at the staged dbdde238 build: /model remained with a newline, failure left text behind, and --keys clear-line refused the occupied composer. Claude acceptance is still blocked until #528 fixes connection-before-focus. Stage clearance is requested and queued behind another live SDLC merge lock.

**Delta:** Reproduce submit using the actual production wrapper and terminal protocol; logical submit stays distinct from newline, and picker Enter must reach a positively recognized picker as confirmation. Recorded and live conformance must cover the same receiver/protocol path rather than treating a native-PTY success as proof of every live-pane path.

Recovery-only --keys means a validated nonempty sequence containing only clear-line and esc. It may start on a positively recognized occupied composer when the wrapper proves no active turn, picker, image, buffered input or competing automatic transaction. Model and generic key drives retain the empty-composer precondition. The same fresh exact-wrapper, sequence, owner and caller checks apply at admission and execution; do not trust an unrecognized composer or a wire-supplied ownership claim. Reuse Request.RecoveryOnly, peerComposerText and a typed Recoverable observation across the pure reducer, wrapper and Couch service (ARCH-DRY, ARCH-ORDER).

Reserve min(3seconds, one tenth of the budget remaining at execution start) INSIDE the existing absolute deadline for model failure recovery. Before that deadline, cleanup may act only on this operation’s acknowledged /model composer text or a recognized owned picker, never on human cancellation, a turn, changed unrelated text, possibly-applied input, unknown write progress or an unknown view. Supply the current coherent screen at recovery ticks, proof-pin cleanup keys, cap picker backout at three separate ESCs, and verify a fresh empty prompt before reporting recovery. If the available proof or budget cannot establish recovery, retain unconfirmed and name inspection/manual recovery; never claim the model change failed or cleanup succeeded without evidence (ARCH-CONSTRAINTS, ARCH-FUNERAL). Capture real multiline-command clearing behavior before choosing cleanup bytes.

Add recorded RED/GREEN tests for submit-vs-newline under the real protocol, recoverable-command cleanup within the same budget, no cleanup over changed/human input, recovery-only admission on occupied composers, continued generic/model refusal, and key recovery followed by normal Couch delivery. Re-run isolated live conformance using the production input writer and matching terminal protocol; request another operator smoke only after the fixes and evidence are committed. No peer slot input or restart is authorized for this implementation work.


### 2026-10-15 — integrate confirmed recovery before the combined smoke

**Reason:** Verified TL receipt473f6e83-ec19-4a8d-ba41-4bebe74fabde authorizes staging after #556 landed, for one operator restart covering both items. Main also brings #572's trimpath build/test contract and #577's environment-probe correction.

**Delta:** Preserve #556's RecoveryConfirmation token, confirmation offers, receipt identity and prepared-recovery routing alongside #544's typed Drive intent, digest and bounded result. Keep both operation declarations and source classifications. Documentation retains current confirmed-recovery behavior and the drive instructions. Native wrapper admission remains on main's exact-grant seam; rerun actual Run conformance with that seam present (ARCH-DRY, ARCH-ORDER). Record a new suite using main's declared make PAIR_GO_TRIMPATH=-trimpath test command. Stage a clean committed merged HEAD and verify both binaries against it. The operator owns the Couch restart; live drive authorization remains enforced by the owning-hub broker guard.


### 2026-10-15 — final restart batch excludes Qoder repair

**Reason:** Verified TL receipt670622a8-d2a5-48e2-892d-9f0a7bca8494 directs immediate merge/restage after #587; do not wait for #569, which rides a later restart.

**Delta:** Integrate origin/main2156c014cee0af8fa1662064ebfe32d746e6568a, carrying #573/#579/#584/#585/#594/#587. Preserve both resource and drive dispatch, source inventories and skill costs at three additive conflict sites. Main's legacy-message serialization, broker-release distinction and capture buffering remain unchanged by reconciliation. Targeted drive/recovery/protocol/resource/native ownership tests plus inventory and actual native Run conformance verify the integration before staging. The operator still owns the Couch restart and the hub runs live model/key acceptance (ARCH-DRY, ARCH-ORDER). No Qoder repair or new live input is part of this restage.


### 2026-10-15 — adapt capture tests to the owned terminal transport

**Reason:** Full-suite/focused RED exposes three #579 pump fixtures missing #544's transport initialization after the merge.

**Delta:** Reuse prepareMasterPumpTransport before launching each new capture pump fixture, matching production descriptor ownership and joined teardown (ARCH-DRY, ARCH-FUNERAL). Preserve capture assertions and all production code. The test-only correction does not require restaging the operator's production binary; record a fresh passing suite before close/landing.
