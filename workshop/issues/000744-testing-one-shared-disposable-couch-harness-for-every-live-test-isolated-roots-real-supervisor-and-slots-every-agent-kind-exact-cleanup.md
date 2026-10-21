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
- [ ] Compose real provisioning/native runtime, then qualify the first shared-helper consumer and audit source-reader/artifact ownership.
- [ ] Document and run required checks, close and land. Resume #722 only after this issue is on main.

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
