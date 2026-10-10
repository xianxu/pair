# TL-callable relaunch / reload-context of a live idle slot — Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `couch --relaunch repo:N` (new Pair binary, same conversation; the
console's Alt+n) and `couch --reload-context repo:N` (a fresh agent
conversation in the same Pair; Shift+Alt+N) run from another agent's shell
against a live slot. Each refuses a busy slot, a dirty checkout, or (relaunch
only) a binary with nothing new, and returns a pollable receipt.

**Architecture:** Both verbs ride the existing remote slot-operation path that
`--resume/--reboot/--reap/--recover` use. The CLI sends a request over the
broker socket, the console queues it, and `PrepareSlotOperation` admits it on
the queue. The CLI then polls the in-memory `OperationReceipt`. Busy is a new
wrapper-owned fact: pair-wrap publishes `Observation.Settled` on its existing
activity frame once output and input have been quiet, the composer is empty and
no turn is open. The supervisor reads that fact at admission, alongside
`ProbeSlotGit` for dirtiness and Go build info for staleness.

**Tech Stack:** Go; couchmessage session frames; couchcore operations; wrapcmd.

---

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `Observation.Settled` | `cmd/internal/couchmessage/broker.go` | modified |
| `wrapperSettled` | `cmd/internal/wrapcmd/peer_settle.go` | new |
| `LiveRestartFacts` / `DecideLiveRestart` | `cmd/internal/couchcore/live_restart.go` | new |
| `BinaryFreshness` / `DecideBinaryFreshness` | `cmd/internal/couchcore/live_restart.go` | new |
| `slotOperations` | `cmd/internal/couchcore/slot_operation.go` | modified (+relaunch, reload-context) |
| `ParseCLI` slot-op case | `cmd/internal/couchcmd/cli.go` | modified |

- **Observation.Settled** — the wrapper's claim "an automatic restart would
  interrupt nothing". It is true only if all of these hold:
  - output and input have been quiet for `SettleInterval` (3s);
  - no turn is open (`notificationLifecycle.Active == false`);
  - the composer reads `PeerComposerEmpty`;
  - no picker, image or buffered input is pending.
  - **Relationships:** 1 per wrapper incarnation, carried on the existing
    activity frame. Any activity clears it at once; it is recomputed only after
    the quiet interval (debounced timer), so streaming output costs nothing
    extra.
  - **Compatibility:** a pre-#421 wrapper never sets it. The field is
    `SettledKnown`+`Settled`, so "unknown" is distinguishable from "busy".
  - **DRY rationale:** it reuses the exact composer reader peer delivery
    trusts (`peerComposerState`), so "empty composer" means one thing.
- **DecideLiveRestart(facts) → (ok, code, detail)** — pure admission for both
  verbs.
  - **Refusal codes:** `busy` (Settled false), `busy-unknown` (an old wrapper or
    no session), `dirty` (git porcelain non-empty), `not-live` (no occupied
    incarnation; the caller should use `--resume`), plus `stale-binary`
    (relaunch only, from `DecideBinaryFreshness`).
  - **Unknown is a refusal, not a fallback.** The 7 slots running pre-#421
    binaries refuse `busy-unknown` until they have been relaunched once by hand
    (or they pass `--force-unknown`, which the receipt records). A wrong guess
    would kill a draft.
- **DecideBinaryFreshness(running, onDisk, sourceHEAD)** — pure.
  - `running` is the wrapper's own `vcs.revision`/`vcs.modified`, sent once
    in a new `Binding.PairRevision`. *(Superseded: the executable's content hash, sent on
    `hello-v2` and never in `Binding`; see Revisions PQ-2 and PQ-3.)*
  - `onDisk` is the build info of `exec.LookPath("pair")` in Couch's
    environment, the same binary `launch_existing.go:91` runs.
  - `sourceHEAD` is `git rev-parse HEAD` of the binary's checkout (the
    parent of `bin/`), when that checkout is a git tree.
  - **Refuse `stale-binary`** when onDisk == running (relaunching changes
    nothing; this was tonight's #418 case).
  - **Note in the receipt** when onDisk ≠ sourceHEAD ("`bin/pair` built from X,
    its checkout is at Y; run make build"). It is not a refusal, because
    relaunching onto a newer-but-still-behind binary is still progress.
  - `--same-binary` overrides the refusal.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| wrapper settle timer | `cmd/internal/wrapcmd/peer_settle.go` | new | pair-wrap clock + terminal snapshot |
| `SlotLiveness` lookup | `cmd/internal/couchcmd/message_service.go` | new | broker actor observation by slot |
| `ProbeSlotGit` | `cmd/internal/couchcore/slotgit.go` | existing | `git status` |
| `BinaryProbe` | `cmd/internal/couchcore/live_restart.go` | new | `exec.LookPath`, `debug/buildinfo`, `git rev-parse` |
| `ReloadContext` | `cmd/internal/couchcore/reload_context.go` | new | SIGUSR2 to the broker-verified wrapper PID *(superseded the pid file; Revisions PQ-1)* |

- **SlotLiveness** is injected into `Couch` (an interface with
  `Liveness(slot string) (Observation, Binding, bool)`), so admission tests use a
  fake. Production reads the broker's `SlotActor` for the exact slot.
- **ReloadContext** resolves the row's tag and data dir to the scoped
  `PairWrapPID` *(superseded: the broker's verified Binding PID; Revisions PQ-1)* (the same artifact family `agentcmd.RunRestart` reads) and sends
  SIGUSR2. It shares one helper with `agentcmd` (ARCH-DRY: extract
  `artifactpath`-scoped `SignalWrapper(dataDir, tag, sig)`).
- **Fakes:** the existing `slot_operations` and console fakes take the new
  ops. The wrapper settle has a clock seam (`now`, timer injection) as
  `peerDelivery` does.

ARCH-FUNERAL: creates nothing durable. Receipts are the existing in-memory,
5-minute, 64-entry `OperationReceipt` map; `Settled` lives in memory per
wrapper.

## Milestones

- [ ] **M1 — the wrapper reports Settled.** `Observation.SettledKnown/Settled`
  and `Binding.PairRevision` *(superseded: `hello-v2` negotiation; Revisions PQ-2)*, with validation and size bounds. pair-wrap's
  settle timer and its predicate (`wrapperSettled`, pure over a snapshot plus
  flags). The broker keeps the latest observation per actor, and `--actors
  --json` shows `settled` per slot (a cheap debugging aid for TLs).
  - Tests: a pure predicate table (turn open, composer occupied, picker,
    buffered input, all clear). A timer test: activity clears at once and
    settle sets after the interval, never during streaming. Frame
    encode/validate with and without the fields (an old wrapper decodes as
    unknown).
- [ ] **M2 — `couch --relaunch repo:N`.** `DecideLiveRestart` +
  `DecideBinaryFreshness`, then the admission branch in `PrepareSlotOperation`
  for live rows (relaunch is not in `ActorActions` for a live row; the new rule
  replaces the offer check for these two ops only). The parser accepts
  `--relaunch repo:N --confirm [--same-binary] [--force-unknown] [--json]`.
  The receipt maps `RelaunchResult.Outcome` and the refusal codes.
  - Tests: an admission table over facts, freshness cases (equal, newer,
    behind HEAD, unknown build info), CLI parse cases, and dispatch through
    the existing slot-operation fake, end to end to a receipt.
- [ ] **M3 — `couch --reload-context repo:N`.** The same admission without the
  freshness rule; execution signals the slot's pair-wrap via the shared helper.
  The receipt is `succeeded` once the signal is delivered *(superseded: succeeded only on a new Binding within 20s, else unknown/unconfirmed; Revisions PQ-4)*. Verifying that a
  fresh conversation actually started is via `--peek`; the receipt says so.
  - Tests: helper extraction, keeping `agentcmd` tests green; dispatch with a
    fake signaller.
- [ ] **M4 — live check, docs.** A live relaunch and reload of a real idle slot
  through the socket: `pair:2` if free, else a disposable Couch, confirmed
  with the TL. A busy refusal while the agent works; a dirty refusal with a
  touched file. Update `atlas/couch.md` and the `couch --skill` text (the
  verbs table, plus "refuses busy/dirty/stale").

## Risks

- **First rollout:** slots on old binaries report unknown. `--force-unknown`
  exists for exactly that, and the TL decides whether to use it.
- **Settle window race:** the operator starts typing after admission and before
  park. Park is fast; the window is the queue latency. Admission runs on the
  console queue (not at socket receipt) to keep the window minimal.
- **Codex turn tracking:** Codex has no progress OSC, so turn-active
  falls back to the composer and quiet interval for Codex. That's acceptable:
  Codex's composer is read from the same recognizer.

## Revisions

### 2026-10-09 — TL ariadne:1 approval notes
- "Behind" compares against the source the launcher builds from: the checkout
  that owns the `pair` binary on Couch's PATH (pair:0 in practice, built from
  `cmd/pair-go`), not the slot's checkout. Both the `stale-binary` refusal and
  the behind-HEAD note name the fix verbatim: `make build in <that checkout>`.
- `--force-unknown` stays. The first rollout of #421 itself is a manual Alt+n
  once per slot.

### 2026-10-09 — plan-quality round 1 (PQ-1..PQ-6)
- **PQ-1 (signal target).** reload-context never reads the pid file. It takes
  the broker's live `Binding` for the exact slot, re-checks
  `OSProcOps.Identity(Binding.PID) == Binding.Start` (the identity the wrapper
  published, `peer_runtime.go:136`), and only then sends SIGUSR2. The shared
  helper is `SignalVerifiedWrapper(proc, binding, sig)`; `agentcmd` keeps its
  in-slot pid-file path, which is correct there.
- **PQ-2 (wire contract).** Neither `Binding` (the broker actor key) nor any
  frame an old peer can see changes. Session frames decode strictly (unknown
  fields and ops are rejected), and an old wrapper treats any byte after the
  ack as a lost session. Negotiation is therefore try-new-then-fall-back:
  - A new wrapper first sends `hello-v2`, which carries the Binding plus
    `Build` (its executable's sha256 and vcs revision).
  - An old Couch refuses the unknown op with an error ack. The new wrapper
    then reconnects with plain `hello` and never sends Settled.
  - A new Couch acks `hello-v2` and accepts `Observation.Settled*` on that
    session only.
  - An old wrapper keeps sending plain `hello`; a new Couch reports it as
    unknown.
  - Tests cover all four pairings over the in-memory transport.
- **PQ-3 (freshness rule).** Identity is the executable's **content hash**,
  computed by the wrapper once at startup, compared with the hash of
  `exec.LookPath("pair")` in Couch's environment.
  - Equal hashes → `stale-binary` refusal, naming `make build in <checkout>`.
  - Differing hashes → allowed. The note "built from X, checkout at Y" uses
    vcs.revision against the checkout HEAD; a vcs.modified build adds "from a
    dirty tree".
  - Unknown build info or an unreadable binary → no refusal; the receipt says
    freshness was unverified.
- **PQ-4 (uncertain outcome).** reload-context waits, up to 20s, for the
  slot's session to re-establish with a new Binding (a new Start or Nonce).
  It reports `succeeded` only then; otherwise the receipt is `unknown` with
  code `unconfirmed`. relaunch already gets its evidence from `ResumeContext`.
- **PQ-5 (test strategy, one line each).**
  - `DecideLiveRestart`: an exhaustive table over the product of facts
    (settled × known × dirty × live × op).
  - `DecideBinaryFreshness`: a table over hash equal/unequal ×
    revision/HEAD × modified × unknown.
  - Settle timer: injected clock and timer with interleaved activity events,
    asserting it never settles inside the interval.
  - Negotiation: the four old/new pairings.
- **PQ-6 (non-goals).**
  - No fleet or batch verb (the TL loops over slots).
  - No automatic `make build`.
  - No durable receipts (the existing 5-minute in-memory ones).
  - No re-check of Settled between admission and park.
  - No change to Alt+n or Shift+Alt+N behavior.
