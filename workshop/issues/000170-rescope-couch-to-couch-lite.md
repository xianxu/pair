---
id: 000170
status: working
deps: []
github_issue:
created: 2026-09-02
updated: 2026-09-02
estimate_hours:
started: 2026-09-02T11:03:39-07:00
---

# Rescope couch to couch-lite

## Problem

couch has consumed ~172 measured hours across twelve closed issues (#145, #146,
#149, #151, #152, #154, #155, #156, #158, #159, #161, #167), of which ~139 went
into the switcher and identity layers. The operator has driven it for a day or
two and it is currently generating papercuts (#163, #164, #165, #168, #169)
faster than value. It replaced the substrate — tabs became a switcher — without
yet adding a capability.

**Root cause: no razor-clear view of what couch is.** That gap got filled by
generality. `cmd/couch` + `cmd/internal/couchcore` is ~22k lines carrying
admission control, supervisor leases, start grants, park transactions, a
write-ahead journal and fail-closed projections — distributed-systems machinery
defending one operator on one host. The oscillation showed up as repeated
redesign of threading structure and agent selection. pair avoided the same
ambiguity by exposing CLI options and letting usage reveal the right
configuration; couch decided in advance and encoded the decision in types, so
every later discovery meant changing the ontology instead of adding a flag.

The estimate ratios separate cleanly along that seam:

| shape unknown (deciding while building) | shape known (building a behavior) |
| --- | --- |
| #146 0.28x, #149 0.32x, #154 0.27x, #151 0.38x | #156 2.27x, #158 1.72x, #159 1.19x, #167 1.95x |

Secondary: ephemeral runtime state is harder to get right (timing-dependent)
and more opaque to a coding agent than repo state. Failures like #169 are
transient — by the time anyone inspects, the subprocess error is gone.

## Spec

Code name **couch-lite**. The binary stays `couch`. The target is a switcher
over a group of coding tasks — closer to cmux than to an actor cluster — whose
unit is a **pair session**, not a terminal. That is the one thing that cannot be
bought, and it is the boundary that keeps the scope from creeping back.

### Behavior

1. **Start or resume in a folder.** `couch` in a directory starts a
   preconfigured agent for that tree, or resumes an existing session there.
   Resuming a **live** session is new: today only parked sessions are
   resumable.

2. **Singleton directory.** couch remains a singleton and lists both parked and
   **detached** sessions; either can be resumed or reattached. "Detached" means
   couch's own child running without the tty. That state is currently
   unreachable in a healthy run because couch exit always parks, so it needs an
   `alt+d` detach mirroring pair's.

3. **Notifications.** Sessions notify as they do today: the actor's colour
   changes in the status row and in the switcher. `ctrl-space` opens the
   switcher **focused on the actor with the latest notification**, so following
   a page is one key plus Return.

4. **A single `previous` slot.** Switching records the actor being left;
   `ctrl+backspace` returns to it. One slot, not a stack — a stack the operator
   cannot see is a stack they will lose track of.

5. **Notification hops are ephemeral.** Arriving via ctrl-space + Return *on an
   actor that had a pending notification* is notification-handling mode, and
   such an actor never becomes `previous`. So chasing two pages, or detouring
   manually to spot-check a third actor, still leaves `ctrl+backspace` pointing
   at the actor the operator was actually working in.

### The switch rule

All of the above is one rule. Carry a single boolean on the current actor,
`entered_via_notification` — set only when arrival was ctrl-space + Return AND
the target had a pending notification — and on **every** switch, whatever the
mechanism:

```
if !current.entered_via_notification { previous = current }
current = target
```

Consequences, all derived rather than special-cased:

- First hop from working actor A pins A.
- N1 to N2 leaves A pinned.
- A manual detour from a notification actor to C leaves A pinned.
- `ctrl+backspace` out of a notification actor lands on A with `previous == A`,
  so the next `ctrl+backspace` is a no-op. **Intended**: the operator is home
  and there is nowhere to bounce to.

An actor does not notify while the operator is attached to it.

### Keys

- **`ctrl-space` = switcher, and nothing else.** It no longer means "up one
  level"; the child -> root-actor -> panel ladder goes away, and with it the
  root-actor/home concept (`couchtty/console.go:68`).
- **`ctrl+backspace` = previous.** This is the key labelled `delete` on an Apple
  keyboard, not forward-delete — no `fn` in the chord.
- **`alt+d` = detach.**

Encodings verified by probe in Ghostty, outside zellij, 2026-09-02:

| key | legacy | kitty flags=1 (what zellij pushes) |
| --- | --- | --- |
| plain backspace | `7f` | `7f` |
| **ctrl+backspace** | **`08`** | **`\x1b[127;5u`** |

Distinct in both modes, and both are exact strings, so they go into
`knownSequences` as two entries — the same dual-encoding shape ctrl-space
already has (`NUL` plus `\x1b[32;5u`). No new parser, no timing window.

Two existing sites currently swallow it and must gain the modifier branch:

- `couchtty/panelkeys.go:98` — `case b == 0x7f || b == 0x08` consumes the
  legacy form as `KeyBackspace`.
- `couchtty/panelkeys.go:198` — `case 127, 8: return KeyBackspace` ignores the
  `modified` flag computed at :193, so `\x1b[127;5u` decodes as a plain
  backspace.

Missing either gives a home key that works everywhere except inside the
switcher, which is where it is most used.

**Accepted cost:** in legacy encoding `0x08` *is* `^H`, so intercepting
ctrl+backspace also takes ctrl-h from the child (readline and nvim insert-mode
treat it as backspace). Under the kitty protocol they separate
(`\x1b[104;5u` vs `\x1b[127;5u`), and zellij pushes the protocol, so this only
bites with the protocol off. Deliberate, not a discovery.

`alt+d` reuses `workbenchshortcut.ChordEncodings(ChordAltD)` rather than
duplicating protocol literals — the pattern couch already uses to intercept
pair's `ChordAltX` as `seqPark` (`couchtty/keys.go:69-75`). pair already binds
`ChordAltD` to `ActionConfirmDetach` (`workbenchshortcut/shortcut.go:120`).

### Out of scope

- **One LLM stack, one path.** No cross-stack codex support; it has already
  produced #144, #161 and #166.
- **No cluster transport or query dialect (#147)**, no capability manifests, no
  vocabulary-derived response shapes.
- **No brain advisor (#148)**, no mesh, no relay.
- Machinery that exists only to defend multi-owner or multi-host cases —
  admission incumbency, start grants, park transactions — is a deletion
  candidate, subject to the plan below.

## Done when

- `couch` in a directory starts a preconfigured agent for that tree, or resumes
  the session already there, live or parked.
- The switcher lists parked and detached sessions; both reattach.
- `alt+d` detaches a session without parking it, and the detached session is
  listed and reattachable.
- `ctrl-space` opens the switcher focused on the actor with the latest
  notification; with no notifications pending it opens on a defined default.
- `ctrl+backspace` returns to `previous`, and a unit test over the switch rule
  asserts that a notification-entered actor never becomes `previous` across the
  N1 -> N2 -> manual-detour sequence.
- `ctrl+backspace` is recognised in both encodings, including inside couch's
  own panel.
- `workshop/projects/couch.md` carries a scope event recording the rescope, and
  #147, #148 and #153 are dispositioned against it.

## Plan

- [ ] Claim, then design the rescope via `superpowers-writing-plans` into
      `workshop/plans/000170-rescope-couch-to-couch-lite-plan.md`. The hard part
      is not the new behavior but deciding what of `couchcore` is deleted; that
      needs a read of the current surface before it is committed to.
- [ ] Switch rule: `entered_via_notification` plus the single `previous` slot,
      as a pure model with tests, before any key wiring.
- [ ] Key layer: `ctrl-space` becomes switcher-only; add `ctrl+backspace` (both
      encodings) and `alt+d`; fix the two `panelkeys.go` sites.
- [ ] Detach: `alt+d` path, and detached sessions listed and reattachable.
- [ ] Resume a live session in the current tree.
- [ ] Notification focus in the switcher.
- [ ] Delete the machinery the rescope orphans.
- [ ] Scope event in `workshop/projects/couch.md`; disposition #147, #148, #153.

## Log

### 2026-09-02

Opened from a brain session working over
`brain/workshop/pensive/2026-08-20-01-pensive-couch-agent-switcher.md`. The
rescope is the operator's call; this issue records the reasoning so the
oscillation that caused the overrun does not restart.

Key-encoding probe run in a plain Ghostty tab outside zellij. Outer terminal
replied `\x1b[?0u` to the kitty-protocol query (nothing pushed at that level).
A third probe phase at kitty flags=15 produced a finding worth keeping even
though it is not work: bit 8 is "report all keys as escape codes", so `ctrl-c`
never reaches the tty as `0x03` and `isig` never fires, and every press also
emits a release event (`\x1b[113;1:3u`). A press-only exact-string table would
miss at that level, and a tolerant parser would fire twice per press — which is
exactly what the `couchtty/keys.go` comment already warns about. zellij pushes
flags=1, so this belongs in the table's comment rather than in the work.

**Raised and not decided**, carried here so they are not lost:

- couch-lite does not solve the problem the project was opened for. The
  original pain was *forgetting a thread exists*, with a dated cost — the rogii
  submission whose 2026-08-05 deadline passed unnoticed. A switcher does not
  catch that; a durable `{tree, what, when}` list plus a clock would, and needs
  none of the fleet, transport or advisor machinery. Whether couch-lite keeps
  that piece is open.
- Whether a durable append log of operation attempts (`{op, args, outcome,
  error}`) generalises #169. The failures being chased are transient, so
  live introspection helps only the hung case; a journal helps the case that
  already went wrong. `couchcore/storejournal.go` and pair's existing jsonl
  ledgers are the existing muscle.
