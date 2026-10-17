---
id: 000641
status: working
deps: []
github_issue:
created: 2026-10-17
updated: 2026-10-17
estimate_hours:
collaboration-mode: collaboration
card_mirror: '777c77d3518d43fca61b9888a2721c34763e4478' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-17T09:59:02-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
---

# Qualify Claude PR footer background work

## Problem

#632 preserves unknown for unfamiliar native footer fields. A real Claude slot with a free composer and background shells includes a native PR field, so this valid variant currently remains unknown and keeps the previous fading behavior.

## Spec

Qualify the native Claude PR footer variant from the exact captured input below, retaining #632's strict complete-footer grammar and independent admission policy. Do not accept arbitrary fields or infer work from body/completion prose. Obtain complete native terminal-frame/geometry evidence before expanding the production recognizer; an unfamiliar or malformed field remains unknown. No protocol, Settled, admission or timer change is needed.

Inline text fixture, captured read-only with Couch peek from pair:2 on 2026-10-17 at about02:13 PDT (tag1-pair-8, agent Claude; free composer, two background shells). This is the exact rendered footer text, not original VT bytes; native CLI version and original synchronized VT frame were not retained. Preserve that provenance when turning it into a recorded test fixture:

```json
{"fixture":"claude-pr-background-footer","source":"couch peek pair:2, live rendered terminal","footer":"⏵⏵ bypass permissions on · PR #387 · 2 shells · ← for agents · ↓ to manage"}
```

The same read-only observation showed an empty composer above the footer and the native completion line reporting two shells still running. Those corroborate the observed state; the footer is the fixture input. Repository policy is text-only, so evidence is preserved inline rather than as a file attachment. No unrelated transcript content is copied.

## Done when

- The exact captured PR/footer variant is preserved as a regression fixture with its provenance and qualified composer geometry; complete native capture evidence is recorded before parser expansion.
- A behavioral red test demonstrates the valid PR footer currently returns unknown, then passes with active background evidence while the composer remains free for a successful peer receipt.
- Malformed/truncated PR fields, arbitrary prefix/suffix fields and counters outside the qualified footer stay unknown. Existing no-shell, zero, completion/lifetime and older-transport behavior remains covered.
- Qualification and native coverage limits are documented; no loose substring fallback or admission change is introduced.

## Plan

- [ ] Qualify the preserved native archive before parser changes: native version, 94×39 geometry, synchronized frame boundaries, exact PR hyperlink/footer and free-composer cursor. Preserve a minimal recorded composer/footer fixture plus source hashes/offsets and the original inline observation's weaker provenance.
- [ ] Red/green: recorded PR footer reports active work while its composer stays free through confirmed peer submission. Extend the existing strict complete-footer parser only for canonical positive-decimal PR #N at the observed position; reject duplicates, malformed/truncated numbers, arbitrary fields and unqualified PR-only absence.
- [ ] Keep body/draft/partial-frame, zero/no-shell, completion/lifetime and older-transport regressions; update qualification docs and run targeted/race plus recorded full suite before close.

ARCH-DRY/PURE/PURPOSE: reuse backgroundWorkState, terminal replay and peer-delivery fixtures; one strict grammar, no substring fallback. This is display evidence only: no protocol, admission, Settled or timer change. ARCH-SECURE: capture only composer/footer evidence from the native archive, excluding unrelated transcript and credentials; incomplete evidence stays unknown. ARCH-MOCK: actual native evidence qualifies the existing fake/replay seam. ARCH-CONSTRAINTS: existing bounded frame dimensions and footer scan remain; no runtime IO. ARCH-ORDER/FUNERAL: no new cross-event state, artifacts or resource owners; existing terminal/frame lifetime rules remain authoritative.

## Log

2026-10-17: follow-up from #632 close/landing. Report52b2dbea-d7a5-404e-ada3-5e31438cefac submitted to the hub. Verified submitted TL74bd9c72-9451-4c36-9963-38eba67e0491 directs landing #632 as reviewed and filing this PR #N variant separately with the capture as fixture; do not loosen the parser without a capture. Filed unclaimed for later dispatch; no implementation started.

### 2026-10-17 — capture investigation

Claimed pair:7 and entered planning. Pair:6 confirms only the original rendered peek was retained there (dd478cee-34e0-4348-869a-4f25b58a45d8). Read-only discovery found the original tag's parked native recording at repos/990458a6956a804b/parked-scrollback-1-pair-8-20261017T024328.raw with capture/events sidecars. Preliminary archive evidence: Claude Code v2.1.301, 94×39, synchronized frames; PR number is OSC8-linked. Current live pair:2 is idle without the PR footer and now reports v2.1.302, so it cannot substitute for archived provenance. Parser unchanged while the exact qualifying frame is located.
