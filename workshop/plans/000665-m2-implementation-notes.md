# #665 M2 implementation notes — proposal

Prepared read-only while M1 awaits its recorded suite. No M2 code or native/browser runs have started. The approved duration policy is R6; new observation seams and unresolved native/browser boundaries below require hub review before implementation.


# Pair #665 M2 priority implementation plan — read-only proposal

Scope:23 exact debt entries in couchcmd4/couchcore4/couchtty13/wrapcmd2.
Broadcast excluded. No source edits or tests/native runs performed for this plan.
M1 must land first. Proposed disposition:17 event/poll/lifetime repairs,
4 exact duration-subject entries retained with reasons,2 external-agent readiness
entries explicitly unresolved pending hub decision. No new clock seam.

## Two bounded event witnesses (nil in production)

A. messageService processed Registry event observer.
- Add one private atomic callback slot on messageService, installed by serviceRig
  after constructor returns and before any wrapper starts. Atomic installation
  avoids a race with already-running service loop; default nil changes no policy.
- At the end of loop step, AFTER effects, snapshots and recursive follow-ups, call
  it with copied scalar evidence: event kind/token/observation sequence and
  registry.Phase(token) result. Do not expose mutable Registry or call service
  reentrantly. Callback only updates a mutex-protected test probe and nonblocking
  wake channel; no new goroutine and no blocking production-loop callback.
- Probe records exact event keys and current phase. Tests arm before sending.
  All waits use testwait hang guards. service.Close joins loop/session workers
  before probe teardown; no channel is closed while producer might still send.

B. Console processed decoded input event observer.
- Add one private callback field set ONCE before Run. newFixtureBeforeRun already
  supplies that installation point; its common fixture installs a test-only probe
  before existing beforeRun customization. Other production consoles leave nil.
- In Run's processEvents loop, call observer only AFTER routeInputEvent returns
  for that exact fully decoded event. Match the complete SGR release Raw bytes,
  with event sequence newer than a probe snapshot armed BEFORE the stdin write.
  Do not match arbitrary read chunks: parser may split a sequence across reads.
- This retains real pipe/pump/decoder/mouse routing and works under private panel
  input capture. No independent terminalCommands noop is treated as stdin fence.
- Probe stores scalar sequence/last-match state under mutex, wakes nonblocking;
  existing fixture Stop/pipe-close/Run join owns teardown. No new worker/channel
  close race. Callback has no t.Fatal (Run executes on worker goroutine).
- Negative state assertions follow routing acknowledgement. For screen/clipboard
  assertions also Presenter.Flush after ack; it completes already requested
  publication, not a new selected-frame or synchronized-output policy.

These are test observation seams only; no clock, protocol ack, transport retry,
request nonce, public API or user-visible behavior change. Their need comes from
actual asynchronous boundaries, not the lint rule itself.

## Couchcmd concrete changes

| Test / exact debt | Implementation / signal / teardown | First RED or mutation control |
|---|---|---|
| TestLifecycleDelayedFramesFromOldIncarnationIgnored —200ms | Capture actual old Session token from s.liveness before replacement. Await new admitted token (different even for exec/identical binding). Send old Update99; await A's processed SessionActivity(old,99). ONLY THEN Submit100; await processed SessionSubmit(old,100). Snapshot/assert original stale-marker, next-row and branch conditions. This prevents Update+Submit coalescing from silently testing only Submit. Real sockets and SessionClient remain; r.wrapper cancellation and s.Close keep ownership. | Controlled old-token attribution mutation must corrupt replacement in exec case and fail AFTER both precise event acknowledgements. Probe unit controls prove unrelated token/sequence cannot release a waiter; no sleep-based substitute. |
| TestLifecycleStartDetachReattachReplaceRestart —100ms | Capture old actual token; start exec wrapper and await new admitted token before stop(). stop joins client; then await A's SessionClosed(old) with Phase found=false, and assert replacement still connected/send accepted. Keep all detach/reattach/restart phases. | Mutation that lets old-token closure disconnect new byte-identical binding must fail send/connected assertion deterministically after Closed acknowledgement. |
| TestMessageAdmissionRejectsMismatchesThenGoesDormant —100ms | Each retry callback fires only when queued. After last retry, await A's final AdmissionDone for target token with phase Dormant, then assert exact retry count, queue empty and disconnected. Do not use a send as the fence: it would wake Dormant and change the no-extra-attempt assertion. | Force final rejection to queue another retry / stay Rejected; phase/count control fails. Keep mismatch table and reason-specific fake world state. |
| TestMessageIdleMultiSlotRunsNoProbes —2500ms | KEEP EXACT sleep and counters/membership assertions. | Duration subject. Reason per JSON entry: "Bounded idle observation spans removed 1s heartbeat/reconcile periods; counter stability over time is the subject, and no existing scheduler seam covers every hypothetical idle timer." Not proof of forever; no new clock merely to remove sleep. |

## Couchcore concrete changes

| Test / exact debt | Implementation / signal / teardown | First RED or mutation control |
|---|---|---|
| TestRunnerConformance_SignalIgnored —100ms | KEEP EXACT sleep after existing ready file proves trap '' INT installed; retain real/fake alive comparison and real Kill/Wait cleanup. | Duration subject. Reason: "Real SIG_IGN survival is sampled after signal delivery; ignored signals provide no handler acknowledgement, and replacing trap-ignore with an acknowledging handler changes conformance subject." Do not delete wait or silently change ignored to caught. |
| TestContinuationZellijAgentStandIn —20s | Before publishing ReadyRecord, arm signal.NotifyContext(t.Context(), SIGTERM, os.Interrupt); defer stop. After readiness, wait for that owned termination context instead of guessed20s lifetime. Parent already explicitly Quiesces session and asserts stale receipt no longer authorizes readiness. Existing ControlledZellij.Close owns real process/server teardown. | Owned helper subprocess control: parent observes ready marker, then sends TERM and requires ordinary joined exit; no natural timeout expiry used as success. Failure-path control ensures termination armed before readiness. Private Zellij lane needed for unchanged parent continuation conformance. |
| TestRecoveryLiveAgentStandIn —40s | Same owned termination-before-ready ordering; retain seed, nonce, identity and launch-ordinal assertions. Parent recovery fixture owns created Zellij session and verified teardown. Reuse only tiny test lifetime helper if helpful; no new transport. | Same helper control, plus existing real recovery fresh/warm variants after native-lane permission. No production query/birth bounds changed. |
| TestQuiesceLeavesTheBindingAndEndsTheSessionLive —100ms polling | Replace hand-built10s/100ms loop with existing testwait.Wait/Until on exact PairSession predicate (same binding, Present=false), keeping last observation/error in failure. Final DetachedSessions empty assertion remains. | Adapter/stateful fake negative control: Present/unknown must not satisfy predicate. Real Zellij qualification needs lane; unknown/malformed replies are refusal/inconclusive, never absence. This is existing positive condition polling, no new retry policy. |

## Couchtty concrete changes

| Test / exact debt | Implementation / signal / teardown | First RED or mutation control |
|---|---|---|
| TestBroadcastClickPastTheCellDoesNotStop —100ms | B acknowledges exact click-release, then phase must remain Live. Existing fixture teardown joins Run. | Misroute outside-span click to broadcast toggle; phase immediately Stopping/Off after acknowledged route, test fails. |
| TestBroadcastOffHasNoClickTarget —100ms | Arm/ack each exact release (or final release after all writes) using B; then existing Off/nil session/empty ColumnSpan checks. | Expose click target while Off; original assertions fail after routing fence. |
| TestWebCastCodesOnTheChromeCell —100ms | Arm B for Alt+click release, issue original mouse bytes, await route then Presenter.Flush; retain no-menu/unchanged phase and later real menu-content assertions. OSC52 copy count can remain extra positive witness, not sole fence (wrong no-op implementation must not cause a hang instead of testing no-menu). | Route Alt+click on capability to Codes menu; no-menu assertion must fail after decoded release. |
| TestLiveConsoleNvimPreservesContentAndChrome —500ms startup | Supply private VimEnter ready-file autocmd via --cmd, installed before startup; await file plus existing first-screen observation. No arbitrary settling interval. File path under fixture root, parent retains real nvim process via existing Console teardown. | Hold test-only VimEnter completion gate / suppress startup witness; no early command phase is admitted. Native nvim/PTY qualification required, not Zellij. |
| Same —250ms per-command | Keep PTY stimuli set number / insert hello / set nonumber. Append command-specific Vimscript writefile acknowledgement after command processing (and redraw!); after each, await corresponding actual frame predicate: number cell visible, hello visible, then number absent with hello retained. File acknowledgement alone is insufficient: renderer ingress can lag it. Final native-vt hello/brain oracle retained. | Hold one redraw/delivery phase until explicit release; next command must not advance early. Wrong number toggle or missing insertion mutation fails actual frame predicate. Do not replace PTY input with RPC-executed commands. |
| TestNotesMenuTakesNoClickUnderAPanel —100ms | B observes hidden-menu release AFTER switcher/menu-cleared positive states. Flush, then require no clipboard copy/no copied notice. B works when panel intercepts printable sentinels. Existing note copyLink/setNotice are synchronous during routing. | Remove hidden-panel menu refusal; exact copy/notice checks fail after routed release. |
| TestPointerDroppedWhileSwitcherOpen —100ms | Assert HTTP status, then inspect marks directly. Existing exact chain Server.point -> Session.acceptPoint -> Console.onPoints -> runTerminalCommand -> applyPoints completes before204. Keep prior actual positive mark/fade and panel-open evidence. | Remove applyPoints panel guard; mark Live assertion fails immediately after HTTP completion. No new observer needed. |
| TestPointerMarksFade —100ms polling | Existing waitFor on screenBg!=nil until absent, bounded by testwait. Keep real product MarkLife and fade tints; no fake clock replacement, duration shortening or new clock seam. | Suppress scheduled fade refresh; actual background never clears and named positive wait fails under short owned test timeout. |
| TestPointerOnTabBar —100ms | Keep positive tab mark. After guarded-LIVE HTTP response, call Presenter.Flush, then assert guarded cell unmarked. | Remove protected LIVE draw exclusion; actual frame assertion fails after flush, not before repaint. |
| TestPointerStressNoDeadlock —1s | KEEP EXACT one-second concurrent workload window with reason. Start all3 producers from one ready/start barrier; retain6 toggles and all joins. Replace unlisted80ms toggle-readiness guesses with actual phase-flip acknowledgements/B as needed, without serializing producers. | Reason: "Concurrent pointer/output/toggle stress is intentionally observed for one second; the duration defines workload exposure, not readiness." Controlled stalled producer must trigger joined-ownership hang guard; no t.Fatal from producer goroutines. |
| TestWebcastClicks —100ms | B acknowledges off-cell release before FakeBrowser.State Live assertion. Existing ToggleLive runs synchronously in routed click, so acknowledgement precedes its state effect. | Misroute off-cell to toggle; PhaseLive assertion fails after route. |
| TestNotificationPTYHelper —1s | Extend existing private PAIR_NOTIFICATION_GATE content with done phase. Helper sends both original broker/OSC notifications, then waits for done marker using testwait. Parent writes done only after focused/inactive suppression assertions and joins child.Done. Cleanup always writes done/release even if assertion fails before focus phase, then Console joins/kills remaining child normally; no helper writer outlives TempDir. | Gate lifecycle control: done not released before second parent assertion; missing notification/copy-leak controls remain. Force delayed parent assertions beyond old1s in owned subprocess control; old lifetime would exit before parent release, new remains live. No Zellij. |
| runPressureTrial —250ms | KEEP EXACT250ms offset in existing2s pressure experiment. Do not change first-input probe time,1500ms censor/menu boundary, offered-load cadence or measured-window interpretation. | Reason: "The pane-input latency probe is deliberately scheduled250ms into the fixed2s pressure trial; changing it changes the measurement experiment." Existing readiness guard stays separate; env-gated fake/PTY experiment qualification only if its semantics otherwise change. |

## Wrapcmd:2 unresolved readiness debts, not duration exceptions

- TestAgentDriveLiveCapture: exact1500ms before pending model/picker input.
  Existing Classify already requires PeerComposerEmpty, exact model/effort menu
  labels, styled selected row and phase; those prove draw-time appearance, not
  native post-render handler mount completion. Input callback's sleep is the only
  current settlement mechanism. Do not change it into a nominal duration subject.
- TestPeerLiveConformance: exact1s before trustInput. Existing observer sees
  Accessing workspace:<private cwd>, selected No/Yes rows and trustStep; comment
  explicitly records subsequent mount effects resetting an immediate selection.
  Paint/selection recognition is insufficient as a handler-ready generation.

Decision needed from hub: (a) authorize a bounded private native investigation
against installed agents to find/qualify an event witness (native lifecycle log or
explicit input-accepted/handler generation; no guesses from unchanged frames), or
(b) explicitly defer these2 exact debt entries to a followup, recording readiness
cause and blocker individually while landing remaining M2. No native run now,
no extra navigation/consent retries, and no new fixed quiet/stable-frame interval.
A followup must be called unresolved, not claimed fixed by this milestone.

## Implementation order / reviewable first slice

After M1 lands and hub accepts this revision: implement2 observer probes and pure
probe controls; repair3 Couchcmd lifecycle sleeps and5 Console stdin negatives;
then existing synchronous HTTP/flush and real fade polling; helper lifetimes and
native nvim require their dedicated qualification. Keep4 exact duration entries
with reason fields and preserve2 external readiness debts only by explicit hub
choice. Source-level temporary mutations run sequentially and restore exact bytes;
ordinary/race targeted checks precede any native lane request or final suite.


# #665 M2 utility sleep-debt implementation plan (read-only preparation)

Authority: current internal/repolint/sleep-debt.json. Scope excludes couchcmd, couchcore, couchtty, wrapcmd, webcast and layout. There are **17 entries**, not16: broadcast6; couchmessage4 (two entries in one background-client test); hostty1; launcher2; procutil2; terminal1; ttyio1.

No repository edits, tests or native runs were performed for this preparation. Implementation waits for M1 landing. No new clock seam is proposed. Existing caller deadlines and product timing remain unchanged.

## Deterministic readiness/lifetime replacements (13 entries)

### 1. broadcast/cloudflared_test.go — TestReapOrphans (100ms ordinals1–4)

Cases in source order: live owner; recycled PID/identity mismatch; wrong command; identity-less dead-owner record. Preserve every existing survival, record and private-directory assertion, plus the positive dead-owner kill case.

ReapOrphans -> runRecords.reapLocked -> killIfOurs completes all signal decisions synchronously; there is no asynchronous production reap worker to join. A bare immediate procutil.Alive check is insufficient because a wrongly issued SIGKILL may not yet have been reaped. Give each survivor fixture a parent-to-child request pipe and child-to-parent reply pipe. After ReapOrphans returns, send a fresh nonce and require its echoed acknowledgement from that same PID before retaining the liveness/identity assertions. A killed process cannot acknowledge the post-decision nonce.

For the cloudflared cases, adjust startOrphan's named shell fixture to retain its cloudflared command identity while reading/echoing requests. Keep its separate process group, initial settled-command witness and positive kill control. For wrong-command case use an equivalently responsive child whose argv still does not match cloudflared. Do not change the wrong-command test into a cloudflared fixture.

Teardown: own cmd.Wait result channel rather than unjoined go cmd.Wait; on failure close pipes, signal only the owned group, then join Wait with testwait.After guard; on normal survivor completion release via pipe EOF and join. Positive kill case awaits joined exit as its event. Register cleanup immediately after Start and before any assertion.

First RED/mutation: force killIfOurs to signal a protected survivor despite its guard (one of the four cases). The post-reap acknowledgement must fail by EOF/exit, not pass an immediate Alive check. Restore production source before other work/tests.

### 2. broadcast/host_test.go — *failingTunnel.Open (100ms ordinal1)

Test consumer: TestHostFailedOpenReachesEveryWaiter. Add entered/release channels to this test-only tunnel. First Open signals entered and waits for release or ctx cancellation, then returns errOpen; subsequent Open still delegates to LocalOnly.

Start first mount and await entered; start second mount. Existing Host exposes h.waiting and h.attempt under h.mu: use existing bounded waitFor to observe exactly two counted waiters joined to the same opening attempt before releasing the failure. Require both returned errors to match errOpen; retain subsequent successful third mount, response body and unmount.

Teardown: release channel via sync.Once on all exits, join both result goroutines, unmount successful mount. A test failure must not leave Open held.

First RED/mutation: suppress attempt.err propagation to later waiters, or allow a second Open while the first attempt is held. Both-error assertion / one-opening-attempt witness must fail deterministically.

### 3. couchmessage/background_protocol_test.go — TestBackgroundClientCopiesCoalescesAndClearsEvidence (120ms ordinals1–2)

These setters are synchronous state transitions, and both calls intentionally leave the observed evidence unchanged. After each existing awaitBackgroundFrames barrier, acquire c.mu and snapshot the setter-owned state (backgroundWork value/presence, dirty, have and submit). Invoke unchanged BackgroundWork(true), then unchanged BackgroundWork(nil) at their respective original positions, and immediately compare the protected state: no dirty transition and no evidence mutation. Preserve existing wire-frame counts and content assertions at the next real publication barriers, copied-bool check, false-vs-nil distinction, burst outcome and final Submit frame. Do not replace sleeps with an immediate frame-count-only assertion; it would merely avoid observing an incorrect scheduled frame.

For unchanged-setter wake behavior, a separate unstarted SessionClient can have its wake channel drained before the call and be inspected synchronously afterward; this needs no production sender-cycle hook. The running client test keeps real socket/wire coverage and the separate idle-duration test below retains the unsolicited-frame observation window. Avoid claiming a synchronous setter test proves arbitrary future silence.

Teardown: wrap c.Run with done channel; cleanup cancels context, waits for Run done and waits for sessionLog closed event before server/socket cleanup. Preserve real transport frame handling; no mock sender or clock added.

First RED/mutation: make BackgroundWork mark dirty and/or signal on an unchanged true or nil value. Protected-state and standalone wake assertions fail immediately, independent of OS scheduling. A heartbeat mutation is covered by the retained idle-duration test, not these setter assertions.

### 4. couchmessage/session_transport_test.go — TestPeerPIDIsTheDialingProcess (200ms ordinal1)

Existing private Unix connection is the source of truth. Dial goroutine reports its connection and blocks on release until the parent has accepted and checked PeerPID equals os.Getpid. Parent then releases it and joins the goroutine. Use t.Context/testwait.After for hang bounds, not a guessed lifetime.

Teardown: listener/accepted/client connections all owned; cleanup releases once and closes listener on every path, then joins the dial goroutine. Register cleanup before Accept. Preserve socket peer-credential assertion, do not substitute a fabricated PID.

First RED/mutation: return a foreign PID from the PeerPID seam; existing equality assertion must fail while the client is verifiably held alive. Add a fixture teardown control covering parent assertion failure/release before accept if helper factoring warrants it.

### 5. hostty/hostty_test.go — TestHostsAgreeAfterClose (100ms ordinal1)

OSHost.Close already stops signal delivery and joins watchDone; FakeHost.Close closes resized/terminated synchronously. After the existing close and twenty post-close stimuli, directly assert both Resized channels are closed (closed channel, not merely no event), then retain Write -> os.ErrClosed and idempotent Close assertions. No new signal-drain wait is needed after the joined lifecycle boundary.

First RED/mutation: fake post-close SetSize sends to its closed channel / does not check closed (historical BR-18), or Close fails to close its notification channel; test fails deterministically without waiting100ms. Existing real OSHost sends SIGWINCH only to its own test process.

### 6. launcher/osruntime_test.go — TestSidecarProcessGroupProbe (30s ordinal1)

Consumer: TestSpawnDetachedKeepsCouchSidecarInOwnedProcessGroup. spawnDetached deliberately binds all stdio to /dev/null, so replacing Sleep with stdin.Read would immediately see EOF and break the witness.

Parent owns a compact private Unix listener and passes its pathname in extraEnv. Child helper connects, sends PID and PGID, then blocks reading a release byte/EOF. Parent Accept/read is readiness, compares received PGID with syscall.Getpgrp exactly as today, then releases child. Keep real spawnDetached and childEnviron behavior, no alternate launch adapter.

Teardown: close listener and accepted connection on every exit; child treats EOF as release. Parent waits for owned PID disappearance after release (spawnDetached already launches its own cmd.Wait goroutine). On failure kill only the witnessed child PID with its recorded identity, then await disappearance; never kill the inherited actor group. Ensure helper child endpoint is absent when its special environment is unset. The existing marker path can be retired in this test-specific protocol, not production.

First RED/mutation: force Setsid=true for Couch scope/tag. PGID comparison fails deterministically while child is held. Teardown regression: release-before-read / parent closing connection must let child exit without its former30s lifetime.

### 7. launcher/session_quiescence_live_test.go — TestSessionDetachLive (100ms ordinal1)

The existing loop already asks the correct state predicate and stops if the positively listed session disappears. Replace its inline poll/sleep with existing testwait.Until (or existing fixture event wait if it has an equivalent state callback) for SessionDetached, retaining every attached/detached narrowedFormsAgree assertion and the fatal established-absence path. No new retry policy, transport workaround or longer product deadline. Existing ten-second fixture bound must not be silently lost; if helper cannot honor it, retain an explicit existing-bound context around the predicate.

Unknown/malformed replies stay unknown under #647; never convert unknown to absence or a success. A native run that encounters unknown is inconclusive/failure under the existing oracle, not permission to retry.

First RED/mutation: state remains attached, or established absence is incorrectly accepted; bounded state wait / disappearance assertion fails. Real execution requires explicit hub native-lane grant. No native execution during prep.

### 8. procutil/resource_darwin_test.go — TestResourceEnvironmentOfAChild and TestResourceCwdOfAChild (10s each)

Child roles emit one ready byte on an owned stdout pipe after their role check, then block reading parent-owned stdin. Parent starts child with explicit stdin/stdout pipes, waits for ready, then performs the existing real NativeResourceReader.Table and Environment/Cwd checks against that exact PID/generation. The child is alive in its final inherited environment/current directory for the whole observation. Preserve environment requested-key filtering and exact values; preserve filepath.EvalSymlinks canonical cwd and separate gone-child refusal test.

Prefer one post-readiness table query rather than startup polling: child readiness is emitted after Go startup and the configured child cwd/environment are already established by exec. If a platform can still return unknown, diagnose it rather than reintroducing startup delays.

Teardown: close parent stdin to release child, join cmd.Wait and stdout reader; failure cleanup may cancel the existing command context/kill only owned child, then join. Register cleanup immediately after Start, including readiness-failure paths.

First RED/mutation: helper exits immediately after readiness / sends readiness before entering intended helper role; actual process identity/environment/cwd observations fail. Existing expected-value mutation controls (wrong env key or wrong cwd) must still fail. These are normal disposable OS tests, not Zellij-lane tests.

## Genuine duration subjects retained with exact debt (4 entries)

1. broadcast/manual_test.go / TestManualViewerServer / time.Sleep(time.Second) ordinal1: gated manual viewer intentionally publishes changing frames once per second for configured BROADCAST_MANUAL_SECONDS. The cadence is the feature under observation, not an inferred readiness boundary. Retain this exact site with that reason; do not disguise it as an otherwise equivalent ticker solely to shrink the debt count.

2. couchmessage/session_transport_test.go / TestSessionCoalescesActivityAndSendsSubmitAtOnce / time.Sleep(700*time.Millisecond) ordinal1: this exact window verifies an already-ready, already-published, otherwise idle real socket sends no unsolicited frames for more than two configured300ms coalescing intervals. It is an absence-over-duration subject, not startup synchronization. No controlled clock/processed-cycle seam exists; do not add one solely for debt removal, or replace the window with an immediate count. Retain the exact entry with this reason, strengthening owned Run/server teardown. Its classification should be surfaced explicitly to the parent/hub because the dispatch estimated only three couchmessage sites; the authority has four. Existing burst/submit wire assertions remain. Mutation: inject periodic idle activity; retained interval detects frames. Do not claim universal future silence from700ms.

3. terminal/presenter_stall_test.go / TestPresenterSurvivesTransientParentStallAndRestoresModes / time.Sleep(2500*time.Millisecond) ordinal1: real PTY master deliberately stops draining beyond historical two-second cutoff yet resumes inside WriteTimeout. Keep actual elapsed lower bound, successful paint and exactly-one trailing parent mode reset. Replacing Sleep with timer-triggered channel would not remove the duration requirement. Mutation: restore a two-second inner write cutoff; test fails.

4. ttyio/writer_test.go / TestFileWriteSurvivesTransientHostStallWithinCallerDeadline / time.Sleep(2500*time.Millisecond) ordinal1: real PTY must accept one exact patterned payload after a host stalls beyond the historical two-second transport limit, while caller sets no deadline. Keep accepted-prefix/full-length, elapsed lower bound, byte-exact once-only payload checks. PollWaits can witness backpressure, but cannot advance real kernel time; no new poll/clock seam solely to remove debt. Mutation: reintroduce a two-second transport-owned timeout; test fails.

## Qualification after authorization

Targeted normal/race tests for changed utility packages; actual native Zellij test only after hub grant. No whole-suite run in this delegated scope. Guard every new block/join by existing testwait bounds and preserve all old assertions/domain; do not add fixed waits or extend deadlines. Add per-entry durable reasons to the retained debt through the parent-owned schema/test mechanism rather than inline suppressions. Cross-file shared sleep allowlist is parent-owned; return exact removed/retained identities for its single update.


# #665 M2 — webcast sleep implementation plan

Read-only source audit, 2026-10-18. Scope: the current 57 entries under cmd/internal/webcast/ in internal/repolint/sleep-debt.json. No implementation, browser, native command or tests run for this audit. Hub bf84e7de governs: readiness becomes observations/completions; real duration subjects stay with entry-specific reasons; no clock added merely to erase sleeps. Keep any existing real-time conformance when using an existing fake clock.

## Concepts and boundaries

| Entity / integration | Location | Proposed status / precise responsibility |
| --- | --- | --- |
| Test readiness predicates | live_test.go, live_record_test.go, viewer_live_test.go | test-only: reuse testwait.Wait/Until, actual expected URL, document selector, recorded segment and renderer state. Last observed state belongs in failures. |
| Recorder queue acknowledgement | Chrome.WithTab, Recorder.Stats, CDP.Conn.SyncEvents | reuse; distinguish their scopes below. No global Flush API. |
| Driver browser-policy completion | chrome_drive.go:setDriving | proposed narrow completion seam: per transition/session, completion after download-behavior and file-chooser command replies; report error/cancellation/ended. Never wait on Chrome.work.Wait while its loop can add work. Requires scoped hub approval if it changes production contract. |
| Viewer input completion | web/drive.js:mountDrive return value | expose existing batcher.pending via diagnostic accessor, without bypassing pace/flush timers or issuing an extra input. Pair queue-empty with observed input dispatch and source effect/response. Return unavailable distinctly when batch was stopped. |
| Annotation post completion | web/annotate.js:flush/endStroke | narrowly scoped idle/pending observation of existing stroke POST promises; do not force an extra flush or drop outstanding posts. Needed for full-stroke/no-duplicate oracle. |
| Draft status | notedraft.Service.Status, Server.noteOwnerScope | reuse with a test-only captured Owner from the actual authenticated request; no new clock, service wait API or HTTP busy loop. Keep final HTTP status/auth assertions. |
| Browser action terminal evidence | chrome_drive.go, live_drive_test.go | unresolved for denied download and native chooser. Qualify actual Chrome event semantics in a later disposable headless lane after M2 admission before choosing a seam; visible native operation needs explicit operator permission. |

Existing facts: SetLive acknowledges the Chrome loop phase, not recorder opening/document capture. Opened means first output opened, not current target/doc readiness. WithTab acknowledges its queued callback and exposes the actual recorded page. Stats acknowledges its query on that queue; it does not pull future browser events or wait every asset worker. SyncEvents fences only events already read and forwarded, including the recorder barrier. A CDP reply does not in general flush future async browser effects. couchViewer.ready awaits renderer resources; also require the intended URL/segment/canonical state. SetDriving currently launches policy commands through goCall and returns before their replies. MemController's clock func timestamps records; its grace timer is real time.AfterFunc. Board route admission uses time.Now, with no injectable clock.

## Implementation order and first RED controls

1. Add test-only readiness helpers around existing seams. First RED: hold the desired document/recorded segment back while the previous page exists; ensure the helper stays blocked, then publish the expected state and require completion. Cancellation/close must return failure, never count as ready. Use original time budgets or testwait's test-deadline guard; no deadline inflation. Preserve last URL/segment/selector/canonical state in diagnostics.
2. Replace positive readiness waits and existing fixed polling intervals with these predicates. For recorded state require WithTab callback state or a test-owned tee sink notification; a subscriber's initial catch-up is not evidence for a newly requested state. Preserve real browser input and disk readback.
3. Add narrowly scoped browser-policy and viewer/annotation completion observations only after reviewing their exact contracts. First RED: hold the actual CDP policy reply / HTTP POST promise; the completion must remain pending. Resolve/refuse/cancel it and assert correct completion/error. Old-generation/stopped queue must never report successful completion. Unit tests use existing JS deferred-promise and fake Chrome protocol infrastructure, then existing headless conformance covers the real adapters.
4. Convert five negative-absence sites only with the evidence below. Mutation controls must delay the forbidden effect past an ordinary CDP reply and ordinary SyncEvents fence, then release it: the repaired oracle must still fail. Never replace a delayed absence assertion with an immediate zero snapshot.
5. Draft waits: capture Identity/Scope/Generation at the real server boundary for the accepted prepare request (scope via existing noteOwnerScope, no copied hashing/auth model). Poll Service.Status on that Owner/request, then execute the original HTTP draft-status request and assertions once settled. Do not poll HTTP at testwait's 2ms cadence: all draft routes share the note limiter (2/s, burst5), so that creates 429 failures. First RED: block CaptureSource, verify status remains pending; release and wait for persisted ready status. Capture returning is not persistence completion. Save uses the same status evidence, and still checks saved note/submission on disk. Retain foreign/expired/unauthenticated requests and HTTP refusal assertions.
6. Retain the four genuine duration entries listed below, with exact identity/reason in the issue Log (not inline lint suppressions, which M1 forbids). Shrink debt only for proven removals. Test inventory, assertions, opt-in dispositions and fixture domains stay intact. Do not edit other fixed/dynamic/sub-threshold sleeps as a side quest.

## Five negative-absence oracles

- TestLiveDrive popup: install existing loopState.tap before input, observe Target.targetCreated for this opener, then the matching Target.targetDestroyed and recorder/loop acknowledgement; only then assert getTargets lacks that target and retains the cast. Capture the event synchronously into a nonblocking bounded test channel; no t.Fatal on the Chrome loop. Mutation: omit/delay closeTarget and ensure it cannot pass because the initial snapshot contained no popup.
- TestLiveDrive driver download: FIRST qualify a real terminal Browser.downloadProgress canceled outcome for the exact URL/GUID when behavior=deny. Current command omits eventsEnabled; propose diagnostic event enablement alongside the actual policy command, not a second test command that re-applies deny and hides a broken production transition. Preserve positive operator download first. Wait terminal outcome plus filesystem observation, then retain unchanged no-new-file assertion. Mutation: allow the driver's download and delay its write; must fail. If deny emits no terminal event, this needs a different browser-boundary oracle/explicit re-plan; a Network.loadingFailed/ERR_ABORTED alone also occurs when downloads are handed off and is not proof.
- TestLiveDrive restored operator chooser: wait actual interception=false command acknowledgement, then actual click completion. There is currently no proven terminal native-chooser signal when interception is off. Runtime.evaluate and SyncEvents are NOT sufficient. Plan a focused protocol fake test proving the exact disable command/session/order, plus separately qualify whether the real Input dispatch response guarantees chooser decision completion. If unavailable, retain the existing test pending an authorized native oracle or explicit scope revision; do not silently remove/soften the no-new-fileChooserOpened assertion.
- TestDriveRealCast right-button refusal: wait viewer event handlers, then batch settlement; include a known allowed gesture marker after the right click, require its injector acknowledgement, and fence source events already emitted before the marker. Also observe viewer queue epoch/generation and zero pending; do not call control/revoke to clear queued work. Mutation: permit right click and delay it in the viewer queue; ctx must become1 and fail the unchanged ctx==0 assertion. Need the batcher completion seam because mountDrive currently hides pending().
- TestAnnotateLiveTwoViewers full stroke/no duplicate: after release await all stroke POST responses through annotation completion observation, then ordered board writer/mem records. Require endpoint coverage and exactly one stable stroke ID with original points assertion. First rendered pixel on B is not final chunk completion. Mutation: first echo ends local stroke / split ID on final chunk; must fail after all posts settle. Browser event and POST completion must be causal, not an arbitrary microtask drain.

Paused recording changes also need acknowledged paused state and resume document, although the existing assertion is segment/order rather than a separate no-event count. Do not claim SetLive alone closes the recording boundary.

## Exact debt disposition ledger

IDs follow current JSON order within webcast. File paths below are relative to cmd/internal/webcast/. `#n` is the ordinal for that exact canonical call in that function. REMOVE means proposed replacement after its RED and regression evidence; HOLD means implementation is gated on the unresolved completion proof. RETAIN means actual duration subject, kept as exact tracked debt with reason.

| ID | Exact debt entry | Disposition / preserved evidence |
| --- | --- | --- |
| 01 | `live_drive_nav_test.go :: TestLiveDriveNavigation :: time.Sleep(1500 * time.Millisecond) #1` | REMOVE — wait recorded navigate segment/new URL and attributed click before stopping; read disk after Close/Done. |
| 02 | `live_drive_nav_test.go :: TestLiveDriveNavigation :: time.Sleep(300 * time.Millisecond) #1` | REMOVE — Grant is acknowledged through WriteControl; assert granted control/page via WithTab rather than sleeping. |
| 03 | `live_drive_nav_test.go :: TestLiveDriveNavigation :: time.Sleep(500 * time.Millisecond) #1` | REMOVE — WithTab must show intended live document and #next before coordinates. |
| 04 | `live_drive_nav_test.go :: TestLiveDriveNavigation :: time.Sleep(time.Second) #1` | REMOVE — source readyState complete, expected URL and #next before go-live. |
| 05 | `live_drive_real_test.go :: TestDriveRealCast :: time.Sleep(100 * time.Millisecond) #1` | REMOVE — navigation predicate via testwait.Wait; retain injected-batch/refusal diagnostics. |
| 06 | `live_drive_real_test.go :: TestDriveRealCast :: time.Sleep(100 * time.Millisecond) #2` | REMOVE — castSays predicate via testwait.Wait; preserve each source-page value assertion. |
| 07 | `live_drive_real_test.go :: TestDriveRealCast :: time.Sleep(150 * time.Millisecond) #1` | REMOVE — hover moves wait their queue/source effect; no asserted 150ms cadence contract. |
| 08 | `live_drive_real_test.go :: TestDriveRealCast :: time.Sleep(500 * time.Millisecond) #1` | REMOVE — viewer model scroll/canonical geometry must match first source scroll; ready plus animation frame. |
| 09 | `live_drive_real_test.go :: TestDriveRealCast :: time.Sleep(500 * time.Millisecond) #2` | REMOVE — viewer model/geometry must match second source scroll before link hit-test. |
| 10 | `live_drive_real_test.go :: TestDriveRealCast :: time.Sleep(500 * time.Millisecond) #3` | REMOVE — post-page recorded document/selectors and viewer geometry before center(). |
| 11 | `live_drive_real_test.go :: TestDriveRealCast :: time.Sleep(time.Second) #1` | REMOVE — right-click negative oracle; viewer queue epoch/settlement and allowed marker required (see prerequisites). |
| 12 | `live_drive_real_test.go :: TestDriveRealCastFlick :: time.Sleep(2 * time.Second) #1` | REMOVE — await viewer input queue settlement and spy final arrival; retain JS100events/10ms cadence and95% injected-wheel assertion. |
| 13 | `live_drive_real_test.go :: startRealCast :: time.Sleep(1500 * time.Millisecond) #1` | REMOVE — recorder live page URL/root plus viewer readiness, not merely Opened. |
| 14 | `live_drive_real_test.go :: startRealCast :: time.Sleep(time.Second) #1` | REMOVE — source readyState and expected path/selectors before SetLive. |
| 15 | `live_drive_test.go :: TestLiveDrive :: time.Sleep(1500 * time.Millisecond) #1` | REMOVE — matched popup create/destroy completion (negative oracle above). |
| 16 | `live_drive_test.go :: TestLiveDrive :: time.Sleep(2 * time.Second) #1` | HOLD — denied-download terminal browser outcome unqualified; retain original no-file assertion until evidence/re-plan. |
| 17 | `live_drive_test.go :: TestLiveDrive :: time.Sleep(300 * time.Millisecond) #1` | REMOVE — actual driver policy command replies for deny/intercept-on; driver string alone is insufficient. |
| 18 | `live_drive_test.go :: TestLiveDrive :: time.Sleep(300 * time.Millisecond) #2` | REMOVE — actual intercept-off/download-default command replies for revoked generation. |
| 19 | `live_drive_test.go :: TestLiveDrive :: time.Sleep(300 * time.Millisecond) #3` | REMOVE — dialog closed signal plus existing capability outcome; retain uncertain input fencing assertion. |
| 20 | `live_drive_test.go :: TestLiveDrive :: time.Sleep(500 * time.Millisecond) #1` | REMOVE — live recorder expected page ready before Request/Grant. |
| 21 | `live_drive_test.go :: TestLiveDrive :: time.Sleep(500 * time.Millisecond) #2` | REMOVE — explicit recorder result/queue evidence before Close, then closed disk validation; no generic flush. |
| 22 | `live_drive_test.go :: TestLiveDrive :: time.Sleep(time.Second) #1` | REMOVE — expected source document/selectors ready before positive operator download. |
| 23 | `live_drive_test.go :: TestLiveDrive :: time.Sleep(time.Second) #2` | HOLD — restored operator chooser completion unqualified; exact policy ack plus native/dispatch semantic evidence required. |
| 24 | `live_notedraft_test.go :: TestLiveDrawCaptureProtocolsAndStoredSubmission :: time.Sleep(250 * time.Millisecond) #1` | REMOVE — existing Service.Status owner/request readiness then final authenticated HTTP status; avoid overpolling limited route. |
| 25 | `live_notedraft_test.go :: TestLiveDrawCaptureProtocolsAndStoredSubmission :: time.Sleep(500 * time.Millisecond) #1` | REMOVE — existing Service.Status persisted saved outcome then final HTTP status and submission evidence. |
| 26 | `live_record_test.go :: <package> :: time.Sleep(1500 * time.Millisecond) #1` | REMOVE — scenarios[browse] initial expected page/model ready, not a fixed startup pause. |
| 27 | `live_record_test.go :: <package> :: time.Sleep(1500 * time.Millisecond) #2` | REMOVE — scenarios[segments] new target attached/session and new document recorded before second navigation. |
| 28 | `live_record_test.go :: <package> :: time.Sleep(2 * time.Second) #1` | REMOVE — scenarios[segments] terminal cross-document URL/segment present before returning. |
| 29 | `live_record_test.go :: <package> :: time.Sleep(700 * time.Millisecond) #1` | REMOVE — scenarios[segments] paused state observed; mutate heading synchronously and verify source value before resume;700ms is not asserted. |
| 30 | `live_record_test.go :: <package> :: time.Sleep(900 * time.Millisecond) #1` | RETAIN — scenarios[browse]900ms between six wheel gestures is the real-time fidelity recording input schedule, not readiness. Keep distinct from initial1500ms startup. |
| 31 | `live_record_test.go :: <package> :: time.Sleep(time.Second) #1` | REMOVE — scenarios[segments] initial document/recorded page ready. |
| 32 | `live_record_test.go :: <package> :: time.Sleep(time.Second) #2` | REMOVE — scenarios[segments] pushState hash observed in source and recorded URL before pause. |
| 33 | `live_record_test.go :: <package> :: time.Sleep(time.Second) #3` | REMOVE — scenarios[segments] resumed recorded segment/document observed before creating tab2. |
| 34 | `live_record_test.go :: TestLiveRecording :: time.Sleep(1200 * time.Millisecond) #1` | REMOVE — observe required input/scroll/sheet/hover/ptr/mutation/assets/layout facts on test-owned recorded stream before pause; preserve all final required kinds and content. |
| 35 | `live_record_test.go :: TestLiveRecording :: time.Sleep(1500 * time.Millisecond) #1` | REMOVE — first live recording doc/frame prerequisites; Opened alone insufficient. |
| 36 | `live_record_test.go :: TestLiveRecording :: time.Sleep(2 * time.Second) #1` | REMOVE — expected /next navigation segment/doc and preceding click on recorded stream, then Stats for losses. |
| 37 | `live_record_test.go :: TestLiveRecording :: time.Sleep(300 * time.Millisecond) #1` | REMOVE — paused record boundary acknowledged; source mutation value observed before resume (no minimum paused gap asserted). |
| 38 | `live_record_test.go :: TestLiveRecording :: time.Sleep(800 * time.Millisecond) #1` | REMOVE — resumed segment/doc appears before clicking link; preserve exact recorded segment/order assertion. |
| 39 | `live_record_test.go :: TestLiveRecording :: time.Sleep(time.Second) #1` | REMOVE — source URL/document/forms/frame prerequisites before initial go-live. |
| 40 | `live_record_test.go :: TestLiveStrokeReachesDiskWhileLive :: time.Sleep(100 * time.Millisecond) #1` | REMOVE — testwait predicate for board accepting segment1; retain failure and recorder diagnostics. |
| 41 | `live_record_test.go :: TestLiveStrokeReachesDiskWhileLive :: time.Sleep(100 * time.Millisecond) #2` | REMOVE — testwait predicate reads pointer.jsonl containing this stroke while still live; keep actual disk read, not tee-only success. |
| 42 | `live_record_test.go :: TestLiveStrokeReachesDiskWhileLive :: time.Sleep(time.Second) #1` | REMOVE — source document ready before go-live and board admission. |
| 43 | `live_record_test.go :: recordScenario :: time.Sleep(1500 * time.Millisecond) #1` | REMOVE — recordScenario URL/document readiness before recording; preserve secs-time.Since(start) duration subject (not a debt entry). |
| 44 | `live_replaced_test.go :: TestLiveReplacedElements :: time.Sleep(2500 * time.Millisecond) #1` | REMOVE — require recorded layout carrying canvas400x100 and required replaced-element boxes/frame; do not require guessed number of shots. |
| 45 | `live_replaced_test.go :: TestLiveReplacedElements :: time.Sleep(3 * time.Second) #1` | REMOVE — source .ap-term-text present with expected content and video metadata dimensions, then recorded embed subtree/layout. Keep asynchronous fill unchanged. |
| 46 | `live_replaced_test.go :: TestLiveReplacedElements :: time.Sleep(500 * time.Millisecond) #1` | REMOVE — viewer.ready plus exact placeholder/hidden/shadow-canvas predicates before assertions. |
| 47 | `live_replaced_test.go :: TestLiveReplacedElements :: time.Sleep(time.Second) #1` | REMOVE — source expected URL/replaced selectors ready before go-live. |
| 48 | `live_test.go :: liveHarness.clickControl :: time.Sleep(100 * time.Millisecond) #1` | REMOVE — existing clickControl element discovery through testwait.Wait; retain errors and last control geometry. |
| 49 | `lock_live_test.go :: TestLiveLockPage :: time.Sleep(100 * time.Millisecond) #1` | REMOVE — existing lock-page evaluate predicate through testwait.Wait; retain last value and route/lock assertions. |
| 50 | `server_board_test.go :: TestBoardAnswers :: time.Sleep(time.Second) #1` | REMOVE — StopEnd passes synchronously through c.flush/w.Put; post immediately and retain exact410/refuseEnded. Prior cases consume3note tokens (malformed, stale segment, unknown state), less than NoteBurst5; wrong content type consumes none and oversized stroke uses separate stroke budget. No refill is needed; if later cases exhaust it, split ended check into its own fixture rather than changing admission. |
| 51 | `server_board_test.go :: TestBoardColorLimitAndRate :: time.Sleep(100 * time.Millisecond) #1` | RETAIN — HTTP rate-refill integration is itself tested: exhaust real note budget, wait100ms between probes until refill permits all8notes then color-full, require busy>0. No route clock seam exists; retain real refill coverage. |
| 52 | `server_drive_test.go :: TestDriveOnlyUnderTheDriveLink :: time.Sleep(300 * time.Millisecond) #1` | RETAIN — wait beyond actual100ms seat grace while a view-only stream remains open; proves view stream does NOT hold the seat. Timestamp clock cannot control grace. |
| 53 | `server_drive_test.go :: TestDriveSeatFollowsItsStream :: time.Sleep(300 * time.Millisecond) #1` | RETAIN — wait beyond actual100ms grace with drive stream open; proves attached seat survives timer window. Keep post-disconnect real expiry and pending cleanup assertions. |
| 54 | `server_notedraft_test.go :: TestDraftOwnerUsesLiveGateProof :: time.Sleep(600 * time.Millisecond) #1` | REMOVE — capture Owner from actual authenticated host request, wait existing Service.Status pending→ready, then retain HTTP ready/foreign-cookie/spoofed-header assertions. |
| 55 | `viewer_annotate_live_test.go :: TestAnnotateLiveTwoViewers :: time.Sleep(200 * time.Millisecond) #1` | REMOVE — await final stroke POST completion and ordered writer records; preserve exactly-one-ID and full endpoint assertions (fifth negative oracle). |
| 56 | `viewer_annotate_live_test.go :: TestAnnotateLiveTwoViewers :: time.Sleep(300 * time.Millisecond) #1` | REMOVE — existing waitEval assertions already observe paused pins/disabled rail/dialog; wait model pause transition instead of pre-sleep. |
| 57 | `viewer_live_test.go :: waitEval :: time.Sleep(100 * time.Millisecond) #1` | REMOVE — general waitEval uses testwait.Wait with last got/error diagnostics; predicate/renderer promise decides readiness, no fixed100ms precondition. |


Inventory target:51proposed removals,4retained duration subjects,2hard negative-oracle holds. This is a plan, not a claim that51are already safe: right-click/stroke and driver-policy observations also need their narrow completion contracts reviewed in the complete M2 revision. The headless normal suite may validate those seams after admission; no sleeps are deleted ahead of proof. Current57stay unchanged during planning.

## Teardown / failure ownership

- Test waiters use caller/test context and testwait deadline guard; no goroutine calling t.Fatal. Register subscriptions/taps before the action; cancel/unsubscribe every path, including assertion failure. Preserve previous tap if a shared fixture already uses one. Notifications must be bounded/nonblocking so the Chrome loop cannot deadlock; overflow fails the test with explicit missing-evidence diagnostics.
- All headless Chrome profiles and recorder directories remain test-owned TempDir. Preserve c.Close/Done join. If a helper launches followDriving, wrap it with a done channel in the fixture; ctrl.Close closes its subscription, then join followDriving before Chrome.Close. Do not globally wait on Chrome.work while producers remain active. Join any new test-owned observer goroutine before cleanup destroys its channels.
- HTTP streams: defer/cancel request, close response body, await route done or subscribed controller leave, then close client connections/server; controller.Close joins its work. Keep real grace timers in both duration tests. A predicate must not mutate Release repeatedly merely to probe readiness if it can change the tested state.
- Draft service: retained svc.Close cancels/joins its worker. Test-only controlled capture channel select must include ctx.Done; release on cleanup even after early failure. HTTP server closes connections before service destruction. Never close a notification while a producer can still send.
- An idle observation is scoped to a stable queue generation and already-dispatched action: no generic window.couchViewer flush, no arbitrary extra control event to clear queues, no global monkeypatch of timers/fetch. Diagnostic accessors do not change normal scheduling and expose no page/checkpoint content.

## Admission and verification handoff

Root takes one complete M2 revision to hub before implementation. In that revision name the policy-completion, viewer batch and annotation-post seams, and identify the2hard oracle holds. Normal disposable **headless** browser qualification requires no extra permission once implementation is admitted; it can determine whether the exact deny-download event and input-dispatch semantics exist. A visible/operator/native picker lane requires explicit operator permission; do not start it under this planning task or use the operator profile/session. If headless evidence cannot establish native chooser completion, stop that specific repair and ask hub for the native lane or revised scoped oracle. Do not relabel a readiness guess a duration subject.

After admission: run focused pure/JS protocol tests first (node --test tests/webcast/drive.test.mjs tests/webcast/annotate.test.mjs plus any added focused contract), focused Go server/Chrome fake-protocol tests and race checks, then the existing gated headless tests for the changed functions. Preserve opt-in gating and test inventory. Mutation controls: hold/lose doc readiness, broken driver guard command, delayed popup close, allowed delayed download, intercepted chooser after revoke, admitted right-click queued late, split/early-ended stroke, dropped recorded navigation, missing current-size layout, stalled draft persistence. Each must fail for the named behavioral assertion rather than a compile error. Final debt scanner run proves exact shrink/subset and unchanged retained identities; no inline sleep directives or threshold evasion. Root owns recorded suite/review gates and artifact updates. No test/native/browser command was run by this planning subtask.
