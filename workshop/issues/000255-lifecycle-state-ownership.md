---
id: 000255
status: open
deps: []
github_issue:
created: 2026-09-14
updated: 2026-09-15
estimate_hours:
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

- A durable terminal-contract design names supported capabilities, transformations, state authorities and concrete production boundaries, including deliberate handling of unsupported features.
- Child-requested, selected-view and parent-terminal states have authoritative owners and coherent identity/output-position semantics; emitted effects and uncertain outcomes remain distinct.
- Production switching and output admission prevent stale-source input/output and mode leakage; the historical race has a committed regression if still present.
- Composed conformance tests enforce chunk independence, switch preservation, UI isolation, input fidelity and ordered effects as specified above, including background mode changes and replay eviction.
- Tests exercise independent terminal interpretation, forced scheduling, partial IO and supported live terminal/multiplexer behavior; local parser/reducer tests remain supporting evidence rather than the only oracle.
- Atlas documents the abstraction and #207/#224/#241/#252/#254 responsibilities, with lifecycle concerns handed off to #256 and no duplicated owners.

## Plan

- [ ] Revalidate terminal findings and map existing Couch/Pair/Zellij boundaries, capabilities, state and output paths.
- [ ] Claim/start-plan and author an approved durable terminal-abstraction design with supported semantics, ownership, transition ordering and implementation boundaries.
- [ ] Add conformance regressions for the terminal acceptance properties using independent interpretation and forced production-boundary schedules.
- [ ] Implement the approved abstraction and reconciliation changes, coordinating linked terminal issues.
- [ ] Verify composed/live behavior, update atlas and close through SDLC.

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

Operator challenged the explanation that adding UI and interception inherently makes interference unavoidable: a faithful terminal abstraction should preserve inner-program behavior. Accepted that correction. The missing requirement is a semantic terminal contract, not merely more locks or single-owner fields. Current #252 reproduction violates chunk independence. The #207 trace records click-only writes during panel display and subsequent takeover with no restored mouse modes; this supports a mode-restoration gap, while the initial background mouse-off source remains unresolved. These observations do not establish the disconnect cause. No production changes or live repairs were made for this issue update.

## Revisions

### 2026-09-15 — Make terminal semantics explicit within generic ownership scope

Reason: operator requested a rigorous terminal abstraction rather than attributing interference to inevitable layered complexity. Added a dedicated terminal-contract design requirement, separated child-requested/selected-view/parent-terminal state, defined observable acceptance properties and connected the concrete mouse, UTF-8, filtering and mode-reconciliation issues. Preserved the original lifecycle, external-outcome and uncertainty scope and historical audit evidence. Implementation remains open and requires the existing durable-design approval; this update does not select a full emulator or authorize a blanket1003 change.

### 2026-09-15 — Narrow to terminal abstraction

Reason: the operator clarified that terminal state management and a faithful terminal abstraction are the central purpose of #255. Supersedes the earlier revision retaining a broad lifecycle umbrella. Retitled and rewrote current Problem/Spec/Done when/Plan around terminal semantics and conformance. Moved generic lifecycle authority, process/attachment outcomes and observation uncertainty to #256; retained terminal IO outcomes and switch-ordering concerns here. Historical log and audit evidence remain as provenance, not additional current scope. No implementation was started.
