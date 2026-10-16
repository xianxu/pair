---
id: 000604
status: working
deps: []
github_issue:
created: 2026-10-15
updated: 2026-10-16
estimate_hours:
card_mirror: '22b7204910faa5d41e6b1facc4ede86de5fe925c' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-16T15:06:21-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
---

# Couch switcher: Ctrl+Y 'copy last notice' never reaches the clipboard (OSC 52 only); use the system clipboard locally

## Problem

The operator (2026-10-15): "the ctrl+y to copy last notice in switcher never worked for me." Couch's console
(`couchtty/console_menu.go` `copyLastNotice`) copies through the terminal's OSC 52 clipboard escape
(`ESC ]52;c;<base64> ESC \`) and then says "Full notice copy requested; your terminal may not support it." On the
operator's setup (Ghostty, with Couch's console possibly behind a multiplexer layer) the clipboard never receives it, and
the UI can't tell. Tests only assert the escape bytes were written, never that a clipboard got them.

## Spec

- On macOS with a local console, copy with the system clipboard directly (`pbcopy`, or NSPasteboard via a tiny
  helper), and report success only when that returns OK; keep OSC 52 as the fallback for remote or unknown setups,
  labelled as unverifiable.
- Find out why OSC 52 doesn't reach Ghostty here (a multiplexer not passing it through, or Ghostty's
  `clipboard-write` setting), and note it in the atlas.

## Done when

- A test with a fake clipboard seam asserts the notice text reaches it; the operator confirms Ctrl+Y works live.

## Plan

- [ ] Red first: drive Ctrl+Y against a private fake clipboard executable; assert complete multiline retained text and repeat copies, with no OSC 52 on confirmed success.
- [ ] Reuse clipcmd's command list (ARCH-DRY) via a context-aware local-copy method that distinguishes unavailable tools; preserve existing ClipboardCopy callers. Explicit SSH skips local copying. Bound the notice attempt to one second plus bounded pipe cleanup; failure falls back to Presenter.Copy without retry.
- [ ] Cover command failure, unavailable tool, cancellation/timeout, remote fallback, empty history and focused-child Ctrl+Y; no tests touch the actual clipboard. Keep copy feedback out of retained history.
- [ ] Update the existing atlas account with confirmed local versus unverifiable OSC results and the read-only transport/config evidence. Run the recorded full suite and close review; coordinate operator Ctrl+Y acceptance through the hub before landing.


## Log

### 2026-10-15

### 2026-10-16 — bounded design

Plan posted to the hub; proceeding under the bounded quick-flow dispatch. The existing method only writes OSC 52. A private executable on PATH supplies the clipboard seam, avoiding a second tool registry or test-only production API (ARCH-PURE/ARCH-DRY). One serial user action creates no durable state or background work; the subprocess is joined before returning. Timeout/failure may be uncertain, so fallback is an explicitly requested second transport, not proof that a local copy never happened; feedback claims confirmed copying only on successful exit.

Read-only investigation found installed Ghostty defaults allow clipboard-write and no local permission override; Zellij documents OSC 52 writes. The affected live chain/loss layer is not proven, so atlas must not blame either without measurement. Live acceptance remains required.
