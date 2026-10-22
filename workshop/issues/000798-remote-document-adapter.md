---
id: 000798
status: open
deps: [pair#758, parley.nvim#405, pair#759]
github_issue:
created: 2026-10-22
updated: 2026-10-22
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'e887995210693c60fbcc83a27aa001ad4a02fd5d' # card fields mirrored from issue-cards; edit via sdlc
---

# Remote document adapter and security integration

## Problem

The endpoint producer in #758 intentionally lands before parley.nvim#405. The original remote-session adapter and real-editor/security integration need their own issue after the producer contract lands, rather than keeping a partly landed issue branch open.

## Spec

Implement the registered-document DocumentBackend behind #759's typed security boundary using #758's authoritative endpoint bindings and parley.nvim#405's private nvim API. Support eligible hub session files and registered side docs; read/subscribe text and existing highlights, revision-safe paragraph edits re-wrapped at file width, save-and-trusted-submit, eligible link/parent/jump navigation, and bounded headless nvim fallback. No generic RPC/file/shell route. Revalidate stable document ID, editor incarnation and binding generation at use and final effect enqueue, including already-established channels after supersession. Preserve #759 Permit kill ordering, origin/audit provenance, native transcript headers and owned completion/stream cleanup. Keep process/socket/editor uncertainty explicit, with desktop attach taking priority over headless.

Plan before code; coordinate the exact adapter with parley.nvim#405 and #759. Operator-assisted real Access/iPhone edit/submit/navigation and desktop/headless handover is required before claiming the remote editor MVP works; #760 owns phone UI.

## Done when

- Stateful endpoint/editor/security integration tests exercise real production seams, generation/kill/revision fences and late completion/stream cleanup.
- Actual private nvim conformance verifies buffer fidelity/reflow/highlights/save+submit and desktop/headless replacement; no operator sessions used for fixtures.
- Operator-assisted integrated phone acceptance and native transcript/audit evidence pass before landing, with explicit dependencies on #760 for UI.

## Plan

## Log

2026-10-22: split from #758 by TL receipt79aac81f; #758 is the independently landable endpoint producer only. This issue preserves its original adapter/security-integration purpose and remains unclaimed pending prerequisites.
