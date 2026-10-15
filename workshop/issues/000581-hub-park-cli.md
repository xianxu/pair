---
id: 000581
status: open
deps: []
github_issue:
target: slot-recovery
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
severity: high
card_mirror: '9b1f65fa50ac480f01c5842dcd1abea6e3c4c210' # card fields mirrored from issue-cards; edit via sdlc
---

# Expose park transaction modes through the owning hub CLI

## Problem

The hub TL cannot drive an existing park transaction through Couch CLI. During the operator's 2026-10-15 acceptance, ariadne:5 retained an open requested park after the slow-commit refusal tracked in #580. The operator authorized the TL/worker to Retry the exact attempt, but `couch --internal park ariadne:5 --mode=retry` refuses at parsing: unknown internal operation park. The only supported path is selecting the row's UI park action. Agent-first lifecycle operation must not require the person to click a menu.

## Spec

Expose a supported hub CLI park verb covering normal, retry, recover and abandon through the existing lifecycle controller and transaction authority. Reuse existing exact slot resolution, owning-hub authorization, stop confirmation and operation receipts; do not create a parallel park protocol or make internal hidden entrypoints the public contract. Retry/recover/abandon must operate only on an existing exact transaction; a stale nonce/generation must not accidentally park a replacement or begin a new transaction. Keep the different effects honest: normal/retry/recover may stop the agent through existing lifecycle semantics; abandon gives up the transaction and leaves running resources occupied. Provide exact actionable CLI recovery advice in errors and help. A lost or uncertain reply must be inspected before retrying, with read-only receipt/state verification.

## Done when

- The owning authenticated hub can request park and select retry/recover/abandon for an exact slot without the UI, with appropriate explicit confirmation and operation receipt.
- Existing transaction nonce/generation is guarded at effect time; stale/no-transaction requests, wrong hub, changed owner, duplicate/in-flight requests and uncertain replies have tested safe behavior.
- Stateful tests pin the requested/awaiting/unknown/completed/abandoned transitions through the public CLI and real queue/controller seam; UI and CLI share semantics, with no duplicate quit or replacement launch.
- Help and failure advice name reachable exact commands; headless acceptance and a scratch-only operator script verify the path. No working slots are used for tests.

## Plan

## Log

2026-10-15: filed unassigned at hub request bef78679-9c7f-44da-8d8a-55d79e0559e3. Source inspected at staged e13111215270762629c27e860584429b7ae66a98: couchcore/ops.go declares park PresentationTUI with mode normal/retry/recover/abandon. couchcmd/cli.go admits --internal only for PresentationInternal; couchcore/slot_operation.go's shared remote operation list omits park. Actual CLI attempt exited2 before effects. No supported hub CLI command exists for the recovery advice printed by relaunch.

Couch.Park already uses parkJoinMode to convert normal to Retry when a transaction is open; menu Tab -> park -> confirm is therefore the current workaround. Do not change the native lifecycle or broaden stop authority to fill the transport gap. Separate from #580's post-commit1s cutoff. No implementation, slot mutation or new recovery attempt is authorized by this filing.
