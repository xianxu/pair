---
id: 000744
status: working
deps: []
github_issue:
created: 2026-10-20
updated: 2026-10-20
estimate_hours: 8.337
collaboration-mode: collaboration
card_mirror: 'a6df4fea1bea2d49afe5ef94de0bb53eca2a832d' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-20T14:46:17-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
---

# testing: one shared disposable-Couch harness for every live test (isolated roots, real supervisor and slots, every agent kind, exact cleanup)

## Problem

Live tests each build their own disposable Couch: pair#722's three-slot drain matrix, pair#726's held-cold resumes,
pair#740's global park, pair#741's hub-resume conformance, and pair#743's conformance in a VM. Each reinvents
isolation (state folder, socket namespace, HOME, queue root) and cleanup, and a mistake in any of them can touch the
live fleet. Operator, 10-20 (ops side doc `2026-10-20-19-conformance-structure.md`, decision 1): one shared harness.

## Spec

Build one shared `cmd/internal/testcouch` helper for the native disposable fleets needed by #722, #740 and #741. TL2d5e80a3 explicitly selects real installed Zellij topology under isolated roots and dispatches this issue independently from main. No #722 commit or drain API is a dependency. Its claim/PR remain parked until this helper lands.

The helper owns a fresh root lease, empty environment, actual Git primary/remote and N numbered workspaces, one real supervisor/steward, actual Pair/wrappers and native Zellij sessions. Real installed SDLC/Weave provision minimal repositories without Brewfiles/Makefiles; no invented setup markers or caller bindings. Public CLI runs inside the controlled agent's inherited registered environment. All mutations and cleanup target only the owned fleet. A deterministic fake-agent engine has Claude and Codex personalities; these are fixtures, not installed models or behavioral conformance to real agents. Qoder and real-model/network runs are outside this dispatch.

Normal tests qualify pure confinement/ownership/agent reducers with stateful process/filesystem doubles. A named opt-in native test consumes the shared helper, starts a genuine steward and three non-hub workers, exercises actual messaging for both supported personalities, detach/restart and exact cleanup. It is the first live-test consumer of the helper; the full #722/#740/#741 scenarios resume in their own issues. Existing bespoke tests migrate when next touched, rather than a fleet-wide refactor here. Full design: [shared native harness plan](../plans/000744-shared-native-couch-plan.md).

## Done when

- One shared helper creates only private roots/environment, actual native supervisor/Pair/wrapper/Zellij and a configurable bounded number of genuine numbered workers; provisioning and caller identity use production routes.
- Deterministic Claude/Codex fixture personalities support recognized input, controlled work/picker/idle, public CLI execution and native conversation artifacts for consumer-owned lifecycle scenarios, without credentials or network.
- Every direct child and IO pump is joined; daemon/session shutdown is fenced to the run's private namespace and observed process generations. Startup failure, cancellation and PID reuse cannot target a foreign process or erase unknown live state.
- A guard test rejects live/inherited/empty/symlink-escaping paths before effects and proves derived messaging sockets differ from the live namespace. State/operation counts at N and 2N defend bounded fanout.
- A real opt-in consumer runs successfully for Claude and Codex with a genuine steward plus three non-hub workers, public message/receipt paths, detach/restart, and verified cleanup. It emits exact-head/native-version/isolation/log evidence. Cheap tests stay in the ordinary suite.
- Atlas documents this as the shared live-fleet path; reader/artifact inventories, required recorded suite, mandatory close review and normal PR landing pass. No protected local-gate declaration or staging of operator slots is included.

## Plan

- [x] TL reviews the durable design and state/recovery table before change-code; derive the estimate only after plan-quality.
- [x] Implement confinement and owned lifecycle plus deterministic agent/CLI protocol with focused adversarial tests.
- [x] Compose real provisioning/native runtime, then qualify the first shared-helper consumer and audit source-reader/artifact ownership.
- [x] Document verification and prepare the recorded-suite/close handoff. Publication remains subject to the suite, close and merge gates.

## Estimate

Produced via brain/data/life/42shots/velocity/estimate-logic-v3.1.md against baseline-v3.1.md. Method A only. The calibration source is marked stale by estimate-source; values are provisional ship wall-clock, not a deadline.

Decomposition: issue/spec shaping; root guard/environment; ownership reducer; Claude/Codex fixture paint/transcripts; correlated control protocol; candidate build/provisioning adapter; native fleet/actor adapter; exact cleanup adapter; qualification consumer; isolated transport seam; atlas/inventory work; one close review; three external API discovery units (Zellij, SDLC/Weave provisioning, native transcript/resume). Existing testpair, procutil, terminal, captures and provisioning contracts avoid greenfield library work. Design uses the thorough-plan ×0.2 discount on implementation components, plus a 15% buffer. Impl values are already v3.1's 40% scale. Familiarity 1.5 reflects native but bounded integration; do not apply another implementation scale.

```estimate
model: estimate-logic-v3.1
familiarity: 1.5
item: issue-spec design=0.5 impl=0.08
item: smaller-go-module design=0.06 impl=0.2
item: greenfield-go-module design=0.2 impl=0.32
item: tui-screen design=0.2 impl=0.4
item: smaller-go-module design=0.06 impl=0.2
item: api-integration design=0.2 impl=0.6
item: api-integration design=0.3 impl=0.6
item: api-integration design=0.3 impl=0.6
item: smaller-go-module design=0.06 impl=0.2
item: smaller-go-module design=0.06 impl=0.2
item: atlas-docs design=0.04 impl=0.08
item: milestone-review design=0 impl=0.2
item: real-api-discovery design=0 impl=0.36
design-buffer: 0.15
total: 8.337
```

Design 1.98 ×1.15 + impl 4.04 ×1.5 = 8.337 hours. Setup/runtime/cleanup budgets in the plan are separate resource limits, not estimates.

## Log

### 2026-10-20 — clean waiting checkpoint

Final audited PTY test hash refresh passed the reader inventory/fixed-preflight check (1test5.201s); helper races after the lifecycle creation-gate correction passed13.453s. Documentation, artifact/reader declarations and cheap guards are ready. TL sequencing is recorded above; wait for landed #747, integrate only landed main, then run both native personalities and the required pushed-head suite before mandatory close review/merge. No composite build, stage, native pass, full suite or SHIP is claimed. Claim retained; #722 remains parked until #744 lands.

### 2026-10-20 — documentation and guard qualification while waiting for #747

Added atlas/couch-testing.md, linked the shared helper and isolated message parent, and documented make test-couch-fleet in README. Audited artifact sources and reader classes; renamed the internal intermediate executable fixture-agent to avoid artifact-vocabulary collision. The main-package ignore guard now treats an existing source-directory collision as the go-build refusal it is, preserving file/missing-path negative controls. Artifactpath passed10.018s, reader audit9 tests61.265s, affected-runner11 tests15.368s, candidate/installer checks0.875s. Repolint initially found two fixed sleeps in the real-PTY regression and an unused suppression; replaced waits with observed control-turn and resized-frame events, removed the suppression, then native entrypoint races passed3.571s and repolint passed8.294s. Final reader hash refresh pending this test edit; no native retry until landed #747. Full required suite and mandatory close review remain outstanding.

### 2026-10-20 — native worker blocker #747; exact failed-run recovery

Second native target ran clean Pair/Couch candidates at25ac5f6fbf856a337fd03d53108ea4a66f9386ed with real SDLCf8d649244c434aa51a81330a0600d9997d64522b and Zellij0.45.1. Public adoption, genuine provisioning and Claude steward registration passed. Its actual reboot fixture:1 failed with reserved-vs-established registration; filed unclaimed #747 and hub receipt21fa0729 was submitted. Full log: /Volumes/TestScratch/tmp/pair-slot5/pair-744-native-ump2qlja/native.log, SHA256e4ef8082296fb7cbd03b6ad947f4e71fcf8ccf73fdec59256ea5089886c4593a. No passing native fleet claimed.

Cleanup exposed a real native stdin issue: Stop returned but os.Stdin.Close did not unblock its already-blocked TTY reader. Added a real-PTY/self-exec regression (red on prior adapter), fixed Main with cancellable ttyio and cached native fd for resize, then verified fixture races. The driver had already entered Codex after the failed Claude cleanup; it refused the still-live same logical session name and cleaned its own root. The qualification now stops immediately after a failed personality, before another fleet starts. The retained Claude run was recovered with a temporary exact-owner/generation probe, all owned generations confirmed absent, and the probe deleted. Its diagnostic root /Volumes/TestScratch/pc-3857330288 remains with ownership Phasequiescent/zero resources; no live test process is retained. Current focused races pass helper4.965s, fixture1.809s, native fixture entrypoint3.899s and couchmessage30.356s; ordinary consumer opt-out passes0.456s. Native/full-suite/close remain pending #747. #722 stays parked with its claim.

### 2026-10-20 — first native attempt and cleanup review corrections

The first native consumer failed at genuine provisioning before any slot launch: the public provisioning route reads the singleton selection, and a fresh store had not been adopted. Both Claude/Codex roots were removed successfully; no native qualification claimed. Added the real public adoption preview/validated digest/apply route before provisioning, with stateful refusal tests. A bounded cleanup review found marker-only census missed platform-binary descendants, Stop was not joined, and blanket late resource removal could erase unobserved generations. Fixed those classes with indexed ancestry/private-TTY closure, fixture generation publication, actual supervisor/agent shutdown joins, exact observed absence updates and a quiescent-before-root-removal reducer state. All testcouch/fixture race tests passed (10.723s/3.125s). Next: rerun native consumer and retain full log evidence; no full suite/close yet.

### 2026-10-20 — implementation checkpoint before native composition

Full-flow change-code passed at c23595e57 after PQ-1 transport isolation correction and TL3d5e6782 approval. Estimate-quality returned INFO; the 8.337h decomposition includes native debugging/races/required-suite work in the integration adapters, with qualification item covering the consumer itself. Implemented shared root/environment guards, pure owned-generation reconciliation, actual provisioning adapter, candidate/runtime placement and deterministic Claude/Codex fixtures. Focused testcouch packages passed (4.901s helper, 0.893s fixture), and transport full package (26.147s) plus targeted race checks passed. The first installed consumer compiles and correctly opts out without PAIR_NATIVE_COUCH_FLEET; native composition is the next step, with no native/full-suite/close pass claimed. #722 remains parked.

### 2026-10-20 14:50 PDT — native shared-harness plan ready for TL review

#722 is parked at its pushed checkpoint with claim kept. #744 claim/start-plan created this independent branch at fresh main; no #722 commits were imported. Two bounded read-only explorations verified native environment/agent reuse and real provisioning/public worker startup. The proposed helper uses real provision-workspace and recordless reboot, not extra hubs, fake workspace identities or setup markers. Durable plan records confinement/ownership state table, exact execution/cleanup seams, agent personalities, native qualification/first consumer and limits. Await TL review before change-code or implementation; no estimate derived yet.


### 2026-10-20


### 2026-10-20 — landed #747 integration and native delivery failure

Integrated landed main containing PR498 into clean head 4ef7301c08078dee9c7a50f1f5dc0a0552c3c448. Native qualification brought up a genuine Claude steward plus all three workers, then timed out waiting for ordinary receipt submission. Full log /Volumes/TestScratch/tmp/pair-slot5/pair-744-native-k8fbki9y/native.log, SHA256786982c4cbaa630a33122cb2690725153fbb5a5502f7c2a95831257ff938847b. All fixture composers held the Zellij DCS version-reply payload >|Zellij(4501); receipt remained queued for an empty composer. Repeated status queries consumed the fixture's ordinary mailbox budget, which incorrectly blocked Stop too. No matrix pass or close claimed. TL2f8fbc72 approved regression-backed corrections and requires identifying the query producer before classifying the DCS defect as fixture-only.

A temporary exact-owner/generation recovery probe reconciled only run82d0fab94f658ef6d73cd5acd1a8110a. The native test returned PASS0.66s with Phasequiescent and zero owned resources; diagnostic root /Volumes/TestScratch/pc-3751337488 is retained. Probe removed immediately afterward. No retained live test fleet or new native run pending.

### 2026-10-20 — native negotiation provenance

The pinned Claude composer capture itself contains XTVERSION ESC[>0q at byte59, emitted once by agentfixture/run.go before deterministic repaint. Native Zellij returns its solicited DCS response through the same pane PTY to the fixture. Every retained raw scrollback contains one such query; no Pair/Couch-originated or replayed query is demonstrated. The fixture parser dropped the DCS opener but treated its payload as authored input. Whole/every-two-piece-split regressions now prove both personalities ignore the completed bounded control string and still accept a following ordinary message. This is fixture negotiation handling; no independent product bug is established by this evidence.

### 2026-10-20 — negotiation and shutdown regressions verified

DCS whole/split framing and counted shutdown regressions pass, including 128 completed ordinary requests followed by Stop, sixteen stale shutdowns without actor exit or retained-result growth, and unrelated pending-result absence. Shutdown owns one reserved request/result slot; the ordinary quota stays128. Independent full fixture race run passed agentfixture2.229s and native entrypoint3.442s. Audited model_test.go and run_test.go changes read only their existing private fixtures; refreshed those two local_or_literal_fixture hashes individually. Next: commit these corrections and rerun native matrix from that clean head.

### 2026-10-20 — caller deadline correction after bounded CLI failure

Native rerun79a7f25ea072b3ec47218fa6b74bef45fa902242 failed worker3 after fixture runCLI killed its recovery subprocess at its fixed15s inner deadline; public slot recovery permits30s admission plus5m operation polling. Full log /Volumes/TestScratch/tmp/pair-slot5/pair-744-native-f6oqf28t/native.log SHA256b0fa89ae4f1b3aaf9b70c121f1d8b5ff8f8e4e54fae9a6507d8c2f464a9ed474. Cleanup succeeded with all owned generations absent, direct child/IO joined and root removed. No ordinary-message/restart matrix pass claimed.

ActorCLI now transports its caller's remaining bounded deadline to the fixture command; no arbitrary increased inner constant. Direct fixture tests retain a15s default; explicit requests are validated through24h and parent cancellation still joins native children. A readiness audit also found that RestartSupervisor could observe all four reconnected actors before Start's exact-one snapshot; readiness now accepts at least its requested count, while native qualification still asserts exact membership. Stateful observer regression failed on the prior exact comparison and passes after correction. Portable helper5.453s, fixture1.479s/native entrypoint0.512s, artifactpath5.776s and repolint12.863s passed on the preceding correction; current combined races and inventory are in progress before the next clean-head native run.

### 2026-10-20 — complete pre-rerun harness sweep

Native head299c7bf30748c23971207cd5842b1047bf40a174 established four genuine Claude registrations, three concurrent public actor queries, receipt0090a45e-be1d-42ea-bc99-fcca46ed1d65 submitted and independent worker accepted-body evidence for shared-native-fleet-ordinary-message-claude. It timed out at Detach: the helper sent legacy ESC-d, which Couch intentionally treats as ordinary input. Log /Volumes/TestScratch/tmp/pair-slot5/pair-744-native-vavozhhc/native.log SHA256c0f476dd7aed8d8df381e748d622be9b698f107918765b86827c03a3395e4903. Cleanup confirmed all generations absent and removed its root.

Following verified TL69cdbd1e, swept every helper/consumer input, query, timeout, sleep and poll before another native run:

- Host keys: Detach is the only physical-key producer. Encode Ctrl+Space/Alt+d from the actual terminal Endpoint keyboard mode, including CSI-u event flags; observe the real switcher before Alt+d. Regression decodes both negotiated modes and pins the legacy non-chord. Ordinary peer sends retain production paste/submit; both personalities' actual wrapper profiles submit raw CR, which the fixture accepts.
- Capture negotiation: Claude emits its original XTVERSION/DA1 queries; Codex emits CPR, OSC10/11 colors, Kitty and DA1. The fixture consumes bounded fragmented DCS/OSC/CSI responses. No extra product-originated query found. Direct physical CSI-u typing into worker fixtures is not qualified by this matrix; only console keys and real public message delivery are used.
- Caller deadlines: ActorCLI transports the caller's remaining budget. Removed the unrelated per-tool2s version deadline; versions now share the declared build budget. Build/setup180s, runtime120s and cleanup15s are explicit outer scenario budgets, with no increases. The existing runner's TERM grace2s, fleet supervisor TERM2s, process escalation1.5s, terminal post-exit drain500ms and pipe WaitDelay are shutdown policies within their owners, not readiness success allowances.
- Cancellation ownership: build/provision direct-child cancellation was insufficient. Reused couchcore.OSProvisionIO instead of adding a second process-group runner (ARCH-DRY). Both old wrappers failed a marker/generation-controlled descendant regression; both now join cancellation with no running observed descendant. Provision environment/output separation and existing bounded diagnostics remain tested.
- Waits: replaced the paste30ms sleep with observed draft state before asserting no acceptance. Removed the100ms child scheduling assumption; the independent marker-controlled parent-cancel test proves started-child join. Remaining5/20/25/100ms loops are condition/mailbox cadences bounded by their caller; permanent mailbox exhaustion now has a typed error and readiness/native waits return it after one observation, rather than poll to timeout. The128 ordinary-request cap is unchanged, shutdown remains reserved.
- Diagnostics/claims: terminal feed errors now stop input/readiness explicitly. Native logs record cohort, receipt/body and restart elapsed evidence. Resize's ioctl is exercised here; native fixture SIGWINCH/repaint has its separate real-PTY regression, and this matrix does not claim every background pane repainted. Final membership remains exactly one steward plus three workers of the requested personality.

Next: finish combined checks and reader audit, commit/push, then native rerun; no matrix pass/full suite/close claimed.

### 2026-10-20 — sweep checks complete before clean-head native rerun

Independent whole helper race run passed testcouch7.367s, agentfixture2.161s and native fixture entrypoint3.427s. Targeted console/observer/keyboard checks passed; repository lint6.317s and artifactpath6.188s passed. Individually audited changed native consumer (real repository/runtime inputs, non_go_full_fallback), run/terminal/binary tests (only private local fixtures, local_or_literal_fixture), refreshed those four reader hashes, then make test-affected-inventory passed. Issue validation and whitespace checks passed. Commit/push this complete sweep before the next native run; no native/full-suite/close pass yet.

### 2026-10-20 — sweep rerun reaches restart, exposes result-retirement lifetime

Clean native head83f2c0b39939cb9e05b0f1b7011fd153f3d60004 passed exact Claude steward/three-worker personality membership at24.93s, concurrent queries, submitted receiptce13c3ff-8400-4dbb-923d-a85fbec60b8f and independent accepted body at26.04s, then physical detach. Restart readiness hit the unchanged128-request lifetime/result quota and the new typed refusal ended immediately. Full log /Volumes/TestScratch/tmp/pair-slot5/pair-744-native-nkp9k3yo/native.log SHA256c5b166c22a60d032247b8e7785b258e9bb108996bea617f7793eb42ae9904148. Cleanup succeeded; root removed and every owned generation absent.

The durable plan adds explicit reclamation of matched consumed query results, preserving128 unread/pending capacity and reserved shutdown. Active Exec IDs plus unread result files replace lifetime bookkeeping; admission reserves completion capacity. Counted retirement and unread-cap regressions precede the next native run. No timeout/cap increase and no full-matrix/close pass claimed.

### 2026-10-20 — bounded query retirement and concurrent publication guards

Fixture regression reproduced the old lifetime failure at cycle128. The correction completes256 serial consume/retire cycles while retaining at most one pending command and one unread result, with active Exec IDs bounded at4 and no lifetime tombstones/counter. Keep original command/result ID fences through explicit consumption, including stale directory-entry retirement races. ActorCLI retires its own matched result after reading it.

The capacity sweep also reproduced two atomic-publication races: a transient .publish file falsely made127 pending look full; ignoring it without serialized admission let32 concurrent writers admit3 and reach130. One owned per-actor advisory admission inode now serializes check/publication, and counters exclude atomic scratch names. Eight rounds of32 writers admit exactly one request to128. Unread128 still refuses ordinary work, reserved Stop still joins, stale shutdowns remain fenced, and no timeout/cap was increased. Agent fixture race run passed8.260s and native entrypoint3.456s; independent combined verification follows before commit/native rerun.

### 2026-10-20 — result reclamation checks complete

Independent whole helper races passed testcouch7.653s, agentfixture8.243s and native fixture entrypoint3.379s. Repository lint5.998s, artifactpath6.796s, individually audited run_test reader hash, make test-affected-inventory and whitespace checks all passed. Commit/push this bounded lifecycle correction before rerunning native; no full matrix/full suite/close pass claimed.

### 2026-10-20 — real detach ownership refusal after bounded lifecycle correction

Native head5aa2570ca7f9cb1be8771903df943618759ca492 passed exact Claude cohort and concurrent queries, then submitted receiptdf3c25fa-196c-4cc5-9d7a-82afca51f156 with independently accepted body at26.94s. Real switcher detach refused: leave couch: detach1-fixture-2: observe Pair session before retiring its incarnation: session ownership unresolved (remaining diagnostic clipped by the screen). Cleanup again confirmed all generations absent and removed the root. Full log /Volumes/TestScratch/tmp/pair-slot5/pair-744-native-xyneu9_b/native.log SHA2569030df57986142026b443b6bd53158a0f1a29a049bb378813f4a94f457bfb2e1. No complete matrix pass or product classification yet.

Added diagnostic observation only: detect the explicit leave refusal, use the public Ctrl+Y copy-notice action, and capture the owned host's bounded OSC52 effect after a sequence fence. It exposes the full product reason rather than waiting120s on a clipped error. A real-PTY regression proves owned OSC52 capture; no operator clipboard is touched by the private emulator. Registration failure also includes its current screen. Next native run is a deliberate diagnostic reproduction of this product-boundary refusal, not a blind pass retry.

### 2026-10-21 — diagnostic outcome and real composer conformance

Diagnostic headdcdffcea8f14b4465f025c3d40839968082e2a32 completed the full Claude path: exact four-actor membership, submitted receipt36fe0870-02c2-4288-9f27-e46f3e661744 and accepted body, detach/restart30.60s, all owned generations absent and root removed. The prior ownership refusal did not reproduce; filed unclaimed #761 with failed and diagnostic evidence, without claiming a product fix or registering a known flake. The Codex subcase registered all four genuine actors but its ordinary receipt never submitted before the unchanged deadline; all composers remained empty and accepted0. Its full cleanup passed. Combined log /Volumes/TestScratch/tmp/pair-slot5/pair-744-native-diagnostic-phd4gpjx/native.log SHA256c83886c74d8553e3a44fef3082f2d198410bcdb94baad557602a71b69aff2dcb.

The Codex fixture's synthetic shortcuts footer painted the reserved left edge beneath the prompt. Production peerComposerText correctly refused that surface despite the bold prompt itself being recognizable. Added42 tests through the actual production terminal model, peerComposerState and qualifiedComposerText for both personalities, idle/working/picker phases, empty/occupied/multiline/wrapped drafts and three geometries. Eighteen Codex cases were RED before removing only the optional synthetic footer; production recognizers are unchanged. Independent targeted composer race run passed1.960s, and affected-reader inventory passed (new test reads only pure fixture paint/model state; no repository scanner). No native run remained active after the capacity interruption; continued from these retained results, without duplicate fleet launch.

### 2026-10-21 — worker early exit and failure evidence before cleanup

Native3046a4affc8ff1eb9bed7f89bad733a890073d41 stopped at worker2 public recovery: Pair exited before registering (exit1), operation31a19fcf-29b3-441f-a4ea-22ea3f2f70ea, attemptstart-2e7663c377216793. Full log /Volumes/TestScratch/tmp/pair-slot5/pair-744-native-imxp9u7d/native.log SHA256f24995581e7d6379ebca8ca58ac67f0815d7cc6de453f4457361664a763feda4. Cleanup succeeded but removed the referenced private failures.json, preventing classification of the exit. No matrix pass or root cause claimed.

Cleanup now reads the existing bounded production LaunchFailureStore after every owned generation is absent, logging at most the newest four records and their bounded raw-output tails before root removal. It creates no retained diagnostic file or process. A real production-store regression pins newest-record/output preservation. Cancelled only this checkout's queued suite on3046a4af through sdlc local-check cancel because the diagnostic code changed; other runs were left intact. Requeue the final head after native qualification.

Independent race verification also caught an assertion-domain bug: the256-cycle test counted transient .publish hard links as extra retained results. Production counters already exclude those. The regression now deliberately holds such a scratch file and independently counts only final .json request/result files, preserving the same one-request/one-result assertion without a cap/timeout change. Checks in progress before the diagnostic rerun.

### 2026-10-21 — failure-evidence checks complete

Independent whole helper race checks passed testcouch11.950s, agentfixture11.894s and native fixture entrypoint3.541s. Repository lint13.976s, artifactpath12.467s, audited run_test reader update, affected-reader inventory and whitespace checks passed. Commit/push before the launch-failure diagnostic rerun.

### 2026-10-21 — full native qualification PASS; #747 live proof

Clean candidate5484106480a0123bd86913ea39273a339342a196 passed make test-couch-fleet in143.30s (Claude70.02s, Codex60.88s). Each personality used the genuine steward and three real non-hub workers, verified exact personalities/membership, three concurrent actor CLI calls, public receipt submission and independent accepted-body evidence, host resize, public switcher detach, supervisor restart with all four actors returned, and cleanup proving every owned generation absent/direct child+IO joined and both roots removed. Claude receipt62a0f057-9c21-48b0-846b-3e812c363b97; Codex receipt50108e66-b645-465c-8a65-cd72b1428896.

Full retained log: /Volumes/TestScratch/tmp/pair-slot5/pair-744-native-diagnostic-k_3lukca/native.log, SHA25653cbdc9a79cdcfe4ba579b28f86057494a546b7efa6be342b2deb315fcaf4880. Native tools: Zellij0.45.1, nvim0.11.7, Git2.54.0, installed SDLCbdee3f351ca51b0ad9ef65c4ae4e845b00a6a847 modified=false, Weave versiondev. This is the agreed **#747 post-landing live proof**, after integrating its landed PR498; no composite build or candidate override. #747 owner independently verified the log/checksum.

The earlier detach owner refusal remains recorded as unclaimed #761, without a product-fix or known-flake claim. The isolated worker early-exit attempt is also retained above and has not been given a root-cause claim. Current matrix is a complete native pass; required recorded suite and mandatory close review/merge remain outstanding. #722 stays parked until #744 lands.

### 2026-10-21 — close finding class repairs

BR-1 reproduced both service failures and the installed PTY refusal before migrating every real isolated-root consumer: stale_store service tests, installed CLI helper, opt-in stale-session native setup and shell wrapper. testpair.ShortRoot and the shell wrapper allocate owned short roots; no transport limit or live-parent rule changed. Real broker/wrapper listener confinement/cleanup and long-TMPDIR shell regressions pass; complete couchcmd99.641s, installed PTY13.852s and short-root race1.330s passed. Three existing reader hashes were individually audited; the new shell-reading test is non_go_full_fallback.

BR-2 reproduced active ActorCLI returning while its actual child remained alive, queued cancellation failing to settle at four occupied slots, and lack of absolute-deadline decoding. The approved protocol now cancels and acknowledges one immutable request, accounts queue time in its absolute deadline, suppresses canceled queued spawning, and joins actual process-group work before terminal publication/retirement. A pure admission decision derives transitions from existing facts rather than maintaining duplicate state. Queued/running/deadline/racing-result cases and129 cancel/success same-ID cycles cover no inherited marker or retained files. Complete helper/fixture race checks passed10.382s/12.030s and final ActorCLI race1.308s.

BR-3 corrects implemented value/function names, paint location and filesystem correlation classification in the current Core concepts tables with a dated Revision. BR-4 uses durablefile atomic snapshots, reports the first publication failure while retaining in-memory ownership, and still completes proved-clean teardown. Held-reader, failed-write/prior-evidence, and actual owned-process cleanup regressions passed; independent ownership race2.316s. Newly tracked fixture-local tests were individually audited as local_or_literal_fixture; no repository scanners or native-build inputs are hidden by those dispositions. A newly tracked short-root file exposed a stale suppression, removed before the final lint pass. Native final-head rerun, recorded suite and close remain next.

### 2026-10-21 — final native rerun blocked; parked for #761

After BR-1 through BR-4 repairs, clean pushed bfa769c1dec8ee80eabba775220d7084ae1c4983 failed make test-couch-fleet in48.08s during Claude detach. The genuine four-actor cohort and submitted receipt1b2825eb-0193-4b89-84f9-0fcce43274f8 with independent accepted body passed first. Public detach refused: observe Pair session before detach, session 📁1-3 ownership unresolved: invalid live pane JSON, failure operation-392b33c981ed0bbda69e1a8a9d6763ed. New bounded diagnostics preserved that precise product refusal before cleanup. All exact owned generations were absent, direct child/IO joined and root /Volumes/TestScratch/pc-1539241296 removed. Codex was not attempted after this failure.

Retained complete log /Volumes/TestScratch/tmp/pair-slot5/pair-744-native-final.nEqfD0/native.log SHA2561d4b86f5c114a28e372bb025ee7c5eaf30355f05d301802d0f1ecddc420ce7b5. Tool versions match the earlier PASS. This is not a final-head native pass and not a known-flake classification. Verified TL159f625e directs park744, claim761 from main, root-cause/fix/land its product bug, then integrate landed main into744 and rerun final matrix before close/landing. The queued bfa769c1 recorded suite was canceled before running because744 is parked for its prerequisite; other checkouts remained running. Claims retained; PR509 stays open. #722 and #739 wait. Next744 step: integrate only landed761, native matrix, recorded suite, repeat full close, merge.

### 2026-10-21 — resume after landed #761/#763/#764

TL3eba5889 dispatches the final qualification/landing sequence after train232 landed #761 and #766 through PR522. The #761 conductor exited0 and released this checkout to clean main-slot5 before switching. Integrated landed main7fabbeb93589b7c0c44a43fa699e83b09082bd5e, containing the native lookup recovery/raw evidence, per-reader certificates and exact fixture cleanup fix; no candidate override or stacked branch. Sole conflict is main's deletion of the old aggregate. Accepted deletion and regenerated exactly18 previously audited issue-owned reader certificates with make affected-readers, preserving classifications and verifying each merged source still has its audited hash. All other current-main certificates stay intact. New inventory and diff checks pass.

Preflight verification: whole testcouch/agentfixture/native-entrypoint races PASS11.092s/12.847s/3.587s; couchmessage25.174s, testpair11.077s, artifactpath10.516s, repolint8.910s and actual fixture composer conformance0.416s PASS. Commit/push integration before one clean-head make test-couch-fleet run for Claude and Codex, retaining the full log, head/tool versions, receipts/body proof and exact cleanup. Preserve the original143.30s successful #747 post-landing proof and its53cbdc9a79cdcfe4ba579b28f86057494a546b7efa6be342b2deb315fcaf4880 checksum plus the later failed final attempt; the new result does not erase either. Required recorded suite, full close and normal merge follow; #722 remains parked and #739 waits for main.

### 2026-10-21 — final native matrix PASS after prerequisite landing

Clean sourceb60b2f7b064fe5a8b57c38388f23a8e419f4600b passed make test-couch-fleet once: test107.35s (Claude50.57s, Codex49.62s), whole command108.881s, exit0, identical head and clean tree afterward. Each genuine steward plus3 workers passed concurrent ActorCLI, public ordinary message receipt/submitted and independent accepted-body proof, resize, physical detach, supervisor restart returning all4 actors, and cleanup proving every owned generation absent/direct child and IO joined. Claude receiptadb100ec-5ff6-4f12-bb1e-be1495ca96ef; Codexcb696b96-d463-4256-be0c-ac0a1232df5e. Both owned roots were removed. No native retry, timeout increase or remaining live test fleet.

Retained full log workshop/evidence/000744/final-post761-native.log SHA256d444a2fd8f8300ea4fc666ac9f3b43a819ee655546e68b50b7bd68a28e74b6c3, with exact-head command/result JSON beside it. Tools: Zellij0.45.1, nvim0.11.7, Git2.54.0, installed SDLC65859e6cf429aa9e672921a1c781361bc0902452 modified=false, Weave dev. The earlier #747 live proof remains explicitly retained at5484106480a0123bd86913ea39273a339342a196, log53cbdc9a79cdcfe4ba579b28f86057494a546b7efa6be342b2deb315fcaf4880; this post-BR/prerequisite run also preserves that coverage. Prior failed bfa769c1 final attempt stays recorded. Whitespace checks pass for code/docs; byte-exact raw terminal captures intentionally retain their native whitespace and were excluded from that formatting check. Required recorded suite and full close/merge remain next. #739 waits for landing, #722 stays parked.

## Revisions

### 2026-10-20 — #747 native-proof sequencing

Verified submitted TLd31f383f: #747 lands independently on its real emitted-profile executable-boundary proof. Then #744 integrates landed main and runs the native three-worker matrix, doubling as #747 live proof. No composite scratch candidate and no issue-branch stacking. #744 docs and guards proceed while native acceptance waits; no changed Done when or bypass.

### 2026-10-20 — approved base plan; PQ-1 isolation correction pending

Verified submitted TLcf61df78 approved the base native plan and change-code. Full-flow plan-quality then found that directory-wide dead-wrapper collection at message listener startup can touch foreign residue in the shared per-UID parent even with disjoint addresses. No implementation started. The durable plan now proposes extending existing COUCH_ISOLATED_ROOT authority to an owned message parent, with actual-listener foreign-residue regression and compatibility checks. This narrow transport seam corrects the original no-production-change assumption; request TL approval before code, then rerun the remembered plan gate.

### 2026-10-20 14:47 PDT — dispatched native prerequisite for #722

Verified submitted TL2d5e80a3: park #722 with claim, commit/push and next step; take #744 from main on its own branch, plan first, land it, then resume #722 over landed main. Native Zellij topology approved; scope limited to #722/#740/#741 needs. The original broader Spec/Done when/Plan are retained below; the active contract above narrows agent coverage to Claude/Codex and makes the initial native helper qualification its first live consumer. Qoder is a future adapter, not a promised result now.

### Original Spec — historical

One pair test helper package for a disposable Couch fleet:
- private `COUCH_ISOLATED_ROOT`, store, identity, Pair data, HOME, XDG, Zellij socket/config/runtime and sdlc queue
  roots, under TestScratch when present (TMPDIR otherwise); refuses empty, live or inherited paths before starting;
- a real supervisor plus N throwaway worker slots in real disposable checkouts, launched with genuine caller
  identity (no synthesized bindings);
- agent fixtures for **each agent kind** (Claude and Codex, Qoder when available) and a deterministic fake agent;
- exact cleanup: joins every child and removes only the processes, sessions and sockets it created;
- a guard test proving it can never see the live Couch's socket, store or HOME.
Existing live tests migrate to it as they're next touched; new ones must use it.

### Original Done when — historical

- The helper exists with the guard test, and pair#741 (or the first live test to land next) uses it.
- `atlas/` documents it as the one way to write a live test.

### Original Plan — historical

- [ ]

### 2026-10-20 — TL3d5e6782 approves transport isolation correction

The PQ-1 scope correction is approved, conditional on unset-isolation path/sweep compatibility, refusal before effects for invalid/overlong paths, and actual-listener foreign-residue coverage. Rerun the remembered full-flow plan gate. No code has been written.


### 2026-10-21 — separate pre-close checklist from gated publication

The original final Plan row read: “Document and run required checks, close and land. Resume #722 only after this issue is on main.” It mixed pre-close readiness with actions that cannot occur until the close gate itself completes. The active checklist now records the verified implementation/native work and prepared suite/close handoff. Required suite PASS, mandatory close review and normal merge remain mandatory, with pending/passed outcomes recorded in Log and SDLC; this bookkeeping correction neither waives a gate nor changes scope. #722 still resumes only over landed main.


### 2026-10-21 — close round1 revision

Close returned REWORK with BR-1 isolated-consumer socket-root coverage, BR-2 per-request cancellation, BR-3 pure/integration table and BR-4 ownership snapshot publication findings. The queued suite on6d584dc2 was canceled before running because code fixes invalidate it; other checkouts were left running. TL8aa5e124 approved the correlated-cancellation ordering in the durable plan and requires a new final-head native matrix, preserving the earlier PASS. Correct classifications and fix whole consumer/request/publication classes without changing socket limits, native topology or scenario scope. Required suite, repeated full close and ordinary merge remain outstanding.

### 2026-10-21 — BR-2 leader-first completion: proposed retained uncertainty

Close round2 verifies BR-1, BR-3 and BR-4 addressed and independently passes both native personalities, but keeps BR-2 Critical. Root regression TestCLILeaderExitCannotAcknowledgeLiveDescendant is RED: inherited-pipes returns terminal ErrWaitDelay in1.01s with descendant alive; closed-pipes returns success in0.01s with descendant alive. The regression terminates only its observed matching descendant generation afterward. The queued suite lease75766965 was canceled through SDLC before execution; no suite result is claimed. The final native PASS107.35s remains retained, but must be repeated after any lifecycle correction.

Proposed refinement, awaiting TL approval: preserve the current generation-qualified live-leader cancellation; never copy a raw numeric-PGID kill after its identity has disappeared. Observe exact group quiescence after every completion, including success, child error, cancellation and pipe-drain cutoff. A nonempty or unreadable group is a typed CleanupUncertain result, not terminal acknowledgment. RetireResult must refuse it before removing command/result/cancel files. ActorCLI surfaces explicit cleanup uncertainty, records the existing OwnedRun unknown/stuck outcome and refuses new fleet work; existing Close performs private-run census and generation-qualified descendant reconciliation. If cleanup cannot prove absence, preserve the owned root/manifest as it already does. This uses the existing bounded request slot and fleet owner, without another persistent store or timeout increase.

| Internal outcome | Visible result | Recovery | Possible loss |
|---|---|---|---|
| Leader complete and group proved empty | ordinary terminal result; retire correlated slot | continue | none |
| Leader exited, descendants live or group inspection unknown (pipes held or closed) | cleanup uncertain; correlated evidence retained; no new work | Close owned fleet | transient fixture work is stopped during exact cleanup |
| Leader live, deadline/cancel | existing generation-fenced stop, then group observation | retire only if quiescent; otherwise Close | interrupted CLI work |
| Owned cleanup cannot prove generations absent | cleanup failure with retained root/manifest | inspect evidence before another fleet | residue deliberately retained |

Simplest rejected alternative: copy OSProvisionIO's post-WaitDelay numeric group kill; it neither preserves incarnation authority after leader exit nor confirms absence. A new per-request anchor could supply that authority, but adds a process/control protocol when the existing fail-closed uncertainty/Close path already expresses this failure. Regression sweep: leader-first with inherited/closed pipes; ordinary success/error; deadline and explicit cancellation; unreadable group; no retirement/ID reuse of uncertain slot; refusal of new work; owned exact cleanup of a retained descendant. Keep process/IO joins bounded and errors visible, retain original native runs/#747 checksum, then rerun focused races, native matrix, recorded suite and full close. No implementation of this refinement begins until TL approves.
