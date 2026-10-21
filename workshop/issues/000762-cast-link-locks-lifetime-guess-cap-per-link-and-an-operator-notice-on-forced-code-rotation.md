---
id: 000762
status: open
deps: []
github_issue:
created: 2026-10-21
updated: 2026-10-21
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'f73b1535a744494534047233e448fbbe1af5f0de' # card fields mirrored from issue-cards; edit via sdlc
---

# cast link locks: lifetime guess cap per link and an operator notice on forced code rotation

## Problem

## Spec

From pair:6's read-only audit of the cast's link locks (10-21, during pair#759 planning). Today each cast link needs
its URL token **and** a numeric code (4 digits view/pointer, 6 digits drive). Guessing is limited per link to a burst
of 5 then 1 attempt/s (shared across IPs), and every 8 wrong codes force-rotate the code. There is **no lifetime cap**:
someone holding a leaked link URL can keep guessing against successive codes at 1/s. Expected time to unlock a
4-digit view or pointer link is about 3 hours of continuous guessing (8 tries per 10,000-code draw, 8 s per draw); a
6-digit drive link is about 12 days. Each forced rotation also silently invalidates the code the operator gave
legitimate viewers who haven't unlocked yet.

Harden, without changing the normal flow:
1. **Lifetime guess cap per link:** after 3 forced rotations (24 wrong codes) without a successful unlock, the link
   **refuses all unlock attempts** until the operator rotates it explicitly. Already-unlocked sessions keep working.
2. **Tell the operator:** each forced rotation, and reaching the cap, raises one Couch notice naming the cast and link
   kind (view / pointer / drive), so a guessing attack or a mistyped code is visible.
3. No change to tokens, cookies, codes' length, or the per-link rate bucket.

Out: URL-token expiry, the quick-tunnel default, and Access in front (pair#759's remote mode covers Access).

## Done when

- Unit tests: the cap triggers at exactly the 3rd forced rotation, refuses further unlocks (correct code included),
  keeps unlocked sessions, and clears on explicit operator rotation; rotation RNG failure still refuses.
- A notice is raised per forced rotation and at the cap (test through the existing notice seam).
- atlas (broadcast or webcast) states the per-link guess policy in one paragraph.

## Plan

- [ ]

## Log

### 2026-10-21
