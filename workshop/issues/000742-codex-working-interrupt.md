---
id: 000742
status: working
deps: []
github_issue:
created: 2026-10-20
updated: 2026-10-20
estimate_hours:
collaboration-mode: collaboration
card_mirror: '693e630e0e85f0b4c7dbc5b23e1e1968a0757b32' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-20T12:53:03-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
---

# Couch interrupt refuses a working Codex turn

## Problem

Follow-up to #717 (landed in PR #483): the morning live check reported couch --interrupt refusing a Codex turn that was working. The existing admission proof combines reducer-owned working state, recognized composer, current viewport interrupt hint, and automatic-input ownership. Diagnose which proof is missing from the real Codex path; do not weaken idle/panel/picker or second-interrupt protections to make the working case pass.

## Spec

Use the agent's current coherent work evidence, defined once per agent. For Codex, extend the existing RecognizeCodexWorking pure recognizer to the observed Working and Waiting for background terminal status forms; locate the native status relative to the qualified native composer, not the physical viewport bottom. Both the lifecycle observer and interrupt admission consume this recognizer. Claude keeps reducer-owned Working(), no stop grace, a recognized composer and its current interrupt footer.

An incomplete redraw does not emit a Codex stop transition. At admission and immediately before the one write, Codex's positive current work proof authorizes interrupt without requiring cached notification activity flags. Project a positively recognized working Codex screen as turn-active even if the notification lifecycle missed it. The existing picker, complete-frame, operator/peer/automatic-input ownership and lease fences remain. A later fresh complete *idle composer* must be observed to confirm completion; absence of working evidence on an unknown/picker/panel screen never confirms. There is no retry of a possibly written interrupt.

Actions: interrupt once; inspect a refusal or unconfirmed result; request a new interrupt only on a newly observed working turn.

| Internal state / failure mode | Visible state | Recovery action | Possible loss |
|---|---|---|---|
| Recognized working Codex or Claude, exclusive owned input | One Ctrl+C, awaiting turn end | Inspect final result | Current turn is intentionally interrupted |
| Idle, unknown, partial frame, picker/panel or input ownership conflict | Refused; nothing sent | Inspect current slot | None |
| Turn ends or ownership changes before dispatch | Refused/cancelled before write | Inspect current slot | None |
| Ctrl+C written but no fresh idle composer before deadline | Unconfirmed; never resent | Inspect current slot | Work may already have been interrupted |
| Fresh idle composer after the write | Confirmed | Inspect result; second interrupt refuses | Current turn stopped; conversation remains |

ARCH-DRY: Codex's one pure recognizer serves lifecycle and drive proof, reusing codexComposerBounds. ARCH-PURE: retain native capture/PTY IO in tests and wrapper adapters. ARCH-CONSTRAINTS: one bounded viewport/composer scan per completed observation; no new goroutine, cache, store or growing state. ARCH-STATE: keep the existing drive reducer and one input lease; require positive idle completion rather than infer completion from an absent work signal. Creates nothing durable beyond existing bounded receipts because this repair changes only recognition and confirmation.

## Done when

- Reproduce the reported Codex working-turn refusal with captured evidence and an automated regression that fails before the fix.
- A current working Codex turn admits exactly one Ctrl+C, then confirms only the observed turn end. A second interrupt after completion refuses and sends nothing.
- Idle, stop grace, panels, pickers, stale ownership and competing input continue to refuse without a write.
- Targeted tests, the recorded local check, boundary review and a hub-coordinated live Codex check pass before landing.

## Plan

- [ ] Capture coherent owned native Codex Working and background-wait frames; record the failing admission flags and exact composer/status relationship. Do not infer unobserved geometry.
- [ ] Add regressions through handleChunk and the production mailbox/writer with no manually set Codex activity flags: top-anchored and tall composer, both status forms, complete/incomplete redraw and stale lifecycle observations. Assert one Ctrl+C, fresh positive idle confirmation, then second-interrupt refusal.
- [ ] Repair the shared recognizer and per-agent admission/confirmation proof. Preserve Claude stop-grace and all negative/ownership cases. Mutate each corrected guard and require its oracle to fail.
- [ ] Run targeted packages/race and reader inventory; document the per-agent work proof in atlas. Push PR and record affected check; stage the tested head for hub-owned live Codex sleep/interrupt/second-refusal check, then close and merge.


## Log

- 2026-10-20 — Dispatched by ops:0 after #726 landed, before #733. #717 is already done and its old branch must not be reused. Current archived #717 Log describes BR-4/BR-6 fixes but lacks the reported failed live-check detail; asked ops:0 for target/time/refusal while tracing the production path.

- 2026-10-20 12:56 PDT — ops:0 supplied the morning evidence: target parley.nvim:4, Codex freshly switch-agent launched around 09:00 on f3c597f8 wrapper; requested sleep 60. UI showed Working (3s, esc to interrupt), then Waiting for background terminal (15s, esc to interrupt), 1 background terminal running. couch --interrupt refused with the nothing-sent/no-working-turn message. Source audit: Codex already has a visual observer; its recognizer only accepts Working in fixed physical bottom rows, so Waiting is a confirmed unsupported status. A top-anchored composer plus work status is also missed by fixed rows. No evidence yet assigns the first Working refusal to one guard.
- Existing Codex working.raw is a 154-byte status delta without a composer or complete synchronized frame; it cannot establish a composer-relative recognition contract. Capturing a fresh owned native PTY run before defining geometry. ARCH-DRY: reuse one Codex work recognizer for lifecycle and interrupt evidence, and preserve Claude working/stop-grace guards.

- 2026-10-20 12:59 PDT — Owned native Codex 0.163.0 capture reproduced initial Working refusal. In a 120x38 complete frame the status is at zero-based row10, the qualified bold composer at row13 (two blank rows between), footer row15. Diagnostics: composer known, picker false, lifecycle Generation0/Active=false/ActivitySeen=false, Interruptible=false. Coherent frames alternate • Working and ◦ Working; exact bullet matching is a second false-negative. Starting MCP servers appears at the same geometry and must stay excluded by wording. The owned fresh agent executed sleep20 and replied done; no background-wait frame appeared within the bounded70s capture. Child/reader teardown joined, temporary tests and copied login removed. Evidence under /Volumes/TestScratch/tmp/pair-slot2/codex-interrupt-vxjpuco9: capture.raw, working-prefix.raw, working.screen, coherent-statuses.txt, diagnostic.txt, version.txt. Existing targeted interrupt/Codex-recognizer tests pass and miss the combined production path.

## Revisions

- 2026-10-20 12:59 PDT — Fresh capture qualifies the proposal: accept both captured •/◦ spinner markers in the explicit Working/background-wait status grammar; keep MCP startup and arbitrary text excluded. This is evidence for the same per-agent recognition repair, not a broadened interrupt action. Design approval requested from ops:0 before implementation; no production/test edits made.
