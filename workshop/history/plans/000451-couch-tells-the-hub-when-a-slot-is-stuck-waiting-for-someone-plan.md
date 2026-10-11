# Couch tells the hub when a slot is stuck waiting — Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** When a connected slot has been waiting for someone longer than a threshold (default 10 min), Couch sends one notice per waiting episode to the hub slot, so the TL never polls.

**Architecture:** A supervisor-side watcher on the existing `messageService` ticks once a minute over memory the service already holds: the broker's per-actor observation (`LastActivity`, `Pending`), the registry's `Settled` self-report (#421), and the Console's slot-git resting view. Only a slot that has been quiet past the threshold costs one IO, a screen tail read from its own wrapper (the `peek` path, pair#425). A pure classifier decides waiting/not and names the reason from the tail; the notice is an ordinary `Broker.Send` *from the waiting slot's binding* to the hub's exact slot, so it rides the existing delivery, breaker and ledger. Pair still only renders: the judgement lives in Couch, the tail's reader (the tail.go rule).

**Tech Stack:** Go, `cmd/internal/couchcmd`, `cmd/internal/couchmessage`.

---

## Measured facts the design rests on

- `LastActivity` moves on any agent output or operator keystroke (`wrapcmd/peer_delivery.go` `observeOutput`, `operatorInputLocked`, `humanSubmit`). Live `couch --actors --json` on 2026-10-11 13:21 showed idle Claude/Codex slots with `LastActivity` 6–45 min old, so an idle screen (dialog or composer) is byte-quiet; a running turn is not (spinner/timer repaint).
- `Settled` (#421) = no turn open, no picker/overlay, orientation finalized, composer empty. Unsettled **and** quiet for 10 min therefore means something on screen waits for input: a permission/selection dialog, an unsent draft, a stuck orientation.
- Live tails (`couch --peek ariadne:1`, `pair:2`, `pair:0`) show the shape the classifier reads: prose paragraphs, a turn-summary line (`✻ Cooked for …`, Codex `‹dim›Worked for …‹/dim›`), an optional `※ recap:` paragraph, a `────` separator, then the composer prompt line (`❯` Claude, `›` Codex). ariadne:1 was a real stuck slot at the time (off resting, settled, 6 min quiet, "waiting on the operator to commit").

## Rules (the spec, made precise)

Evaluate each connected, non-hub actor whose broker row is `Known` and whose `Settled` is reported (nil = old wrapper = skip; unknown is never "waiting"):

1. `quiet := now - LastActivity`; quiet < threshold → not waiting.
2. `Settled == false` → waiting, reason **prompt**: summary = the dialog title (the nearest non-empty line above the first option line `^[❯›>]?\s*1\.\s`), else the last prose line.
3. `Settled == true`, last prose line ends with `?` → waiting, reason **question**, summary = that line.
4. `Settled == true`, off its resting branch, no message pending → waiting, reason **idle**, summary = last prose line.
5. Otherwise (settled on resting branch, turn ended normally; or idle with a message in flight) → not waiting.

**Last prose line:** strip tail markup (`‹dim›…‹/dim›` content kept only when the line has other text, `‹rev›`, `‹cursor›`, `‹‹`→`‹`); cut at the last composer prompt line (trimmed starts with `❯` or `›`), keep lines above it; split into paragraphs at blank lines and separator-only lines (`─`/`━`/`-`); drop paragraphs whose first line starts with `✻` or `※`, and paragraphs where every line is dim-only; the answer is the last line of the last paragraph left. Question = that line, with trailing `*_`)"'` and spaces trimmed, ends with `?`.

**Episode:** keyed by (binding, `LastActivity`). Once a notice for that key is *admitted* by the broker, no further notice until `LastActivity` changes (the slot worked again). A failed admission (hub busy, hub absent, allowance) leaves it unmarked; the next tick retries. Decisions "not waiting" are cached per (binding, LastActivity, Settled, resting, pending) so an idle resting slot costs one tail read per episode, not one per minute.

**Hub:** the connected binding whose verified workspace root is `messageAuthority.hub`. No hub → no notices. The hub itself is never watched: its operator is the one it would notify, and pasting into the hub's own composer would start a turn answering itself; skipping it is the strict form of #441's "not owed when the hub notifies itself".

**Notice body** (essentials first; pair#211 keeps head+tail): `slot ariadne:3 waiting 12 min on a prompt: Do you want to proceed? (couch --peek ariadne:3; mute: add ariadne:3 to <store>/wait-notice-off)`. Reasons render as `on a prompt`, `after asking`, `idle off its resting branch`. Summary capped at 200 bytes.

**Threshold:** `COUCH_WAIT_NOTICE_AFTER` (Go duration, ≥1m) read at supervisor start; default 10m; invalid → default with the error in Couch's diagnostic log line. Tick: 1 min.

**Opt-out per slot:** `<store>/wait-notice-off`, one slot reference per line, `#` comments. Couch only reads it (on each tick that would notify). Operator/TL-owned; no CLI verb.

## Lifecycle (ARCH-FUNERAL)

- Episode/decision map: in memory, one entry per connected binding, pruned each tick to the bindings in the listing; dies with the service. A Couch restart forgets it, so a slot still waiting is re-announced once — accepted.
- `wait-notice-off`: creates nothing durable from Couch; the operator's file, bounded by slot count.
- Notices: ordinary messages, so they reuse the broker's receipt retention and the session-sync ledger's existing bound.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `waitFacts` | `cmd/internal/couchcmd/wait_notice.go` | new |
| `classifyWait` | `cmd/internal/couchcmd/wait_notice.go` | new |
| `lastProseLine` / `dialogTitle` | `cmd/internal/couchcmd/wait_notice.go` | new |
| `waitNoticeBody` | `cmd/internal/couchcmd/wait_notice.go` | new |
| `parseWaitOptOut` | `cmd/internal/couchcmd/wait_notice.go` | new |

- **waitFacts** — one slot's memory facts: quiet duration, `Settled *bool`, resting, pending. 1:1 with a broker `Candidate` + `SlotLiveness`.
- **classifyWait(facts, threshold, tail []string) (reason, summary, waiting)** — rules 1–5. Needs the tail only past rule 1; the caller asks `needsTail(facts, threshold)` first so the tail read stays off the quiet path. DRY: one definition of "waiting" for the watcher and its tests.
- **lastProseLine / dialogTitle** — screen-tail readers over the tail.go markup grammar. Future: agent-specific chrome lists widen here.
- **parseWaitOptOut** — file text → set of slot refs.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `waitWatcher` | `cmd/internal/couchcmd/wait_notice.go` | new | ticker, broker listing, tail RPC, `Broker.Send`, opt-out file |
| `messageService.hubBinding` | `cmd/internal/couchcmd/wait_notice.go` | new | `workspaces` map |
| `startMessageService` | `cmd/internal/couchcmd/message_service.go` | modified | starts the watcher |

- **waitWatcher** — `tick(now)` lists actors via `broker.Actors(ctx, hub)`, joins `liveness`, reads tails through `authority.endpoint(b).(couchmessage.TailReader)` with a 2 s timeout, sends via `broker.Send(ctx, waiting, id, Route{Target: hub.Slot}, body)`. Tests drive `tick` directly over the existing `serviceRig` (real sockets, real broker) with a tail-serving fake endpoint; no timer in tests, so `TestMessageIdleMultiSlotRunsNoProbes` stays true for the rig.

## Tasks

### Task 1: pure classifier

**Files:** Create `cmd/internal/couchcmd/wait_notice.go`, `cmd/internal/couchcmd/wait_notice_test.go`

- [ ] Table tests, using tails copied from the live peeks above: Claude permission dialog (`Do you want to proceed?` + `❯ 1. Yes`) unsettled 11 min → prompt, summary names the dialog; settled turn ending `Should I merge it now?` on resting → question; settled normal ending on resting → not waiting; settled off resting no pending → idle; same with pending → not waiting; quiet 9 min → not waiting; `Settled` nil → not waiting; Codex tail (`›` composer, dim `Worked for` paragraph) → last prose line is the `•` line's paragraph end; recap paragraph skipped; `‹‹` unescaped.
- [ ] Run `go test ./cmd/internal/couchcmd -run 'Wait' -count=1` → FAIL (undefined).
- [ ] Implement `waitFacts`, `needsTail`, `classifyWait`, `lastProseLine`, `dialogTitle`, `stripTailMarkup`, `waitNoticeBody`, `parseWaitOptOut`.
- [ ] Re-run → PASS. Commit `#451: couch: waiting classifier`.

### Task 2: watcher + delivery to the hub

**Files:** Modify `cmd/internal/couchcmd/wait_notice.go`, `cmd/internal/couchcmd/message_service.go` (start it in `startMessageService` with `namespace` as store dir), test in `cmd/internal/couchcmd/wait_notice_test.go`.

- [ ] Integration tests over `serviceRig` (`world.hub = "/repo1"`, restart, hub = connect(1), worker = a hello-v2 wrapper with `SetBuild` + `Settle`, endpoint fake serving `Tail` and recording hub deliveries):
  - permission dialog left past threshold: two ticks → exactly one hub delivery, from the worker's binding, body starts `slot pair:2 waiting` and contains `Do you want to proceed?`;
  - new activity (`Update` with a later `LastActivity`) then quiet again → a second notice (new episode);
  - question ending → one notice; normal ending on resting branch → none;
  - opt-out file listing `pair:2` → none; hub's own waiting → none; no hub → none.
- [ ] Implement `waitWatcher{svc, threshold, optOutPath, now, seen map}` with `tick`, `hubBinding`, `run(ctx, every)`; start it from `startMessageService` (production only) with `COUCH_WAIT_NOTICE_AFTER`.
- [ ] `go test ./cmd/internal/couchcmd -count=1` → PASS. Commit `#451: couch: notify the hub of waiting slots`.

### Task 3: docs, atlas, live check

- [ ] `atlas/couch.md`: a "Waiting notices to the hub (#451)" subsection under Live peer messages (rules, episode, opt-out file, threshold env, restart re-announces once).
- [ ] `cmd/internal/couchcmd/skills/couch/SKILL.md`: tell the TL what a `slot X waiting …` notice is and how to mute a slot.
- [ ] `make build`, full `make -k test` + `go test ./...` (scratchpad TMPDIR), compare failures with main.
- [ ] Live: in pair:0 build, restart Couch (operator), leave a slot on a question/dialog past the threshold (or `COUCH_WAIT_NOTICE_AFTER=1m`), confirm ops:0 receives one notice. Ask the operator to confirm.

## Revisions

- 2026-10-11 (close review round 1): `COUCH_WAIT_NOTICE_AFTER` that is not a duration ≥1m refuses Couch's start, validated before any socket is taken. This replaces "fall back to the default with a log line": the start path has no log seam, and a silent fallback hides a typo. `dialogTitle` keys on the last selected option (`[❯›>] N.`), scanned from the bottom. The live check moved to #452.
