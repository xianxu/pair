# Name binding failures; an unturned fresh launch does not hide the conversation (pair#214) Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A thread whose conversation cannot be resumed says why, in words
that name the failure and its repair. A fresh in-pane restart that never took a
turn no longer hides the conversation it replaced, so the thread resumes it.

**Architecture:** Two independent changes, one at each layer the failure
crosses.
1. **The ledger reader:** `sessioninventory.QueryResumeTargetContext` falls
   back to the previous established generation when the newest launch is a
   chosen-id fresh launch whose session file is proven absent. Couch's proof and
   pair's own resume launch both read this function.
2. **The couch classifier:** the evidence pass keeps the resume-refusal code it
   used to drop, and `ClassifyThread` maps it to three named reasons.

**Tech Stack:** Go; `cmd/internal/sessionledger`, `cmd/internal/sessioninventory`,
`cmd/internal/couchcore`, `cmd/internal/couchtty`.

Issue: `workshop/issues/000214-racing-launch-operations-on-one-thread-leave-it-unresumable.md`.
The inputs are "Scope after 2026-10-06" and the 2026-10-07 design reading.

---

## Operator decisions (2026-10-07)

- **D1: resume the old conversation.** After an in-pane fresh restart whose
  agent never took a turn, resuming the thread continues the conversation the
  restart replaced. The restart left nothing behind, so the old conversation is
  the only real state.
- **D2: name the failure everywhere.** The specific reasons apply wherever the
  ledger shows a binding failure, with or without a park receipt. Such a
  failure no longer reads as plain `session gone`.

## Design

**Fallback rule (item 2, pure).**
`sessionledger.PreviousEstablished(records, owner, before) (Current, bool)`
returns the newest launch for `owner` whose ordinal is below `before`, if that
launch has exactly one bound root. Otherwise it returns not-ok:
- no earlier launch;
- the nearest earlier launch has no binding;
- the nearest earlier launch has conflicting roots.

It does not reach past the nearest earlier launch, because an older generation
behind an unbound one is stale history, not the replaced conversation.
`CurrentLaunch` and its writers in `store.go` keep the newest-launch meaning.

**Where it applies.** `QueryResumeTargetContext` applies the fallback only when
`ResumeTargetForRuntimeLaunch` returned `FreshRequired`. That is a chosen-id
launch whose root file a complete listing proves absent, meaning its agent
never took a turn.
- An incomplete listing stays provisional, so there is no fallback on missing
  evidence.
- A resume-origin launch, a legacy launch, or a codex launch with no origin is
  unchanged.
- `launcher.ReadLedger` (restart decisions) calls `ResumeTargetForRuntimeLaunch`
  directly and is untouched, so pair's restart semantics don't move.

The fallback result is the earlier generation's established target. It is
marked with `FellBackFrom` (the shadowing launch's ordinal) for diagnostics.

**Reasons (item 1).** `ThreadEvidence.ParkedRefusal ResumeDiagnosticCode`
records why the resume proof failed, set in the evidence pass
(`actionableinventory.go:~961`):
- A resolver error with no `ResumeDiagnosticOf` code is an IO failure. The proof
  stays `ProofUnresolved` and the row reads `checking…`. This fixes the existing
  hole where IO failure read as "unbound"; `relaunch.go:123` already guards it.
- A refusal code, or a nonempty `bindingResumeDiagnostic`, is recorded.

`ClassifyThread` maps the recorded code after the parked-proof check and before
the receipt and session-gone fallbacks:

| Diagnostic | Reason (slug) | Label | Repair |
|---|---|---|---|
| `ResumeBindingAmbiguous` | `conversation-ambiguous` | "two conversations claim it — reboot" | reboot |
| `ResumeBindingUnbound` | `no-turn` | "no turn taken yet — reboot" | reboot |
| `ResumeBindingProvisional` | `unconfirmed` | "conversation not confirmed yet — retry after a turn" | retry |
| `ResumeBindingRootMissing`, or none | unchanged: `binding-lost` with a receipt, else `session-gone` | unchanged | — |

The defining words are "two conversations", "no turn" and "not confirmed".
None contains another reason's word: "binding", "agent", "path", "session
gone", "never started" and the rest.

`ArchivableState` is unchanged: the new reasons are unusable states, archivable
as today. "Recoverable" is expressed by reboot and retry, not by blocking
archive (YAGNI).

**ARCH-ORDER.** No state is held between events: the fallback and the mapping
are pure functions over one read. **ARCH-FUNERAL:** nothing new is created;
there are no new artifacts. **ARCH-SECURE:** no new trust boundary. **ARCH-MOCK:**
the fallback test uses the real `QueryResumeTargetContext` over a fixture
ledger and a native directory, through the existing runtime fakes.

## Core concepts

| Name | Lives in | Status |
|------|----------|--------|
| `PreviousEstablished` | `cmd/internal/sessionledger/record.go` | new |
| `ResumeTarget.FellBackFrom` | `cmd/internal/sessioninventory/query.go` | modified |
| `ThreadEvidence.ParkedRefusal` | `cmd/internal/couchcore/actionableinventory.go` | modified |
| `bindingFailureReason` | `cmd/internal/couchcore/actionableinventory.go` | new |
| `ReasonConversationAmbiguous`, `ReasonNoTurn`, `ReasonUnconfirmed` | `cmd/internal/couchcore/threadreason.go` | new |

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `QueryResumeTargetContext` | `cmd/internal/sessioninventory/query.go` | modified | ledger file and native session listing |

---

### Task 1: `PreviousEstablished` (pure)

**Files:** `sessionledger/record.go`; test `sessionledger/record_test.go`.

- [ ] Failing tests, `TestPreviousEstablished`, a table covering:
  - an earlier launch with one root returns it;
  - no earlier launch returns not-ok;
  - an earlier launch with no binding returns not-ok and does not skip further
    back;
  - an earlier launch with two roots returns not-ok;
  - another owner's launches are ignored;
  - the 2026-09-08 shape (launch 26 bound, launch 29 bound to the same root,
    launch 31 unbound) returns launch 29's root.
- [ ] Implement over the same owner filter and binding matching as
  `CurrentLaunch` (reuse its helpers, ARCH-DRY).
- [ ] `go test ./cmd/internal/sessionledger -count=1`.

### Task 2: the owner query falls back

**Files:** `sessioninventory/query.go`; test
`sessioninventory/resume_target_test.go`.

- [ ] Failing tests:
  - `TestAnUnturnedFreshLaunchDoesNotHideThePreviousConversation`: a ledger
    with an established binding, then a chosen-id launch whose file is absent
    under a complete listing. `QueryResumeTargetContext` returns `Established`
    with the old root and `FellBackFrom` set.
  - Unchanged cases: the chosen file exists (usable chosen id); an incomplete
    listing (provisional); no earlier binding (`FreshRequired`, unbound).
- [ ] Implement:
  - `readOwnerLaunch` also returns the parsed records, or a sibling returns
    them; read the ledger once.
  - When `result.FreshRequired`, call `sessionledger.PreviousEstablished`. If ok,
    the result is `ResumeTargetForLaunch(earlier)` with `FellBackFrom =
    current.Launch.Ordinal` and the diagnostics carried over.
- [ ] Run `go test ./cmd/internal/sessioninventory ./cmd/internal/launcher -count=1`.
  `TestRunRestartUnmaterializedChosenIDStartsFresh` must stay green, because the
  restart path is untouched.

### Task 3: name the refusal (couch)

**Files:** `couchcore/threadreason.go`, `couchcore/actionableinventory.go`,
`couchcore/actor_actions.go`, `couchcore/slotstart.go`,
`couchtty/menu.go` (`unusableThreadNotice`). Tests: `threadreason_test.go`,
`classify_test.go`, and the classify shapes that pin reasons.

- [ ] Failing tests:
  - `classify_test.go` shapes: each binding code maps to its reason, with and
    without a receipt (D2). `root-missing` still gives `binding-lost` with a
    receipt and `session-gone` without.
  - **IO-error shape:** a resolver error with no code projects
    `unusable/unknown`, not `no-turn`.
  - `TestEveryReasonIsProducedBySomeShape` and the defining-word test cover the
    three new reasons.
- [ ] Implement:
  - Constants, `AllThreadReasons`, `Label()`, and the defining words.
  - `ParkedRefusal` and the evidence-pass change.
  - `bindingFailureReason` and its call in `ClassifyThread`.
  - Sweep every switch over `ReasonBindingLost` or the reason set:
    `SwitchableState`, the slot resume offer (`actor_actions.go:43`) and its
    mirror test, the lost-slot notice (`slotstart.go:226`), and the Enter
    notice (`menu.go:1276`, which uses the repair from the table above).
  - Update the tests that pin a slug: `menu_render_test`, `recover_action_test`,
    `recoverplan_test`, `slot_operation_test`, `parkedproducers_test`.
- [ ] Run `go test ./cmd/internal/couchcore ./cmd/internal/couchtty ./cmd/internal/couchcmd -count=1`.

### Task 4: close

- [ ] Atlas: `atlas/couch.md`'s reason list (`:1270` area) gains the three
  reasons. The resume section gains the fallback rule, with a pointer to
  `PreviousEstablished`.
- [ ] Full verification per the repo's test notes (unsandboxed; known failures
  checked against the merge base).
- [ ] `sdlc close --issue 214 --verified '<evidence>'`.
