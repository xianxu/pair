---
collaboration-mode: discussion
---

# Couch clipboard copy and retained failure implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy). Execute the warm-context diagnosis in the main session; delegate isolated tests or bounded exploration when useful. Steps use checkbox syntax. One atomic issue, one close boundary; no Mx milestones.

**Goal:** Ctrl+Y really copies the operator's notice, and every failed or unconfirmed attempt remains inspectable and retryable without losing the tool error.

**Architecture:** Reuse clipcmd for the external clipboard boundary, ReduceMenu for authoritative UI transitions, and Couch's existing bounded supervision writer for one typed copy event. Keep the copied notice private and separate from diagnostic metadata. Preserve the current bounded local-copy/OSC fallback policy until a measured actual-path failure justifies changing it.

**Tech stack:** Go, existing terminal Presenter/Console, native clipboard binaries, diagnosticlog, headless Console fixtures.

## Evidence and decision boundary

Known: the local tool error is discarded, stderr is not captured, feedback can be erased by menu events and no copy outcome reaches supervision.jsonl. The existing fake tests confirm only fake-tool behavior. One authorized worker-launched real-tool probe succeeded and restored the clipboard; the historical Couch failure is still unknown.

This plan does not claim a guessed PATH, sandbox, timeout or stdin cause. First expose the proven evidence defects. Then stage through the hub and observe the actual Couch command path. If that reveals a different copy mechanism or budget change, append a measured Revision and obtain TL approval before it. Do not close until the actual Ctrl+Y paste and real-tool conformance succeed.

Recommended: extend the current clipboard boundary and pure menu model. Making every menu error globally sticky would change unrelated workflows; a new clipboard daemon or speculative timeout increase would add a mechanism without explaining this failure. Neither alternative is justified.

## Core concepts

| Name | Kind | Lives in | Status |
|---|---|---|---|
| ClipboardCopyError | PURE | cmd/internal/clipcmd/clipboard_result.go | new |
| CopyFeedback / ReduceCopyFeedback | PURE | cmd/internal/couchtty/menu_copy.go | new |
| MenuState / ReduceMenu | PURE | cmd/internal/couchtty/menu.go | modified |
| copy supervision event projection | PURE | cmd/internal/couchcmd/supervision_copy.go | new |
| OSRuntime clipboard boundary | INTEGRATION | cmd/internal/clipcmd/runtime.go | modified |
| Console copyLastNotice | INTEGRATION | cmd/internal/couchtty/console_menu.go | modified |
| supervision copy wiring | INTEGRATION | cmd/internal/couchcmd/message_service.go | modified |
| stateful clipboard fake | INTEGRATION | cmd/internal/clipcmd/runtime_clipboard_test.go; cmd/internal/couchtty/console_notice_expiry_test.go | modified |
| real clipboard conformance | INTEGRATION | cmd/internal/couchtty/clipboard_live_test.go | new |

ClipboardCopyError describes tool basename/resolved executable, started/exit status, deadline/cancellation, bounded stderr and wrapped cause. A ClipboardCopyResult returned by the same executor retains tool/status on success too; the existing error-only method delegates to it. Formatting and projection are pure; os/exec execution remains in OSRuntime. errors.Is/As behavior and the legacy optional ClipboardCopy contract remain intact.

CopyFeedback is one tagged value owned by MenuState, updated only through ReduceMenu events. It owns at most one immutable payload reference for an explicit retry; it never enters a log DTO. A separate copy banner avoids losing either an ordinary operation error or the retained copy failure. Existing menu row budgeting and cursor/extents must account for the banner.

The fake executable persists copied bytes and invocation counts in an owned directory, and can model nonzero exit with stderr, a timeout, missing executable, cancellation and remote exclusion. The production boundary and fake share exec lookup/stdin semantics. No fake reads or writes the host clipboard.

## Operator projection and transitions

| Internal outcome/event | Operator sees | One recovery action |
|---|---|---|
| No retained notice | No notice to copy | Wait for a notice |
| Native tool exited successfully | Full notice copied | Paste |
| Native copy failed/unavailable; OSC write accepted | Local copy reason; terminal request unconfirmed | Ctrl+Y retries the same payload |
| Native copy failed and OSC failed | Copy failed with local and fallback reasons | Ctrl+Y retries the same payload |
| Retained failure + Escape | Copy banner dismissed; menu stays open | Normal menu use |

Escape is explicit dismissal, not an automatic retry. A success may clear with the next ordinary operator action. Failed/unconfirmed feedback survives navigation, filtering, refresh success/error, frame reconciliation, palette/activity events and leaving/reopening the switcher until Escape, successful retry, or Console shutdown. Ordinary operation notices retain their own existing lifecycle and remain visible alongside it. Do not make all menu errors modal.

The main Console loop executes one bounded clipboard attempt synchronously, outside c.mu. It cannot admit a concurrent Ctrl+Y attempt; no new worker/queue/timer is needed. The selected source payload is frozen for the attempt and explicit retries while its failure is retained. Closing Couch cancels the attempt through c.lifetime; a cancelled or late result must not resurrect a banner after shutdown. OSC write acceptance is always unconfirmed, including a dropped parent write; no automatic retry follows uncertainty. Clipboard assignment of the same value is safe on an explicit user retry.

## Operating envelope, trust and cleanup

- Keep the existing 1s local copy deadline initially; retain actual timeout versus exit information. Changing the budget requires measured actual-path evidence.
- Capture at most 4096 bytes of stderr with a writer that reports consumed bytes while discarding excess; report truncation. Do not use an unbounded bytes.Buffer or CombinedOutput. Resolve tools through the existing search order; record missing/remote/not-started distinctly from an exit status.
- The notice payload is private input. Never include it in diagnostic fields or argv. Retained retry state references one existing immutable notice string and clears on dismissal/success/shutdown. Bound and strip terminal controls before showing tool stderr on the operator screen. Supervision never receives free-form stderr, not even redacted stderr: publish only its total byte length and a short SHA-256 fingerprint of the bounded capture, alongside tool/exit/timeout metadata. Do not dump the process environment. The bounded raw capture remains local to the copy attempt/error; the callback DTO contains only its count/hash.
- Offer one clipboard_copy event per explicit attempt, including native tool/status/timeout, stderr length/hash and a separate OSC outcome. No event contains the copied notice. Use the existing supervisionFactLog nonblocking queue and diagnosticlog generation byte cap; no new file family, background writer or retention protocol. Keep the fact schema free of source prose and free-form tool output. Logging unavailability cannot stop Couch or claim copy failure; retain the UI evidence.
- All fake processes/files belong to their test temp directories and are reaped on every path. Live tests are explicit opt-in, Darwin-only where appropriate, and skip in the normal suite. They must run from a Couch-launched process context and use only an authorized marker, in-memory backup and conditional restore; no clipboard contents are printed.
- ARCH-DRY: one clipboard executor and one existing diagnostic writer. ARCH-PURE: outcome/UI projection are pure; exec and log offers are thin seams. ARCH-PURPOSE: actual copy, persistent failure, structured diagnostics and live conformance all remain acceptance obligations. ARCH-MOCK: stateful fake plus actual clipboard conformance. ARCH-CONSTRAINTS/ORDER/SECURE/FUNERAL: budgets, transition rules, private payload boundary and lifetime owners are stated above.

## Task 1 — preserve native tool evidence

Files: modify cmd/internal/clipcmd/runtime.go and runtime_clipboard_test.go; create clipboard_result.go and clipboard_result_test.go. Register any new production Go file in cmd/internal/artifactpath/manifest.go.

- [x] Test copyClipboard through a stateful private executable: adversarial process outcomes and stderr streams are checked against typed metadata, bounded capture and preserved errors.Is/As; exact stdin and legacy behavior remain controls. Test ClipboardCopyError.Error and clipboardStderrSummary with arbitrary controls/oversized output: sanitize the UI and prove the exported DTO carries only count/hash.
- [x] Run go test ./cmd/internal/clipcmd -run 'TestClipboard' -count=1; retain expected red assertions.
- [x] Implement the typed result and bounded stderr capture; no retry/cascade change after a selected executable fails.
- [x] Rerun the focused package and commit the proven boundary change.

## Task 2 — model retained copy feedback and diagnostics

Files: create cmd/internal/couchtty/menu_copy.go and menu_copy_test.go; modify menu.go, menu_render.go, console_menu.go and console_notice_expiry_test.go; create cmd/internal/couchcmd/supervision_copy.go and supervision_copy_test.go; modify supervision_facts.go and message_service.go. Register new production files in the artifact manifest.

- [x] Test ReduceCopyFeedback/ReduceMenu over adversarial operator/background event sequences: the pure tagged state retains failure and original payload until explicit dismissal or successful retry, without erasing ordinary operation errors.
- [x] Test copyLastNotice through production Console keys and a stateful executable: captured stdin and callback DTO prove payload selection, independent failure causes and one observation per attempt. A controllable child-start handshake followed by lifetime cancellation orders shutdown deterministically; no sleep-based completion race. Test RenderMenuView with both banners at narrow/short dimensions: drawn extents and cursor remain within the fitted block.
- [x] Add typed MenuEvent copy outcomes and the CopyFeedback reducer; route Console results through that authority instead of assigning a notice directly. Render the independent retained banner and adjust budgets.
- [x] Inject a narrow Console outcome callback at the composition root; offer one typed clipboard_copy event through the existing supervision log. Test real service/Console wiring with an in-memory fact sink: clipboardCopyFact must encode tool/exit/timeout, stderr length/hash and separate fallback outcome, never free-form stderr or the copied payload.
- [x] Run focused clipcmd, couchtty copy/menu and couchcmd supervision tests plus artifactpath classification; commit.

## Task 3 — actual-path diagnosis and conformance

Files: cmd/internal/couchtty/clipboard_live_test.go; README.md; atlas/architecture.md. Issue Revisions record any measured mechanism/budget correction.

- [x] Add an opt-in real clipboard conformance check through the production Console Ctrl+Y path in a Couch-launched context. Save/restore only as authorized; verify bytes without printing them. Normal suite never touches the real clipboard.
- [x] ~~Ask the hub for staging: sdlc stage --build only on its word, supply exact git rev-parse HEAD and binary revisions, and show the persistent diagnostic/log path plus retry/dismiss steps.~~ Superseded by the approved rollout Revision: no staging or pre-close real-paste claim; the hub holds the actual result for the next restart.
- [x] ~~Observe the actual tool outcome from the real Couch process. If it fails, diagnose the recorded tool/status/stderr and demonstrate a failing regression for that specific cause before correcting it. If it cannot be reproduced, report that limit and obtain the TL's acceptance decision; do not infer the historical cause from the worker probe.~~ Superseded by the approved rollout Revision: no staging or pre-close real-paste claim; the hub holds the actual result for the next restart.
- [x] ~~Operator verifies real Ctrl+Y -> paste; force a private/fake missing-tool failure and verify persistent banner and structured log. Clear the stage when the hub is finished.~~ Superseded by the approved rollout Revision: no staging or pre-close real-paste claim; the hub holds the actual result for the next restart.
- [x] Update README/atlas around success, unconfirmed fallback, persistent error, explicit retry/dismiss and event location; retain original OSC52-loss uncertainty unless this investigation resolves it.

## Task 4 — verification and close

- [x] Push the implementation and run the declared recorded suite once. For any failure, follow the worker flake rule; fix introduced defects and retain failed attempts.
- [ ] Record all acceptance evidence and lessons; close through sdlc with measured actuals and mandatory review. No separate duplicate boundary review.
- [ ] On SHIP, merge main and land with sdlc merge --yes; report actual Git merge hash/PR to hub. No permission/gate bypass.

## Revisions

### 2026-10-17 — approved scope and rollout

TL approved the full-flow single-pass design (dd9c7097-9be3-4fca-9cc3-c7046cde424a), then clarified gate sequencing (00b6f3ff-12f2-4f3e-8407-9da4c629a989). Task 3 staging and pre-close real-paste requirements are superseded: no pair:0 staging. Implement and test instrumentation, persistence and the opt-in conformance harness, then close and land normally. The hub owns the actual Ctrl+Y acceptance at the next natural restart: real paste, or the measured logged cause filed as a follow-up. The historical failure is not yet fixed; do not claim it is. Keep the 1s/OSC policy unchanged. The issue's revised Done when is the landing contract.

### 2026-10-17 — plan-quality refinements and simpler log privacy

PQ-1 identified that full-payload redaction cannot protect truncated/transformed echoes. TL receipt 8429745d-ecb6-40e5-9325-a6282e0141c0 chooses a simpler strict boundary: no free-form stderr in supervision at all. The operator screen may show bounded sanitized stderr; logs get tool/exit/timeout and stderr byte length plus a short hash only. No overlap or echo heuristics. PQ-2 is addressed by function-level adversarial strategies above, including controlled cancellation ordering and the production Console-to-log seam. These refinements supersede earlier references to logging bounded stderr.
