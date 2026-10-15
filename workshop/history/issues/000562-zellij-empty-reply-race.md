---
id: 000562
status: done
deps: []
github_issue:
created: 2026-10-14
updated: 2026-10-15
estimate_hours:
card_mirror: '77c3343517cbb1a4fbf07984baa41022132e3871' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-14T21:49:51-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: inferred}
actual_hours: 4.49
---

# Zellij CLI queries can finish before their reply

## Problem

Zellij0.45.1 has a source-level race in which a global UnblockInputThread can arrive on a temporary CLI/probe connection before the response that connection expects. A CLI query may finish successfully without its Log; a ConnStatus probe may silently omit a live session. #218 retained inconsistent successful inventories, but did not capture IPC ordering, so the mechanism is not yet established as the cause of those observed failures.

### Captured evidence

- In #218 MATRIX5, two successful commands about50ms apart returned20 then21 session names. The --short reply omitted the selected owned session; the immediate --no-formatting reply included it. Both had empty stderr and exit0. Exact stdout/times/status are retained locally in /tmp/pair218-MATRIX5.vHTGIP/listing-disagreement.json; the issue remains understandable without this temporary artifact. No automatic rerun or concurrency conclusion followed.
- [Main conformance run38488929394](https://github.com/xianxu/pair/actions/runs/38488929394/job/115609240748), source29dce157654ab7850f6a654738d94547bd841ded, macos-26-arm64, pinned Zellij0.45.1: TestRecoveryRealHelperAndSessionConformanceLive/warm failed at2026-10-14T00:15:50Z with `strconv.Atoi: parsing "": invalid syntax`. The failing make target was test-couch-zellij-live.
- CI evidence boundary: the Atoi input came from source-agent.pid, not CLI stdout. At that revision the source script published the PID via shell redirection, while waitFile accepted os.Stat success before reading it. A zero-length publication interval is a separate plausible cause. The CI log alone does not establish an empty CLI reply or the protocol race; classify this independently before using it as upstream evidence. Source pointers: [PID read](https://github.com/xianxu/pair/blob/29dce157654ab7850f6a654738d94547bd841ded/cmd/internal/couchcore/recovery_conformance_live_test.go#L182-L195), [existence-only wait](https://github.com/xianxu/pair/blob/29dce157654ab7850f6a654738d94547bd841ded/cmd/internal/couchcore/conformance_live_test.go#L35-L45).
- MATRIX4 separately stopped on row7 Leave refusing invalid live pane JSON. The failing authority query was not captured in that version; do not label its bytes empty or infer the same cause.

## Spec

Diagnose the reply/discovery race without retries, sleeps or treating unknown ownership as absence. Preserve exact generation and command authority. A native live server and successful CLI discovery are separate evidence.

### Upstream report draft — internal only, not submitted

**Title:** v0.45.1: global input-unblock broadcasts can finish CLI queries and exclude live sessions from discovery.

The server registers temporary CLI and session-probe sockets alongside interactive clients, and broadcasts UnblockInputThread to every registered client. Pane/tab close and plugin reload can emit these broadcasts. The CLI receive loop accepts an unblock as successful completion before an output-bearing Log may arrive. Discovery sends ConnStatus and reads exactly one response; anything other than Connected excludes the socket.

Primary source pointers: [registration](https://github.com/zellij-org/zellij/blob/v0.45.1/zellij-server/src/lib.rs#L949), [broadcast](https://github.com/zellij-org/zellij/blob/v0.45.1/zellij-server/src/lib.rs#L1287), [pane/tab emitters](https://github.com/zellij-org/zellij/blob/v0.45.1/zellij-server/src/pty.rs#L580), [plugin emitter](https://github.com/zellij-org/zellij/blob/v0.45.1/zellij-server/src/plugins/mod.rs#L440), [CLI receive loop](https://github.com/zellij-org/zellij/blob/v0.45.1/zellij-client/src/cli_client.rs#L206-L252), [discovery probe](https://github.com/zellij-org/zellij/blob/v0.45.1/zellij-utils/src/sessions.rs#L145-L170).

Expected regressions: inject an unrelated unblock before query Log and before probe Connected; the query must preserve its output and discovery must retain the live session. These are source-supported mechanisms, not captured runtime ordering for our failures. Explicit --session actions still enumerate globally before connecting ([dispatch](https://github.com/zellij-org/zellij/blob/v0.45.1/src/commands.rs#L404-L451)), so changing our listing format or naming a target cannot eliminate exposure.

External posting, upstream installation and operator environment changes require their own authorization. Do not publish local fixture paths or session identifiers upstream. Follow-up #563 owns passive IPC-frame observation and the renewed parallelism experiment. Historical cold attribution remains #560.

## Done when

- A source-derived stateful executable fake reproduces early successful completion before query output and verifies Pair preserves uncertainty; pinned upstream source establishes the possible query/ConnStatus interleavings. Actual upstream receiver regressions and captured runtime IPC ordering remain an explicit upstream/#563 dependency, not claimed delivered evidence.
- The CI empty-PID sidecar is classified separately, with a regression for complete PID publication if that boundary is responsible; no claim that Atoi proved empty CLI stdout.
- A reviewed correction or explicitly assigned upstream dependency prevents unknown/empty replies from becoming positive absence or detached authority, without retry/sleep workarounds or weakened ownership fences.
- The upstream draft reflects measured findings and clearly distinguishes hypotheses; external submission occurs only with authorization.

## Plan

- [x] Verify the possible interleavings in pinned source, reproduce unanswered-query handling with a stateful executable fake, and classify the independent CI PID publication boundary.
- [x] Design the smallest correction or upstream dependency with the TL; preserve uncertainty and exact-generation effect guards.
- [x] Verify stateful regressions and applicable live conformance, then hand the stable observation contract to #563.

## Log




- 2026-10-15: closed — Post-close main#528 integration: keep guarded no-delete saved-layout reuse; keep native terminal fixture and positive pane JSON; truthful absent/client replies. Actual replacement-fixture regression RED then GREEN; composition race21.366s, full launcher56.966s. Integrated full suite PASS17a7c65f exit0 10m12s log5427de39d465; PR340 conformance38718342015/job116366437010 GREEN2m24s, merge-check GREEN. No recovery/generation relaxation, retries, stage or operator mutation. Corrected annotated-tag versus peeled-commit wording only; upstream receiver work remains explicit dependency. (fixed-then-ship; full suite: Recorded full suite PASS17a7c65fa31acf2907a7393279fc68cac75fb36d: exit0 in10m12s, log5427de39d465; all prerequisites and Go packages. Main7ba4f86bb included, code conflicts resolved and tested.); review verdict: SHIP
- 2026-10-15: closed — Full suite PASS b15c36aa exit0 9m11s log18ad9a98b2bf before BR-1 correction; replacement suite on54d5b2de queued and MUST pass before merge. BR-1 subprocess regressions RED before correction20.434s then GREEN15.273s; original-code overlay independently fails failed-connect/partial-stdout cases (/tmp/pair562-br1-counterfactual.log). Full corrected launcher -race PASS92.439s; eleven real-subprocess validator controls PASS1.028s. Context failures precede parsing; only source-verified exact post-parse absence replies can prove name validation. BR-2 Done-when/Plan reconciled with TL-approved boundary-fake/source-evidence delivery, old wording retained in Revision. Earlier stateful and atomic publication counterfactuals preserved. PR340 previous conformance green; final54d5b2de CI pending. No external post/operator mutation; live look TL-owned morning restart.; review verdict: SHIP
- 2026-10-15: flow upgraded quick → full — 129 added lines in code files (limit 100); an earlier round of this close already ran the full review
### 2026-10-14

Split from #218 during warm-fleet measurement. Filed as an unassigned follow-up; no implementation or upstream post performed. Full evidence and incomplete matrices remain recorded in #218. CI log retrieved through gh; source at the exact failing commit inspected rather than attributing the Atoi message to CLI output.


## Revisions

### 2026-10-14 — dispatched correction and executable plan

TL dispatch sets the correction: empty/whitespace-only successful CLI stdout is **no answer**, never detached/absent/accepted authority. A separately requested query is a fresh observation; no hidden retry, sleep, positive cache or relaxed generation check. Nonempty malformed pane JSON retains its failing classification. Preserve the existing explicit exit1/no-active-sessions diagnostic as a positively observed empty inventory.

Source investigation verified upstream v0.45.1 annotated tag object bebf50a75198fe6c434cbf34cb2ef981a1d959ca (peeled source commit efd8fd5a89a20c07a111d248ad7fce53848d2c18) from local git objects. The individual CLI receive loop accepts an unrelated UnblockInputThread before Log and start_cli_client then completes successfully. ConnStatus reads one response and excludes anything other than Connected. These are possible source-supported interleavings, not proof of the runtime ordering in #218. Nonempty incomplete discovery remains possible and belongs to #563's passive evidence; this correction does not claim globally complete inventories.

**Chosen design:** one typed ErrZellijNoReply at ZellijSource.runContext's read boundary (ARCH-DRY/ORDER). Empty exit0 with optional whitespace fails; the exact known empty-inventory exit1 diagnostic remains distinct. Existing snapshot/presence/quiescence/owner paths propagate uncertainty; no cached poison prevents a later fresh observation. Sweep legacy launcher authority paths that bypass that seam: SessionBlocksReuse blocks reuse on query error; ProbeSessionName does not generalize empty success into monotone name acceptance; ListSessions propagates missing client/listing replies rather than displaying detached. ProbeLiveLayout uses the same query boundary. ShowFamilyExisting's display-only omission helper creates no authority and stays best effort. No mutation acknowledgments are routed through this nonempty-read contract.

Rejected alternatives: auto-repeat the same query would hide uncertainty and can amplify the upstream race; changing Zellij/installing a patched binary requires upstream work/environment changes and would not protect existing installations. Keep the Pair boundary conservative and the upstream report internal.

**Checkable implementation sequence (single atomic close, no milestones):**

1. Stateful fake-zellij executable scripts ordered protocol frames per invocation. U-before-Log produces exit0/empty, records that Log remained unread; the next explicitly requested observation receives Log. Tests enter real ZellijSource/runContext, snapshot/list-clients, SessionOwnerProbe and legacy safety paths. Assert no automatic second query, no authority/side effect from unknown, and recovery through a new query; separate malformed nonempty JSON, explicit empty inventory, cancellation and generation-replacement controls. Observe RED before changing production.
2. Implement the shared no-reply guard and the enumerated legacy consumers, preserving existing exact server-generation guards and malformed reply behavior. Use controlled process/frame barriers, no timing oracle. Expected production addition stays below100lines (quick shell); if it leaves that shell, use full plan-quality before proceeding.
3. Independently fix the CI fixture publication boundary: its current shell redirection publishes an empty source-agent.pid before printf writes, and waitFile watches existence only. Reuse a test-only same-directory temporary+rename publisher for PID and pane identity. A controlled test pauses at rename and proves public path absent until complete values are published; no attribution to CLI stdout. Test files only; no production sidecar format change.
4. Finish the internal upstream draft with precise pinned source links, ordering pseudo-regressions, source-supported versus captured evidence, and explicit PID-race exclusion. No upstream posting or installation. Update the atlas map for observation semantics; give #563 the stable boundary/evidence limitation via the hub.
5. Run affected launcher/core fixture tests with race detector, controlled repetition and guard-removal counterfactual; push and run declared full suite once before close, then close's fresh review. Merge main and land through sdlc merge --yes on SHIP. Ask hub only if actual live/operator setup is needed; do not stage or touch pair:0 without instruction.

**Pure/integration split:** response presence and existing JSON/parser classifications are deterministic; ZellijSource.runContext executes one bounded subprocess. SessionOwnerProbe uses its existing injected IO and before/after server identity. The fake owns persisted per-test frame/invocation state, removed with t.TempDir; no production durable state or goroutine is added (ARCH-PURE/MOCK/FUNERAL). Existing5second per-query budget and caller cancellation remain; number of production queries does not increase (ARCH-CONSTRAINTS). Session/PID identities are rechecked at effect boundaries as today (ARCH-SECURE). This serves the observation error class without attempting an upstream protocol replacement (ARCH-PURPOSE).

**Acceptance:** deterministic private fake/headless scenarios plus the full suite and existing live conformance lane. Live pane/operator mutations are unnecessary to force the chosen interleaving; request hub's acceptance decision with the plan. The separate PID publication regression is mandatory even though that failure is not the Zellij race.

Planning checkpoint: claimed pair:7, start-plan refreshed Ariadne and then succeeded on retry. No implementation or real Zellij session mutation performed.

### 2026-10-14 — approved implementation and negative-name-cache discovery

TL approved855b248f via receipt4efcb022 (submitted). Acceptance is stateful fake/headless plus couch-zellij-conformance CI green on the PR; live check belongs to the operator's morning restart. Atomic PID/pane publication is explicitly included as the independent cause class of CI38488929394. No separate fixture issue found in current active issue text; queried pair:5 (send uncertain, receipt unavailable), so do not assume a peer answer.

Legacy regressions reproduced empty listing freeing a name, empty name probe accepting length, and pair list inventing attached/detached rows after missing listing/client replies. Further class sweep found the bool sessionNameAcceptor also collapses any failed probe into a negative monotone length bound and a misleading shorter-tag instruction. Revision: add a retained observation error to that assignment-local acceptor, do not learn a length bound or issue additional probes after unknown, and propagate it through initial/single-name assignment and prompted length measurement. A new assignment remains a fresh observation. Explicit validator length refusals keep the existing shortening ladder. This is the approved conservative-name-consumer behavior, not a new retry mechanism. Tests must cover both positive and negative monotone poisoning and independently remove each guard.


### 2026-10-14 — completed upstream report draft (internal, not submitted)

**Title:** v0.45.1: unrelated UnblockInputThread can finish CLI queries before their Log and interfere with ConnStatus discovery

**Version/source:** Zellij v0.45.1, annotated tag object bebf50a75198fe6c434cbf34cb2ef981a1d959ca, peeled source commit efd8fd5a89a20c07a111d248ad7fce53848d2c18. This report identifies source-supported interleavings. We have not captured the actual IPC ordering of the inconsistent inventories observed in our application.

**Query ordering:** Temporary CLI connections enter the same [client registry](https://github.com/zellij-org/zellij/blob/v0.45.1/zellij-server/src/lib.rs#L655-L675) as interactive clients. [Connection handling registers before routing](https://github.com/zellij-org/zellij/blob/v0.45.1/zellij-server/src/lib.rs#L945-L956). The [global unblock broadcast](https://github.com/zellij-org/zellij/blob/v0.45.1/zellij-server/src/lib.rs#L1287-L1297) includes every registered client ID, including temporary query connections. An unrelated pane/tab close or plugin event can therefore queue an unblock on one of these connections. In the [CLI receive loop](https://github.com/zellij-org/zellij/blob/v0.45.1/zellij-client/src/cli_client.rs#L212-L258), UnblockInputThread returns None immediately; Log prints its output only if it arrives first. The [caller](https://github.com/zellij-org/zellij/blob/v0.45.1/zellij-client/src/cli_client.rs#L73-L88) treats None as completion, sends ClientExited and returns successfully. Thus exit0 need not mean the requested output-bearing reply was received.

**Discovery ordering:** The [session probe](https://github.com/zellij-org/zellij/blob/v0.45.1/zellij-utils/src/sessions.rs#L148-L168) sends ConnStatus and consumes exactly one response; any message except Connected excludes the socket. The [server's ConnStatus response](https://github.com/zellij-org/zellij/blob/v0.45.1/zellij-server/src/lib.rs#L1617-L1619) can follow a queued unrelated unblock. A live session can then be omitted even when other sessions make the overall listing nonempty. Explicit --session commands still enter [global discovery](https://github.com/zellij-org/zellij/blob/v0.45.1/src/commands.rs#L404-L451), so merely changing listing flags or supplying a session name does not remove this exposure.

**Deterministic source-ordering scenarios requested upstream:**

1. Create an output-bearing CLI query connection. Deliver UnblockInputThread unrelated to that query, then Log with a recognizable nonempty body. Expected: query retains the requested body (or explicitly reports incomplete observation). Current receive transition permits successful completion before consuming Log.
2. Create a ConnStatus probe against a live registered server. Deliver unrelated UnblockInputThread before Connected. Expected: discovery retains the live session after its own response; current one-read probe rejects it on the first frame.
3. Control: Log/Connected before unblock retains expected output/presence. These can be injected at the receiver boundary without relying on a probabilistic concurrency soak or sleep.

**What we measured versus inferred:** Our application retained two successful inventory commands approximately50ms apart returning20 versus21 names, with a selected live session omitted from the first. We did not capture those connections' IPC frames, so that observation is compatible with this mechanism but does not prove it. Our committed Pair regression is a source-derived stateful CLI fake: ordered UnblockInput before queued Log produces successful empty stdout; Pair now refuses to turn it into absence/detached/ownership authority. A later explicit query is a new observation. This test validates our boundary handling, not Zellij's Rust wire or scheduling. Malformed nonempty pane JSON remains invalid. Nonempty incomplete discovery still needs passive IPC evidence and is not fixed by our empty-reply guard.

**Separate CI fixture failure:** The `strconv.Atoi` empty input in our conformance run38488929394 was read from our source-agent.pid file, not CLI stdout. The writer opened the public file before printf supplied the PID while the reader waited only for existence. A deterministic barrier reproduces that empty-publication interval. We changed PID and pane-value fixture publication to sibling temporary files plus rename. That failure is not evidence of this Zellij reply race and should not be cited as an upstream reproduction.

**Potential upstream direction, not a verified patch:** Give query/probe completion its own correlated response semantics, or ensure unrelated input-unblock messages cannot terminate or disqualify temporary query/probe clients. We would welcome a maintainer-provided receiver-level regression for the two sequences above. No upstream patch, installation or external report submission has been performed here; identifiers and temporary artifact paths are intentionally omitted from this draft.

### 2026-10-14 — implementation verification checkpoint

Implemented the one read-boundary ErrZellijNoReply and the enumerated legacy consumers. Only explicit typed validator refusal can update monotone negative name bounds; assignment-local queryError retains unknown without further probes and is propagated at every production caller. Initial and prompted assignments now report actual failed evidence instead of inventing a shorter-tag recovery. Independent regressions also cover an unrelated error containing the words too long, uncertainty during limit measurement after a genuine refusal, and a fresh assignment observing a later answer. Production additions remain91lines, inside the inferred quick shell.

Stateful CLI model RED twice before the central guard; legacy tests separately reproduced5 authority failures. Latest normal/race evidence: launcher full package22.515s in /tmp/pair562-launcher-confirmation.log; integrated launcher/core race49.835s/1.968s in /tmp/pair562-final-race.log; final projection/measurement/fresh-assignment race3repetitions1.447s. Atomic PID/pane fixture race20repetitions4.608s, restoring old direct-public writes fails both cases. Removing the central guard fails snapshot sequence tests; removing owner empty-input guard fails injected-IO authority test; removing negative-name guard fails assignment/prompt regressions; removing post-measurement uncertainty guard fails its independent test. All overlays live only in /tmp and were not committed.

Full launcher testing caught a preexisting fake that represented no sessions as successful empty stdout; it now emits Zellij's explicit exit1/no-active-sessions diagnostic. A new measurement test initially used a candidate shorter than the calibration marker and therefore never entered its intended second observation; enlarged the candidate and verified its exact2probe contract, including a guard-removal red. These were test fixture corrections, not weakened production checks.

The separate PID race from CI38488929394 is fixed by the shared atomic publisher in the live fixture; no claim that this measured CLI stdout. Current active issue search and pair:5's visible follow-ups identify #564 as an unrelated retention-registration budget issue, not a duplicate PID fixture issue. No unrelated card was closed. Upstream report remains internal, and no operator session/stage was mutated. Full recorded suite and couch-zellij-conformance CI on the PR remain pending before landing.

Merged current main141e055b817765a50ab718224bc3d5c72b7769ff before full-suite scheduling; #218 added initial native-owner pass sharing and the bounded query observer. Integration was clean. Updated its observer regression: successful empty client stdout now retains raw exit0 but carries ErrZellijNoReply instead of detached classification; raw bytes/command/status remain observable. Added a pass integration test proving one shared native capture does not cache an unanswered CLI pane reply: an explicit second Probe performs a new CLI observation and establishes ownership. Targeted owner-pass/observer race tests pass4.111s in /tmp/pair562-main-integration.log. #564 is confirmed unrelated retention-registration deadline work and remains open.

Final empty-reply sweep caught one more masked observation: ProbeSessionName used CombinedOutput, so exit0 with empty stdout plus unrelated stderr could teach name acceptance. A new regression reproduced it; stdout/stderr are now kept separate for successful-reply presence while real validator refusal diagnostics and nonzero no-session validation retain their existing interpretation. Targeted race/selected-root tests pass11.597s in /tmp/pair562-stderr-guard-green.log. Production additions are97lines, still inside the quick shell. Canceled the still-queued b24d39c3 suite before acquisition and replace it with this final code; no running suite was stopped. PR340 conformance38689468810/job116271776125 and other CI checks were green on b24d39c3; the final revision must receive its own CI and suite evidence before landing.

### 2026-10-14 — full-suite fixture correction and remainder

First recorded full suite on c2269c22775394e6afaf028d13f050c0ddfa865f ran after1729seconds queued and failed at test-pair-embedded-runtime (exit2 in2m13s, log sha256prefixce675e92ccad). Its shell Zellij fake still represented observed absence and zero clients as exit0/no stdout. Corrected the fake to exact exit1/no-active-sessions for absence and a real client header for zero clients. Swept CLI fixtures across shell tests and shared helpers: StubZellij also emitted empty success for an empty inventory; its independent regression reproduced that bad model, and it now emits the explicit observed-empty diagnostic. No production unknown guard was weakened.

The shell remainder is running with make -k over every remaining prerequisite after the failed target; the next recorded whole-suite run will include the Go package recipe and all prerequisites. make test-pair-embedded-runtime now passes, and shared-stub/core absence tests pass0.261s/5.082s. A direct bash invocation of the embedded test was invalid under inherited Couch scope knobs; using the Make wrapper provided its canonical isolated test environment. Final suite and close remain required; the previous failure is retained here rather than attributed to a known flake.

### 2026-10-15 — composition and public-entrypoint fixture sweep

The shell remainder completed green. The second recorded full suite on171587984ce937aae98d30c62c080ff419178b98 ran after1990seconds queued and failed in14m26s (log sha256prefixee8422ca96db). All shell prerequisites passed; Go completed its other packages. Couchcore timed out at10minutes in TestSpawnComposesProductionPairRegistrationBoundary, leaving its parallel tests pending, and pair-go TestPublicPairCommandFamiliesIgnoreCouchStore failed four launch-create variants. These are not attributed to flakes.

Root cause: two additional executable fixtures still returned empty successful listings. The composition fixture entered post-ack cleanup awaiting a known-absent session, which the new boundary correctly refused to infer. Its original targeted reproduction timed out at45seconds; explicit exit1/no-active-sessions plus a client header passes both ordinary/fresh-slot cases3.502s and race3repetitions11.255s (/tmp/pair562-core-spawn-{red,green,race}.log). The public-command fixture received the same truthful observation model; its complete namespace/store-family test passes20.445s, retaining exactly-one-create-handoff assertions. Swept Couchcore and public entrypoint executable fixtures, including generic launch-only stubs: no production guard or timeout changed. A new recorded whole-suite pass is still required, including the previously blocked parallel core tests. PR340 conformance on17158798 was green (run38693235094/job116283761799); replacement head must pass independently.

Merged origin/main e85237e7e (including #554 render-fixture fix) before replacement-suite acquisition. The c3de83ab run was canceled while still queued, with no test body started; replacement will test this integration. No conflict or operator environment change.

### 2026-10-15 — complete verification and observation handoff

Recorded full suite PASS on b15c36aa4fddcfcaa13a969e189121a8b0a93a0b: exit0 in9m11s after3943seconds queued, log sha256prefix18ad9a98b2bf; couchcore262.439s, launcher22.875s, pair-go15.196s. Every shell prerequisite and Go package completed. All PR340 CI passed, including couch-zellij-conformance run38699358810/job116303506925 (2m47s). Earlier failed runs remain documented above; no failure was waived as a flake.

Stable #563 handoff: ErrZellijNoReply means no observed stdout after a successful read, preserving raw exit0 in the query observer. Exact observed no-active-sessions remains distinct. Each explicit query is a fresh observation; there are no retries or relaxed generation checks. A nonempty incomplete inventory remains possible and must not be interpreted as proof that the upstream interleaving was measured. The internal upstream draft includes source-derived receiver scenarios and the evidence limits. TL owns the morning restart look; no stage, operator Couch mutation or external submission occurred.

After that suite, merged current main965fcb814, including #423's test-only inventory-readiness correction, cleanly. A replacement recorded integration suite will run before landing, as required by the gate; the completed pass above remains valid evidence for #562's behavior.

### 2026-10-15 — close review BR-1 / BR-2

The full close review (automatically upgraded from quick at106 added code lines) returned REWORK with two real findings. BR-1: ProbeSessionName still treats nonzero replies with arbitrary diagnostics as positive validator evidence, allowing failed connections or cancellation with partial output to teach monotone acceptance. Enumerate every outcome: context failure first, explicit length refusal, explicit known validator/no-session evidence, successful nonempty stdout, and all other failures/empty success remain unknown. Add subprocess-level failure/partial-output/cancellation regressions and verify the assignment-local acceptor retains that uncertainty.

BR-2: the original Done-when said “A deterministic protocol regression establishes whether unrelated unblock can finish a query before Log and exclude a live ConnStatus probe”; the first Plan row claimed reproduction of both interleavings. That exceeded TL-approved delivery (receipt4efcb022): a source-derived Pair read-boundary fake, source-verified upstream possibilities, and an internal report requesting actual receiver tests. Updated those two statements to the delivered boundary scope, preserving the old criterion here. No ConnStatus receiver model, upstream Rust test or captured runtime ordering is claimed. The explicit outstanding dependency is the internal upstream draft's receiver scenarios plus #563's passive IPC evidence.

The integration suite on77145de9 was still queued and was canceled before acquisition to avoid testing a revision superseded by the required BR-1 code correction. The completed b15c36aa full pass remains recorded.

BR-1 source check: pinned Zellij0.45.1 zellij-utils/src/cli.rs30–51,76–77 attaches validate_session to --session; src/main.rs20 parses before commands.rs409–449 action dispatch. Thus exact exit1/no-stdout action diagnostics (There is no active session!, or this exact requested name missing with one active-name line) prove only length validation. All arbitrary failures, mixed partial stdout, compound discovery failures and cancellation remain unknown. Context failure is checked before any validator-looking text. Added pure diagnostic classification plus real-subprocess controls. The five original regressions failed in20.434s before correction (positive-bound poisoning and canceled negative-bound poisoning with an extra probe); after correction all five pass15.273s. Eleven explicit reply-shape controls pass1.028s.

Full launcher package with the BR-1 correction passes under -race (92.439s, /tmp/pair562-br1-launcher-race.log). The shared Zellij read boundary already checks context/failure before granting output authority; the remaining display-only helper is unchanged. No other authoritative subprocess outcome classifier accepts failed partial output.

### 2026-10-15 — post-SHIP main integration

Corrected full suite on54d5b2defa4d2c44f827a17cd9100c89f3069afd PASS: exit0 in10m27s after3483seconds queued, log sha256prefixca95d8684ad9. Close round2 SHIP disposed both BR-1/BR-2 with no new findings (evidence a02f0bd47951); final close-head CI conformance38710050422/job116338901052 GREEN1m51s, merge-check GREEN.

Main #528 landed as PR337 during this wait. Integrating7ba4f86bb exposes three conflicts: SessionBlocksReuse now preserves saved layouts without deleting exited sessions, and two CLI test fixtures were replaced/extended for native terminal admission. Keep main's guarded runContext reuse behavior; keep its native terminal composition fixture, updating its unanswered listing/client replies to truthful absence/header evidence; combine selected-root's explicit failure knob and positive pane JSON with #562's explicit empty-inventory/client-header replies. Add an actual native-fixture reply regression. No recovery authority, generation or capture semantics are changed. This code-conflict integration requires a fresh full suite and narrow fix review before landing.

Hub report707b578a-2360-49cc-815b-be8584b7e375 returned outcome uncertain; receipt unavailable on query. Hub peek remains live. No automatic resend was attempted; this Log retains the integration state until the distinct landing report.

Source-identity correction: earlier prose called bebf50a75198fe6c434cbf34cb2ef981a1d959ca a commit. Local cat-file identifies it as the annotated v0.45.1 tag; rev-parse v0.45.1^{commit} yields efd8fd5a89a20c07a111d248ad7fce53848d2c18. Corrected the draft/source description to distinguish both objects; source content and findings are unchanged. Integrated full launcher package passes56.966s (/tmp/pair562-main528-launcher.log).

The actual #528 native registration fixture still emitted empty success before readiness and for clients. The new executable regression independently failed both session-listing modes and clients before correction; corrected helper plus ordinary/fresh-slot production-composition cases pass5.408s and -race21.366s (/tmp/pair562-native-fixture-{green,race}.log). Conflict resolution changes only fixture observation replies, retaining native lease/admission behavior.

Integrated full suite PASS on17a7c65fa31acf2907a7393279fc68cac75fb36d: exit0 in10m12s after2681seconds queued, log sha256prefix5427de39d465; couchcore340.169s, launcher37.636s, pair-go30.820s. All prerequisites/packages completed. PR340 conformance38718342015/job116366437010 GREEN2m24s and merge-check GREEN. origin/main is already contained; post-conflict narrow review is the remaining gate.
