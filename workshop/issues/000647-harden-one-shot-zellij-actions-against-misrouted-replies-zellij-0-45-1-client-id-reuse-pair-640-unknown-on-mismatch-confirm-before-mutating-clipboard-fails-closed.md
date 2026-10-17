---
id: 000647
status: working
deps: []
github_issue:
created: 2026-10-17
updated: 2026-10-17
estimate_hours:
collaboration-mode: collaboration
card_mirror: '357c7d9efaf5501b5a76c6906675b15910547574' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-17T11:09:41-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
---

# Harden one-shot zellij actions against misrouted replies (zellij 0.45.1 client-ID reuse, pair#640): unknown on mismatch, confirm before mutating, clipboard fails closed

## Problem

## Spec

Operator decision, 2026-10-17: harden Pair and Couch against zellij 0.45.1's reply misrouting (pair#640) on our side. There's no local zellij build. Pick up the upstream fix (zellij PR #5630, "do not reuse client ids", merged but unreleased) when it ships.

Background, from pair#640: a reused client ID lets a late cleanup or reply reach the wrong connection. In a raw-request stress test about 3% of replies went to another connection. Periodic reads already recover on their next tick. One-shot actions don't. pair:1's audit on main 15d84ef7 lists them:
- layout and terminal toggles: focus or geometry from `list-panes` and `current-tab-info`;
- clipboard paste and routing, which doesn't always fail closed;
- warm resume and attach checks: `list-clients` counts;
- restart-key ownership and note anchors;
- discovery, which can silently omit a live session from a well-formed reply;
- the ownership proof's pane content.

Hardening, in this order of value:
1. A reply that doesn't fit its request counts as unknown, never as an answer: the wrong kind, a different session or pane than the one asked about, or malformed.
2. A one-shot mutation confirms with a second independent read before acting. If the reads disagree it refuses, and the user sees a refused action, not a wrong one.
3. Clipboard paths fail closed.
4. Where the protocol allows a request to carry its own correlation, use it.

No blind retries, no sleeps, no transport machinery. Reads that already re-check every tick stay as they are.

## Done when

- Each caller in pair:1's audit is classified as re-checked (no change), hardened (rule 1 and/or 2), or fail-closed (clipboard). The atlas records the table.
- A fake-zellij test per hardened caller shows that a misrouted reply of the same kind and of the wrong kind leads to refusal or unknown, never to the wrong action.
- pair:1's native reproduction, run against the real 0.45.1, shows hardened one-shot actions refusing rather than misacting.
- When zellij ships the fix, a note in the issue says to rerun the reproduction on the new version.

## Plan

Durable full-flow plan: [Zellij reply hardening](../plans/000647-zellij-reply-hardening-plan.md). R1 is proposed for TL approval; no implementation has begun.

- [ ] M1 — Strict reply contracts, stateful fake and lifecycle/absence admission.
- [ ] M2 — Go one-shot action and clipboard safety, with per-caller fake refusal tests.
- [ ] M3 — Editor admission, native refusal qualification and complete atlas caller map.

## Log

### 2026-10-17

### 2026-10-17 — full-flow design R1

Claimed and started on a fresh main-based branch, separately from #640's documentation-only closeout. Applying the writing-plans skill; the durable plan classifies every audited caller, preserves periodic/advisory policies, defines a fixed two-observation admission with no retries/sleeps/transport additions, and enumerates deterministic fake cases plus the native acceptance oracle. Positive local sidecar authority remains a fast path. Clipboard OS mirroring of the selected text remains separate from auto-quote admission; uncertainty prevents staging/focus/trigger effects.

R1 explicitly asks the TL to acknowledge the protocol limit: no native request nonce is available, and two identical valid stale replies cannot be authenticated merely by comparison. Wrong kind/context, required-field absence and observable disagreements refuse; local identity and absence fences remain independent. No claim of universal protection from arbitrary correlated false evidence, and no native mutation replay. ARCH-SECURE/ORDER shaped that boundary. No estimate is derived until plan quality passes.

Upstream follow-through: [zellij#5270](https://github.com/zellij-org/zellij/issues/5270), [#5382](https://github.com/zellij-org/zellij/issues/5382), [#5546](https://github.com/zellij-org/zellij/issues/5546), [PR#5630](https://github.com/zellij-org/zellij/pull/5630). When a release includes retired client IDs, rerun the retained raw reproduction, native refusal/healthy controls and original actual-chord conformance before revising compatibility assumptions. No local native build or upstream posting is authorized.
