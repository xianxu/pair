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


### Detach must never strand a thread

`alt+d` is reachable, so the state it produces must be reachable back. Today it
is not: a detached thread whose native binding is incomplete appears **nowhere**
in the switcher, and the only way back in is `zellij attach '<session>'` from
outside couch.

Reproduced 2026-09-03, thread `couch-8d1a4da0f9fe730d`:

- record has no incarnation (detach cleared it) and no `verified_park` (detach
  destroys nothing, so it correctly writes none) — so it is neither `live` nor
  `parked`, the only two states `ActionableThreadInventory` emits;
- `📁pair-couch-24` is alive with no client, and `session-names.jsonl` maps it
  to the thread correctly;
- its ledger holds `launch × 1` and **zero `binding` records** (a healthy
  thread has `binding × 2`), so there is no `NativeID`;
- `ProjectDetachedSessions` requires Agent **and** NativeID to emit a complete
  observation, so it emits none, and the row never appears.

The trigger is ordinary, not a crash: two candidate claude transcripts appeared
at startup (`c7245bc1`, `b36b9e12`), and round-gated identity discovery refuses
to rank or guess between roots, so it bound neither.

**The gating looks wrong on its own terms.** A detached row's Enter is a
*reattach to a live zellij session by name* — the conversation is already loaded
in the running session. `NativeID` is the proof needed to **resume a parked**
thread, where the process is gone and must be reconstructed. Requiring
native-resume proof for a reattach imports a precondition from the wrong
operation. `ProjectDetachedSessions`'s own comment says `SessionDetached` is
"exactly the state an `alt+d` leaves behind and exactly what `pair resume`
reattaches onto" — and that reattach needs the session name, which the index
has.

So: **gate the detached row on the session-name binding, not on the resume
proof.** The fail-closed cases the comment defends (two addresses bound to one
session name; two zellij rows sharing a name) are genuine ambiguities about
*which session* and must still refuse. A missing `NativeID` is not that.

**Invariant to hold, whichever way the gating lands:** if a detached thread
cannot be offered a row, `detach` must refuse rather than produce the state. A
reachable operation must not create an unreachable one.

**The second half — Enter is refused for EVERY detached thread.** Confirmed by
reading the chain: `createflow.go:238` refuses on `ResumeRequired &&
decision.Action != ActionCreate`; `decision.go:32-36` resolves a `ForcedTag`
onto a blocking session to `ActionAttach`; `decision.go:95` counts
`SessionDetached` as blocking. couch resume spawns `pair resume <tag>` with
`ResumeRequired`, so a detached thread always lands on `ActionAttach` and is
refused — a healthy binding and a correct `NativeID` do not help.

So making the row appear does not restore reattach. The blocker is that
`ValidateTrustedLaunchProfile` (`launch_args_policy.go:105`) rejects
`ResumeRequired` with an empty `RequiredSessionID`, so the launch profile has no
way to say *"reattach to this session name; no native proof needed"* — even
though `AttachExistingSession` takes only tag/session/agent and
`RequireNativeResumeBinding` lives only inside `runCreate`. The fix is a second
authority on the profile, not a loosened gate.

**Hazard for that work:** with no `NativeID`, a session that dies between
observation and launch leaves nothing to recover from — it must **refuse**, never
fall through to create, or a fresh empty agent silently replaces the
conversation. `NativeID` is a fallback for *reconstruct*, not a precondition for
*reattach*.

`createflow_test.go:417` (`TestRequiredNativeResumeRefusesAttachRace`) pins the
current refusal and will fail when this is fixed — a deliberate decision, not an
accident. It also passes for the wrong reason today: it sets no binding
statuses, so the unbound path refuses first and the attach-boundary guard it
names is never reached. Its setup needs correcting either way.

### Keys

- **`ctrl-space` = switcher, and nothing else.** It no longer means "up one
  level"; the child -> root-actor -> panel ladder goes away, and with it the
  root-actor/home concept (`couchtty/console.go:68`).
- **`ctrl+backspace` = previous.** This is the key labelled `delete` on an Apple
  keyboard, not forward-delete — no `fn` in the chord.
- **`alt+d` = detach EVERY actor and return to the shell.** Operator decision,
  2026-09-03. The scope mapping is the argument: the gesture means "detach the
  thing I am in", and in couch that is the fleet, not one row — which is also
  what the operator expected it to do before being told otherwise.

  That is `leave`'s existing behaviour ("Detach every active work thread and
  leave Couch", `ops.go:234`), so `alt+d` becomes **`leave`'s chord**, not a new
  operation (`ARCH-DRY`). Keep `leave`'s `ConfirmRequired`: it ends the session.
  The per-thread detach *operation* stays available from the panel; what goes
  away is the per-thread *chord*, so one key does not mean two scopes.

  **Ordering constraint, blocking:** this must NOT land before the attach path
  is fixed. Every detached thread is currently unreattachable
  (`createflow.go:238`), so a fleet-wide `alt+d` strands the WHOLE fleet in one
  keystroke instead of one thread. The detach guard on
  `000170-detach-strand-fix` does not help — healthy threads pass it and are
  stranded anyway. Fix the attach path first, then rebind.

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
- A detached thread always appears in the switcher and Enter reattaches it,
  **including when no native binding was ever written** — asserted with a
  thread whose ledger has zero `binding` records. Not satisfied by the detach
  guard alone: the attach path must stop refusing `ActionAttach` first.
- Reattach with no `NativeID` whose session died between observation and launch
  **refuses**; it never falls through to create.
- Where a detached row genuinely cannot be offered (ambiguous session name),
  `detach` refuses instead of producing an unreachable thread.
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
- [x] M1 — Switch rule and key layer. `SwitchTracker` as a pure model with
      tests before any wiring; `ctrl-space` becomes switcher-only and the
      focus ladder plus root-actor concept go; `ctrl+backspace` in both
      encodings, including the `panelkeys.go` modified-flag fix; `alt+x` on
      the panel becomes `leave couch`; the switcher opens focused on the
      actor with the latest notification.
- [x] M2 — Detach. `alt+d` intercepted from the canonical chord table;
      `ThreadDetached` derived (not persisted) from `launcher`'s existing
      0-client zellij classification; detached rows listed and reattachable;
      `leave couch` detaches every thread instead of parking them.
- [x] M3 — Start or resume in a folder. `SelectUniqueParkedRoot` widens to
      `SelectUniqueResumableRoot` over parked *or* detached rows.
- [x] M4 — Delete the machinery the rescope orphans: admission + the fleet
      policy provider, start grants, legacy migration, the never-instantiated
      actor loop, and the dead registry-era surface. Atlas and project file
      updated; operator smoke on the real stack.
- [x] Operator smoke finding — rebind the lifecycle chords to the 2x2 the
      operator named (see Revisions, 2026-09-03). Key picks the disposition,
      surface picks the scope, leaving is unconditional. Atlas + README follow.

## Revisions

### 2026-09-03 — the lifecycle chords are a 2x2, and leaving is unconditional

Reason: operator smoke on the real stack (the M4 task that was deferred to the
issue close) reported *"there seems no way to quit couch now — when the last
live actor is detached or parked, the user is stuck in the switcher pane."*

The mechanism was not broken. Two tests driving the production input path
(panel focused, last child exits; and park-the-last-actor-then-leave) both
reached `leave` and exited `Console.Run`. What was broken was that nothing on
screen said so, and what the keys meant depended on where you pressed them:

| | in an actor | in the switcher |
| --- | --- | --- |
| `alt+x` | park this thread (agent dies), confirmed | **leave couch** (safe, detaches everything) |
| `alt+d` | detach this thread | **nothing** — `detach: no attached thread` |

So the exit lived on the destructive-looking key, the safe key did nothing on
the surface where the operator most wanted it, and the switcher's own key
legend (`menuControls`) was consumed only by a README test — never rendered.
`Escape` at the root with no live actor refuses with "no live thread can
receive focus", every letter is typeahead, and ctrl-c/ctrl-d are swallowed as
control bytes. Every key an operator would try either refused or filtered.

Delta, decided with the operator:

- **The key picks the disposition, the surface picks the scope.** `alt+x` parks
  (destructive), `alt+d` detaches (safe); in an actor that means one thread, in
  the switcher it means every live thread. Both switcher forms then leave couch.
- **Leaving is an invariant, not a consequence.** A whole-couch action with
  nothing live to act on still leaves — making the exit conditional on there
  being something to act on is precisely the trap reported.
- **Confirmation rides the disposition, not the scope.** Park is confirmed at
  both scopes; detach at neither. The park-all confirmation names its cost
  ("leave couch, parking 3 live threads") on the *item*, because the frame
  title never reaches the screen: `RenderMenuView` overwrites line 0 with the
  breadcrumb.
- **Detaching an actor moves focus to the switcher**, as park already did. That
  also fixes an accident: `onExit`'s `last && !panelFocused` meant detaching the
  *last* actor quit couch outright.
- `Couch.Leave` takes a `LeaveDisposition` (`LeaveDetach` | `LeavePark`) and
  refuses an unknown one rather than defaulting — guessing either way silently
  contradicts the key that was pressed. One operation with a mode, mirroring
  park's existing `mode` argument, rather than a second verb (ARCH-DRY).

This supersedes Spec §Keys' `alt+d` line and design decisions #2 and #3 in the
2026-09-02 design-complete note: `alt+x` on the panel is no longer how `leave`
is reached, and leaving now has two dispositions rather than always detaching.
Decision #3's *reasoning* survives intact and is why detach stayed the default
and the unconfirmed one.

## Log

### 2026-09-03
- Stays OPEN as the root issue for making couch usable, at the operator's
  direction. The chord rebinding landed (see Revisions), but the smoke that
  found it also found that couch cannot reattach a detached-but-running
  session, and that nine of thirteen threads are invisible. Those are pair#181
  (the inventory campaign) over pair#168, #171, #179 and #180. #170 closes when
  couch is usable, not when its own milestones are ticked.

### 2026-09-02
- 2026-09-02: closed M4 — Full `make test` exits 0 (zero FAIL lines). Round-1 findings BR-15..BR-21 all fixed and independently re-verified by the reviewer reverting each fix. Round-2 findings: (a) DeleteUnstartedThread orphan deleted + class-guarded by TestNoProductionSymbolIsReferencedOnlyByTests over couchcore with a categorized allowlist (found 12; 2 deleted, 6 named against pair#173); (b)+(c) docs guard rewritten to match per-paragraph with whitespace normalized over git ls-files instead of a hand-listed file set, re-verified against the ACTUAL pre-fix README from git history (the wrapped `sdlc fleet policy` case the line-oriented version could not fire on), and the two live-voice admission comments it then caught are corrected; (d) deadline added at the seam (repoIdentityTimeout=5s, matching the deleted ExecPolicyResolver bound) and pinned by TestRepoIdentityResolutionIsBoundedEvenWithAnUncancelledContext, which hangs without it; (e) Chunk 4 ticked and Task 13 Step 4 corrected to stop claiming DeletePristineThread was deleted; (f) peer note filed as ariadne#212, committed without touching ariadne in-flight branch work. Minors: .gitignore main-package list now derived by TestEveryMainPackageIsIgnoredAtTheRepoRoot (it had missed both generatecmd packages), resume.go shadowed duplicate removed, lessons.md updated in this same commit. One minor disposed with evidence not code: an empty path returns "spawn: no path given", not drift, because removing the inert worktree arg left WorkingDir with nothing to fall through to. Task 15 Step 4 (operator smoke on the real Ghostty->couch->pair->zellij stack) remains UNTICKED and deferred to the issue close - it is the operator to run.; review verdict: FIX-THEN-SHIP
- 2026-09-02: closed M3 — BR-4 and BR-8 addressed; all prior findings disposed last round. go test ./cmd/... green; env -u PAIR_SESSION_ID -u PAIR_TAG make test exits 0. BR-8: DetachedSessionObservation now carries Agent and NativeID exactly as ParkedResumeObservation does, and detachedResumeProofMatches is parkedResumeProofMatches twin -- both resumable kinds prove themselves the same way in the same PURE layer, so actionableThreadState fails closed on its own rather than trusting the IO shell to have filtered candidates. TestProjectActionableThreadsDetachedRequiresTheResumeProof crosses full proof / missing native id / disagreeing agent / missing session name. The shell attaches the proof it already resolved for the candidate gate, so nothing is queried twice. BR-4 atlas half: the atlas now names BOTH callers of the zellij snapshot with the measured 1.49s -- the switcher refresh is event-driven, renders last-good and converges late, while startup BLOCKS on it because StartInteractive must decide resume-vs-new before attaching, and leave-detaches makes a detach candidate the normal case; pair#172 filed for the parallelization, with the note that the candidate filter decides only whether the snapshot runs.; review verdict: FIX-THEN-SHIP
- 2026-09-02: closed M2 — go test ./cmd/... green; env -u PAIR_SESSION_ID -u PAIR_TAG make test exits 0; go build ./... and GOOS=linux go build ./... both clean. Detach: TestCouchDetach asserts the two properties that separate it from park -- Quiesce/DeleteSession never called, and no SIGKILL ever sent (fake ProcOps signal log) -- plus group signalling, a client that ignores SIGTERM leaving the thread live, a vanished session refusing, and a canceled context signalling nothing. RetireIncarnation refuses recycled PIDs, unknown incarnations, open park/start transactions and stale revisions, and keeps name+LatestLaunchProfile. Reattach: TestDecideResumeAcceptsDetachedWithoutVerifiedPark crosses the detached proof against tombstoned ParkHistory (the class that would otherwise be permanently unreattachable) and confirms the occupied-incarnation gate is unchanged; TestDeleteStartKeepsARecordThatHasEverStarted is mutation-verified against the pre-fix source and reddens for both the unnamed and named cases while a never-started record still deletes. Projection: ThreadDetached requires zero incarnations, so a stale IncarnationLive stays hidden. alt+d driven through the production input path (TestConsoleRunAltDDetachesWithoutConfirmation), which caught reduceParkHotkey discarding its effect. Envelope measured: BenchmarkMenu100 open 100us / filter 89us / navigation 211us / render 197us / refresh 214us against 50ms and 16ms budgets; the 2+N subprocess cost is bounded by TestActionableInventoryAsksOnlyAboutDetachCandidates and TestActionableInventorySkipsTheQueryWithNoCandidates.; review verdict: FIX-THEN-SHIP
- 2026-09-02: closed M1 — go test ./cmd/... green. SwitchTracker drives the Spec sequences (A->N1->N2->manual detour keeps A pinned; the second ctrl+backspace is a no-op by construction). TestEveryArrivalAcknowledgesTheLandedActor asserts the acknowledge rule across all three arrival kinds independently of previous, plus the negative (a switch that does not land keeps the bell). Both ctrl+backspace encodings decode to HitPrevious through the production interceptor at every read split and stay inert inside a bracketed paste; plain 0x7f still reaches the panel. TestDecodePanelKeysDistinguishesCtrlBackspaceFromBackspace pins the panelkeys modified-flag fix. TestLeaveConfirmationNeedsNoLiveThread + SurvivesAnInventoryRefresh cover leave from a couch with no live actor including the async reconcileMenuFrames site; TestParkConfirmationStillDiesWithItsThread pins the counterpart. Ladder deletion proved by go build ./... and GOOS=linux go build ./... with no residue. make test: test-session-watch and test-review fail identically at the merge-base (pre-existing, recorded in memory); every target after them passes.; review verdict: FIX-THEN-SHIP

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

### 2026-09-02 — M1 verification and envelope

Commands run for M1 (window `88fe1de0..HEAD`):

- `go build ./... && GOOS=linux go build ./...` — clean, which is how the focus
  ladder's removal is proved. Grep cannot see an unused method; `actorAlive` had
  to be deleted deliberately.
- `go test ./cmd/... -count=1` — green.
- `env -u PAIR_SESSION_ID -u PAIR_TAG make test` — stops at `test-session-watch`
  and `test-review`, both of which fail **identically at the merge-base**, so
  they are pre-existing rather than regressions. Verified by checking out
  `$(git merge-base HEAD origin/main)` and running each target there. Every
  target after them passes; run them explicitly, because `make test` stops at
  the first failure and would otherwise give a change no coverage at all.
- `git diff --check` — clean over the milestone's own files.

**Operating envelope (ARCH-CONSTRAINTS), measured not asserted.**
`BenchmarkMenu100`: open 99 µs, filter 87 µs, navigation 215 µs, render 202 µs —
against the committed 50 ms open and 16 ms filter/navigation/render budgets, so
roughly two orders of magnitude of headroom. M1's additions to the keystroke
path are one extra byte comparison per input byte (the legacy `0x08` branch),
one extra exact-string row in `knownSequences` (no timing window, and `\x1b[1`
was already a partial prefix via `ChordAltX`), and one `reconcileRootSelection`
per `ctrl-space` — which is O(inventory) once per keypress of that key, not per
keystroke.

**Mutation evidence for the two production seams** the boundary review found
untested: inverting the `arrival` derivation in `ExecuteConsoleOperation`
reddens `TestConsoleRunNotificationHopThenPreviousReturnsHome` (previous becomes
c2 instead of c1), and deleting `Run`'s `case HitPrevious` arm reddens it by
timeout. Both were run and reverted, so the tests are known to pin the seams
rather than merely to pass beside them.

### 2026-09-02 — M3 startup envelope, measured

The M3 review was right that the "unchanged from #167" envelope claim was stale.
Measured on this host (19 zellij sessions, 6 exited, 13 live):

- `ZellijSource.Snapshot`: **1.49 s** (`BenchmarkZellijSnapshotLive`, 5 runs).
- Breakdown: two `list-sessions` runs at ~16 ms, plus one
  `action list-clients` per **live** session at ~100 ms each, serially —
  13 × ~110 ms = 1.43 s, which accounts for essentially all of it.

Where that lands:

- **Startup: blocking.** `StartInteractive` must decide resume-vs-new before it
  attaches anything, so a detach candidate adds ~1.4 s before the first frame.
  M2 made detached the normal resting state, so this is the ordinary case, not
  the edge one.
- **Switcher open: not blocking.** Refreshes are event-driven (no ticker) and
  run on the single-flight worker while the menu renders its last-good
  projection, so the 50 ms open and 16 ms keystroke budgets are unaffected; the
  rows simply converge ~1.4 s later.

The per-session queries are independent, so this is bounded fan-out away from
~150 ms. That is a contained change but outside M3's scope, so it is filed
rather than absorbed — expanding a milestone to fix what its review measured is
how boundaries stop meaning anything.

### 2026-09-02 — M4 would have bricked the live store

The deletion sweep removes fields from types that ARE the on-disk record schema,
and `strictjson.Decode` disallows unknown fields — so a removed field makes every
record still carrying it undecodable. Measured against the live store
(`~/.local/share/pair/couch/threadstore`, 17 records):
`claim_generation` appears in **17/17**, `policy` in **5/5** incarnations,
`reservation` in none.

Dropping `ClaimGeneration` as Task 13 Step 5 describes would therefore have made
every thread in the operator's store unreadable on the next `couch` startup —
including the parked and detached history, which exists nowhere else.

Caught before writing any deletion code, by enumerating the actual keys in the
live store rather than reasoning from the Go types. Approach recorded in the plan
Revisions: the removed fields become decode-only tombstones in `threadrecord`,
so old records keep decoding and shed the fields on their next write.

### 2026-09-02 — the manifest was the worse half of the same hazard

The tombstone finding above covered thread *records*. Applying the same
measurement to D3 found the sharper case: the store **manifest** carries
`legacy_cutover: true` and `legacy_migration_version: 1`, and it is decoded by
the same strict path. A bad record loses one thread; a bad manifest loses the
whole store, because `loadManifestLocked` fails and nothing can be listed,
resumed or reattached at all.

Same remedy, separate guard: `TestPreM4ManifestStillLoads` in `couchcore`,
fixtured from the operator's real manifest. Both guards were mutation-checked
by deleting the tombstone field and confirming the test reddens.

`legacy_actor_id` got a tombstone too even though the live store has none left
(the cutover records it named were parked, which clears incarnations). "Absent
from the one store I measured" is not "absent", and the cost of being wrong is
the store.

### 2026-09-02 — two tests were passing vacuously

`TestTrackedLaunchCancellationAfterAcknowledgementReapsAndRollsBack` asserted
`ErrThreadNotFound` at a **hardcoded** address that was never the address
actually allocated, so it proved nothing. D2 shifted the entropy the tag derives
from, the address collided with the real record, and the vacuum showed. The true
behaviour is the correct opposite of the test's name: after acknowledgement the
target has executed and Pair has registered, so occupied-or-proven-free keeps
the record as an `unknown` incarnation rather than deleting it. Renamed, and now
asserted over the whole snapshot so it cannot go vacuous again.

The journal test retargeted off `CutoverLegacyActors` had the same problem in
its first form: it still passed with `withLock`'s recovery removed, because the
interrupted record's file was already on disk and the next commit clears the
journal either way. Manifest membership is the only assertion that separates
recovery from coincidence.

Rule for the sweep, applied from here on: after retargeting a test off deleted
surface, mutate the thing it now claims to pin and confirm it reddens. Three of
the retargets needed a second attempt.

### 2026-09-02 — D2 was a dead-code deletion that surfaced a live bug

The start-grant token was genuinely redundant (in-process, single-owner). But
removing it exposed that three call sites each rebuilt "the arguments that
reproduce this resolution" by hand, and getting it wrong is silent: passing the
*resolved* agent where the operator requested none changes `AgentSource`, so the
commit re-resolves to a different fingerprint and fails with
`ErrStartResolutionChanged` — a drift error for a drift that never happened.
`StartResolution.CommitArgs` now owns that contract (`ARCH-DRY`), and
`MenuFrame` holds the resolution itself instead of a seven-field copy of it.

### 2026-09-02 — M4 deletion sweep complete

D1–D5 landed in five commits. `make test` exits 0. Full detail in the plan's
`## Revisions`. One finding deliberately **not** acted on: the worktree
`NamingTable` looks dead too (live registry has `"names": {}`, no production
writer survives D5), but cutting it is wider than this sweep — left as a
follow-up.

### 2026-09-02 — M4 boundary review: FIX-THEN-SHIP, 1 Critical + 6 Important

The Critical is the one that matters: **deleting admission also deleted an
invariant**, not just its code. `AllocateThreadTag` claims the Pair artifact and
writes a pristine reservation; admission owned rolling both back on failure.
After D1 the three post-allocation failure sites called
`releaseClaimIfThreadAbsent`, which returns nil whenever the record exists — and
there it always does. Protection that can never fire, leaking a record the
switcher hides and the reconciler skips, so nothing ever reclaims it.

The lesson generalizes past this instance: when deleting a subsystem, enumerate
the *invariants* it enforced, not only the symbols it defined. A deleted
guarantee leaves no compile error.

Two related repairs:

- The tombstone work was half-done. Decodability is not enough — a pre-M4
  incarnation keeps its repository identity *inside* `policy`, so ignoring the
  tombstone loads it empty and `advanceSuccessfulStart` refuses, inside `New()`,
  for the whole store. Tolerating a legacy field is not the same as carrying its
  value forward.
- `make test-couch-policy-live` survived D1 and **reported green**: `go test`
  exits 0 when `-run` matches nothing. I had deleted the CI workflow that would
  have failed loudly and left the target that fails silently — exactly backwards
  — and then wrote in this Log that the target was gone. Both are now mechanical:
  `TestEveryMakefileTestSelectorMatchesALiveTest` and
  `TestDeletedVocabularyIsNotDescribedAsLive`, each mutation-checked against the
  case that motivated it.

Also: a 4.5 MB Mach-O binary reached a commit through `go build` at the repo root
plus `git add -A`. Dropped from the branch before it could propagate to the
dependent repos this base layer feeds.

`make test` exits 0 after all seven fixes.

### 2026-09-02 — M4 review round 2: the guards were weaker than they looked

No Criticals; the previous round's Critical was independently re-verified by the
reviewer reverting each fix and watching the test redden. What this round caught
was subtler and more useful: **both mechanical guards I added were narrower than
the rules they encoded.**

The sharpest is the docs guard. It matched per line, and Markdown prose wraps —
the real pre-fix README broke ``(`sdlc`` / ``fleet policy`)`` across a newline,
so that row could never fire on the text it was written for. My mutation check
had passed because I wrote the term on one line. **A guard verified against a
convenient reconstruction proves nothing about the real artifact**; re-verified
now against the actual file from git history.

Its file set was hand-listed too, which is the same recall step the guard exists
to remove — it omitted Go sources, and two live-voice admission comments
survived. And the Go half of that rule was never mechanised at all: deleting
`rollbackUnforkedStart` orphaned `DeleteUnstartedThread` *in the commit that
closed the orphan finding*.

The other real catch: I claimed "a hung git no longer hangs the start form".
That was false. `RunContext` carried cancellation, but the CLI passes
`context.Background()`, so nothing bounded the call — strictly worse than the 5s
timeout it replaced. **"Carries a context" is not a time bound.** The deadline is
now at the seam and reddens when removed.

Also filed `ariadne#212`, the peer-repo note Task 15 Step 0 required and the
first close skipped: ariadne's `sdlc fleet policy --json` arm lost its only
programmatic consumer when couch deleted admission, and the cross-repo
conformance target went with it. Filed as a note, not an edit — the disposition
is ariadne's.

`make test` exits 0.

### 2026-09-03

Detach strands a thread with no native binding — full evidence in the new
*Detach must never strand a thread* section above. Found from the operator
detaching a live pair thread and finding no row to reattach to; recovered with
`zellij attach '📁pair-couch-24'` outside couch.

### 2026-09-03 — detach guard landed; attach path outstanding

Branch `000170-detach-strand-fix` (`e7ecf245`, off `8bfdd846`) adds the third
precondition to `Detach`: resolve the native binding through the same
`NativeBindingResolver` seam Resume uses, refuse when it is not one established
root, before the SIGTERM so refusing costs nothing. Tests cover unbound,
ambiguous and established-with-no-root-id, each asserting the incarnation is
untouched and no group signal was sent; reverting only `detach.go` fails all
three.

That stops detach *manufacturing* the unreachable state. It does **not** restore
reattach — see *The second half* above. The invariant is not met until the
attach path lands.

Unrelated and pre-existing at `8bfdd846`: `./cmd/internal/couchcore` deadlocks
(`pair-launch-helper: acknowledgement unavailable: EOF`), reproducing with the
above changes stashed and outside the sandbox, so it is neither new nor an
environment artifact.

### 2026-09-03 — alt+d becomes fleet-wide

Operator: "detach in couch should apply to all actors as well." Recorded in
*Keys* above with the ordering constraint. Raising the priority of the attach
path: with `alt+d` fleet-wide, the existing reattach refusal turns a one-thread
mistake into a whole-fleet one, and `leave` already calls detach on every thread
on the way out — so that exposure exists today, before any rebinding.
