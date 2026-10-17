---
id: 000622
status: working
deps: []
github_issue:
created: 2026-10-16
updated: 2026-10-16
estimate_hours:
card_mirror: 'e60acf923f81c394c448ccf87f55887cb352bdd9' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-16T17:03:46-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
---

# Couch server spends ~1 core emulating every hosted client's output in a vt parser, visible or not

## Problem

Profiled by pair:2 for pair#618 (2026-10-16, macOS `sample` of the Couch server, 5 s): the Couch server burns about
one core continuously, and ~500 of its busy samples are terminal emulation: `ptychild.(*Child).ingest` →
`internal/terminal` + charmbracelet's vt `ansiparser`. Couch parses the full output stream of every hosted session
client (~27 `pair --couch-session-v1 resume` children) in a vt emulator, so every spinner redraw on every slot costs
Couch CPU all the time, whether or not anyone is looking at that slot.

## Spec

- Measure first: per hosted client, ingest bytes per second and CPU, at idle and during a busy turn; which consumers
  need the emulated screen (console view, peek, waiting/idle detection, resource monitor).
- Then the smallest change that keeps every consumer correct. Options to weigh: emulate only the slot(s) the console
  is showing and keep a cheap raw ring for the rest (replaying it when a slot is shown or peeked); or parse lazily
  (append raw bytes, emulate on demand). Idle/waiting detection must keep working for off-screen slots.
- No new background workers; bounded memory per client.

## Done when

- The Couch server's steady CPU with ~27 hosted clients drops by a measured amount (before/after `sample` and CPU%),
  with the console, `--peek` and waiting notices behaving as before (tests plus a live check).

## Plan

- [ ] M1 — land permanent per-hosted-client ingest-byte diagnostics through read-only resources; no parser-policy change.
- [ ] M2 — after the combined restart, measure natural idle/busy rates and server CPU, choose the smallest correct optimization with TL approval, then implement and verify it.


## Log

### 2026-10-16

### 2026-10-16 — initial measurement and consumer audit

Claimed/started on fresh main b95df08cebf86cc71cd86a27dfdf2fdb96f1b91d. Slot-private Ariadne refresh advanced its clean checkout from 1fbc10d2 to e1bb1ba5 after start-plan's fleet refresh refused the unrelated untracked pensive; the pensive stayed untouched. No feature code changed.

Live supervisor PID71982, launched 14:57:39, has 23 direct hosted resume children at the new observation. Cumulative CPU rose 107:20.99 → 107:40.38 over 30.1086775s, or 64.40% of one core. Identity unchanged. This is a natural mixed-load observation, not an idle-fleet result. Baseline /tmp/pair622-live-baseline.json SHA256 956d9eb679e309e51ab0b874c0f436b812cb2ea95541bb87c50f9dec1c5bd7d4. Read-only 5s native sample at 17:06:26 retained /tmp/pair622-couch-71982-native.sample.txt SHA256 414d34db020424e9ccccfd9c69776a8fae33a4fc03393dea7b80c956ba0101aa. Its call graph includes Child.ingest and VT parsing; no percentage is inferred by dividing inclusive counts or summing overlapping stacks. The binary currently at the path reports 9fc035e718b1, but its mtime is later than server launch; that is not proof of the loaded image's revision.

Read-only consumer audit: selected console frame/history, mouse/key encoding, query replies, effects, resize and EOF need the hosted Endpoint authority. Off-screen waiting/picker notices and peek use wrapper tails/status; resource idle uses wrapper settled/picker/broker state; idle fading uses thread activity. These do not require the hosted screen. The raw diagnostic ring is bounded at 128KiB and discards arbitrary prefixes: existing TestEndpointRetainsFrameAndModesAfterDiagnosticRingEviction proves screen/modes predate its tail. Replaying that tail cannot safely replace the authoritative emulator. Lazy parsing must also preserve prompt queries, exactly-once effects, ordered geometry and EOF. Details and tradeoffs are in the durable plan.

No exact per-hosted-client ingress counter exists in this running binary. Existing capture could provide bytes but requires startup opt-in. TL instead directs permanent always-on atomic byte counters as M1, exposed read-only through resources/debug; optional emulator CPU timing only if cheap. They land before the #618/#617 combined restart, then measurement and optimization design follow. The posted direction's canonical sender/body matched receipt 9929f33e; its delivery outcome is indeterminate, even though this turn received the text. No capture restart was performed.

## Revisions

### 2026-10-16 — measurement milestone before optimization design

TL direction 9929f33e replaces the proposed capture choreography with permanent M1 ingress diagnostics. Original whole-issue CPU reduction and live consumer acceptance remain owed in M2; M1 does not claim them. One atomic byte increment per Child.ingest call; no per-chunk clock/syscall/worker or raw payload retention. CPU attribution uses the live server sample until a cheap accurate per-client CPU seam is established; wall elapsed time will not be labelled CPU. Full-flow milestone review is used because an explicit M1/M2 sequence lies outside the binary's quick-flow shell; the implementation target remains a small diagnostic diff.
