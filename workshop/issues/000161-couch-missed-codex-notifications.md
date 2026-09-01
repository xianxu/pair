---
id: 000161
status: working
deps: []
github_issue:
created: 2026-09-01
updated: 2026-09-01
estimate_hours: 6.93
started: 2026-09-01T13:35:54-07:00
---

# Couch misses Codex completion notifications

## Problem

During Couch dogfood testing, Codex completed work in Pair but Couch did not
surface a completion notification. No notification appeared in either the
status bar or the switcher.

The same joint failure can occur for Claude. One confirmed marker is:

```text
✻ Sautéed for 34s · done 1:39 PM
```

Claude marker mode currently requires the verb after `✻` to match ASCII-only
`[A-Za-z]+`. The accented `é` therefore prevents this otherwise valid marker
from reaching the shared notification sink. Codex remains a separate failure
mode because it currently relies on native OSC notification output.

A captured Codex completion renders as `─ Worked for 3m 57s ─…`. The same
capture also showed a copy prefixed by `> `. Investigation must distinguish the
live colored status line from quoted or replayed copies before granting either
form notification authority.

## Spec

### First slice: Claude Unicode marker

- In `endOfTurnByAgent["claude"]`, accept a marker verb made of one or more
  Unicode code points whose General Category is Letter (`L`). Precomposed
  `Sautéed` therefore matches; combining marks (`M`), digits, punctuation,
  whitespace, and an empty verb do not. Do not add Unicode normalization.
- Preserve the existing marker prefix and duration grammar. Optional trailing
  status/time text remains outside the parsed prefix.
- Preserve the current ownership boundary: only a newly finalized colored span
  can authorize this signal through `finalizeSpan -> emitOuter`. Quoted history,
  uncolored text, duplicate-span suppression, rate limiting, and delivery remain
  unchanged (`ARCH-DRY`, `ARCH-PURE`).

### Deferred: Codex completion

- Surface Codex completion notifications for Pair sessions in Couch's status
  bar and switcher.
- Keep the two surfaces consistent so a completed agent is not silently missed.
- Use the Codex rollout transcript as the primary durable completion source.
  Reuse Pair's existing `sessionwatch`/`sessioninventory` authority: scanner-
  validated root `session_meta`, current launch generation, spawned process
  incarnation, and descendant open-file evidence establish the rollout binding.
  Process evidence corroborates but never independently selects a transcript;
  never use newest-file or cwd-only guessing. Tail new records and treat
  `task_complete` / `turn_complete` as turn completion, including short turns
  whose TUI does not visibly retain a `Working` bar (`ARCH-IDENTITY`).
- Keep native notification OSC as an immediate accepted source. Treat the
  rendered `• Working (… esc to interrupt)` present-to-absent transition as a
  secondary signal and a previously-active 60-second idle timeout as last-resort
  recovery. `Worked for…` is supporting evidence, not completion authority.
- Feed every source through one deduplicating notification sink so a single turn
  produces at most one unread/outer notification (`ARCH-DRY`). Do not overwrite
  Codex's single legacy `notify` hook.

### Later signal hardening: Claude activity

- Interpret Claude OSC progress state `9;4;3` as working and `9;4;0` as work
  stopped. After the explicit stop, briefly allow a richer native notification
  or finalized `✻ <verb> for <duration>` marker to arrive before emitting a
  generic completion.
- Retain the colored completion marker as compatibility fallback. Do not infer
  completion merely because no progress event arrived for ten seconds; arm any
  longer idle recovery only after working activity was observed.

## Done when

### First-slice acceptance

- A production-path regression proves
  `✻ Sautéed for 34s · done 1:39 PM` reaches Claude's existing notification
  sink through `finalizeSpan -> emitOuter`.
- Existing ASCII markers retain behavior.
- Focused grammar tests prove an empty verb and any verb containing a non-`L`
  code point do not match.

### Later issue acceptance

- PID-bound transcript tests prove both a short direct response and a longer
  response emit on `task_complete` / `turn_complete`, without cross-session
  attribution when multiple Codex processes run concurrently.
- Native OSC, transcript completion, rendered activity transition, and timeout
  converge on one notification per turn; unobserved prompt idleness does not
  notify.
- A Codex completion in Pair produces a visible notification in the Couch
  status bar.
- The same completion is visible in the Couch switcher.
- Automated coverage reproduces the missed Codex path and prevents regression.

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only. `sdlc estimate-source` reports the calibration
source as stale, so the number is provisional but uses the required method.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.30 impl=0.08
item: smaller-go-module design=0.06 impl=0.20
item: tui-screen design=0.40 impl=0.40
item: cross-cutting-refactor design=0.20 impl=0.20
item: greenfield-go-module design=0.40 impl=0.32
item: api-integration design=0.60 impl=0.60
item: smaller-go-module design=0.06 impl=0.20
item: tui-screen design=0.40 impl=0.40
item: api-integration design=0.20 impl=0.24
item: atlas-docs design=0.20 impl=0.20
item: milestone-review design=0.08 impl=0.12
item: milestone-review design=0.08 impl=0.12
item: milestone-review design=0.08 impl=0.12
item: milestone-review design=0.08 impl=0.12
design-buffer: 0.15
total: 6.93
```

## Plan

- [x] M1 — Correct and regression-test Claude's Unicode marker-verb grammar.
- [x] M2 — Add the pure turn lifecycle reducer, semantic deduplication, Claude
  OSC progress transitions, and activity-gated timers.
- [ ] M3 — Extend the existing authorized Codex session watcher to tail the
  bound rollout and publish transcript completion observations to `pair-wrap`.
- [ ] M4 — Add rendered Codex `Working` recovery and verify canonical
  notifications through Couch's status and switcher surfaces.

## Log

### 2026-09-01
- M2 implementation: RED covered absent lifecycle types, OSC progress at every
  split in marker/native modes, one proxy-owned timer, resumed work cancelling
  stale grace, and transcript start activating a submitted turn. GREEN:
  `go test ./cmd/internal/wrapcmd -count=1` and `git diff --check`. The pure
  reducer now deduplicates native/marker terminals per generation; Claude
  `OSC 9;4;3;` arms activity and `OSC 9;4;0;` enters a 250 ms richer-message
  grace, with a tokenized 60-second activity watchdog (`ARCH-PURE`, `ARCH-DRY`).
- M2 boundary review reopened the milestone. The send boundary had been coupled
  to a bare-CR encoding even though Pair's semantic send chord is Alt+Enter and
  each harness owns its output bytes. RED production-flow tests now cover legacy
  and KKP Alt+Enter for Codex and Claude, plain Enter non-submission, native
  completion after a real send, source-order deduplication, keyed mismatch/new
  turns, and distinct abort fallback. Added the lifecycle source to the
  exhaustive artifact inventory (`ARCH-PURPOSE`).
- 2026-09-01: closed M1 — Judgment actual: 0.25h because telemetry reported no transcript events. RED: Unicode grammar rejected Sautéed and production path emitted no OSC. GREEN: focused grammar/production/fuzz tests and go test ./cmd/internal/wrapcmd -count=1 pass; git diff --check clean. No atlas: pure regex bugfix with no new architectural surface and no atlas restatement of the old grammar.; review verdict: SHIP

Captured from initial Couch dogfood testing. Codex finished work in Pair, but
no notification appeared in the status bar or switcher. Investigation was
explicitly deferred to issue implementation.

Claimed the issue and entered planning. Inspection showed Claude uses a
finalized colored-span marker while Codex relies on native OSC. Captured
`✻ Sautéed for 34s · done 1:39 PM` as a concrete Claude failure: the ASCII-only
verb class rejects `é`. The operator approved a first slice that widens only
the verb to Unicode letters and leaves Codex discovery open.

Reviewed sibling `cmux` as prior art. cmux accepts OSC 9/99/777 and documents
Codex's `notify` command as one completion source, but its more robust agent
integration uses a PATH wrapper it controls plus PID-anchored transcript state
and optional injected hooks. Its design explicitly notes Codex has one legacy
`notify` slot that may already belong to another integration, so a later Pair
design must not silently overwrite it; viable alternatives are transcript
state, chaining the existing handler, or verified multi-hook injection.

Fresh-context spec review found that the Claude-first slice was not independently
finishable because Codex criteria still governed `Done when`, and that "Unicode
letters" and malformed input were underspecified. Split immediate and deferred
acceptance, named the pure grammar and production delivery seams, and defined the
verb as General Category `L` code points only.

Captured a concrete Codex stopped-state rendering: `─ Worked for 3m 57s ─…`,
with another observed copy prefixed by `> `. No current Codex text marker exists
in `pair-wrap`; Codex still takes the native OSC path. This evidence narrows the
later reproduction, but does not yet establish that rendered text is a stable or
authoritative completion protocol.

The operator clarified that `Worked for…` appears for sufficiently long Codex
turns, while short responses can render output directly. Adopted cmux's stronger
pattern: Pair already owns the spawned Codex PID, so bind that process to the
rollout JSONL it writes and derive completion from `task_complete` /
`turn_complete`. Visual `Working` transitions and native OSC remain useful fast
signals, but all sources deduplicate through one sink. For Claude, use explicit
OSC `9;4;3` working and `9;4;0` stopped state rather than treating a ten-second
progress silence as completion; retain the colored marker for richer text and
compatibility.

Planning reconnaissance corrected one integration detail from the cmux analogy.
Pair already has stronger transcript identity in `sessionwatch` and
`sessioninventory`: validated root metadata and launch generation are the
authority, while PID/open-file evidence only corroborates the causal candidate.
The implementation will extend that pipeline and publish launch-scoped lifecycle
observations rather than add a second resolver inside `pair-wrap` (`ARCH-DRY`,
`ARCH-PURE`). It also identified the existing 500 ms `emitOuter` limiter as
delivery throttling, not turn deduplication; semantic dedup belongs in a pure
turn lifecycle reducer before the sink.

Fresh-context plan review required the lifecycle contract to become executable,
not merely directional. The plan now pins `task_started(turn_id)` and user submit
as generation boundaries; keyed/keyless late-arrival rules; sessionledger-style
journal commit and indeterminate-write recovery; the unique causal-round terminal
watermark; tokenized proxy-owned timers; bounded durable-channel backpressure;
and a 100 ms watcher + 50 ms tailer latency budget. These close the duplicate,
cross-turn, replay, stale-timer, and first-completion ambiguity classes before
code begins (`ARCH-PURPOSE`, `ARCH-CONSTRAINTS`).

A second plan review found the binding handoff could publish a completed short
turn without its earlier keyed opener. The plan now publishes the validated
causal round's `task_started(turn_id)` and matching terminal together, in order,
after binding, then follows only records beyond the terminal watermark. A
transcript-only short-turn regression pins this first-turn behavior.

The SDLC plan-quality gate then required three cross-cutting plan rules: name
adversarial-input/mechanical-guard strategies per risky function instead of
enumerating test cases; explicitly state excluded behaviors and why; and extend
the existing opt-in native-session live conformance test with Codex lifecycle
identity/envelope assertions on the pre-release and scheduled macOS cadence.
The durable plan now carries all three (`ARCH-PURPOSE`, `ARCH-MOCK`).

The next gate round accepted the non-goals and conformance contract but found
the function-level test-strategy rule was still duplicated by scenario lists and
diff recipes in later task steps. Removed those restatements across every chunk;
the plan now retains one adversarial-input/mechanical-guard strategy per risky
function plus architectural contracts and production acceptance only.

M1 used strict TDD. The grammar and production-path regressions both failed
before implementation: precomposed `Sautéed` was rejected and the colored span
emitted no outer notification. Replacing only the ASCII verb class with
`\p{L}+` made the focused tests and the full `wrapcmd` package pass. A rune-level
fuzz property pins the General Category `L` boundary; no atlas text restated the
old grammar (`ARCH-PURE`).

## Revisions

### 2026-09-01 — add confirmed Claude failure mode

Expanded the issue after dogfood evidence showed the two missing notification
surfaces are not Codex-only. Added a bounded first slice for Unicode Claude
marker verbs without changing the still-open Codex completion design.

### 2026-09-01 — separate first-slice acceptance after review

Separated the Claude patch from deferred Codex acceptance and plan ordering.
Specified the exact Unicode category boundary and the pure-parser and
production-delivery seams the regression tests must exercise.

### 2026-09-01 — add captured Codex stopped-state line

Recorded the observed `Worked for` rendering and its `> `-prefixed duplicate so
later Codex design tests the actual terminal stream and does not mistake quoted
or replayed text for a live completion.

### 2026-09-01 — choose transcript-authoritative Codex lifecycle

Replaced the unresolved Codex source choice with PID-bound rollout transcript
events as the durable authority. Added native OSC and rendered activity as
deduplicated fast signals, activity-gated idle recovery, and Claude's explicit
OSC progress transition with its existing marker fallback.

### 2026-09-01 — align transcript binding with Pair identity authority

Replaced the cmux-derived direct PID selection wording with Pair's established
scanner-authorized root binding, using process evidence only as corroboration.
Split implementation into four genuine review boundaries: Claude grammar,
lifecycle core, Codex transcript authority, and rendered/Couch recovery.

### 2026-09-01 — make lifecycle and journal contracts executable

Specified exact keyed/keyless generation transitions, durable append/replay
outcomes, causal-round authorization watermark, proxy-loop timer/channel
ownership, and enforced polling/resource bounds after plan review.

### 2026-09-01 — preserve the first causal turn opener

Required the binding handoff to publish both the validated causal round opener
and terminal so a short turn completed before binding still opens, completes,
and notifies exactly once without visual or native rescue.

### 2026-09-01 — satisfy executable-plan and live-conformance rules

Compressed test prose into named function-level risk strategies, added explicit
non-goals, and made installed-Codex lifecycle envelope checking part of Pair's
recurring native-session conformance seam.

### 2026-09-01 — apply executable test strategy across all chunks

Removed remaining scenario inventories and procedural diff recipes so the named
function-level risk strategies are the single plan authority for test design.
