---
id: 000550
status: codecomplete
deps: []
github_issue:
created: 2026-10-14
updated: 2026-10-14
estimate_hours:
card_mirror: '2b58318890588b3da1bb9bfb3a48d73580ee2540' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-14T08:52:41-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "dd1c99a1", done: "3847ab18"}
actual_hours: 0.60
---

# Qoder slots never connect to Couch: wrapper creates no message endpoint and never settles (pair:4, 10-14)

## Problem

First Qoder trial, 2026-10-14. The operator switched idle pair:4 (tag 1-pair-6) to Qoder (`qoder --dangerously-skip-permissions --session-id …`, "Auto Model", YOLO). Qoder runs and its screen renders (`couch --peek` shows source `recording`), but the slot never joins Couch:
- `couch --actors` doesn't list pair:4; `--peek` says "live tail: unavailable: no wrapper is connected for this thread"; `--send-to` can't reach it.
- The wrapper (pid 14097, started 07:53) created **no** `wrapper-<pid>-*.sock` in `/tmp/pair-message-501/`, while every Codex/Claude wrapper has one.
- After a Couch restart (08:42, main 895c7868) the slot still didn't connect; `couch --relaunch pair:4 --confirm --same-binary --force-unknown` restarted it (new session 📁1-243) but failed "unready: … did not settle within 2m0s", still not connected.
- `agent-ready-1-pair-6-qoder.json` exists (launch_ordinal 63, pid 14123); wrap events show `start`, `child-start`, `agent-restart-request`, `proxy-defer-close`, `winsize`, no idle/settled.
Qoder's first-run folder-trust dialog also appeared (accepted at the operator's direction).

## Spec

Find why a Qoder wrapper never brings up its message endpoint / registration and never reports Settled: likely the Qoder adapter's readiness or idle detection (screen markers differ: "Type your message or @path/to/file", footer "Auto Model · ctx …"), gating endpoint creation. Fix in the Qoder adapter, with a recorded-screen test, and check the trust-dialog case (first run in an untrusted folder).

## Done when

- A Qoder slot appears in `couch --actors`, settles when idle, and receives `couch --send-to` messages; `--peek` shows source `live`.
- Recorded-screen tests cover Qoder's idle prompt and its trust dialog.
- Every AgentInventory entry has an explicit receiver policy; enabled receivers require a captured empty-composer parser, so new agents cannot silently miss admission.

## Plan

- [x] Freeze bounded Qoder 1.1.71 idle and 1.1.66 folder-trust repaint fixtures from pair:4's existing raw recordings, with source metadata; exclude conversation bodies and credentials.
- [x] Add failing recorded-screen and generated-input tests for Qoder peer registration, empty versus occupied multiline composer, placeholders, trust/permission/question dialogs, slash commands, settling, and exactly one bracketed paste plus bare-CR submit.
- [x] Admit Qoder to the receiver profiles; share its ruled-box specification between Return recognition and peer parsing (ARCH-DRY). Derive its text origin and empty cursor position from captured geometry; recognize only the captured placeholder presentation, keeping drafts occupied (ARCH-PURE).
- [x] Exercise Qoder through the existing stateful harness fake and broker/socket delivery seam, including trust -> idle -> paste -> submit -> clear and interruption by operator input/picker (ARCH-MOCK, ARCH-ORDER). Reuse existing delivery transitions and runtime teardown; add no new lifecycle or artifacts (ARCH-FUNERAL).
- [x] Update the Couch atlas support statement. Run targeted wrapper tests, make -k build, and the suite through sdlc local-check run --name suite against the pushed final head. Terminal parsing stays bounded by the existing snapshot dimensions and existing delivery deadlines (ARCH-CONSTRAINTS); captured external screen bytes fail closed on unknown shape and fixtures carry no secrets (ARCH-SECURE).
- [x] With hub authorization only, relaunch pair:4 on the built selected head; prove actor registration, Settled, live peek and one harmless peer receipt. Close via the mandatory SDLC review and report the outcome to ops:0 (ARCH-PURPOSE).

## Log

### 2026-10-14
- 2026-10-14: closed — Required sdlc-local/suite passed on b54b991dc3824a007a5520e186f35c1a3c4f38ac (exit 0, 10m36s); wrapper and focused race passed. Hub live Qoder smoke passed registration, live peek, receive/reply and settling on that clean build; stage cleared. Recorded screens and real endpoint/broker tests pass. BR-1 fixed by sweeping README receiver support and submission prose against production; only documentation changed after the tested head.; review verdict: SHIP

- Refresh completed on main-slot3; design base from git rev-parse HEAD: 41861430868ce99ad949ce01b1154b61ec584077. Claimed and started #550 in pair:3.
- Root cause: peer_runtime.go explicitly admits only codex and claude, returning before endpoint creation for qoder. peer_composer.go likewise has no qoder case and assumes column 2 for all input origins. Endpoint startup is not gated by observed screen readiness; Qoder lacks a receiver profile entirely.
- Live inspection: couch peek pair:4 still reports source recording, no connected wrapper, and an idle YOLO composer. Current raw stream identifies Qoder 1.1.71; parked 20261014T084549 stream preserves the first-run folder trust dialog from 1.1.66. No relaunch or input sent.

## Revisions

### 2026-10-14 — root cause established

The initial readiness-gate hypothesis resolves to missing Qoder receiver admission and peer-composer support. Add the receiver profile while retaining unknown-screen refusal and proving trust-dialog blocking; reuse the existing Return geometry and delivery lifecycle. Implementation awaits hub plan approval.

### 2026-10-14 — approved design

Hub ops:0 approved commit 2c62b3c20e32e1472e59bc9d654ff7800b43596d (receipt 3aaacf4f-8c1d-42fc-b7c5-3393d4b8951b). Additional requirement: assert every AgentInventory entry has an explicit peer receiver policy, so a future harness cannot silently fall through admission. Existing non-receiver harnesses retain explicit disabled policies pending their own captured composer support; this issue adds Qoder only. Pair:4 relaunch and harmless live message remain reserved for explicit hub authorization after build/test evidence.

### 2026-10-14 — implementation and regression evidence

Qoder receiver policy is enabled; Return and peer input share qoderComposerSpec. Input begins at column 3. The placeholder requires the captured wording, empty origin cursor with painted background, and hint foreground matching that background (theme-relative; no hardcoded RGB). Bright authored placeholder words, multiline input, slash/shell commands and picker glyphs cannot read empty. All six AgentInventory entries now have explicit policies; the parity test refuses future omissions.

Recorded trust opens the existing Enterselect overlay marker; bare operator Enter consumes it before idle output, matching production. Private registry test executes startPeerRuntime and reads live tail through its endpoint. The existing real broker/socket test now runs for Qoder as well as Claude. Tests prove one bracketed paste, one bare CR, clear confirmation, settling, operator input/draft/picker refusal. Mutation disabling Qoder admission and its composer case correctly fails startup, idle, settle and broker delivery tests; restored focused tests pass. make -k build passed. Full wrapper and race checks are running; full suite and live smoke remain pending. No pair:4 input or relaunch occurred.

- Full wrapper package passed: go test ./cmd/internal/wrapcmd -count=1 (80.804s). Focused race check passed: go test -race ./cmd/internal/wrapcmd -run 'TestPeer(Qoder|ReceiverInventory|RuntimeBroker)' -count=1. Enabled receiver coverage additionally requires a captured empty-composer parser for every admitted agent; deliberately disabled agents remain explicit.

### 2026-10-14 — live smoke accepted by hub

Hub ops:0 confirmed the authorized live smoke PASSED (receipt dfe32439-0383-45b8-a92f-983bfedd2779): pair:4 was relaunched onto staged b54b991dc3824a007a5520e186f35c1a3c4f38ac; Qoder appeared in actors as active, peek reported source live, received the hub's harmless message, replied to the hub (receipt prefix 406d4cb5), and settled. Staged pair:0 bin/pair embedded that revision with modified=false; SHA256 b0f73760fae0a157d115069b1ba30429d70d1e1cf1a2f66ece46d57b4af74dcb.

Ran sdlc stage --clear after hub authorization; pair:0 returned to main, releasing it for #528's smoke. Required suite on b54b991dc3824a007a5520e186f35c1a3c4f38ac remains queued behind machine-wide suite holders. No further source changes are needed for live conformance.

### 2026-10-14 — required suite passed

sdlc local-check run --name suite passed on b54b991dc3824a007a5520e186f35c1a3c4f38ac, exit 0 in 10m36s, after a 380-second machine-slot wait. Success recorded as sdlc-local/suite. Full log has no anchored FAIL/--- FAIL/make Error entries; SHA256 fd0b0445004a2f4b1a81529d2f49f97dce52df802861a18c3a98a34b4798dce8. Subsequent commits are documentation-only: live-smoke record and placeholder wording clarification.

### 2026-10-14 — close review BR-1

Round 1 found no correctness defect; focused tests and a fresh full wrapper run passed (136.264s), both fixture hashes matched, and ARCH-DRY/PURE/PURPOSE passed. BR-1 identified README's receiver paragraph as still listing only Claude/Codex. Swept README receiver-support references plus atlas/couch.md and peer_runtime.go. README now includes Qoder and its recorded/native evidence. The same paragraph's old echoed-text validation claims contradicted the existing fixed-delay delivery path, so it now describes bracketed paste, delayed submit and clear/turn confirmation; no executable behavior changed. The ledger blocked finalization despite FIX-THEN-SHIP, so close must rerun to dispose BR-1.

### 2026-10-14 — README documentation revision

Extend the documentation checklist to sweep README's receiver-support and submission paragraph alongside atlas/couch.md. This resolves the user-facing-surface-documentation class; all changed prose was checked against peer_runtime.go and dispatchPeer/advancePeer. The required passing suite remains applicable because only documentation changed after its tested head.
