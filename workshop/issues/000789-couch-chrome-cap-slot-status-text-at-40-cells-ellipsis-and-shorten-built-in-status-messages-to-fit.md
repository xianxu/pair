---
id: 000789
status: working
deps: []
github_issue:
created: 2026-10-21
updated: 2026-10-22
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'a5dba37ce0ebc1e4e90f2cd6bbe6d1616c736532' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-22T06:59:35-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:4
    worktree: /Users/xianxu/workspace/worktree/pair-slot4/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "5b879924", done: "e2fe8944"}
---

# Couch chrome: cap slot status text at 40 cells (ellipsis) and shorten built-in status messages to fit

## Problem

## Spec

Operator, accepting pair#779 (10-21 19:1x): "one caution is we should limit the status text to be certain length,
otherwise, a long text will wrap into 10 lines, destroy the layout? maybe limit status message to be 40 chars,
maybe another task to clean up messages showing up there to be within that limit".

pair#779 caps the tab bar at two rows, but a slot's status text in the chrome can still be arbitrarily long and
crowd out every other tab. Two parts:
1. **Display cap:** the chrome renders a slot's status text at most 40 display cells (rune-width aware), cut with
   `…`; the full text stays available where it already is (peek, the switcher or hover). One pure function,
   unit-tested, used by every place the chrome draws status text.
2. **Producer clean-up:** list the status-text producers (agent status lines, wait notices, lifecycle labels) and
   shorten their messages to fit 40 cells so the cap rarely bites; a test fails if a built-in status string exceeds
   it.

## Done when

- A 200-character status renders as 40 cells ending in `…`, with layout unchanged (test); full text still reachable.
- No built-in status message exceeds 40 cells (test over the catalog).

## Plan

- [ ] Add a pure 40-cell sanitized/ellipsized chrome projection, used for both notice fields; retain the source text and use the existing Ctrl+Y copy action for full capability diagnostics too.
- [ ] Inventory notice producers, shorten built-in chrome instructions, and enforce their literal/template contribution with an AST-derived catalog; dynamic names, paths and diagnostic suffixes retain their full source text.
- [ ] Pin ASCII/CJK/combining/control inputs, shared legacy/two-row rendering, full-text copy and source-catalog negative controls; run recorded checks, close and land.

## Log

### 2026-10-21

## Revisions

### 2026-10-22 — implementation scope

The cap counts display cells including the ellipsis, after row sanitization. It applies to Notice and CapabilityNotice in the shared renderer, never to actor labels, menu details, feed storage or error records. Catalog enforcement counts built-in literal/template text; dynamic labels/errors may exceed 40 and are bounded only in their chrome projection. Ctrl+Y retains full feed text and copies the currently displayed capability diagnostic when no selected operation/error overrides it.

### 2026-10-22 — design and producer inventory

ARCH-DRY/ARCH-PURE: reuse rowtext.Sanitize/Fit and textwidth.Width behind one pure ChromeStatusText function; both legacy and composed status renderers share renderStatusCells. Alternatives were truncating producers/storage (loses detail), or relying on terminal-width clipping (does not enforce 40); neither meets the contract. Work is O(B) in input bytes per projection, with no IO, timers, retained state or persisted changes. Existing status packing counted tests retain their bounds; counted input-size tests pin the shared projection's scan bound at N/2N. ARCH-SECURE retains the existing sanitizer before width measurement. ARCH-MOCK/STATE/FUNERAL/EVOLVE add no external effects, lifetime owner or persisted format; copy reuses the current clipboard seam.

Producer inventory: Console.setNotice and ExitNotice; cast capabilityNotice and driveNotices; broadcast/pointer/webcast/cast-code instructions; action/lifecycle refusals. Couchcmd also forwards lifecycle/supervision/stale-session/archive/renice/crash diagnostics as arbitrary full-text Notice values. Menus and detailed diagnostic records remain unrestricted. A source-derived test covers direct setNotice/Notice.Body templates plus named helper producers, with negative controls; runtime substitutions use the display cap. This is presentation policy, not a lint ban on diagnostic length.

Catch-up generated a separate merge-check.yml change (SIGPIPE-safe metadata grep); preserved in stash 'pair789: preserve catch-up generated merge workflow separately', excluded from this issue.
