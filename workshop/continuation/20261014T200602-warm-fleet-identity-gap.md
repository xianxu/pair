---
type: continuation
slug: warm-fleet-identity-gap
agent: codex
created: 2026-10-14T20:06:02
supersedes: captured-page-landed
branch: 000218-couch-thread-startup-takes-8-85s-and-nothing-has-measured-where-it-goes
worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
issues: [000218, 000560]
---

# Continuation: warm-fleet-identity-gap

## NEXT ACTION

Investigate the stopped #218 matrix before rerunning anything. Read the active issue Log, then inspect procutil snapshotSelected/identity_darwin and retained /tmp/pair218-MATRIX-_15pmlgy/results-summary.json (never cat samples.jsonl: it contains megabytes of child terminal data). Sample4 refused PID23658 because identity became empty after successful argv read. Preserve fail-closed uncertainty while distinguishing positively gone processes; no retry, sleep or authority relaxation. Retained evidence cannot identify23658 as owned/unrelated. Add regression from a diagnosed contract before a fix. Also preserve diagnostics before fixture cleanup and expose the existing context-only parent launch timing observer to the fleet fixture so lease/record/fsync under load is measured. Then rebuild final binaries including merged #559 and run a separately versioned balanced9 matrix. The old incomplete experiment is retained, never filtered or presented complete.

## State of play

Working pair#218, full flow, branch000218-couch-thread-startup-takes-8-85s-and-nothing-has-measured-where-it-goes, workspace /Users/xianxu/workspace/worktree/pair-slot5/pair. M1 closed SHIP at82c0eea5782ad527d2fcd077af66ac2e9204bc8a; M2 implementation committed, not reviewed/closed. Current HEAD687f9b7548f4fdac7bbb198d4e9bd61970126631 merges origin/main2d8a7dd988d456bc67be33506ab0a1cee58e0e5f including #559 PR336. Constructor conflict combined ownerScope with SessionIndexes/OwnerLaunches caches. Targeted integration command go test ./cmd/internal/couchcore ./cmd/internal/couchcmd -run 'SessionOwnerPass|Fleet|wireResolver' -count=1 PASSED: core16.552s, couchcmd0.300s; no running test session remains. No live fixture operation remains. Agent /root/baseline218 finished and all other prior agents are idle/done; no duplicate experiments.

Only unrelated untracked pensive workshop/pensive/2026-10-13-01-pensive-captured-page-evidence.md remains; preserve it. Current work/evidence flushed into issue and committed. Do not create another capture merely to duplicate it. #560 cold-start attribution is published, unassigned/open; no work on it. #547 already landed/acknowledged, no restart resends.

## Thread arc & user model

TL assigned warm restart attach/detach because21slots repeated full native process scans. Operator approved one shared snapshot per pass and requested6/20/40 measurements. TL explicitly limited #218 to warm paths (receipt e6621a25-d14b-4826-8a8e-edcccd2e2e4a); cold historical8.85s cannot be reconstructed and belongs to560. User wants full attribution and safe lifecycle semantics, not a scan-only speedup claim. About12x75ms scans explain~0.9s only: lease, durable record, client spawn, real first frame and host/CouchCPU need measurement. Production effect fences and #553 exited-means-gone semantics must survive. Avoid operator windows/VM/services; all fixtures private/headless and exact cleanup only.

## Artifact map

Read workshop/issues/000218-couch-thread-startup-takes-8-85s-and-nothing-has-measured-where-it-goes.md then workshop/plans/000218-startup-detach-observation-plan.md. They hold scope, approved milestones, native baseline, commands and evidence. M1 gate artifacts use the longer issue stem under workshop/plans. The old .couch/continuation.md about525/526 is superseded by subsequent218 assignment.

M1 native narrowing: procutil.SnapshotNamed uses metadata only as negative filter, argv plus generation decides, unknown names remain candidates. Server consumers use named zellij snapshots, full diagnostics/nvim inventories retained. Native Darwin/Seatbelt /bin/ps-denied evidence and Linux fixture/cross-build pass; nativeLinux untested (no runner/VM). M2 launcher.SessionOwnerPass captures lazy initial ServerStates once with independent caller cancellation and bounded parent scope; fresh after-pane and Revalidate remain live. couchcore per-checker/root context scope covers startup and LeaveDetach, registration explicitly bypasses cached absence. Console scope factory opens lazily, releases outside mutex and cancels queued effects. Existing tests cover21callers capture1/live42, fresh competitors/replacements/gone, registration bypass mutation,21slotdetach, cancellation/races. See session_owner_pass.go across launcher/couchcore plus console wiring. Actual native counters opt-in via PAIR_STARTUP_TIMING=1; count API calls, not inferred rows.

Baseline complete single attach source /tmp/pair218-attribution.TZ7J81 archived886000bba plus timing-only overlays, frozen /tmp/pair218-attributed-baseline-bin. Parent1.649786s (claim829ms, prepare289ms, spawn7ms, record/baseline124ms, registration200ms, promotion136ms, residual64ms). Child590ms through handoff, prelude159/discovery77/verify192/retention60/revalidate57ms. Attached proof2.079s is not first frame. Detach490.559ms (owner/read170, retirement310, processwait11ms). Parent/child/nested times overlap; never sum them together. Full scan1374rows/1356argv/1058identity75.307ms versus named1375/48/48=2.393ms. Exact commands and limitations in issue.

M2 frozen production binaries /tmp/pair218-candidate-mpwn2106 from4b3ccc608f47322ce3e4f13d408e41700ce5a05d EXCLUDE559. Fleet harness cmd/internal/couchcmd/startup_fleet_timing_live_test.go exercises actual wireResolver/Console/boundedworkers: root plus20background,21privateZellijservers, persistent two-pane marker stand-ins, real terminal-model alternate screen with agent marker required for first frame; delivery and kernel client births separate. ActualCPU/hostlogs/nativecalls retained. Smoke override validated6/20/40 only; default schedule[6,20,40,20,40,6,40,6,20]. Harness fixes committed9f1158045: ThreadRecord revision1/layout2 and Couch layout2 matching seed; guards unchanged. TestFleet passes. No parent fsync fleet observer yet (existing private launchTimingObserverKey in couchcore/launch_existing.go).

Smokes: /tmp/pair218-SMOKE-aw4k478w missingrevision failedsetup; /tmp/pair218-SMOKE2-nm6wcyu6 layoutmismatch refused before spawn; /tmp/pair218-SMOKE3-xgr7yc1s PASS32.72s, root890.20ms/all21frames15400.59ms/adoption15410.72ms/Leave4681.67ms/peak6. CouchCPU10078startup3357leave ms, peak117%, onehost8.4%idle.21exactclientbirths, server/agent/sentinel survival and private cleanup verified. Keep failedattempts.

Balanced matrix /tmp/pair218-MATRIX-_15pmlgy retained source.json with full source snapshot/hashes, results-summary.json, native-counters-by-phase.json, native-counters-by-sample-pid.json, rawlog, samplesJSONL, hostlogs, ownedmanifest. Samples1/2/3 limits6/20/40: all21frames22.740/17.669/14.222s; Leave9.565/3.969/4.058s; actualpeaks6/20/20. Sample4limit20 FAILEDprepare, only2frames; recorded3.456s is partial, not all21. No Leave or samples5–9. Error process identity unavailable after command read PID23658; no retained identification, PIDgone at cleanup. Private launch diagnostics unfortunately deleted with fixture tree: futurecopybeforecleanup. All scoped processes/root absent aftercleanup. Actual startup table/argv/identitycalls:233/47838/41549;225/46819/40517;222/45875/39575 (manyfullprelude scans remain, attribute rather than claim eliminated). Leavecalls77/6171/6159;68/5525/5476;69/5807/5759. Parent/childcounters separated, temporalphase boundaries labelled. No concurrencydecision from one sample each.

## Live deliberations

Native identity API currently returns empty for both disappearedPID and unreadablemetadata. snapshotSelected reads command then identity; emptyidentity with successfulcommand becomesuncertainty. Could be normal exit race but not proven by sample; investigate/error-bearing contract without treatingpermissiondenialasabsence. #559 is nowmerged and materially changes CPU, so finalmatrix must use rebuiltversion with allchanges. #528 pair:2 stillunlanded lastcheck, preserve reachableidentity/exactgeneration semantics whenintegrating. Merge main often.

## Decisions & dead ends

M1 reviewer nativehandoff once said No session found despite positive generation/socket. Subsequent identicalrerun passed. Investigation10attempts hadoneinvalidpaneJSON, then sharednamespace12/12 andisolated10/10passed; causeunclassified. PrivateTMPDIR containment is not demonstratedrepair. No retry/sleep/prodguardweakening. Newmatrixidentityfailure is distinct; don'tassume samecause.

Full suite PASS recorded on71e3380d773ddda553af77a5933dad1884f2aa42 (12m10s, logSHAe43d9c7c917c) beforeM1close; noM2fullsuiteyet. make -k buildpassed4b3ccc6. Finalexactpushedhead needs sdlc local-check run --name suite, thenM2milestone-close, issueclose,PR/merge no bypass. Binaryowns mandatoryfreshreview. Localgateslabel operatoronly. Stage/build:0 requireshubcoordination; do nottouchoperatorworkspacewithoutit. Reportrealhashfromgitrev-parse.

## Lessons learned

Fresh discovery must retain name exclusivity even when initialsnapshotshared. Registration must bypass prelaunchabsence. Native tracing must preserve transparent serverexec. Kernelserverpresence andZellijRPCdiscovery differ. Frame timingrequires meaningfulterminalrender, notstderr/attachedquery. Failedmatrix rows andhostCPUarefirstclassevidence, notoutlierstodelete. Save diagnostics before privatefixturecleanup.

Hubops:0 via bare couch --send-to hub. Last report971227f8-822f-4a01-99c2-532b5a3e428c verified submitted; reports matrixfailure/stopped/integrated559 andnextinvestigation; no resend needed. Earlier a572112e askedpair7hash, nowresolvedthrough fetchedmain PR336. Newrestartprompt b85d00d5 verifiedsubmitted canonical restartreminder, no independentnewwork. Read couch --skill/sdlc --help/tl-worker --help fresh and sdlc state onresume. This writerownsrestart; don'trunpaircontinueseparately.
