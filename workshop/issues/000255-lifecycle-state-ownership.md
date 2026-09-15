---
id: 000255
status: working
deps: []
github_issue:
created: 2026-09-14
updated: 2026-09-15
estimate_hours: 18.025
started: 2026-09-15T09:20:10-07:00
---

# Establish a faithful terminal abstraction for Couch and Pair

## Problem

Couch and Pair must present a defined terminal abstraction to inner programs. Composing UI, switching children and routing input must preserve that abstraction. The current combination of selective parsing, passthrough, interleaved control writes and bounded replay has incomplete contracts for terminal state and transformations. Correctness requires more than locks or consistent ownership of existing fields.

Concrete failures motivate the design: #252 reproduces control insertion inside split UTF-8 characters; #207 captures click-only mode emission followed by takeover without restored mouse modes. The historical terminal audit also identified:

1. **Confirmed TTY data race and incomplete switch transaction.** `cmd/internal/couchtty/console.go:565` reads `p.replayCutoff` outside `mu`, while `onChunk` writes it under `mu` at line 1405. The temporary race test below failed on those exact accesses. `switchTo` publishes active/focus at lines 530–547 and performs takeover afterward; the operation worker reaches it while Run continues input/output handling (`operation_queue.go:64`, `console.go:2217`). A stale output-source decision can outlive a focus change. The data race is reproduced; stale-screen/input misrouting is an architectural interleaving risk, not a reproduced incident.
5. **Sequence coverage does not span the composition.** Local menu/reattach/orientation reducers have generated sequence tests; park tests include timeout followed by late success and stale attempts. The audit found no comparable composed console/lifecycle model spanning attach, switch, queued output, child exit, input failure, partial host write, operation completion and stop. Tests confined to a locked write transaction do not exercise the ownership decision made before entering it.

Those audit findings refer to baseline `5ebb381f` plus then-uncommitted #250 recovery work; revalidate before implementation. The race was reproduced, but neither it nor the other terminal findings establishes the disconnect's cause.

## Spec

Define and faithfully implement the terminal abstraction at Couch and Pair boundaries, with explicit state ownership, supported protocol semantics and observable conformance. Preserve existing useful parsers, reducers, typed output paths and Child-owned geometry where they meet the contract (ARCH-DRY, ARCH-ORDER). A dedicated durable design must choose implementation boundaries before code changes.

### Proposed architecture (2026-09-15; awaiting design approval)

Recommend an authoritative virtual terminal per child and one shared parent compositor for Couch and Pair. Render semantic screen frames rather than replaying raw child drawing commands into shared terminal state. Preserve existing PTY/geometry/Host seams and qualify the existing x/vt backend before adopting it; source inspection found compatibility gaps, so it is not yet a selected production backend. A stateful passthrough translator and moving all UI into Zellij are the alternatives considered.

Durable proposal: [terminal abstraction plan](../plans/000255-terminal-abstraction-plan.md). Its first checkpoint qualifies required protocol behavior and the backend; later milestones implement the shared abstraction and migrate both consumers. Qualification alone cannot close #255. Detailed production steps, numeric budgets and remaining input policies must clear design review before code changes.

### Terminal abstraction contract

Specify the terminal abstraction presented to each inner program before assigning implementation fields to owners. Adding a status bar, switching threads or intercepting shortcuts must preserve that abstraction within a declared supported protocol. Serialization and race freedom alone do not establish correct terminal semantics (ARCH-ORDER). Selective parsing plus passthrough needs an explicit preservation contract; terminal complexity is not an exemption from it.

Document supported features, capability advertisement and deliberate transformations, including geometry, cursor/save state, scrolling regions, screen buffers, rendering attributes, UTF-8/control framing, synchronized drawing, mouse and keyboard modes, and terminal queries/replies. Define how unsupported sequences are forwarded or handled without corrupting framing or falsely advertising support. This is a design requirement, not a decision to build another complete emulator or to filter additional sequences.

Separate three kinds of state:

- **Child-requested state:** derived from each child's continuous output, including background output; exposed as a coherent snapshot tied to an output position and child identity.
- **Selected-view state:** region ownership, selected source, geometry, input destination and the output/replay boundary for a switch.
- **Parent-terminal state:** desired configuration and evidence of emitted effects, with explicit partial-write/error uncertainty. A scanner's belief and successful byte acceptance are not a terminal-state query.

Each state has one transition owner; multiple parsers are legitimate when they represent distinct terminal connections or delivery stages. Consumers must not reconstruct competing versions of the same authority from different stream fragments. One compositor owns parent output and mode reconciliation. Switching must order old-source admission, pending output, establishment of new state/view and admission of the new source; generation/position checks prevent queued old output from becoming current. Bounded replay may restore content but is not authority for persistent terminal modes.

Input modes apply to a terminal connection, while event routing follows region/focus and an explicit press/drag/release policy. Couch's child is the Zellij client; Zellij owns routing among its panes. Couch need not infer whether Claude, Codex or nvim occupies an inner pane. If Couch requests a superset such as mouse mode1003, define filtering to each child's requested event set and encoding. Using1003 is a candidate policy, not a substitute for ownership, switch restoration or input conformance.

### Terminal acceptance properties

- **Chunk independence:** every split of equivalent valid bytes produces equivalent child-visible and parent-rendered behavior; control injection cannot divide UTF-8 characters or terminal sequences. Malformed/incomplete streams have explicit bounded handling.
- **Switch preservation:** after switching away/back, the selected child's view and input contract match its current state, including background mode changes and commands that aged out of replay.
- **UI isolation:** composing Couch/Pair UI preserves the inner program's cursor/save state, attributes, scrolling region, buffers and drawing semantics under the declared geometry mapping.
- **Input fidelity:** only the intended destination receives an event, in the negotiated encoding/event set; parent-only shortcuts are consumed and mouse coordinates/drag ownership remain consistent.
- **Ordered effects:** live output, replay, focus changes, resize and failed/partial writes cannot silently publish contradictory ownership or confirmed host state.

Test these through production composition with an independent terminal interpreter/stateful fake and live conformance where practical. Force byte splits and switch/output schedules; checking internal fields with the same parser is insufficient. Preserve existing local reducers and parser tests as supporting coverage (ARCH-MOCK, ARCH-DRY).

### Scope and related work

In scope: terminal connections and capabilities; child-requested, selected-view and parent-terminal state; input routing/encoding; geometry; parsing and control insertion; rendering/replay; switching and output ordering; terminal read/write failure effects and recovery.

Out of scope: durable thread lifecycle transition APIs, store mutation authority, process liveness classification and generic attachment/process lifecycle redesign. Those findings are preserved in #256. Share necessary outcome types at the boundary rather than inventing a second lifecycle system. #253 supplies disconnect telemetry; it is not a replacement for terminal-state correctness.

- #224: typed console output ownership; coordinate its writer API with the full selection/output transition.
- #207: mouse state, parent tracking and restoration on switching.
- #241: private-mode reconciliation across pane takeovers.
- #252: UTF-8/control framing and chunk independence.
- #254: explicit minimal-filtering policy under the supported terminal contract.
- #250: completed recovery integration to preserve and revalidate during terminal design.

These are coordination and acceptance references, not blanket blocking dependencies. Decide which work is implemented here or in linked issues during design; do not duplicate state owners. No live-session repair or production refactoring is authorized by this issue edit.

## Done when

The primary acceptance is operator-visible: the ongoing display corruption and mouse-selection highlight loss are gone in actual Couch/Pair use. Architectural completion and green component tests alone cannot close this issue.

- Sustained use shows no recurrence of the reported replacement glyphs, corrupted pane borders or flashing/stray rendering artifacts attributable to Pair/Couch processing. Any remaining reported artifact must be investigated and resolved or explicitly reviewed with the operator; do not assume #252 explains every symptom.
- Mouse selection highlights continuously during dragging in the agent pane and right-side shell/nvim panes, including after panel open/close, repeated thread/tab switches and reattachment; it must not wait for mouse release.
- Isolated automated reproductions cover the observed failures, followed by an operator smoke test under the triggering workflows and extended-session conditions. Record duration, operations, terminal/build versions and observed result. Absence during a brief test is not proof of sustained resolution; final close requires operator acceptance of the live result.

- A durable terminal-contract design names supported capabilities, transformations, state authorities and concrete production boundaries, including deliberate handling of unsupported features.
- Child-requested, selected-view and parent-terminal states have authoritative owners and coherent identity/output-position semantics; emitted effects and uncertain outcomes remain distinct.
- Production switching and output admission prevent stale-source input/output and mode leakage; the historical race has a committed regression if still present.
- Composed conformance tests enforce chunk independence, switch preservation, UI isolation, input fidelity and ordered effects as specified above, including background mode changes and replay eviction.
- Tests exercise independent terminal interpretation, forced scheduling, partial IO and supported live terminal/multiplexer behavior; local parser/reducer tests remain supporting evidence rather than the only oracle.
- Atlas documents the abstraction and #207/#224/#241/#252/#254 responsibilities, with lifecycle concerns handed off to #256 and no duplicated owners.

## Plan

- [x] M1 — Qualify the required terminal contract and candidate backend; record failures, untested obligations and an evidence-based adoption decision.
- [ ] M2 — Implement the shared endpoint/presenter after qualification and detailed design approval.
- [ ] M3 — Migrate Couch and Pair, including wrapper transformation conformance, to the shared contract.
- [ ] M4 — Complete composed/live conformance, measured rollout verification and publication.

## Log

### 2026-09-14 — Audit capture

Created at the user's request after the read-only audit. Left open; no implementation begun. Existing selected orientation, ptychild, termcmd and couchcore tests passed; the TTY audit's selected fake/reducer tests also passed. A temporary Go overlay test failed under the race detector with `onChunk` writing console.go:1405 and `switchTo` reading console.go:565. No Pair source or live sessions were changed by the audit.

The temporary files were `/tmp/pair-state-audit/{overlay.json,audit_test.go,race.log}`. They are not durable dependencies: the complete test source is preserved below. It uses existing package fixtures, concurrent fake-child output and forced switching; production's operation worker permits this concurrency. Promote it to a deterministic scheduling-boundary regression during implementation.

```sh
go test -race -overlay /tmp/pair-state-audit/overlay.json ./cmd/internal/couchtty -run '^TestAuditConcurrentOutputAndSwitch$' -count=1
```

```go
package couchtty

import (
 "strings"
 "sync"
 "testing"
)

func TestAuditConcurrentOutputAndSwitch(t *testing.T) {
 f := newFixture(t, 24, 80)
 waitFor(t, "console ready", func() bool { return strings.Contains(f.host.Written(), "\x1b[") })
 var wg sync.WaitGroup
 wg.Add(1)
 go func() {
  defer wg.Done()
  for i:=0;i<200;i++ { f.child.Feed([]byte("output\r\n")) }
 }()
 for i:=0;i<60;i++ { f.con.switchTo("c1",true,arrivalOrdinary) }
 wg.Wait()
}
```

The overlay maps an additional `cmd/internal/couchtty/audit_temp_test.go` to the temporary source above. Observed result: `WARNING: DATA RACE`, followed by test failure. This demonstrates the field race only; it neither reproduces the random disconnect nor proves every hypothesized interleaving.

### 2026-09-15 — Terminal abstraction discussion
- 2026-09-15: closed M1 — Full Go suite PASS after implementation corrections; focused race on terminalqualify, probe, artifactpath PASS after BR-5 test-only correction; four partition mutations detected; probe 53 pass, 15 fail, 14 not-covered rejects unchanged adoption; git diff --check PASS. No production fix claimed. Actual unavailable: sdlc actual found no transcript events.; review verdict: SHIP

Operator challenged the explanation that adding UI and interception inherently makes interference unavoidable: a faithful terminal abstraction should preserve inner-program behavior. Accepted that correction. The missing requirement is a semantic terminal contract, not merely more locks or single-owner fields. Current #252 reproduction violates chunk independence. The #207 trace records click-only writes during panel display and subsequent takeover with no restored mouse modes; this supports a mode-restoration gap, while the initial background mouse-off source remains unresolved. These observations do not establish the disconnect cause. No production changes or live repairs were made for this issue update.

- 2026-09-15 M2 implementation progress: checked-in backend fork now passes all 68 original executable qualification cases without weakening predicates. Added erase regressions for ED1 preserving the cursor-row suffix, ED2 background, ED3 history-only clearing, and bounded ECH. Shared immutable Frame/View/Render, incremental input decoder, context-aware descriptor transport, FIFO input writer and Endpoint are implemented; focused normal/race tests pass. Independent xterm-headless renderer oracle also passes. Presenter and attributed integration qualification remain in progress; production Couch/Pair migration has not begun.
- Endpoint verification now covers hidden query origin, actual decoder-to-negotiated-input delivery, every split of Unicode/control output across input/snapshot operations, synchronized-output recovery, no-prior-snapshot sync capture, once-only effects, unsupported clipboard reads, acknowledged resize and teardown rejection. Transport tests include blocked/partial/zero-progress writes and joined cancellation.
- M2 provisional microbenchmarks on Apple M2 Max: feed+owned snapshot 80×24 ≈90µs/443KB and 240×80 ≈893µs/4.31MB per operation; renderer unchanged frame ≈49µs/496µs respectively with zero allocations. These are per-operation allocation measurements, not retained RSS or sustained latency acceptance; M4 must measure the complete path.

- 2026-09-15 M2 pre-gate verification: final `go test ./... -count=1` passed; shared terminal/ttyio/terminalqualify race tests passed; independent xterm-headless renderer oracle passed; fork normal/race tests passed. Qualification is 81 pass, 0 fail, six correctly uncovered M3/M4 obligations. Updated the exhaustive source inventory and replaced the wrapper test that deliberately expected split-ZWJ corruption with every-split correct-cluster assertions. Presenter in-session review fixes stable mouse modes during drag, joined cancellation, hidden refresh isolation, typed origin retirement, panel release, and owned keyboard cleanup. Native scrollback export is explicitly retained as M3 work. SDLC M2 boundary review is next.

- 2026-09-15 M2 boundary round 1 returned REWORK with BR6–BR8. BR6 reproduced chrome/panel/orphan mouse gesture leakage; View now represents parent/child/no ownership and Presenter admits mouse events atomically against a backend negotiation epoch. BR7 reproduced truncated CSI effects (including exactly32 parameters losing one); the fork now retains bounded overflow evidence and rejects the entire CSI/DCS command before dispatch. BR8 reproduced stale cursor shape/visibility after reset, restore and buffer switching; Endpoint and the qualification observer now read copied authoritative cursor state. Each class has a failing-before regression and passing-after verification. Extended qualification adds three cursor-state cases (84 pass, six consumer/live obligations uncovered). Wider verification and the second boundary review follow.

## Revisions

### 2026-09-15 — Make terminal semantics explicit within generic ownership scope

Reason: operator requested a rigorous terminal abstraction rather than attributing interference to inevitable layered complexity. Added a dedicated terminal-contract design requirement, separated child-requested/selected-view/parent-terminal state, defined observable acceptance properties and connected the concrete mouse, UTF-8, filtering and mode-reconciliation issues. Preserved the original lifecycle, external-outcome and uncertainty scope and historical audit evidence. Implementation remains open and requires the existing durable-design approval; this update does not select a full emulator or authorize a blanket1003 change.

### 2026-09-15 — Narrow to terminal abstraction

Reason: the operator clarified that terminal state management and a faithful terminal abstraction are the central purpose of #255. Supersedes the earlier revision retaining a broad lifecycle umbrella. Retitled and rewrote current Problem/Spec/Done when/Plan around terminal semantics and conformance. Moved generic lifecycle authority, process/attachment outcomes and observation uncertainty to #256; retained terminal IO outcomes and switch-ordering concerns here. Historical log and audit evidence remain as provenance, not additional current scope. No implementation was started.

### 2026-09-15 — Planning started

Claimed #255 and ran start-plan. Read-only parallel architecture mapping confirmed Screen is a selective observer, replay/mode snapshots can differ in position, and selection precedes output takeover in both consoles. Backend inspection found existing x/vt reusable screen APIs but keyboard, query/effect and framing conformance gaps. Recorded virtual-terminal/compositor proposal and qualification-first boundaries; no runtime changes, dependency upgrades, or live probes.

### 2026-09-15 — Pair coverage explicit

Operator emphasized that Pair needs the same abstraction. The proposal covers both Couch and pair term as shared compositors and pair wrap as an explicit observation/transformation boundary. Added wrapper filter, Return, notification and query/reply audit plus composed-path acceptance; a Couch-only implementation cannot close #255.

### 2026-09-15 — Architectural proposal reviewed

Fresh spec review approved the proposed direction and M1 qualification, with no blockers at that stage. Incorporated advisory input/reply serialization, presentation admission, wrapper observation and budget timing. Awaiting operator architectural approval before qualification implementation; backend suitability and detailed later milestones are not yet approved.

### 2026-09-15 — Qualification approved

Operator approved the architectural direction and qualification phase. Added executable M1 tool/matrix plan, explicit negative qualification semantics and bounded candidate lifecycle; later production migration remains subject to qualification and detailed plan approval. Replaced generic plan rows with the four actual review boundaries from the approved proposal.

- 2026-09-15 M2 second boundary review: BR6–BR8 independently verified addressed; BR9 blocks closure. Failed resize revived an already-canceled child gesture. Sweep all cancellation callers (release, failure, selection, panel, negotiation, resize), committing ownership revocation separately from delivery and subsequent operation success. Interrupted delivery must retain the existing release admission rather than enqueue a duplicate on retry. M3 remains pending this gate.

- 2026-09-15 M2 BR9 correction verified: explicit pending cancellation commits child-to-parent ownership before admission, retries Flush rather than enqueue, and Release attempts parent cleanup while reporting child cancellation errors. Red evidence: `/tmp/pair255-br9-red.log`, `/tmp/pair255-br9-callers-red.log`, `/tmp/pair255-br9-child-red.log`; final shared terminal/ttyio/qualification race checks and independent renderer oracle passed (`/tmp/pair255-m2-round3-{race,oracle}.log`). Six-caller matrix plus actual interrupted Select, physical remainder, fresh press, failed resize and permanent child failure regressions pass. Previous full root suite remains valid for unchanged consumers. M3 history discovery prototypes are preserved under `tests/terminal-oracle/discovery`; all six probes passed, with typed/dirty smoke repeated after oracle error/cleanup hardening. These are preparation evidence, not production qualification.

- 2026-09-15 M2 third boundary review: BR9 independently mutation-verified addressed; terminal implementation checks passed. BR10 blocks closure because newly preserved discovery probes retain temporary directories without a lifecycle. Correct the shared driver to remove all owned artifacts after process/PTY teardown on success and failure, with cleanup regressions. No retained-directory mode is needed. Separately prepared baseline binaries from b11ab67f under `/tmp/pair255-baseline-b11ab67f/out`; preliminary direct-PTY synthetic startup/input samples are `/tmp/pair255-baseline-startup.json` (includes cold first launch; not final M4 comparison).

- 2026-09-15 BR10 verified: three cleanup regressions first reproduced leaks; all five success/spawn/runtime/teardown/capture tests now pass. TemporaryDirectory owns the entire run; both PTY descriptors close on spawn failure; diagnostic capture retains at most 8KiB. Typed-history and dirty-rebuild probes still pass; process snapshot found no surviving disposable Zellij processes. Evidence `/tmp/pair255-br10-cleanup-tests.log`, `/tmp/pair255-br10-{typed,dirty}.jsonl`, `/tmp/pair255-br10-processes-after.log`. Terminal production code unchanged since passing race/oracle checks.

- 2026-09-15 M2 fourth boundary review disposed BR10 but reproduced BR11 (parent restoration). Independent production Select/Release regression now fails at accepted prefix93: ordinary text overwrites the last column because autowrap remains disabled. Sweep renderer/setup state and test every accepted output prefix, including hyperlink/rendition/cursor style and input-reporting cleanup. Red: `/tmp/pair255-br11-oracle-red.log`.

- 2026-09-15 BR11 corrected: centralized parent release inventory restores autowrap, origin/full margins, OSC8, rendition and default cursor style/visibility after aborting partial framing, while retaining confirmed-owned keyboard pop. Every-prefix production Select/Release test now passes in independent xterm, including ordinary post-release text, modes, styles, link closure and cursor shape/blink. Shared terminal/ttyio/qualification race checks pass; evidence `/tmp/pair255-br11-{oracle-green,race}.log`. No live symptom acceptance claimed.

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only. This is approved M1 qualification only; M2–M4 require later estimates after their designs settle. Calibration is marked stale by estimate-source, so the result is provisional.

Candidate integration uses the existing vt library: 1.0 design ×0.5 library ×0.2 thorough-spec =0.10; implementation0.8 ×0.4 =0.32. Matrix/independent expectations are a separate greenfield concern with no library for the oracle:1.0 ×0.2 =0.20, impl0.8 ×0.4 =0.32. Report and CLI are two smaller modules, each0.3 ×0.2 =0.06 design and0.5 ×0.4 =0.20 impl. Docs0.2 ×0.2 =0.04 design and0.2 ×0.4 =0.08 impl. Review0.1 design and0.5 ×0.4 =0.20 impl. One real-API discovery allowance0.6 ×0.4 =0.24 impl covers behavioral qualification of the unfamiliar backend. Familiarity1.0; design buffer15%. Total0.56 ×1.15 +1.56 =2.204h.

```text
model: estimate-logic-v3.1
familiarity: 1.0
item: greenfield-go-module design=0.10 impl=0.32
item: greenfield-go-module design=0.20 impl=0.32
item: smaller-go-module design=0.06 impl=0.20
item: smaller-go-module design=0.06 impl=0.20
item: atlas-docs design=0.04 impl=0.08
item: milestone-review design=0.10 impl=0.20
item: real-api-discovery design=0 impl=0.24
design-buffer: 0.15
total: 2.204
```

### 2026-09-15 — Operator-visible acceptance takes precedence

Operator clarified that #255 acceptance is the ongoing display corruption and loss of selection highlight going away. Promoted these to primary Done when criteria, requiring both causal regressions and sustained actual-use acceptance across panes/switches. The abstraction is the means, not a substitute deliverable. M1 qualification still cannot close #255.


### 2026-09-15 — M1 qualification implemented, negative backend result

Added the isolated terminal qualification probe and literal screen/input/query fixtures. Current result: 41 pass, 15 fail, 14 not-covered; unchanged adoption is rejected. Controlled blocked/failing reply transport, cancellation, isolation, teardown and race tests pass; a comparator mutation is detected. Production Couch/Pair behavior is unchanged. The qualification report records the Pair wrapper raw/transformed-stream audit and the integration obligations that remain. Final acceptance remains sustained absence of display corruption and continuous mouse-drag highlights in both panes, confirmed by the operator. M1 review is pending; M2–M4 require the approved backend re-plan checkpoint.

Full Go suite passed after generating runtime assets; focused normal/race and artifact-inventory checks passed. One existing orientation reply test failed intermittently in an earlier full run, then passed 30 focused repetitions and the final suite; recorded without claiming a fix. `sdlc actual` could not find transcript events, so M1 uses the specific unavailable-telemetry exception rather than guessed hours.


### 2026-09-15 — M1 boundary review round 1: REWORK

Four findings addressed before resubmission: BR-1 complete observation equivalence, BR-2 non-color text attribute coverage, BR-3 bounded structured JSON evidence, BR-4 README probe documentation. Added regression tests first; focused race tests pass. Updated matrix: 53 pass, 15 fail, 14 not-covered, still rejecting unchanged production adoption. Added the general qualification lesson to workshop/lessons.md. No REWORK verdict is recorded as a completed review boundary.


### 2026-09-15 — M1 review round 2 partition-test correction

Round 2 disposed BR-1 through BR-4 and raised BR-5: partition regression tests counted calls without proving delivered bytes. Added literal partitions for empty, single-byte, multi-byte UTF-8 and CSI inputs, plus independent byte-preservation and every-boundary assertions over all 256 byte values and mixed Unicode/control streams. The production RunCase executor path is checked with the same invariant. Four mutations are detected by failed assertions: repeating whole input, dropping a byte, skipping alternating boundaries and omitting the first boundary. No production implementation changed in this correction. Final full Go suite passed after round 1 corrections; focused race verification covers these additional tests.


### 2026-09-15 — M1 complete, implementation decision pending

Third boundary review: SHIP, all five findings disposed. Qualification result remains 53 pass, 15 fail, 14 not-covered. The approved qualification phase is complete; backend re-plan is next. Production migration and the user-visible acceptance gate remain open.

### 2026-09-15 — Operator authorizes M2–M4 continuation

Continue autonomously through M4, including backend re-plan and all implementation/review gates. Then pause for operator smoke test before merge. This supersedes the earlier stop after negative M1 qualification; it does not waive final sustained display/selection acceptance. Durable plan Chunks 2–4 select a checked-in narrow x/vt fork plus shared endpoint/presenter, both compositor migrations, wrapper audit and isolated sustained conformance. Production installation and merge remain held for the final smoke test.


### 2026-09-15 — Expanded estimate after M2–M4 plan-quality pass

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only. Preserve the historical M1 estimate of 2.204h above; the executable derivation below includes those unchanged item values plus M2–M4. The earlier fence is now historical text rather than a second active derivation.

Source ranges were read from v2/v2.1 and the v3.1 implementation scaling. New Go concerns use the upper 0.8h implementation range ×0.4=0.32; TUI concerns use 1h×0.4=0.40; API harnesses use 1.5h×0.4=0.60; smaller/refactor use 0.5h×0.4=0.20. Design uses the documented library half-discount for usable vt/ultraviolet/ANSI/syscall/oracle libraries, then ×0.2 for settled decisions; grapheme/bounds/fd/composed-harness decisions use ×0.5 because implementation details still need discovery. Familiarity remains 1.0 for this now-audited Go/backend/repo surface, with two explicit real-API discovery allowances. Review/docs are counted once per remaining milestone, and sustained/native verification has its own API-harness items. This is a provisional calibration estimate, not a deadline or measured actual.

| Scope | Primitive | Design h | Impl h |
|---|---|---:|---:|
| M2 grapheme repair | greenfield-go-module | 0.50 | 0.32 |
| M2 keyboard state | greenfield-go-module | 0.20 | 0.32 |
| M2 protocol repairs | smaller-go-module | 0.06 | 0.20 |
| M2 backend bounds | greenfield-go-module | 0.50 | 0.32 |
| M2 endpoint | greenfield-go-module | 0.20 | 0.32 |
| M2 fd/packet transport | greenfield-go-module | 0.50 | 0.32 |
| M2 input decoder | greenfield-go-module | 0.20 | 0.32 |
| M2 view transitions | tui-screen | 0.40 | 0.40 |
| M2 renderer | greenfield-go-module | 0.20 | 0.32 |
| M2 presenter | tui-screen | 0.40 | 0.40 |
| M2 wire oracle | api-integration | 0.10 | 0.60 |
| M3 child migration | cross-cutting-refactor | 0.20 | 0.20 |
| M3 Couch output | tui-screen | 0.40 | 0.40 |
| M3 Couch input | tui-screen | 0.40 | 0.40 |
| M3 Pair terminal | tui-screen | 0.40 | 0.40 |
| M3 wrapper | cross-cutting-refactor | 0.20 | 0.20 |
| M3 composed harness | api-integration | 0.50 | 0.60 |
| M4 sustained harness | api-integration | 0.20 | 0.60 |
| M4 performance | api-integration | 0.10 | 0.60 |
| M4 candidate handoff | smaller-go-module | 0.06 | 0.20 |
| M2 documentation | atlas-docs | 0.04 | 0.08 |
| M2 review | milestone-review | 0.10 | 0.20 |
| M3 documentation | atlas-docs | 0.04 | 0.08 |
| M3 review | milestone-review | 0.10 | 0.20 |
| M4 documentation | atlas-docs | 0.04 | 0.08 |
| M4 review | milestone-review | 0.10 | 0.20 |
| OS nonblocking API discovery | real-api-discovery | 0.00 | 0.24 |
| independent oracle/native conformance discovery | real-api-discovery | 0.00 | 0.24 |

Cumulative design 6.700 ×1.15 + implementation 10.320 = 18.025h. Added scope estimate: 15.821h.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: greenfield-go-module design=0.10 impl=0.32
item: greenfield-go-module design=0.20 impl=0.32
item: smaller-go-module design=0.06 impl=0.20
item: smaller-go-module design=0.06 impl=0.20
item: atlas-docs design=0.04 impl=0.08
item: milestone-review design=0.10 impl=0.20
item: real-api-discovery design=0.00 impl=0.24
item: greenfield-go-module design=0.50 impl=0.32
item: greenfield-go-module design=0.20 impl=0.32
item: smaller-go-module design=0.06 impl=0.20
item: greenfield-go-module design=0.50 impl=0.32
item: greenfield-go-module design=0.20 impl=0.32
item: greenfield-go-module design=0.50 impl=0.32
item: greenfield-go-module design=0.20 impl=0.32
item: tui-screen design=0.40 impl=0.40
item: greenfield-go-module design=0.20 impl=0.32
item: tui-screen design=0.40 impl=0.40
item: api-integration design=0.10 impl=0.60
item: cross-cutting-refactor design=0.20 impl=0.20
item: tui-screen design=0.40 impl=0.40
item: tui-screen design=0.40 impl=0.40
item: tui-screen design=0.40 impl=0.40
item: cross-cutting-refactor design=0.20 impl=0.20
item: api-integration design=0.50 impl=0.60
item: api-integration design=0.20 impl=0.60
item: api-integration design=0.10 impl=0.60
item: smaller-go-module design=0.06 impl=0.20
item: atlas-docs design=0.04 impl=0.08
item: milestone-review design=0.10 impl=0.20
item: atlas-docs design=0.04 impl=0.08
item: milestone-review design=0.10 impl=0.20
item: atlas-docs design=0.04 impl=0.08
item: milestone-review design=0.10 impl=0.20
item: real-api-discovery design=0.00 impl=0.24
item: real-api-discovery design=0.00 impl=0.24
design-buffer: 0.15
total: 18.025
```
