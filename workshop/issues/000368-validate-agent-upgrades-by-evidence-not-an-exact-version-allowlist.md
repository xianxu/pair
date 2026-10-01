---
id: 000368
status: open
deps: []
github_issue:
created: 2026-10-01
updated: 2026-10-01
estimate_hours:
card_mirror: 'daa81d9d85c59d05c1f1940676f1616beb19d0a4' # card fields mirrored from issue-cards; edit via sdlc
---

# Validate agent upgrades by evidence, not an exact-version allowlist
## Problem

Agent CLIs auto-update constantly. Pair adapts to each harness across the
aspects in `atlas/how-to-bring-up-a-new-harness-cli.md` (return remap, overlay
detection, session watching, composer recognition for peer delivery, faint
suggested-prompt text, cursor detection, ...), and any update can silently
change the behavior an adaptation keys on.

#353 guarded peer delivery with an exact-version allowlist (Claude Code
2.1.286, Codex CLI 0.159.2). After the agents auto-updated to 2.1.287 and
0.159.3, every newly started slot quietly stopped registering for peer
messages; nothing told the operator (found in #360's smoke, 2026-10-01). #360
removed the allowlist as a stopgap, so any installed version now receives, with
only the composer-state checks guarding delivery.

## Spec

Assume an upgrade works, and prove it from evidence gathered in normal daily use.

- **Evidence per agent version.** Each adaptation that depends on harness
  behavior records, keyed by agent and version, whether its expected signal was
  observed. Examples the operator named: the faint suggested-prompt text still
  uses SGR 2; the composer and cursor are still detected; peer-delivery
  composer recognition and submit still fire. Extend the existing flight
  recorder (`$PAIR_ADAPT_LOG_PATH`, `cmd/internal/adapt`, outcomes
  fired / bypass / near-miss / fail) rather than inventing a second channel.
- **Durable across sessions.** The flight recorder truncates per launch. Version
  evidence needs an aggregate that survives sessions (per agent+version: which
  checklist signals have been seen firing, when last, and any near-misses),
  with a stated size bound and removal policy (ARCH-FUNERAL).
- **Deterministic checks, not vibes.** Each checklist item maps to a concrete
  signal with a pass condition, so "version X is evidenced" is computed, not
  judged.
- **Surface in the doctor first.** `doctor/doctor.sh` and `:PairDoctor` report,
  per installed agent version: evidenced signals, missing ones, and
  near-misses. This was `:PairDoctor`'s original purpose, before perf testing
  was added to it.
- **Operator notification later.** How and when to warn (status line, couch,
  refusing automation) is decided after evidence accumulates.

## Done when

- Every harness-dependent signal in the bring-up checklist that daily use can
  exercise emits evidence keyed by agent and version, including the peer
  delivery composer and submit signals and the faint-text styling.
- A durable per-version aggregate exists with a bound and removal path.
- `doctor/doctor.sh` and `:PairDoctor` show, for the installed versions, which
  signals are evidenced, missing, or near-missing, with a test per state.
- A new agent version with no evidence yet is reported as unevidenced, not as
  broken.

## Plan

- [ ] Inventory the harness-dependent signals in `atlas/how-to-bring-up-a-new-harness-cli.md` that daily use exercises, plus peer-delivery composer/submit and faint-text SGR 2, each with a deterministic pass condition.
- [ ] Emit each as a flight-recorder signal carrying agent and version (`cmd/internal/adapt`).
- [ ] Durable per-agent-version aggregate with a size bound and removal policy.
- [ ] Report evidenced / missing / near-miss per installed version in `doctor/doctor.sh` and `:PairDoctor`, with a test per state; an unevidenced new version reads as unevidenced, not broken.
- [ ] Decide operator notification after evidence accumulates (separate follow-up).

## Log

### 2026-10-01

- Filed from #360's live smoke. The exact-version peer allowlist was removed
  in #360 (`peerReceiverAgents` in `cmd/internal/wrapcmd/peer_runtime.go`);
  this issue replaces it with evidence.
