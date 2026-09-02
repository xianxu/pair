---
id: 000170
status: working
deps: []
github_issue:
created: 2026-09-02
updated: 2026-09-02
estimate_hours: 10.69
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

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec                 design=1.50 impl=0.12
item: issue-spec                 design=0.10 impl=0.04
item: greenfield-go-module       design=0.20 impl=0.28
item: smaller-go-module          design=0.04 impl=0.14
item: smaller-go-module          design=0.04 impl=0.14
item: smaller-go-module          design=0.06 impl=0.16
item: smaller-go-module          design=0.03 impl=0.10
item: smaller-go-module          design=0.06 impl=0.18
item: smaller-go-module          design=0.06 impl=0.16
item: smaller-go-module          design=0.06 impl=0.20
item: smaller-go-module          design=0.04 impl=0.16
item: smaller-go-module          design=0.06 impl=0.20
item: smaller-go-module          design=0.06 impl=0.20
item: smaller-go-module          design=0.04 impl=0.16
item: smaller-go-module          design=0.04 impl=0.16
item: tui-screen                 design=0.20 impl=0.24
item: tui-screen                 design=0.20 impl=0.28
item: tui-screen                 design=0.16 impl=0.20
item: cross-cutting-refactor     design=0.08 impl=0.14
item: cross-cutting-refactor     design=0.10 impl=0.20
item: cross-cutting-refactor     design=0.10 impl=0.18
item: cross-cutting-refactor     design=0.06 impl=0.16
item: cross-cutting-refactor     design=0.06 impl=0.16
item: cross-repo-refactor-small  design=0.04 impl=0.08
item: real-api-discovery         design=0.00 impl=0.18
item: scope-pivot                design=0.50 impl=0.20
item: ux-rename-iteration        design=0.15 impl=0.06
item: ux-rename-iteration        design=0.15 impl=0.06
item: milestone-review           design=0.00 impl=0.20
item: milestone-review           design=0.00 impl=0.20
item: milestone-review           design=0.00 impl=0.20
item: milestone-review           design=0.00 impl=0.20
item: milestone-review           design=0.00 impl=0.20
item: atlas-docs                 design=0.02 impl=0.05
item: atlas-docs                 design=0.02 impl=0.05
item: atlas-docs                 design=0.02 impl=0.05
item: atlas-docs                 design=0.03 impl=0.08
design-buffer: 0.15
total: 10.69
```

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only.*

**One line per instance, deliberately.** Repeated same-slug `item:` lines are
legal, and aggregating them would let a close-time miss be attributed to the
issue rather than to a primitive — which is the one thing the ledger exists to
prevent. In order, the instances are:

| Slug | Instances |
| --- | --- |
| `issue-spec` | plan authoring (this session); the M2 follow-up issue for the crash reconciler |
| `greenfield-go-module` | `Couch.Detach` + its typed operation |
| `smaller-go-module` | `SwitchTracker`; ctrl+backspace both encodings; `DetachedSessionResolver` + fake; `ProcOps.SignalGroup`; `RetireIncarnation` + the `DeleteStart` predicate; `ThreadDetached` projection; the three resume gates; `SelectUniqueResumableRoot`; `CommitStartClaim`; the `Policy`→`RepoIdentity` migration + its characterization test; the `couchcore.New` signature change + probe repair; the M2 refresh benchmark + conditional snapshot cache |
| `tui-screen` | leave as a global frame; previous + notification focus + acknowledgement; `alt+d` wiring + menu rows |
| `cross-cutting-refactor` | retire the focus ladder; D1 admission/policy sweep; D2 start-grant collapse; D3–D5 deletions; `plan_contract_test.go` digest-ledger + `NonArtifactSources` repair |
| `cross-repo-refactor-small` | ariadne policy-arm disposition, plus dropping the workflow and `make test-couch-policy-live` |
| `real-api-discovery` | zellij detach conformance (a session surviving its client's death) |
| `scope-pivot` | one in-flight design discovery (see below) |
| `ux-rename-iteration` | two TUI smoke rounds |
| `milestone-review` | M1, M2, M3, M4, issue close |
| `atlas-docs` | M1, M2, M3, M4 + project file |

**Where judgment entered:**

- **`issue-spec` for plan authoring takes NO spec-quality discount.** Step 3's
  discount exists because the spec pre-resolved a primitive's decisions, and the
  spec-authoring primitive cannot be pre-resolved by its own output. The issue
  Spec fixed the behavior but explicitly left the hard question open ("deciding
  what of `couchcore` is deleted ... needs a read of the current surface"), plus
  the detach mechanism and three key-layer ambiguities. Range top, ×1.0.
  Empirical check: `started:` is 11:03 and the plan cleared its gate at ~12:07,
  all of it design, so ~1.07h was already consumed before implementation began —
  against 1.85h budgeted here. An earlier draft of this block discounted it ×0.5
  and budgeted 0.98h, which the clock had already overrun at gate time.
- **Every other design line takes ×0.2.** The plan fixes files, signatures, test
  strategies and mechanical guards per task, so the remaining design cost is
  reading rather than deciding — the condition Step 3 names.
- **`ux-rename-iteration` ×2, at ×0.5 design.** `estimate-logic-v2.1.md`'s Known
  Limitations names UX iteration round count (3–5 typical for TUI features, not
  1) as a documented unparameterized bias, and this is a keyboard-UX issue whose
  five-layer smoke (Ghostty → couch → pair → zellij → claude) covers behaviour
  the plan admits no test proves end to end. Two rounds rather than three,
  because the Spec pinned the keys and the switch rule unusually precisely — the
  iteration is about feel, not about decisions.
- **`scope-pivot` ×1, at full design (×1.0).** A pivot is by definition not
  pre-resolved by the spec. It is budgeted prospectively, not retrospectively:
  this plan discovered three load-bearing mechanisms *during pre-flight review*
  — `RetireIncarnation` (round 1), `ProcOps.SignalGroup` and the `DeleteStart`
  predicate (round 2), the `arrival` argument and the tombstone branch order
  (plan-quality round 1) — and a design with that profile discovers a fourth in
  flight. `pair#152`, the nearest sibling, budgeted the same line at the same
  values. This is not double-counting the `issue-spec` line, which covers design
  already done; this covers design not yet done.
- **Design buffer +15%, not +30%** (v2.1 Step 6): the plan doc is thorough, and
  +30% on top of a ×0.2 discount double-counts the same thoroughness.
- **`impl=` values are already v3.1-scaled** to 40% of the v2/v2.1 table. No
  separate scale field.
- **`familiarity: 1.0`** on its own terms: couch is twelve recently closed issues
  of established patterns, so the impl multiplier is neutral. (Plan thoroughness
  is deliberately *not* an input here — v2 Step 5 keeps familiarity off design,
  and Step 3 is where spec quality is already paid. Using it twice would
  double-count.)

**Known risks to this estimate**, stated rather than buried:

- **M4 is the tail.** The deletion milestone has the widest blast radius and the
  least mechanically verifiable cost up front — dominated by what the compiler
  finds after `policyresolver` leaves, across four packages plus a probe binary.
  If this estimate misses, that is where.
- **`milestone-review` sits at the model's ceiling and may still be low.** Five
  boundaries at 0.5h × 0.40 is twelve minutes of ship wall-clock per boundary,
  covering an auto-dispatched fresh-context review *and* its fix round-trips.
  Review overhead is the least AI-compressible primitive in the table, so the
  flat 0.40 impl scale is least supported here — `baseline-v3.1.md` open question
  #2. Left model-conformant rather than hand-tuned.
- **Direction: contested, and the closest prior argues LOW.** An earlier draft of
  this block concluded "more likely high than low" from the issue's own ratio
  table (shape-known couch issues at 1.19–2.27x). That comparator set is wrong for
  this issue: #156/#158/#159/#167 are all 6–21 items and 0–1 milestones. The
  structural match is **`pair#155`** — same v3.1 model, 20 items, 3 milestones,
  three `cross-cutting-refactor` lines over a migration touching every Go, shell,
  launcher and Neovim consumer — and it closed **7.85 estimated against 14.01
  actual (0.56x)**, under-calling by ~1.8x. This issue is larger: 4 boundaries,
  five packages, a deletion milestone. So the honest read is that `cross-cutting-
  refactor` at this scale has one precedent and it ran nearly double; the
  `issue-spec` line's ~0.7h of slack does not cover that. The number is left as
  derived rather than hand-inflated — bending items to hit a feared total is the
  back-fitting the gate exists to catch — but the risk direction is recorded here
  so the close-time ledger reads it correctly.

## Plan

Design landed at `workshop/plans/000170-rescope-couch-to-couch-lite-plan.md`.
Four review boundaries; each is independently operable.

- [x] Claim, then design the rescope via `superpowers-writing-plans` into
      `workshop/plans/000170-rescope-couch-to-couch-lite-plan.md`. The hard part
      is not the new behavior but deciding what of `couchcore` is deleted; that
      needs a read of the current surface before it is committed to.
- [ ] M1 — Switch rule and key layer. `SwitchTracker` as a pure model with
      tests before any wiring; `ctrl-space` becomes switcher-only and the
      focus ladder plus root-actor concept go; `ctrl+backspace` in both
      encodings, including the `panelkeys.go` modified-flag fix; `alt+x` on
      the panel becomes `leave couch`; the switcher opens focused on the
      actor with the latest notification.
- [ ] M2 — Detach. `alt+d` intercepted from the canonical chord table;
      `ThreadDetached` derived (not persisted) from `launcher`'s existing
      0-client zellij classification; detached rows listed and reattachable;
      `leave couch` detaches every thread instead of parking them.
- [ ] M3 — Start or resume in a folder. `SelectUniqueParkedRoot` widens to
      `SelectUniqueResumableRoot` over parked *or* detached rows.
- [ ] M4 — Delete the machinery the rescope orphans: admission + the fleet
      policy provider, start grants, legacy migration, the never-instantiated
      actor loop, and the dead registry-era surface. Atlas and project file
      updated; operator smoke on the real stack.

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

### 2026-09-02 — design complete

Plan at `workshop/plans/000170-rescope-couch-to-couch-lite-plan.md`, four
milestones.

**Four decisions taken with the operator, each closing a Spec ambiguity:**

1. `ctrl-space` from an actor opens the switcher; `ctrl-space` *inside* the
   switcher still opens the start form. The ladder that dies is child ->
   root-actor -> panel; the panel's own `ctrl-space` is not a rung of it, and
   it is the only route to starting a thread (`couchtty/menu.go:318`).
2. `alt+x` on the panel means `leave couch`. Removing the root-actor concept
   removes `leave`'s only trigger (`couchtty/console.go:1101`). Deriving it
   from focus — alt+x quits what you are looking at — needs no new key and
   reuses the existing typed `leave` confirmation.
3. `leave couch` detaches every thread instead of parking them. Raised by the
   operator: `alt+d` is categorically safe, `alt+x` is not, because park kills
   the agent process — so parking on the way out kills every agent, including
   ones mid-turn. This makes *detached* the normal resting state, which is what
   makes M3's startup reattach worth having.
4. couch keeps intercepting `alt+x`. Interception does not add the risk:
   `pair quit` kills the agent identically whether couch intercepts or pair's
   own `PairConfirmQuit` handles it. What interception buys is the durable park
   transaction — without it couch sees only a child exit, the incarnation
   becomes conservative `unknown`, and `ProjectActionableThreads` hides the row
   (`couchcore/actionableinventory.go:106-126`), so the thread disappears from
   the switcher. Not intercepting is strictly worse.

**The deletion decision** (the plan item flagged as the hard part). Applying the
issue's own razor honestly — *machinery that exists only to defend multi-owner
or multi-host cases* — deletes less than the Problem section's framing suggests,
because several named candidates defend a single-host failure instead. Deleted:
admission + the ariadne fleet-policy provider (with its stateful fake and the
`test-couch-policy-live` conformance target), start grants, legacy migration and
cutover, the never-instantiated actor loop, and the dead registry-era surface in
`couch.go`/`couchcmd`. Kept, each with a stated reason in the plan: the
supervisor lease (it *enforces* the singleton couch-lite asserts), the park
transaction (a two-process handshake is protocol defence, not multi-owner, and
`VerifiedPark` *is* the parked row), the write-ahead journal (single-host crash
safety; its non-journal helpers must survive regardless), the fail-closed
actionable projection (the switcher's only data source — M2 extends it), the
artifact controller (only `Claim`/`Release` are "collision", and they serialize
against standalone pair), and the start transaction (single-host crash recovery;
deleting it means rewriting the start path, which is the ontology churn this
rescope exists to stop).

**Two findings that shaped the design, both from reading rather than guessing:**

- `launcher` already models detached sessions: `SessionDetached` is a live
  zellij session with zero clients (`launcher/session.go:10`,
  `launcher/list.go:23`, `launcher/zellij.go:30`) and `launcher/decision.go:33-37` already
  reattaches onto one. So couch's detached state needs no new observation
  machinery and no new `ThreadRecord` field — it is *derived* on each inventory
  refresh, consistent with couch's existing "liveness is recomputed, never
  stored" rule. Detach is therefore just the existing process-group teardown
  minus the session deletion, with the surviving session as its success proof.
- Deleting the policy provider threatened to orphan `path-preferences/` (the
  operator's per-path agent+argv memory), because `advanceSuccessfulStart` keys
  those files by `incarnation.Policy.RepoIdentity`
  (`couchcore/threadstore.go:613-618`). It does not have to: `repo_identity` is
  just the git common dir — verified against the live store
  (`"repo_identity": "/Users/xianxu/workspace/tools/.git"` for
  `physical_path: /Users/xianxu/workspace/tools`) and against ariadne's own
  fixtures (`fleet/json_test.go:61`) — and couch already has a `GitRunner` seam,
  so the identical value is derivable locally. `ThreadIncarnation.Policy`
  becomes `RepoIdentity string` and every existing preference file stays
  readable; a characterization test pins the digest before anything moves.

`ARCH-DRY` (one selector, one observation authority, chord bytes from the
canonical table), `ARCH-PURE` (the switch rule and the detached rule are pure;
teardown and observation stay in the shell), `ARCH-CONSTRAINTS` (the switcher's
committed keystroke envelope is the budget; the new session query runs once per
refresh on the existing single-flight worker, off the keystroke path).

**Plan review round 1** found a material hole in the detach design, worth
recording because the fix changed the design rather than the prose. The draft
claimed detach needed no durable transition. It does: `FinalizePark`
(`couchcore/threadstore.go:391`) is the only path that removes an incarnation,
and `reconcileInterruptedStarts` only touches records with an open start
transaction — so killing the pair client would leave a dead-PID
`IncarnationLive` on the record forever. That hides the row in the projector,
and `DecideResume` refuses any occupied incarnation at `couchcore/resume.go:73-86`
*before* it reaches the verified-park check, so the thread could never reattach.
Both detach Done-when bullets would have failed. Fixed by adding
`ThreadStore.RetireIncarnation` — `FinalizePark`'s removal half without the park
transaction — and requiring **zero** incarnations for `ThreadDetached`, which
keeps the projector fail-closed instead of teaching it to tolerate stale state.

Two further corrections worth keeping: detach must **not** reuse `handleCleanup`
(`couchcore/couch.go:450-479`), which SIGKILLs unconditionally and says so in its
own comment — under the leave-detaches-everything decision that would run against
every thread on every exit, contradicting the safety argument the decision rests
on; detach now sends SIGTERM only and fails safe. And `LookupTrees`/`knownTrees`/
`Describe` were wrongly listed for deletion: they are live via `ResolveRef` <-
`couchcore/operationdispatch.go:197`, the `stop` arm.

One pre-existing gap is named and excluded rather than absorbed: a couch that
dies *without* leaving cleanly still leaves stale `IncarnationLive` records whose
threads are invisible and unresumable. That is today's behavior, not a
regression; `RetireIncarnation` is the transition a startup reconciler would use,
and a follow-up issue is filed at M2 close.

**Plan review round 2** verified those fixes and found seven more. Two would
have shipped broken and one destroyed data:

- `ReconcileResumeAdmission` (`couchcore/admission.go:183`) is a *second*
  `VerifiedPark == nil` refusal, reached from `ResumeContext` after
  `DecideResume` returns. Widening only `DecideResume` would ship a detached row
  whose Enter fails "is not verified parked". Both are widened now, and M4's
  `CommitStartClaim` carries the precondition forward so the fix does not
  silently regress when admission is deleted.
- Detached resume could **delete the thread record**. `DeleteStart`
  (`couchcore/threadstore.go:724-756`) keeps the record when a verified park
  exists and otherwise falls through to `deleteThreadIf` — so any post-claim
  failure would remove the record, its label, description and
  `LatestLaunchProfile` while the zellij session kept running.
  `starttransaction.go:83-86` names the premise: the verified park *is* the
  rollback authority. Fixed as the class rather than the instance
  (`ARCH-PURPOSE`) — a record carrying a `LatestLaunchProfile` is durable
  history and is never deleted.
- Detach needed a seam that does not exist: `ProcOps.Signal` is single-PID, and
  under couch the sidecars deliberately share the actor's process group
  (`launcher/osruntime.go:399-408` suppresses `Setsid`), so detach would orphan
  the session-watcher and title-poller every time. Added `ProcOps.SignalGroup`,
  implemented by lifting the existing `signalOwnedProcessGroup`
  (`couchcore/runner.go:180-190`) out of `execHandle`.

Also: `leave`'s confirmation is thread-bound and fails at five sites with no
live actor — one of them asynchronously on the next refresh — so it becomes a
**global** frame, the shape the menu already uses for the start form;
`DetachedSessions` takes addresses rather than returning a whole set, because
the session-name index is per repo scope; and the switcher's envelope claim was
wrong — `ZellijSource.Snapshot` spawns 2 + N subprocesses
(`launcher/zellij.go:15-41`), now bounded to detach *candidates* with a
measurement step before M2 closes.
