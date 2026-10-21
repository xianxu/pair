---
id: 000762
status: working
deps: []
github_issue:
created: 2026-10-21
updated: 2026-10-21
estimate_hours:
collaboration-mode: collaboration
card_mirror: '9cf2f3059d8a23d9f0bc418832e1ec6ede412cdb' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-21T02:33:09-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:4
    worktree: /Users/xianxu/workspace/worktree/pair-slot4/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: operator, spec: "71129b84", done: "de31820a"}
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

- [x] Add the per-lock third-forced-rotation refusal under its existing mutex; only successful operator rotation clears it. Preserve live sessions and RNG-failure refusal.
- [x] Publish every current guessing rotation through the existing Couch notice seam, with one combined cap notice; retain ended-cast/stale-generation guards and remove the quiet window.
- [ ] Pin exact thresholds, successful-unlock persistence, per-link isolation, session survival, explicit rotation/RNG failure and notice behavior; update the broadcast policy paragraph, verify and land.

Actions: unlock, rotate explicitly, end cast.

| Internal state / failure mode | Visible state | Recovery action | Possible loss |
|---|---|---|---|
| Fewer than three forced rotations | normal code entry; every forced change has a notice | unlock with current code | old unredeemed code superseded |
| Third forced rotation | code changed; link locked until operator rotation | rotate explicitly | new unlocks refused; existing sessions retained |
| Randomness fails at spent budget or operator recovery | refusal retained; no false rotation success | rotate explicitly after repair | none; current sessions preserved on failed rotation |
| Successful operator rotation | normal unlocks restored, old sessions revoked | unlock with new code | previous sessions disconnected, as before |
| Cast ended | no valid unlock/session | start a new cast | in-memory locks end with cast |

The existing Lock mutex orders attempt/counter/rotation changes (ARCH-ORDER). One bounded counter per link and a cap bit on the existing Rotation event add O(1) work per attempt, no scan or extra worker/store. Removing the quiet-window maps reduces residue. Tokens, lengths, rate limiter and session cap stay unchanged (ARCH-DRY/CONSTRAINTS/FUNERAL).

## Log

### 2026-10-21

- Traced shared castlock.Lock, broadcast.LinkLocks and Couch onCodeRotated. Guessing currently rotates indefinitely; Couch has a30-second coalescing window. Scope is bounded changes in these existing owners; no new effect path or artifact family. Quick-flow plan posted to hub, proceeding unless objected.

## Revisions

### 2026-10-21 — lifetime count and cap notice clarified

TL approval receipt2ea832e9-f345-48e2-ab4b-986934a358fd supersedes “without a successful unlock” in the Spec: successful unlocks do not reset the forced-rotation count. Only successful explicit operator rotation clears it. A legitimate viewer’s typos may eventually trip the cap during a long cast; accepted tradeoff is operator rotation. The third forced rotation emits one combined notice (“code changed; link locked until you rotate”), not two rows.

- TDD reproduced correct-code admission after24 wrong codes in both4/6-digit locks; the new counter/cap fixes it under the existing mutex. Successful unlocks between rotations do not reset it. Tests cover exact threshold, no fourth rotation, per-link isolation, existing-session/channel survival, operator-reset/session revocation, failed operator recovery and failed third forced rotation. Counted32/64 capped attempts draw no additional entropy. Lock race tests passed3.337s; broadcast/webcast race packages passed112.417s/18.185s.
- Notice tests reproduced the missing rapid second notice and asynchronous old-notice overwrite before fixes. Removed the quiet maps/window and published through the shared locked notice seam while the existing generation guard is held; repaint/flash remain outside locks. Targeted notice race tests passed2.367s for both cast kinds/all link kinds, concurrent callbacks, cap text/privacy and stale/ended guards. Final package/lint checks running before recorded suite and close.

- Final targeted checks passed: couchtty race214.803s, artifactpath26.554s and repolint21.642s. Reader inventory and whitespace checks passed. Implementation adds35 production lines across existing files, within the quick shell. Preparing pushed-head recorded verification and mandatory close review; landing remains gated.
