---
id: 000641
status: open
deps: []
github_issue:
created: 2026-10-17
updated: 2026-10-17
estimate_hours:
collaboration-mode: collaboration
card_mirror: '0bfad2e727b2849ab62d015b32089e06c5f51b95' # card fields mirrored from issue-cards; edit via sdlc
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

- [ ]

## Log

2026-10-17: follow-up from #632 close/landing. Report52b2dbea-d7a5-404e-ada3-5e31438cefac submitted to the hub. Verified submitted TL74bd9c72-9451-4c36-9963-38eba67e0491 directs landing #632 as reviewed and filing this PR #N variant separately with the capture as fixture; do not loosen the parser without a capture. Filed unclaimed for later dispatch; no implementation started.
