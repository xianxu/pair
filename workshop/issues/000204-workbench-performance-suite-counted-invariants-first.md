---
id: 000204
status: working
deps: []
target: workbench-latency
github_issue:
created: 2026-09-06
updated: 2026-10-15
estimate_hours: 2.246
card_mirror: '7cc0859f79dbdb9ba9d4a108cb6f2a7face19259' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-15T05:25:29-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
---

# workbench performance suite: counted invariants first

## Problem

Three interactive-latency defects (`#201`, `#202`, `#203`) shipped and survived
for months. None would have been caught by review, and none had a failing test —
because nothing in the tree measures what an interaction costs.

The operator's account of how they were found is the requirement: *"I didn't ask
for perf test as I thought couch can't be that slow."* Intuition about a
multi-process chain is unreliable — a keystroke crosses 8–10 process wake-ups and
each hop reads like a function call at its call site. Absent measurement, the
first signal is an operator saying typing feels slow, months late.

**The trap this issue must avoid is a timing suite.** Every wall-clock number
taken during the debugging session moved 2–8× with ambient activity:

| measurement | busy host | quiet host |
|---|---|---|
| `zellij action` round-trip | 145 ms (max 467) | 17.6 ms |
| `alt+Return` | ~290–930 ms | 35 ms |
| process wake-up delay | 4.92 ms | 2.50 ms |

A CI job asserting thresholds on those would flake within a month and be muted.
Worse, the session's one *controlled* experiment returned a confident false
negative — 12 CPU burners moved wake-up delay 2.51 → 2.21 ms, i.e. nothing —
because steady CPU burn does not represent a real build's process-spawn storm.
Reproducible and wrong.

**The counted facts from the same session held under every condition**, and each
names its defect directly rather than a symptom:

| | invariant | today |
|---|---|---|
| `#201` | `alt+Return` performs **one** zellij round-trip | two |
| `#202` | the agent span file is read **once per change** | once per keystroke |
| `#203` | build fan-out is bounded by **cores**, not sessions | sessions × cores |
| `#209` | a thread/tab switch issues a **repaint request**, not only a replay write | landed 2026-09-09 |

`#209`'s row is the fourth, and it arrives with the operating envelope
`ARCH-CONSTRAINTS` asks for rather than a prose assurance. The request is a
SIGWINCH nudge (shrink one row, settle, restore), and `cmd/probes/zellijrepaint`
measured what it costs and what it needs on zellij 0.44.3 / macOS:

| | measured |
|---|---|
| bytes zellij re-renders per nudge, single pane | 6.6 KB back-to-back, 19.3 KB after a 1.5 s gap |
| repaint rate with NO settle between the two ioctls | **6 of 12 runs** — a coin flip |
| repaint rate with a 1 ms settle | 5 of 5 |
| nudges per switch | 1 |
| event-loop time per nudge | **0** — the settle runs off the caller's goroutine |
| in-flight nudges per child | 1; a request arriving during one is dropped |

The first two rows are the ones worth a tier-1 count. The byte figure is a
single-pane session and couch runs 10+ panes, so it is a floor rather than the
number; and the coin-flip row is why the count must be of a nudge that ACTUALLY
repaints, not of a `Resize` call — a counted invariant over the call would have
been green for a fix that worked half the time.

The last two rows exist because the envelope was stated once and then changed
underneath itself. "Two SIGWINCHes cost one extra repaint" was costed when the
nudge was two ioctls; adding the 20 ms settle made it 20 ms of an event loop per
switch, which held key-repeat turns into a proportional stall. Moving the settle
off the caller and dropping a request that arrives during one takes it back to
zero. **An envelope stated for one event has to be restated when the per-event
cost changes** — that is the rule this row is here to make hard to skip.

## Spec

Build the three counted Couch release invariants from the 10-14 Revision through their actual consumers, reusing #559's read/decoder counters and #218's process-reader accounting. Counts run in the normal suite; no timing assertions or operator environment access.

- Idle UI activity: every unchanged completed pass over20 owners with valid30MB ledgers reads and decodes zero historical ledger/catalog/transcript body bytes.
- Peer messaging: every confirmed unchanged request reads and decodes zero session-index/historical evidence body bytes at that20-owner scale. Fresh PID/readiness/process-generation evidence remains required and separately asserted.
- Selected Couch attach: at most one full process-table enumeration through handoff (parent0 + helper1), successful exact attachment and zero creates. Independently scheduled inventory and subsequent client lifetime/cleanup are outside that request boundary.
- Other-path ratchets: standalone attach<=2, continuation fallthrough<=3, owner pass<=1+2N full enumerations for N Probe+Revalidate callers. Named snapshots still count as full enumeration; targeted PID reads do not. Keep all safety fences; lowering these budgets belongs to the follow-up.

Cold and changed-owner controls must prove counters are connected; each warm operation must independently be zero; temporary counterfactuals must demonstrate each consumer/budget regression can fail. Share fixture/counter support and production adapters instead of copying their behavior. Document reproducible commands and extension/acceptance guidance. Earlier #201/#202/#203 and nonblocking tier2 goals move to #566; attachment scan reduction is a separate approved follow-up.

## Done when

- Three named consumer-level counted invariants run in the normal suite: unchanged idle UI pass and peer-message requests at20 owners x30MB have zero historical evidence read and decoder-input bytes, and selected Couch attach uses<=1 full scan (parent0/helper1).
- Standalone<=2, continuation<=3 and owner-pass<=1+2N count ratchets run normally and preserve successful behavior and authority fencing.
- Cold/mutated controls observe positive work; unchanged recovery is zero; independent counterfactuals fail the named invariant without weakening security checks.
- The normal full suite and applicable targeted race tests pass; no timing threshold or host-load manipulation is required.
- The extension guide/runbook names count boundaries and reproducible commands, links the workbench-latency target, and keeps live CPU acceptance separate from CI.
- Earlier goals are preserved in unassigned #566 and #567; no deferred goal is claimed delivered here.

## Plan

- [x] Extract/reuse #559 fixture and independent read/decode counters, preserving cache invalidation/security controls.
- [x] Add actual Console activity and socket-message consumer regressions with cold/mutation/warm/counterfactual controls.
- [x] Add selected Couch attach strict budget and standalone/continuation/owner-pass ratchets at the process-reader boundary.
- [x] Document release commands, count boundaries, extension pattern and separate live CPU acceptance; preserve released follow-up backlog.
- [x] Run targeted/race and recorded full suite; close through review, merge main and land on SHIP via the binary checkpoints.

Durable plan: workshop/plans/000204-workbench-performance-suite-counted-invariants-first-plan.md.

## Log

### 2026-09-06

Filed out of the session that produced `#201`/`#202`/`#203`. The suite's design
is downstream of that session's own failures rather than of a methodology
preference: three of its wall-clock findings were off by 2–8×, one controlled
experiment produced a confident null from an unrepresentative load shape, and one
promising code-read hypothesis (`#202` as the cause of typing lag) died to a
0.97 ms benchmark. The counted facts were the only ones that survived contact
with a different machine state.

Governed by `workshop/targets/workbench-latency.md`, written alongside this
issue, and by `ariadne#216`, which amends `ARCH-CONSTRAINTS` so a measured basis
must name the probe that reproduces it.

## Revisions

- 2026-10-14 — add (operator: "think of how to catch those issues before it's released"): a fourth case shipped today. After pair#528 M1, Couch burned 40–100% CPU at idle (pair#559) because every UI activity tick re-decoded each row's launch ledger (15–33 MB), catalog and transcripts, and every peer message re-decoded session-name indexes. Review and the full suite passed; the operator found it through timeouts. Add **counted invariants** for Couch's steady state: (1) bytes read/decoded per idle UI tick with N=20 synthetic slots and 30 MB ledgers = 0 for unchanged files; (2) the same per peer message; (3) process-table scans per attach ≤ 1 (pair#218); (4) an idle-CPU smoke in the acceptance runbook: restart over a 20-slot fixture, Couch CPU < 5% averaged over 60 s after attaches settle. Counted invariants run in the normal suite (deterministic); the CPU smoke is part of every Couch acceptance session.


### 2026-10-15 — dispatched counted-invariant delivery (proposed)

TL receipt2e776e5f-723a-4479-8930-8339f014cd38 dispatches the 10-14 Revision's initial three counted Couch invariants, normal suite, no timings, reusing #559/#218. Claimed pair:7 and started the issue; start-plan refreshed only the slot's Ariadne substrate tocea78653e4003d0ca6fa64fbf9ae3b726f5d9b37 and was rerun as instructed. No product implementation yet.

Read-only inspection finds #559 already has the 20-owner scale tests below the consumers: threadactivity.SessionActivity and a hand-built messageAuthority.current adapter. The missing release protection is actual Console scheduling/message-service wiring and an attach count at the real process-reader boundary. Proposed plan: workshop/plans/000204-workbench-performance-suite-counted-invariants-first-plan.md. It reuses rather than duplicates the counters and makes every zero assertion prove cold/mutation work and successful behavior.

Two scope points require TL plan approval: (1) close this delivery on the three dispatched Couch invariants, with earlier #201/#202/#203/tier2 goals explicitly retained outside its completion claim; (2) strict total process scans<=1 applies to selected Couch attachment (parent0 + helper1). Standalone attach currently scans2, continuation can scan3; #218's owner pass is one initial capture plus2N full-table live fences, not a strict total1. The plan recommends the current selected Couch path, and requests a scope decision before any standalone/fence redesign. Original Done-when/Plan above are not marked fulfilled; update them with their original text preserved in this Revision after the TL settles this delivery.

### 2026-10-15 — TL-approved delivery and ratchets

Receipt dcb3f338-41ac-45ae-a098-39ff395ae578 approves plan227c8394, closes this issue on the three Couch invariants, and adds the exact other-path count ratchets (standalone2, continuation3, owner1+2N) without lowering them here. Earlier interaction/tier2 backlog filed unassigned as #566; scan-redesign follow-up filed unassigned as #567 with current counts. Replaced the active Spec/Done-when/Plan so completion is evaluated against this approved delivery; the superseded sections are preserved below. No runtime implementation has started.

<details>
<summary>Superseded original contract (preserved)</summary>

> ## Spec
> 
> **A performance suite whose tier-1 assertions are counts, not times.**
> 
> - **Tier 1 — counted invariants (blocking).** Deterministic, fast, host- and
>   load-independent, runnable in CI. Assert operation counts at seams: subprocess
>   spawns per interaction, file reads per keystroke, threads requested per
>   session. These are the promises, and the three above are the opening cases.
> - **Tier 2 — timings (non-blocking).** A regression net. Every reading records
>   the conditions that qualify it — for this workbench, **the number of
>   concurrently working agents**, which moved latency 8× on 2026-09-06 with the
>   session population unchanged. A timing without its agent count is not
>   recorded, because it cannot be compared to anything.
> 
> Tier 2 never fails a build. It reports drift; a human decides whether the drift
> is real. This is the `under-specify` discipline applied to measurement: spend
> the operator's attention on three promises that cannot flake rather than on
> thirty thresholds that will.
> 
> ### To settle in the plan
> 
> 1. **Where the counting seam goes.** Counting subprocess spawns and file reads
>    needs an injection point. `ARCH-PURE`'s existing seams are the candidates —
>    `RunZellijAction` is already an interface method (`termcmd`), and the nvim
>    side would need something equivalent. Prefer counting at a seam the
>    production path already goes through over instrumenting the OS.
> 2. **Whether the nvim side can be counted in-process.** `#202`'s invariant is a
>    Lua-level file read; `nvim/annotate_test.lua` is the precedent for testing
>    Lua directly without a buffer.
> 3. **What tier 2 runs against, and how often.** A fake workload with a known
>    agent count is reproducible where the operator's real fleet is not.
> 
> Out of scope: fixing `#201`/`#202`/`#203` — this issue builds the net that would
> have caught them, and each fix lands against its own issue. Ideally the
> invariants land **first**, red, and each fix turns one green.
> 
> 
> ## Done when
> 
> - The three invariants above exist as tier-1 tests, and each **fails against
>   today's tree** before its fix lands — a green test on unfixed code is not a
>   test of anything.
> - Tier-1 tests pass on a busy host and a quiet one with identical results;
>   demonstrated, not assumed.
> - Tier 2 records agent count alongside every timing, and cannot fail a build.
> - Adding a fourth invariant is documented in a way that does not require reading
>   this issue.
> - `workshop/targets/workbench-latency.md` is the referenced commitment
>   (`target:` frontmatter set), and the suite's structure follows it.
> 
> 
> ## Plan
> 
> - [ ] Settle the three plan questions above — especially the counting seam.
> - [ ] Land `#201`'s invariant red; likewise `#202`'s and `#203`'s.
> - [ ] Verify identical tier-1 results on a loaded vs quiet host.
> - [ ] Add tier-2 timing capture with mandatory agent-count recording.
> - [ ] Document how to add an invariant.

</details>

### 2026-10-15 — plan-quality round1

PQ-1/PQ-2 request concrete process-source ownership and named measurement-oracle unit-test strategies. Refined the durable plan with one zero-native procutil.Source/Reader instance propagated through ancestry, orphan sweep, initial owner capture and every native live fence; no global hooks or observation caching. Named Counter.RecordRead/RecordDecode/Snapshot and Snapshot.Delta strategies use an independent accounting oracle, immutable snapshots and joined concurrent updates under race. No implementation before the gate passes.

## Estimate

Produced via brain/data/life/42shots/velocity/estimate-logic-v3.1.md against baseline-v3.1.md. Method A only; source is marked stale, so calibration is provisional. Reuse existing Go/std-library fixtures, socket service and native process capture; no novel external library/API discovery. Familiarity1.0. Thorough approved plan applies x0.2 design discount and15% design buffer; implementation picks are40% of v2's primitive table as v3.1 requires.

Three consumer integrations each use API-integration base design1.0/impl0.5 => design0.20/impl0.20. Fixture/counter extraction uses smaller-Go-module base0.20/0.50 =>0.04/0.20; source wiring uses cross-cutting-refactor base0.60/0.50 =>0.12/0.20; docs base0.20/0.20 =>0.04/0.08; one close review base0.20/0.50 =>0.04/0.20. Design subtotal0.84 plus buffer0.126; implementation1.28; total2.246h. Queue wait is not an extra implementation primitive.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: api-integration design=0.20 impl=0.20
item: api-integration design=0.20 impl=0.20
item: api-integration design=0.20 impl=0.20
item: smaller-go-module design=0.04 impl=0.20
item: cross-cutting-refactor design=0.12 impl=0.20
item: atlas-docs design=0.04 impl=0.08
item: milestone-review design=0.04 impl=0.20
design-buffer: 0.15
total: 2.246
```

### 2026-10-15 — counted consumer implementation checkpoint

change-code passed the full flow after PQ-1/PQ-2 refinement and the derived2.246h estimate. Estimate-quality INFO notes compressed source/attach/verification implementation allocations; this is retained as calibration risk, not a reason to guess actuals. Native Source is zero-value compatible, instance-local and never caches observations; explicit owner IO still wins. Shared test artifacts/counters reuse canonical artifact paths/ledger recognition and keep independent read and decode totals.

UI consumer20x30,000,000-byte valid ledgers passes12.019s before extracting the row adapter; production now calls the same couchtty.ThreadActivityProbe used by the actual request/advance/finish/menu-reducer test. Post-extraction activity tests pass9.261s; UI race31.260s. Recreating activity/launch caches per row in an overlay fails warm-byte invariants (37.88s), and disconnecting the observer fails cold controls (5.24s). Each unchanged completed generation is measured separately; one ledger mutation causes only its owner work, followed by zero again.

Message consumer uses real private socket requests, broker submission receipts and endpoint delivery through the shared production recordedMessageAuthority. Twenty owners at30MB and60 sends measure zero warm historical/index bytes per request; cold read/decode29641bytes proves the seam. Cache-bypass and disconnected-observer overlays both fail. Existing smaller authority tests retain replacement/PID/readiness/thread refusal; fresh small security reads are not optimized away. Normal and targeted race message tests pass.

Selected attach parent0/helper<=1 and ratchets standalone<=2, continuation<=3, owner<=1+2N are connected to one low process Reader through real production ancestry, snapshot parsing and live fencing. Owner formula covers N1,3,21. Normal/race source/launcher/core tests pass. Additional-enumeration overlays fail selected2>1, standalone3>2, continuation5>3; owner5>3,13>7,85>43; all six public parent shapes fail4>0. Bounds allow future reductions; connected source and successful exact attach/no-create assertions prevent empty tests.

New production source inventory initially failed for moved test-support files and Source; corrected explicit classifications and reused artifactpath.IsLedgerHistorySidecar instead of inventing filename vocabulary. Artifactpath and support tests pass14.051s/0.408s. Shared counter arithmetic/race tests and a negative-delta-clamping counterfactual pass/fail as expected. The lower #559 idle test is reduced to3x4096 while its benchmark and allocation-history comparison remain; actual20x30MB scale belongs to the consumer test.

Atlas now documents normal-suite commands, count boundaries, adding an invariant and a separate TL/operator CPU smoke. No stage, live operator action or machine setting change. Integrated targeted group is running before the pushed-head recorded full suite; close/landing remain pending.

Integrated targeted group PASS: UI12.212s, message1.348s, launcher0.184s, core9.676s, procutil0.167s, threadactivity1.156s, support0.321s (/tmp/pair204-integrated-targeted.log). Temporary counterfactual sources live only outside the checkout.

### 2026-10-15 — first full suite and canonical fixture root

Recorded full suite on52d9b7daf6b97538fce58e44ccf756acca6da49b completed all recipes in9m4s after280s queue wait. Its only failure was TestNoNativeAuthorityShadowInInteractiveConsumers: the extracted test-support fixture duplicated the Codex native storage path. Saved failure log e74cededd3e7ff175d5ee7bbbf6dc708.log, SHA256 prefix67142c241ca3. Every other package passed; no remaining recipes or flaky-failure waiver.

The fixture now obtains its native root from sessioninventory.NewOSRuntime.NativeRoots instead of defining another path (ARCH-DRY). The guard stays unchanged. Targeted native-authority guards, shared fixture and both20x30MB consumer invariants pass: sessioninventory3.828s, support0.280s, UI11.937s, message1.989s. A new recorded full suite is required before close.

### 2026-10-15 — full verification green

Recorded full suite PASS on021cd810983d981f2e58ccda109302a29ed28506: exit0 in8m48s after258s machine-gate wait, log SHA256 prefixe63fe49ae2d2 at .git/sdlc/local-checks/021cd810983d981f2e58ccda109302a29ed28506/suite.log. Every script/editor recipe and Go package completed, including sessioninventory10.642s, core267.644s, UI47.552s, messaging47.491s and launcher44.081s. No failures waived. PR344 conformance and merge-check both pass (runs38740723867/38740723986). Targeted races and deliberate counterfactual failures are recorded above. Close review/landing are the remaining binary checkpoints; the live CPU runbook is guidance, not a claimed operator acceptance.
