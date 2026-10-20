---
id: 000622
status: working
deps: []
github_issue:
created: 2026-10-16
updated: 2026-10-16
estimate_hours: 3.55725
card_mirror: '292820b92517d7e8cac87c0a964a0560c4c86c0f' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-16T17:03:46-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
---

# Couch server spends ~1 core emulating every hosted client's output in a vt parser, visible or not

## Problem

Profiled by pair:2 for pair#618 (2026-10-16, macOS `sample` of the Couch server, 5 s): the Couch server burns about
one core continuously, and ~500 of its busy samples are terminal emulation: `ptychild.(*Child).ingest` →
`internal/terminal` + charmbracelet's vt `ansiparser`. Couch parses the full output stream of every hosted session
client (~27 `pair --couch-session-v1 resume` children) in a vt emulator, so every spinner redraw on every slot costs
Couch CPU all the time, whether or not anyone is looking at that slot.

## Spec

- Measure first: per hosted client, ingest bytes per second and CPU, at idle and during a busy turn; which consumers
  need the emulated screen (console view, peek, waiting/idle detection, resource monitor).
- Then the smallest change that keeps every consumer correct. Options to weigh: emulate only the slot(s) the console
  is showing and keep a cheap raw ring for the rest (replaying it when a slot is shown or peeked); or parse lazily
  (append raw bytes, emulate on demand). Idle/waiting detection must keep working for off-screen slots.
- No new background workers; bounded memory per client.
- M2 retains eager authority and optimizes only proven printable-ASCII grapheme
  operations. TL642b580a authorizes landing after suite/review and the qualified
  complete-workload CPU experiment; production acceptance follows the next
  coordinated Couch restart, with TL scheduling and this worker verifying it.

## Done when

- The restarted production supervisor is identified as running M1/M2, with its hosted-client census and omission status recorded.
- Production cumulative CPU and native samples are retained with workload/identity limits; live peek, waiting notices and operator-observed console rendering/selection continue to behave normally.
- M1/M2 semantic, race, recorded-suite and SHIP evidence remain recorded. Causal controlled CPU/byte acceptance belongs to the separately claimed #723, per TL44449afb; #622 does not claim a live workload-matched CPU reduction.

## Plan

- [x] M1 — land permanent per-hosted-client ingest-byte diagnostics through read-only resources; no parser-policy change.
- [x] M2 — after the combined restart, measure natural idle/busy rates and server CPU, choose the smallest correct optimization with TL approval, then implement and verify it.
- [x] Verify restarted supervisor/clients, record production CPU/sample, and confirm console/peek/waiting; hand controlled CPU acceptance to #723 before whole-issue close.


## Log


- 2026-10-20: closed — Deployment acceptance complete: loaded M1/M2 supervisor identified,20 matched hosted clients and zero ingress omissions, cumulative CPU/native samples retained without causal unmatched-workload claim; coherent live peeks and TL-checked waiting notices; operator console/selection attestation recorded via TL eef6af5c. Controlled replay acceptance moved by approved Revision to separately landed #723. Runtime already landed PR382/406 with semantic/race/suite/SHIP proof. Continuation suite passed9m58s on56fe4e03dcf861612fcc68cab9a42a95d86e0236; Git diff confirms all later changes workshop-only. Exclude actual from calibration: legacy suite attempts292295564ecd3c67d9068eb4ca0f98e6 and8891def18d42afa41f0cfc838ff1da7f never recorded exact queue endpoints; no timestamps/hours can be reconstructed. Current close explicitly directed --no-actual for these records, consistent with documented TL approval0b14e067. New authenticated TL dispatch a57e4713 authorizes this follow-up close/merge after ariadne#483.; review verdict: SHIP
### 2026-10-19 23:00 PDT — operator console attestation completes live acceptance

Canonical submitted TL receipt eef6af5c-d683-4dee-88b9-8988567ef1a7 relays the operator's console attestation: “nothing wrong observed since that restart or since the latest restart just now”. This is operator testimony, not an automated visual check. Together with the identified loaded M1/M2 supervisor, twenty matched hosted clients, retained production CPU/native profiles, coherent live peeks and TL-checked waiting notices recorded below, it completes this issue's deployment/consumer acceptance. No working slot was relaunched or driven for these observations. Controlled CPU/byte acceptance remains exclusively #723; its twelve retained observations do not turn the unmatched live CPU measurements into a causal comparison.

The close records actuals as unavailable because historical suite attempts 292295564ecd3c67d9068eb4ca0f98e6 and 8891def18d42afa41f0cfc838ff1da7f lack exact queue endpoints; elapsed totals cannot recover timestamps that were never recorded. No hours or queue timestamps are synthesized, and this close is excluded from velocity calibration. Runtime code already landed with M1/M2 semantic, race, recorded-suite and SHIP proof; this branch changes workshop acceptance documentation only.

Close attempt after recording the attestation refused before review: the gate requires the original canonical issue branch (its tip is absent), while this fresh follow-up is 000622-live-acceptance / PR474. The same merged-branch continuation gap previously blocked change-code. The issue remains working, not codecomplete; the text above describes the intended precise actuals exclusion, not a completed close. Reported to TL without branch aliases or gate bypasses. All deployment/consumer acceptance evidence is complete; SDLC follow-up-branch support remains the close blocker.



- 2026-10-17: closed M2 — Required suite on b498b3c91696570946b54e5c7932052cdb010d2c passed12m53s, log f015c1dc152dff569bab9043151aab45dfb3b8fa8acc8cc81068ef826a483732; later prose only. Full VT and terminal/ptychild races pass. Differential/generated semantics pass. Quiet28-case fixed-work ABBA complete ASCII redraw CPU -13.68 to -15.57%, mixed -2.84 to -6.30%, every candidate below both baselines. Native CI #636 initial failure retained, unchanged attempt2 passes. Atlas updated; TL owns portfolio. TL642b580a authorizes M2 landing before next restart; live server CPU/sample and console/peek/waiting remain pending whole-issue acceptance. TL0b14e067 approves no-actual: legacy suite attempt 29229556 lacks exact queue endpoints; excluded from calibration per ariadne#420.; review verdict: SHIP
### 2026-10-16
- 2026-10-16: closed M1 — M1 diagnostics only: recorded suite passed on aaf17569ea4005aa59502de3a3e47ecacb01dada (10m46s, log e13259c07f61); generated/concurrent counter and race checks, parser-lock barrier, production socket/JSON/text CLI conformance and exact encoded frame bounds pass. #626 retains the unrelated affected-package failure plus unchanged controls. README/atlas updated; TL owns project records. TL e6511b06 approves landing before the coordinated live restart; M2 CPU reduction and live metrics remain owed.; review verdict: SHIP

### 2026-10-16 — initial measurement and consumer audit

Claimed/started on fresh main b95df08cebf86cc71cd86a27dfdf2fdb96f1b91d. Slot-private Ariadne refresh advanced its clean checkout from 1fbc10d2 to e1bb1ba5 after start-plan's fleet refresh refused the unrelated untracked pensive; the pensive stayed untouched. No feature code changed.

Live supervisor PID71982, launched 14:57:39, has 23 direct hosted resume children at the new observation. Cumulative CPU rose 107:20.99 → 107:40.38 over 30.1086775s, or 64.40% of one core. Identity unchanged. This is a natural mixed-load observation, not an idle-fleet result. Baseline /tmp/pair622-live-baseline.json SHA256 956d9eb679e309e51ab0b874c0f436b812cb2ea95541bb87c50f9dec1c5bd7d4. Read-only 5s native sample at 17:06:26 retained /tmp/pair622-couch-71982-native.sample.txt SHA256 414d34db020424e9ccccfd9c69776a8fae33a4fc03393dea7b80c956ba0101aa. Its call graph includes Child.ingest and VT parsing; no percentage is inferred by dividing inclusive counts or summing overlapping stacks. The binary currently at the path reports 9fc035e718b1, but its mtime is later than server launch; that is not proof of the loaded image's revision.

Read-only consumer audit: selected console frame/history, mouse/key encoding, query replies, effects, resize and EOF need the hosted Endpoint authority. Off-screen waiting/picker notices and peek use wrapper tails/status; resource idle uses wrapper settled/picker/broker state; idle fading uses thread activity. These do not require the hosted screen. The raw diagnostic ring is bounded at 128KiB and discards arbitrary prefixes: existing TestEndpointRetainsFrameAndModesAfterDiagnosticRingEviction proves screen/modes predate its tail. Replaying that tail cannot safely replace the authoritative emulator. Lazy parsing must also preserve prompt queries, exactly-once effects, ordered geometry and EOF. Details and tradeoffs are in the durable plan.

No exact per-hosted-client ingress counter exists in this running binary. Existing capture could provide bytes but requires startup opt-in. TL instead directs permanent always-on atomic byte counters as M1, exposed read-only through resources/debug; optional emulator CPU timing only if cheap. They land before the #618/#617 combined restart, then measurement and optimization design follow. The posted direction's canonical sender/body matched receipt 9929f33e; its delivery outcome is indeterminate, even though this turn received the text. No capture restart was performed.


### 2026-10-16 — M1 implementation and targeted evidence

Plan-quality resolved PQ-1 CLEAN; change-code admitted at 242bd40216171712db036069ca5aae4f0c381366. Estimate-quality INFO calls the resources/check budget optimistic; 1.509h is cumulative M1 effort including initial shaping, not a remaining-work or whole-issue forecast. M2 remains unestimated until its measured design exists.

RED checks confirmed the missing counter, Console method and resources types/wiring before implementation. GREEN: fresh fake Child → Console → production resources handler → real socket → JSON/text CLI totals advance independently of an intentionally stale process sample; request leaves cache unchanged. Encoded maximum-detail response remains in MaxFrameBytes and reports hosted omissions without mutating its source. Clone/projection identity/isolation checks pass at 0/1/27/256 rows. Counter tests cover generated chunk partitions, eviction, later refusal/publication failure, independent incarnations and concurrent exact totals. Real Child observer barrier proves the diagnostic snapshot completes while the parser mutex is held. Targeted race command go test -race ./cmd/internal/couchcmd ./cmd/internal/couchtty ./cmd/internal/ptychild -run 'TestResourcesHosted|TestResourceHosted|TestResourceProjection|TestHostedIngest|TestChildIngest|TestResourceInventory' -count=1 passes all three packages. No parser policy, live restart or per-client CPU attribution changed.

Counter benchmark control: exact committed pre-counter baseline 242bd40216171712db036069ca5aae4f0c381366 and current working source ran the identical complete-ingest benchmark serially, GOMAXPROCS=3, GOFLAGS=-p=3, three 300ms repetitions. Median ns/op baseline→current: 1-byte 6170→3115, 64-byte 40004→18244, 4096-byte 1116835→959324. Baseline 1-byte spread 3584–9665ns/op and changing contention prevent credible incremental overhead attribution; apparent speedup is inconclusive. Raw logs and metadata retained in /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair622-ingest-bench.hqUsTx/. Temporary isolated baseline worktree removed by its owner; no peer runner was signalled. Source inspection confirms the incremental hot path is one atomic addition without new allocation/clock/native call. Counter race rerun after test-budget guards passed (1.229s).

### 2026-10-16 — PR and pending gate evidence

M1 candidate aaf17569ea4005aa59502de3a3e47ecacb01dada published as PR382, https://github.com/xianxu/pair/pull/382. Main integration check was already current. Required recorded suite is queued as sdlc local-check run --name suite on that pushed code head, local runner sdlc-local-2321322240; no suite pass or milestone acceptance claimed yet.

Affected-package run encountered TestWorkbenchPerformanceIdleUITick: activity worker did not finish, test50.43s/package101.425s. Unchanged targeted repeat on the same code head passed30.16s with read/decode accounting intact. This fixture creates no Child and does not request resources; relevant activity/runtime/fixture files are unchanged from pre-counter242bd402. Fixed30s test guard versus sequential20×30MB cold ledger probes is a demonstrated tight boundary, but the original scheduling/progress cause remains unproven. Filed unassigned #626 with both attempts and repeat log. No unrelated fixture/production change or blind full-suite retry was made; the first recorded suite remains queued. Remaining affected packages continue to completion.

### 2026-10-16 — affected-package census complete

The remaining affected packages completed: couchcore passed531.520s and artifactpath passed11.803s. The aggregate command exited1; its only failure was TestWorkbenchPerformanceIdleUITick (#626). Preserve it as a failed run, alongside the unchanged targeted pass, without calling the aggregate green. Full recorded suite is still queued.

### 2026-10-16 — recorded full suite passes

Required sdlc local-check run --name suite passed on pushed code head aaf17569ea4005aa59502de3a3e47ecacb01dada, execution10m46s after queue1671s. Recorded sdlc-local/suite; log /Users/xianxu/workspace/pair/.git/sdlc/local-checks/aaf17569ea4005aa59502de3a3e47ecacb01dada/suite.log, SHA256 prefix e13259c07f61. All shell/editor checks and Go packages completed. couchcmd51.455s, couchcore316.270s, couchtty45.657s, ptychild0.837s, wrapcmd49.269s. The earlier #626 affected-package failure remains retained; the unchanged full recorded run passes, including that activity test. No retry, timeout increase, test exclusion or production change was used. PR382 CI conformance/merge/local-gates checks also passed on the docs-only descendant. Preparing M1 boundary review, keeping M2 unchecked.

The two interim hub outcome reports a523d633 and20027db5 expired waiting for a recognized composer; those queue-state reports are now stale. A fresh final outcome will be sent after the M1 gates/landing; no peer keys or operator stage were used to force delivery.

### 2026-10-16 — M1 boundary SHIP

M1 milestone-close adopted measured1.21h and returned SHIP with no Critical, Important or Minor findings. Fresh reviewer independently passed targeted race/resource tests, complete ptychild tests and diff checks; all eight ARCH lenses pass. Evidence committed by the binary as53f3ee0de35a. M1 is checked; M2 remains unchecked and the issue working. Proceeding to the explicitly authorized partial landing through sdlc merge --yes. No whole-issue close, CPU reduction or live acceptance is claimed.

Clarification of #626 evidence:30.16s is the targeted test's total duration including fixture preparation, not a measured worker-only time. The original timeout's cause remains unproven; neither that duration nor host load establishes causality.

### 2026-10-19 — read-only post-restart production observation

Canonical TL dispatch06f1bbea authorizes observation after the21:01 restart, with no relaunches. Supervisor93340 launch21:01:29.823; hosted metadata starts21:01:30.504261. Loaded executable inode832400381 matches current disk inode, mtime19:13:37, clean build4b3105cb2d8d4b4a99ec29e8e7d91a3974de820a. Git ancestry proves M2 is included. Unlike the prior baseline, this measurement establishes the loaded image identity. Twenty matched hosted clients and zero ingest omissions/replacements in all snapshots; cached process-detail omissions are separate.

Main matched interval22:14:53.061172–22:16:14.860601 has81.799429s counters at0.775089MB/s. Adjacent cumulative CPU19:33.08→19:56.71 across81.917814s monotonic gives23.63 CPU seconds,28.845984% of one core. RSS854848→854896KiB; load1/5/15 at endpoints89.01/49.69/47.00→98.47/60.95/51.47. Natural mixed fleet, with concurrent build/suite/merge processes; server GOMAXPROCS and selected console are unknown. Exploratory prior counter window65.173589s at0.780600MB/s has only45.792033s CPU brackets (26.074405%); retain the20s skew rather than treating it as a paired workload interval.

The prior baseline was34.00/64.82% at1.098/1.616MB/s for22 matched clients; neither workload, byte mix nor client population is controlled across restarts. These lower absolute current CPU numbers do not establish a causal optimization gain. Native5s profiles at22:14:07 and22:14:53 show eager ingest/VT/capture/validation, plus GC/runtime and wait stacks; inclusive symbol counts are neither additive shares nor CPU percentages.

Live peeks for pair:1/5/7/8 and ariadne:0 preserve coherent Claude/Codex/Qoder tails and cursor/status. parley.nvim:0 explicitly reports recording-only/no connected wrapper; it is not counted as a successful live peek. TL receipt9be71c1b confirms accurate post-restart notices at about21:15 (parley.nvim:4 prompt),21:45 (ariadne:2 unanswered report),22:00 (ariadne:2 idle off resting branch), each checked by TL peek. Operator console-render/selection attestation is pending.

Raw evidence is local-only at /Volumes/TestScratch/tmp/pair-slot5/pair622-live.7rGWDf: resources0/1/2.json, cpu0/1/2.json, peek-start/end.json, server-a/b.sample.txt and derived.json. Profile SHA256s3c75121806ae7905a69aa24c050f0ce2919cb2494a4ac5d3bd8524bc8f646aa8 /811560883ad3a09ae80241d753a55bb553f4f9af7ac0637231c14ef73fcf21ad. Raw tails/profiles are not uploaded.

## Revisions

### 2026-10-16 — measurement milestone before optimization design

TL direction 9929f33e replaces the proposed capture choreography with permanent M1 ingress diagnostics. Original whole-issue CPU reduction and live consumer acceptance remain owed in M2; M1 does not claim them. One atomic byte increment per Child.ingest call; no per-chunk clock/syscall/worker or raw payload retention. CPU attribution uses the live server sample until a cheap accurate per-client CPU seam is established; wall elapsed time will not be labelled CPU. Full-flow milestone review is used because an explicit M1/M2 sequence lies outside the binary's quick-flow shell; the implementation target remains a small diagnostic diff.

### 2026-10-16 — M1 plan posted; baseline controls pass

Concrete M1 plan 044ec5dc278c07223a413253e32ba445e0a59fa9 sent to hub as 9c72573b-6098-4f22-aa21-833ce60f5e42; submitted confirmed. The message names fresh read-only counter snapshots, server/client identity, explicit wire omissions and omission of misleading/expensive CPU timing. Awaiting response before feature code. Existing authority/projection controls pass: go test ./cmd/internal/ptychild ./cmd/internal/couchcmd -run 'TestEndpointRetainsFrameAndModesAfterDiagnosticRingEviction|TestChildNotificationIsTypedOnceIndependentOfRawRetention|TestResourceProjection' -count=1 (ptychild 0.217s; couchcmd 0.356s). No production/test implementation changes yet.

### 2026-10-16 — M1 approved; plan-quality refinement

TL e6511b06-0db1-4dae-a695-ca1c364c46f5 (submitted) approves M1 as planned, explicitly without per-feed CPU timing and with full gates. Plan-quality round 1 raised only Important PQ-1: replace test-case inventories with function-named adversarial strategies. Refined Child.ingest/IngestBytes, Console.HostedIngestSnapshot, projection and request/CLI strategies with independent oracles and mechanical guards. No scope/mechanism change; no implementation started before gate admission.

### 2026-10-16 — M2 built; repeatability guard stops qualification

TL approval4f9a9b31 confirmed, canonical-branch recovery6d5eb120 applied. Plan-quality CLEAN; cumulative3.55725h estimate recorded via setter, estimate-quality INFO retains optimistic verification-budget caveat. Baseline qualification a9cf0d192590ab9195aecded9ba3f043dca8154c, candidate b721874da1b87dbc731b7b6fc6bb2da4e8bf1e4e. Both shortcuts preserve existing state and safety boundaries. Full VT module and targeted frame semantics pass; the independent oracle caught a non-distinguishing test-only Prepend mutant and was corrected to a real keycap continuation before production.

[Complete-ingest ABBA evidence](../plans/000622-m2-benchmark-evidence.md) shows no repeated complete-redraw benefit. B1 ingest4096 improves11.5%, but ASCII redraw is mixed and mixed/chunk64 regresses; B2 broadly slows by an order of magnitude while host load rises16.41→35.19. All runs retained; no causal scheduling attribution or selective exclusion. Approved stop/re-plan condition fired. Candidate held, no full suite/close/stage/landing/live acceptance attempted.

Proposed measurement-only revision brackets fixed identical work with owning-child cumulative user+system CPU outside the hot path, alongside wall time and ABBA order per workload. This needs TL approval before execution. Existing temporary benchmark worktree/binaries/logs remain invocation-owned for that decision; no runner remains and no process was signalled.

### 2026-10-16 22:51 PDT — approved qualification waiting for quiet machine

TL477ed170 approved one revised fixed-work CPU qualification, conditional on load below about8. The prepared runner's source/hash and once-only output guard are recorded in the durable plan. It is currently the owned execution session25768, running run-fixed-cpu.py in /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair622-m2-bench.AMTYLq; poll that session before starting anything else. It has launched no benchmark: load initially fell19.91→11, then rose and remained near50 through22:51. No fixed-cpu-once attempt directory exists yet. The runner waits30s between load checks and will run exactly the approved28 child cases once admitted. Read fixed-cpu-once/results.json and every retained log on completion; an existing attempt must never be silently repeated. No new candidate, full suite, close or live acceptance is admitted until these results are assessed. If complete-workload CPU does not improve repeatably, shelve these shortcuts as directed.

### 2026-10-16 23:39 PDT — partial qualification preserved; idle for relaunch

The owned runner25768 exited1 at22:59:28 after its load guard fired: two ingest1 control runs completed at starting load7.9238, then load8.3301 stopped the experiment. No complete ABBA or complete-ingest redraw workload was measured. Partial CPU/log evidence and hashes are appended to the benchmark report; no repeat or acceptance claim. No background runner, queued/running suite, test or merge remains. TL2e95b559's rolling relaunch request is verified; report idle for relaunch before another long wait. Candidate remains held; one permitted revised attempt is incomplete, and any further experimental disposition belongs to TL after relaunch. An initial couch receipt read returned command-not-found transiently; subsequent bare couch lookup and receipt verification succeeded, with no alternative messaging channel or binary rebuild.

### 2026-10-16 23:41 PDT — one rerun authorized; unrelated file preserved

Verified TLc283e6e7: original partial qualification lacked a quiet window; one fresh rerun is authorized under the same guards and shelving rule. Durable plan records the separate runner/hash and fixed-cpu-authorized-rerun output. Owned execution session94823 waits for60s continuously below load8; startup load83.35, no benchmark admitted. Preserve original fixed-cpu-once partial evidence. Poll session/results before starting anything else; no automatic second retry. The relaunch-blocking 10-13 pensive predates this item and is marked unrelated/preserve by saved continuity; authorship unconfirmed, so leave untouched. Hub notified with submitted receipt2d59eda1. Relaunch also waits for #618 M2.

### 2026-10-17 10:49 PDT — background qualification intact after Couch restart

Verified restart dispatchd7bc80ee. Runner94823 had completed normally at00:03 PDT:28/28 cases valid, all log hashes verified, exact iterations retained, one-minute load4.01–5.01. Complete ASCII redraw CPU reduced13.68–15.57%; mixed redraw2.84–6.30%, with both candidate CPU observations below both baseline observations for every case. Full results and limitations appended to the benchmark report. No lost background work and no rerun. Shelving condition does not fire. Races, full suite, reviews and coordinated live acceptance remain owed. Checkpoint then report idle for the previously requested relaunch before starting a long suite; preserve unrelated pensive.

### 2026-10-17 — integrated validation and known native CI flake

Integrated main without conflict, documented eager authority and TL642b580a's landing/live order, and opened PR406 on b498b3c91696570946b54e5c7932052cdb010d2c. Full VT module passed (VT0.696s, ansiparser0.159s); go test -race ./cmd/internal/terminal ./cmd/internal/ptychild -count=1 passed (64.259s/2.869s). Required recorded suite queued on that pushed head via make PAIR_GO_TRIMPATH=-trimpath test; do not start another suite or manually chase main while it waits.

Remote CI run39055625434 attempt1 failed warm recovery: list-panes returned no reply, immediate fixture query succeeded with live wrapper/source panes. Native terminal/nvim/notification tests and earlier launcher+park conformance passed. This is the established #636 family, with prior unchanged-source reproduction recorded there; #622 changes none of the recovery fixture/probe sources. Retained failed-step log ci-39055625434-failed.log under the benchmark evidence directory, SHA256b9cf86c1e16cddf0df865f64c925ad65b98036058a3d7fc30333a04b7c74f029. Linked #636 and notified its owner pair:6 (receiptdfe486c3); requested one unchanged failed-job rerun on the same head under flake policy. No local native diagnostic, skipped assertions, query retry or guard relaxation. Retain both CI attempts; further unrelated flakes on this head require hub disposition.

### 2026-10-17 — unchanged native CI rerun passed

Run39055625434 attempt2 completed successfully on the unchanged b498b3c91696570946b54e5c7932052cdb010d2c head, including native terminal and recovery conformance. Both attempts remain evidence; #636 was not repaired by this branch. Successful complete log ci-39055625434-attempt2.log retained under the benchmark directory (66046bytes, SHA2564c86f13f0d8771e1bd57a842e4814965e0333f4ce9f5a8691c64c3e9edd62cf3). Required local suite remains queued; its running handle is5106, worktree sdlc-local-1584895658. Do not duplicate it.

### 2026-10-17 — required recorded suite passed before M2 review

Recorded suite passed unchanged pushed code head b498b3c91696570946b54e5c7932052cdb010d2c: make PAIR_GO_TRIMPATH=-trimpath test, exit0,12m53s execution after644s machine-budget queue. Log saved under .git/sdlc/local-checks/b498b3c91696570946b54e5c7932052cdb010d2c/suite.log, recorded sdlc-local/suite success. Every later branch change is issue/plan prose only. All Go packages, editor/runtime/packaging/lifecycle checks passed. VT module and terminal/ptychild race proof are recorded above. Remote native CI attempt1 known #636 failure and unchanged attempt2 pass are retained. No new code since these checks; enter mandatory M2 milestone review, then normal landing under TL642b580a. Whole-issue live acceptance remains pending, not waived.

### 2026-10-17 — actuals exclusion approved for legacy queue evidence

M2 close refused before review because M1 integrated suite attempt292295564ecd3c67d9068eb4ca0f98e6 predates exact queue endpoints. Its elapsed/queue totals do not prove exact queue idle; no timestamps were synthesized and no evidence file changed. TL0b14e067 confirms known ariadne#420 migration behavior and explicitly approves --no-actual: legacy suite attempt29229556 lacks exact queue endpoints; excluded from calibration per ariadne#420. No evidence recovery required. Mandatory review, suite and remaining gates stay enabled.

## Estimate

Produced via brain/data/life/42shots/velocity/estimate-logic-v3.1.md against baseline-v3.1.md. Method A only; calibration is flagged stale by estimate-source, so provisional. Original M1 derivation was1.509h. The following cumulative derivation now includes approved M2 through live acceptance and closure; it is not a remaining-work estimate. M1's original primitives remain unchanged below.

Decomposition: issue-spec/audit uses design 0.6 ×0.5 (initial mixed-density dispatch) and impl 0.3 ×0.4. Three smaller-go-module extensions are counter, Console snapshot, and resources transport/projection/rendering: each design 0.2 ×0.2 and impl 0.5 ×0.4. Atlas docs use design 0.1 ×0.2 and impl 0.15 ×0.4. Milestone review uses design 0.1 ×0.2 and impl 0.5 ×0.4, including required checks. Existing sync/atomic, fake Child/Console and JSON budget helpers cover the stack; no greenfield/library discovery. Familiarity 1.0, thorough-plan design buffer 15%. Design subtotal 0.46; implementation subtotal 0.98; 0.46×1.15+0.98=1.509 hours.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.3 impl=0.12
item: smaller-go-module design=0.04 impl=0.2
item: smaller-go-module design=0.04 impl=0.2
item: smaller-go-module design=0.04 impl=0.2
item: atlas-docs design=0.02 impl=0.06
item: milestone-review design=0.02 impl=0.2
design-buffer: 0.15
item: issue-spec design=0.375 impl=0.12
item: smaller-go-module design=0.06 impl=0.2
item: smaller-go-module design=0.06 impl=0.2
item: smaller-go-module design=0.06 impl=0.2
item: atlas-docs design=0.02 impl=0.06
item: milestone-review design=0.04 impl=0.2
item: milestone-review design=0.04 impl=0.2
total: 3.55725
```

M2 addition: issue-spec/audit covers post-restart measurement, profile comparison and proposal (design0.75×0.5, implementation0.3×0.4). Three smaller-go-module extensions cover VT differential shortcut, frame validation differential shortcut, and reproducible complete-ingest benchmark qualification (each design0.3×0.2, implementation0.5×0.4). Existing ANSI/UAX29 library and real fake-terminal seams suffice; no novel library or external API. Atlas/docs add design0.1×0.2, implementation0.15×0.4. Two review primitives cover M2 milestone and whole-issue close/landing with full-suite and coordinated live acceptance (each design0.2×0.2, implementation0.5×0.4). Familiarity1.0, design buffer15%. M2 design0.755 and implementation1.18 yield2.04825h; cumulative design1.215×1.15 plus implementation2.16 yields3.55725h. Benchmark, review, suite and live execution consume ship wall-clock; no parallelism or operator-attention discount. Review/check time remains optimistic under machine contention; uncertain operator scheduling is not a new code mechanism.

## Revisions

### 2026-10-16 — M2 approved and cumulative estimate derived

TL approval4f9a9b31 confirmed; plan-quality CLEAN at change-code. Estimate-quality INFO correctly identifies that M1-only1.509h does not cover M2, so derive cumulative3.55725h before implementation. TL6d5eb120 supplied the canonical branch continuation: start-plan recreated the deleted M1 branch at fresh main, plan/measurement commit cherry-picked, ad-hoc branch removed. Landing awaits ariadne#418's matching fix. No review bypass.

### 2026-10-16 — post-restart measurements and M2 proposal

M1 PR382 landed; TL restart dispatch4306f6e3 verified submitted to this incarnation. Fresh M2 branch000622-hosted-ingest-m2 is based on origin/main a991ba8fb2b1e30b199013aebdcdab9d237d7ed9, avoiding reuse of the merged M1 branch. start-plan delivered architectural principles without an issue argument because its default branch stem would reuse M1. No production code changed.

[Live measurement and comparison](../plans/000622-live-measurement.md): server99807, 22 matched clients across75.354s and133.023s. Confirmed idle ariadne:0/parley.nvim:0 emit41–71kB/s; matched aggregate1.616/1.098MB/s. Cumulative server CPU104.16→153.46→198.65s over adjacent monotonic76.055/132.910s brackets (64.82%/34.00% of one core). Both populations changed, zero omitted; no idle-fleet or exact per-client CPU inference. Native5s samples, host contention, counter/CPU timestamp skew, failed initial broad peek and loaded-executable/disk inode mismatch are preserved. No input, process signal, rebuild, stage or restart was performed.

ARCH-DRY/ORDER: reject lossy hidden rings; exact lazy replay would require preserving prompt protocol/effect/geometry authority and still consume continuous output. Profile/source support a narrower eager candidate: printable ASCII handlePrint and validateCell segmentation shortcuts, preserving all general Unicode and provisional-cluster semantics. Durable plan appends differential/generated controls, baseline/candidate complete-ingest benchmarks, full gates and coordinated live acceptance. Await TL approval before change-code/code. No whole-issue performance claim or estimate revision yet.

### 2026-10-19 22:17 PDT — controlled replay replaces unmatched live comparison

TL2cfd2adc explicitly replaces the CPU reduction criterion with controlled A/B: one recorded few-minute real stream, fixed clients, median of three runs per build, at most four parallel processes; production CPU/sample and consumer conformance remain recorded. This changes the measurement method, not the required CPU benefit. Original Done when preserved:


- The Couch server's steady CPU with ~27 hosted clients drops by a measured amount (before/after `sample` and CPU%),
  with the console, `--peek` and waiting notices behaving as before (tests plus a live check).
- M2 may land before that production check under TL642b580a; its landing gate is
  qualified CPU reduction, semantic/race tests, required recorded suite and SHIP
  review. The live criterion remains pending and whole-issue closure waits for
  the next TL-coordinated restart; fixture timings do not satisfy live acceptance.

The controlled procedure is appended to the existing durable plan. No new optimization, fleet input, live capture enablement, restart or product state is authorized. Fresh documentation branch000622-live-acceptance is based on main; start-plan ran without an issue argument because the canonical #622 stem already has merged PRs. This avoids reusing those branches.

### 2026-10-19 22:21 PDT — CPU criterion moved to independently claimed #723

TL44449afb directs #622 closure on verified deployment/consumer checks and moves controlled performance acceptance to a new issue because PR406 already landed and SDLC cannot express its follow-up branch cleanly. Filed and claimed #723, “pair#622 CPU acceptance: controlled replay A/B”, with committed revision3a6c39219 as its Spec. No driver code or replay experiment was created on this branch. Existing temporary scratch contains only measurement observations and the new issue body. The SDLC gap belongs to TL, not this issue. Operator console/selection attestation remains pending.

The immediately prior Done when is preserved below, superseded by the active contract above. Its controlled criterion is retained in #723 rather than represented as satisfied:


- Controlled offline replay of one retained three-minute real scrollback stream, at fixed client count/geometry/chunking, reduces median CPU per byte over three pre-#622 runs versus three current-build runs; no more than four measurement processes run concurrently.
- Production CPU and native samples are recorded with workload/identity limits; console, live peek and waiting continue to behave normally after the TL-coordinated restart.
- M1/M2 semantic, race, recorded-suite and SHIP evidence remain the implementation gates; the controlled replay supplies causal performance acceptance rather than attributing unmatched fleet CPU to #622.
