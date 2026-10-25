# Historical #865 capture before measured M2

Source: preserved commit01bdfbe72710456ee1170dd54c5b86deba50f3e5. This is historical evidence, not the active implementation contract.

````markdown
---
id: 000865
status: working
deps: []
github_issue:
created: 2026-10-24
updated: 2026-10-24
estimate_hours: 2.92325
card_mirror: 'c2760f428b48d58b38fcb8ca4965925094a3ca17' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-24T02:51:38-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:9
    worktree: /Users/xianxu/workspace/worktree/pair-slot9/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: inferred}
---

# Live Couch sustains ~1.3 cores: attribute and bound it

## Problem

**The live Couch spends about 1.2 cores continuously, mostly in the Chrome cast recorder (ops:0, 10-24 02:15).**
Measured on the operator's Couch (`~/workspace/pair/bin/couch`, up 5.5 h): 115–140% CPU sustained over several
samples, while `fseventsd` also ran at 120–140%. The machine's load average was ~22 on 12 cores, and ariadne's
suite shards ran ~1100 s instead of ~500 s, so Couch's share directly slows every recorded suite.

A 3-second `sample` of Couch, top of stack (Couch frames): `webcast/recorder.LowRes`, `image/png` decoding
(`readImagePass`, `filterPaeth`), `image/draw.DrawMask`, `image.(*NRGBA).SetRGBA64`, plus terminal emulation
(`vt.(*Emulator).handleGrapheme`, `terminal.plainText`, `ptychild.ingest`) and GC (`scanObject`). The Chrome cast of
echo's board was left recording; that page repaints every 100–200 ms (pair#849's finding).

Find the cause and bound it: e.g. decoding and downsampling every captured frame even when nothing changed, or
recording at full rate with no viewer. Also check whether Couch's writes drive the `fseventsd` load.

## Spec

Measure the sustained live Couch cost before choosing an optimization. The original recorder attribution was not supported by the denser sample: image functions were about3.4% of counted symbolized leaf samples, while GC/memory, terminal and JSON were larger. The current cast also differs from the originally observed echo board. Preserve these limits and do not infer fseventsd causality.

M1 adds launch-only COUCH_PROFILE_DIR, disabled by default: one finite CPU window plus allocation snapshots and build identity, private local files, no HTTP listener. Profile an isolated disposable Couch through the existing native Claude/Codex harness first. Only the operator may approve relaunching live Couch; TL asks only if disposable workloads fail to reproduce.

After attribution, M2 returns to TL with the measured owner, workload, growth driver and budget before optimization code. Deliver the actual cost correction, not merely the diagnostic switch. The canonical plan is [000865-live-couch-cpu-plan.md](../plans/000865-live-couch-cpu-plan.md); initial evidence is [measurement summary](../evidence/000865/initial-measurements.md).

## Done when

- CPU and allocation profiles identify the owner under a pinned executable and described workload, with disposable native controls first and measurement limits recorded.
- Profiling is off by default, launch-only, finite, private, non-clobbering and lexically joined; deterministic cancellation/IO/ownership controls and real pprof parsing qualify it.
- A TL-approved M2 correction meets a stated CPU/work budget for the measured workload, with counted scaling regression and before/after evidence.
- The fseventsd question is answered only to the extent supported by measurement; unknown causality stays explicit.
- Operator live relaunch requires their separate approval; no approval is inferred from profiling-plan acceptance.

## Plan

- [ ] M1 — Launch-only bounded profiling, deterministic controls, disposable attribution and evidence; mandatory review.
- [ ] M2 — Append measured owner/workload/budget design, obtain TL approval, implement and verify the cost bound; mandatory review.

## Log

### 2026-10-24

### 2026-10-24 — measurement and approved M1

Read-only attribution while #860 held this checkout measured135.69% CPU over25.00s, five PNGs and180.4KB/s whole-process physical writes. This does not attribute physical writes to the recorder or fseventsd. Matching-binary Console control passed30.17s; CPU profile13.78s/30.28s, sampled cumulative allocation10,396.63MB (test-host and resize/replacement overhead included). Two existing terminal benchmarks also passed; component costs are not live-call-rate evidence.

TL f34c93f4-ed6c-49cc-8ea7-bcc6a7f34e2d respecified attribution first; d515e211-98ad-4923-8896-0b50468764e0 approved M1, disposable first, M2 approval before optimization. #860 landed PR582/train350; #865 starts from refreshed main, without stacking. #863 landed109986901f7410e0120135121f9e906054603634 restores the shared explicit-hub harness; #866 native startup and #867 Codex idle remain known limits, not qualifications. #868 was briefly queued then withdrawn as pre-#825 stale-head evidence; no claim was made. Both claims #860/#865 overlapped while the former awaited landing.

ARCH-DRY uses the existing Runtime/Getenv, Console notice and testcouch owners; ARCH-ORDER requires profile acquisition before Stop ownership and joined cancellation. Source inspection found Go StopCPUProfile cannot report its writer failure, so the bounded output writer retains first failure.

## Revisions

- 2026-10-24 07:12 PDT — TL revised the original recorder hypothesis after the denser measurement: original filing is retained under Problem; active Spec/Done when now require attribution before optimization. M1 approved as above; M2 remains a measured design gate. No production code yet.

### 2026-10-24 — plan gate round1

Plan-quality returned PQ-1 explicit-transition-ownership and PQ-2 test-strategy-compression. Added the pure StepProfile lifecycle and single serialized effect interpreter, including cancelled successful acquisition requiring release; compressed prose case inventories into one adversarial strategy per risky function. No implementation admitted or written. No change to TL-approved public M1 behavior.

## Estimate

Produced via brain/data/life/42shots/velocity/estimate-logic-v3.1.md against baseline-v3.1.md (calibration marked stale by estimate-source). Method A primitives plus Method B for the unresolved M2 design. These are ship wall-clock hours, not operator attention.

M1 lifecycle is one greenfield single-concern module: table midpoint design1.25, halved for available runtime/pprof, then thorough-spec x0.2 gives0.125; midpoint implementation0.6 x0.4 gives0.24. Runtime/file integration: table midpoint design2 x0.2=0.4, implementation1 x0.4=0.4. Existing native controls: smaller-module upper design0.3 x0.2=0.06, implementation0.5 x0.4=0.2. One runtime/native conformance discovery budget uses0.6 x0.4=0.24. Two genuine review boundaries use0.4 x0.4=0.16 each. Atlas work uses design0.1 x0.2=0.02 and implementation0.1 x0.4=0.04.

M2 has three unresolved decisions (measured owner, minimal transition correction, workload/budget regression), Method B3 x0.15=0.45 design, code-heavy x1.5 x0.4=0.27 implementation. That is a provisional allowance, not approval to implement an unknown optimization. Revisit after the M1 measurement and TL approval; append any scope/estimate revision rather than fit the work to this allowance. Familiarity1.0; thorough-plan design buffer15%. Sum design1.055, implementation1.71, total2.92325.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: greenfield-go-module design=0.125 impl=0.24
item: api-integration design=0.4 impl=0.4
item: smaller-go-module design=0.06 impl=0.2
item: real-api-discovery design=0 impl=0.24
item: method-b-decisions design=0.45 impl=0.27
item: milestone-review design=0 impl=0.16
item: milestone-review design=0 impl=0.16
item: atlas-docs design=0.02 impl=0.04
design-buffer: 0.15
total: 2.92325
```

### 2026-10-24 — plan gate round2

CLEAN/high: PQ-1 and PQ-2 addressed, no new findings. Derived estimate only after this clearance. The M2 allowance remains provisional pending measurement and TL's design approval.

### 2026-10-24 — admitted design and pure lifecycle

change-code completed exit0 at93b8d5593: plan-quality CLEAN, estimate-quality INFO. Estimate notes question experimental allowance and provisional M2 sizing; re-derive when the measured M2 design is known. First TDD unit: TestProfileLifecycle tests failed on absent model/event APIs (/tmp/pair865-model-red.log), then passed0.380s after implementation (/tmp/pair865-model-green.log). Pure StepProfile is isolated in profiling_model.go. Exhaustive event sequences independently count acquisition/release and require joined finished ownership; no production runtime/file adapter exists yet. No recorded check or merge runner is active.

### 2026-10-24 — M1 adapter checkpoint before native collection

Adapter and launch missing APIs failed first (/tmp/pair865-adapter-red.log, /tmp/pair865-launch-red.log); focused adapter/launch controls passed0.397s. Runtime conformance used the real profiler and all three outputs parsed with real go tool pprof, PASS1.480s. Lifecycle/output/launch controls race count20 PASS1.895s; child opt-in leak was independently RED0.523s then corrected in singleton.go, expanded controls race count20 PASS2.636s. The package race run passed204.812s (compiled before the final child-environment line; that line is covered by expanded race controls).

Audited profiling.go and profiling_model.go: only in-memory state and explicit diagnostic outputs, no Pair artifact families; artifact source coverage RED1.856s on the two missing classifications, then GREEN after individual manifest classification. Repolint PASS8.625s before the child-env addition. Reader audit found only singleton_runtime_test.go changed: existing temporary root/child output inputs are unchanged; the delta sets one literal env key and checks one literal child output field. Re-recorded only that local_or_literal_fixture certificate, then inventory PASS.

Native control was implemented as a bounded independent test task by native_profile_control; root reviewed the file. It reuses shared Build/New/Prepare/Start/StartWorker/Resize/SetState/Close/AssertCleaned, retains one matching executable and bounded outputs, and stops its matrix on any failed case. Compile-only go test -c passed; no native claim yet. Planned invocation uses PAIR_NATIVE_COUCH_PROFILE=1, a fresh explicit retained-evidence directory, TestDisposableCouchProfile and a15m watchdog. Quiet/busy controls have cast off and no Chrome coverage; actual command timings go in the manifest. This is collection, not the M2 acceptance budget. No recorded pre-close/check/merge runner is active.

The staged native consumer required its own reader audit. It invokes testcouch.Build's selected checkout/build scripts and native binaries as well as temporary profile outputs, so it is recorded non_go_full_fallback rather than treated as a literal-only fixture. Its added-reader refusal was observed before the precise certificate was recorded. Final staged inventory passed. Native retained evidence root is /Volumes/TestScratch/tmp/pair-slot9/pair865-native.f2WNZr/evidence (new leaf).

### 2026-10-24 — negative native result and scaled revision

Native matrix PASS415.469s at88c6b3288e1ac5dadfdf707b36b776e9621c5c13, exact binary and all profile files retained; per-case CPU1.20–2.22% of a core and all cleanup proofs pass. [Summary](../evidence/000865/native-terminal-controls.md). Final source guards passed repolint15.995s/artifactpath12.526s. This is not the sustained live-cost reproduction.

TL01a28fc1-4c1c-41ea-bed0-0ac980b58d11 asks for N1/5/10/20 steady redraw, long scrollback and same-Couch age. Planned the existing-owner extension and received approval9f0916d7-6ff1-4cc4-abde-6e1745f3a52a before new code. Added the complete M1 scaled-control revision to the plan. Native/Couch profiling remains disposable; no optimization or operator relaunch approved. No test or workflow runner is active at this checkpoint.

### 2026-10-24 — scaled-control implementation checkpoint

Implemented the approved M1 extension through three bounded existing-owner tasks: shared0..19 worker validation/current-supervisor identity and workload client; pure workload/Run integration; scaled native consumer. All preserve the existing production actor/mailbox/CLI/output limits and exact-generation cleanup. No optimization code or operator fleet modification. The native consumer selects one personality initially and retains the entire cohort, actual acknowledged workload counters, profiles, external counter windows and cleanup proof.

Workload RED controls covered missing API, duplicate completion, early clock signal, oversized output and initial prefill geometry. Final held-clock/Run workload controls and race count10 passed18.119s. Shared owner/client targeted controls passed5.922s and initial race8.538s. Root conformance caught a separate real regression: ASCII spinner patches destroyed Codex Working recognition (RED0.386s); alternating existing accepted bullet markers passed real wrapper Working/composer/cursor controls0.433s. Prefill is exactly2000 lines of118 printable columns plus CRLF (240000 bytes), below256KiB with Working repaint.

Root combined focused run passed agentfixture2.689s, fixture command0.256s, testcouch0.788s and wrapcmd0.484s. Artifact source guard passed28.534s after auditing workload.go as no artifact paths. Repolint caught a new fixed sleep in the client observation test: removed the timing assumption, retaining receipt/counter monotonicity while controlled-clock Run tests establish tick production. Corrected catalog check passed8.395s. Client/identity/options/provisioning race recheck count3 passed14.926s (/tmp/pair865-scale-client-race.log).

Reader inventory audit: new scaled consumer compiles the selected checkout and invokes its build/native assets, so non_go_full_fallback. fleet_test adds NewOwned refusal coverage and requires the same conservative native-helper fallback even though its invalid options refuse allocation. provision_test still reads only its generated temporary Git topology and literal fixture manifest; local_or_literal_fixture retained. Recorded only these three certificates, then inventory passed. No recorded pre-close, milestone review, merge runner or native fleet is active. Next: finish the client race result, commit clean harness/docs/certificates, then run the approved Claude scaled matrix from that clean head with a fresh retained evidence leaf and45-minute watchdog.

### 2026-10-24 — resumed scaled collection admission

Recovered with sdlc state on the clean issue branch after continuation424ecac0ea56c6af0eeca557c617bd5466941166. The approved scaled design and current Log agree: Claude-only disposable qualification, no optimization or live relaunch. Fresh existing parent /Volumes/TestScratch/tmp/pair865-scale.HgmR9S was created with mktemp and verified absolute/nonempty; evidence is its new leaf /Volumes/TestScratch/tmp/pair865-scale.HgmR9S/evidence. Commit this invocation before collection so the native build has source-clean compiled identity.

Exact invocation: `env TMPDIR=/Volumes/TestScratch/tmp GOTMPDIR=/Volumes/TestScratch/tmp PAIR_NATIVE_COUCH_PROFILE_SCALE=1 PAIR_COUCH_PROFILE_EVIDENCE=/Volumes/TestScratch/tmp/pair865-scale.HgmR9S/evidence go test ./cmd/couch -run '^TestDisposableCouchProfileScale$/claude' -count=1 -v -timeout=45m > /tmp/pair865-scale-collection.log 2>&1`. Monitor at most60s apart; the matrix stops on a failed subcase and the harness owns exact-generation cleanup. Preserve partial manifests and cleanup proof before any later run. No recorded check or merge runner started.

### 2026-10-24 — scaled run stopped; honest partial outcomes

Claude scaled matrix failed514.825s at the admitted clean candidate. N1 valid3.576% external CPU/30s, writes0; N5 subtest passed4.708%/20480bytes but INVALID steady-scale qualification: workers1–4 stopped96–97ticks at~9.7s, before profile. N10 failed worker5 reboot/caller steward positive-list authority during setup; no profiled supervisor or workload was created. N20/history/age skipped by existing fail-stop. All attempted fleets positively cleaned every exact generation, direct child and IO joined; manifests cleanup_proved=true without errors. No native runner remains and no retry was made. [Partial summary](../evidence/000865/scaled-claude-partial.md); original log copied to retained evidence/collection.log before scratch expires.

TL deliverycd2e1c48-72b3-4798-815f-b4536f6b4c76 authorizes investigating/fixing early worker waveform termination, verifying actual ticks before capture, classifying the captured startup failure and filing separately if a real scale defect, then measuring up to the largest reliably starting N. Two read-only bounded subagents investigate independent mechanisms. TL deliverya3f5541b-bcd7-4b7e-bfab-db32abed7e36 separately requests a new unclaimed profiler extraction item for pair:7 to land from main; #865 retains measurement/optimization. No live relaunch or M2 optimization approved.

### 2026-10-24 — TL parks synthetic controls pending live profile

Read-only diagnosis found no15-minute workload expiry: stopped differs from expired, and Run ends redraw when genuine stdin submission changes Working to Idle. A fresh supervision coordinator emits startupResendPrompt once per worker on its15s watcher tick; reattach08:18:23.809 plus delivery matches worker stops08:18:39.40. Hub/steward is excluded. This is a strong producer/timing inference; exact accepted payloads were removed with native roots, so do not call it a recovered payload proof. Genuine submission semantics should remain unchanged. The collector's weak nonzero-ticks/finish check accepted early stops; no harness repair was implemented.

N10 classification: worker5 reboot was admitted; later operation-status polling rejected steward positive-list authentication. Initial warming refusal was safely retried before admission. Actual session loss versus listing projection remains unresolved; bounded logs omit the failing steward and raw listing. Filed this distinct symptom unclaimed as pair#872 (workshop/issues/000872-reboot-receipt-caller-positive-list.md on origin/main), without assigning #866's panic root cause.

Filed profiler extraction unclaimed as pair#871 (workshop/issues/000871-couch-opt-in-launch-profiler.md on origin/main), delivered selective file/hunk seam to pair:7; receipt67bfdc60-513e-4187-8d40-70c38e0f0f2f submitted. No later production profiler fixes exist beyond the original a40b597e/88c6b328 pair. #871 owns landing the finite diagnostic independently, not the CPU correction.

TL delivery8443c3a7-e008-4e83-be76-433b6d4e37ff overrides the proposed harness repair: STOP additional measurement-harness work and park #865, keeping its claim. Wait for an operator-approved restart after #871 lands and a real live profile before resuming attribution/M2. No new workload observation action, tick barriers, retry or live relaunch was implemented. Both bounded investigations completed read-only. All native fleets are positively cleaned; no check/review/merge runner is active. M1 review and M2 optimization remain outstanding. Next active dispatch is pair#869, separate main-based branch; #865 branch preserves the complete parked work.

````
