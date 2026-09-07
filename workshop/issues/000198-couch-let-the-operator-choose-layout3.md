---
id: 000198
status: working
deps: []
github_issue:
created: 2026-09-06
updated: 2026-09-06
estimate_hours: 3.81
started: 2026-09-06T12:06:08-07:00
---

# couch: let the operator choose layout3

## Problem

couch always launches `pair resume <tag> --layout2`. The operator wants
`--layout3` — pair's own right-hand user terminal — available under couch.

**The literal is one line** (`couchcore/launch_existing.go:50`), and the code
anticipated this request:

> The layout: pinned to layout2 by operator decision 2026-08-22. couch owns
> terminal switching now, so layout3's third pane — pair's own user terminal —
> is the layer couch replaces. Provisional ("for now"), which is why it is a
> literal here rather than a knob nobody has asked for.
> — `couchcore/couch.go:427`

So this is the sanctioned reversal of a provisional decision, not a fight with
the design. But the change is **not** one line, for four reasons the tree
already records.

**1. The pin is a decision with a rationale, recorded in five places.** The
literal at `launch_existing.go:50`, the rationale at `couch.go:427-431`, and
`atlas/couch.md` at `:201`, `:815`, `:880`, `:903-904` ("Layout stays pinned to
layout2: couch owns terminal switching, so layout3's third pane is the layer
couch replaces"). Editing the literal and leaving those is exactly the stale-map
failure — the atlas would teach a rule the binary no longer follows.

**2. The neighbouring path is destructive.** `launch_existing.go:46-49` (#179):

> `--layout2` is dropped [for a warm reattach] for the same reason: a running
> session already has its layout, and asking for a different one sends Pair down
> a conflict path that offers to **DELETE the live session** — destroying the
> agent this exists to preserve.

The existing `in.Warm` branch already sends no layout at all, so it is safe
today. Any change here must preserve that invariant exactly: the layout flag is
for a COLD start/resume boundary and must never reach a warm reattach. That is
what makes this a change to review rather than a literal to swap.

**3. Scope of the choice is a real fork, not a detail.** `StartArgs` is
persisted "so a revival reproduces the launch without the operator restating
it" (`couchcore/startargs.go`). So:

- **Per-thread** — layout belongs in `StartArgs`, and a revived thread comes
  back in the layout it was started with. Matches how agent and argv already
  work.
- **Namespace-global** — a `couch --layout3` flag applying to every thread it
  starts, held wherever couch keeps per-namespace preference.

These differ in observable behavior after a park/resume cycle, so the plan must
pick one deliberately. Per-thread is the shape the surrounding data already
has, and is the recommendation absent a reason otherwise.

**4. ~18 couch-side test references pin layout2**, several with comments
asserting the pin is intentional (`couch_test.go:1666`: "Layout pinned to
layout2 (operator decision 2026-08-22)"; `cmd/couch/main_test.go:154`: "want
**exactly** `pair resume <generated-couch-tag> --layout2`"):

| file | refs |
|---|---|
| `cmd/internal/couchcore/couch_test.go` | 5 |
| `cmd/internal/couchcmd/run_test.go` | 5 |
| `cmd/couch/main_test.go` | 3 |
| `cmd/internal/couchcore/runner_test.go` | 3 (generic argv fixtures, not pins) |
| `cmd/internal/couchcore/warmresume_test.go` | 1 (pins the #179 omission — must stay) |
| `cmd/internal/couchcore/resume_launch_test.go` | 1 |

Their **premises** need correcting, not their assertions flipped: "couch pins
layout2" becomes "couch sends the thread's chosen layout at a cold boundary and
none at a warm one". The `warmresume_test.go` case is the one that must keep
asserting exactly what it asserts today.

**Mechanism is confirmed to work.** `pair resume <tag> --layout3` parses:
`ParseArgs` runs `extractLayoutRequest` first (`launcher/args.go:51`), stripping
layout flags before the positional guard, and `launchArgsAcceptLayout` admits
them for resume. `couch.go:433-439` records this as measured, not reasoned —
including the correction of an earlier comment that claimed the opposite.

**Honest sizing:** a small feature — flag, plumbing, the persistence decision,
the test premises, and the doc sweep — not a one-line change.

## Spec

The operator can start a couch thread in layout3, and it stays layout3 across a
park/resume cycle.

- Layout is **per-thread**, carried in `StartArgs` alongside agent and argv,
  defaulting to layout2 (unchanged behavior for every existing thread and every
  persisted record).
- The flag reaches the argv **only at a cold start/resume boundary**. A warm
  reattach continues to send no layout flag at all — the #179 invariant is
  preserved verbatim, not re-derived.
- Old persisted records with no layout field read as layout2.

Open for the plan: the operator-facing surface — a `couch` CLI flag at start, a
menu affordance on the thread, or both. Whichever is chosen, it sets the
persisted per-thread value rather than a global mode.

**Worth deciding explicitly, not assuming:** the pin's rationale was that couch
*replaces* layout3's third pane. Running layout3 under couch means the operator
has both couch's terminal switching and pair's right terminal at once. That is
presumably the point of the request — but it is the substance of the reversal,
so record the answer in the issue rather than letting the code imply it.

## Done when

- A thread started with layout3 launches `pair resume <tag> --layout3`, and the
  operator gets pair's right terminal inside couch.
- That thread comes back as layout3 after a park/resume cycle; a thread started
  without the flag is layout2.
- A **warm reattach sends no layout flag**, pinned by the existing
  `warmresume_test.go` case still passing unmodified.
- Persisted records written before this change load as layout2 with no error.
- Test premises corrected across the couch-side files above; no test still
  asserts couch pins layout2 as policy.
- `atlas/couch.md` updated at all four sites (`:201`, `:815`, `:880`,
  `:903-904`) and the rationale comment at `couch.go:427` replaced with the new
  rule — the 2026-08-22 pin is recorded as reversed, with the reason.

## Plan

Superseded in shape by
`workshop/plans/000198-couch-let-the-operator-choose-layout3-plan.md` (ten
tasks); kept here as the outcome ledger.

- [x] Decide the operator surface (CLI flag / menu / both) and confirm
      per-thread persistence.
      → **CLI flag only, and NOT per-thread.** The operator chose a couch-global
      setting with no mixing; a menu affordance would have implied the
      per-thread fork they rejected. See `## Revisions`.
- [x] ~~Add the layout field to `StartArgs`~~ **→ `ThreadRecord.Layout`**, with a
      layout2 read for records that predate it; test the old-record path.
      → **The Spec's mechanism did not exist.** `StartArgs` is rebuilt at resume
      from `ThreadRecord` + `LatestLaunchProfile`, so a field there would have
      been dropped by the very park/resume cycle it had to survive. The witness
      is normalized at `ProjectActionableThreads`, pinned by
      `TestProjectionNormalizesAbsentLayoutAndDoesNotBlockDefaultStartup`.
- [x] Thread it to `launchTrackedThread`'s cold-boundary argv, leaving the
      `in.Warm` branch untouched.
      → argv takes `c.Layout.Flag()`; the warm branch is byte-identical and now
      records no witness either.
- [x] **Added by the Revisions:** the couch-global flag and the startup guard
      refusing a layout that would mix with a session-holding thread.
- [x] Correct the test premises; verify `warmresume_test.go` passes unmodified.
      → `warmresume_test.go` untouched (`git diff --stat` empty). One assertion
      strengthened rather than reworded: run_test.go checked only `--layout2`,
      which would now miss couch leaking `--layout3` onto a warm path.
- [x] Sweep `couch.go:427` and the four `atlas/couch.md` sites.
- [x] Manual: start a layout3 thread under couch, park it, resume it, confirm
      the layout survives and the live session is never offered for deletion.
      → Operator-run on the real six-thread store; see the verification Log
      entry. No deletion prompt at any point.

## Log

### 2026-09-06

Filed from an operator request, with the question "I think it's just one line
change?" — the honest answer is that the literal is one line and the change is
not, for the four reasons in Problem. The comment at `couch.go:427` explicitly
left room for this knob, so the design does not resist it; what makes it more
than a swap is the destructive warm-reattach neighbour (#179), the persistence
fork, and eighteen test references plus five doc sites that encode the old pin
as policy.

**The three connected pieces.** This issue is the flag; `#199` builds the tab
strip `pair term` draws in its own pane; `#200` makes that strip clickable on
one shared mouse arbitration. Only this one touches couch — `#199`/`#200` are
`termcmd` + `hostty` work — so **neither gates couch-lite's close**.

### The reversal, on the record

`#198`'s Spec asked for the reason rather than letting the code imply it. Two
independent reasons, and they point the same way.

**1. The pin's rationale did not survive the rescope.** The 2026-08-22 pin reads
"couch owns terminal switching now, so layout3's third pane — pair's own user
terminal — is the layer couch replaces". That was an actor-cluster-era claim,
made when couch was specified to own everything on the host
(`workshop/projects/couch.md`, 2026-08-21). `#170` rescoped couch to
**couch-lite** on 2026-09-02: a switcher over live coding sessions, with
Admission — fleet capacity and incumbency, its cross-repo provider dependency,
its stateful fake and its live conformance target — deleted whole. couch-lite
switches *agent sessions*; it does not hand the operator a shell at their cwd.
So the layer couch was said to replace is one couch-lite never took over. The
reversal is a correction the rescope already implied, not a change of taste.

**2. The operator's three original reasons have each moved** (2026-09-06):

| original reason for layout2 | what changed |
|---|---|
| keep it simple initially | couch-lite works well and is close to done (2026-09-03 scope event: one remaining issue) |
| unsure whether the right pane should host a web browser too, cmux-style | dropped — the pane is terminal-only, and constant access to a terminal at the same cwd is wanted |
| reservations about the right pane's quality: zellij tabs, the title mechanism, mouse | `#199`/`#200` — couch's status bar generalizes into a tab bar the pane manages itself |

The third reason is the one that was a genuine blocker, and `#172` is why it is
cheap to answer now: it turned a reserved interactive row from an idea into
working code, with its mouse-mode hazards found and fixed across four rounds
(BR-16 → BR-22 → BR-26 → BR-33/`#196`).

**Relationship to #194** (`open`, "Toggle a terminal pane in layout2"): both
answer "the operator wants a terminal", from opposite directions. #194 argues
layout3's permanent 50%-width third pane is the wrong posture and adds an
on-demand terminal to layout2 instead; this issue makes that permanent pane
available under couch. Not contradictory — different postures for different
sittings — but whoever picks up the second of the two should check whether the
first changed the want. Recorded rather than resolved: that is the operator's
call.

### Planning, and a measurement caveat for the close

Planned on 2026-09-06. `sdlc start-plan` → durable plan at
`workshop/plans/000198-couch-let-the-operator-choose-layout3-plan.md` → three
plan-quality rounds. Round 1 blocked on a Critical worth recording, because it
was a real design defect and not a wording problem: the guard compared the **raw**
persisted layout, so a pre-change record's `Layout("")` would not equal
`Layout2` and **every existing thread would have blocked a default `couch`
startup**. `ParseLayout("") -> Layout2` was defined and then never wired in. The
fix was the class, not the site: an enumeration of the two places a raw layout
string becomes a decision input, each with its normalizer, plus `NormalizeLayout`
and the `LayoutUnknown` sentinel for the projection that has no error return
(ARCH-SECURE).

**Measurement caveat (from the estimate-quality check).** At planning time
`sdlc actual --issue 198` reads ~0.29h for the window, against 1.97h of buffered
design — because the planning artifacts were still untracked and had not crossed
a commit boundary. Today's spans are also being split across seven issues
(`#112, #172, #196, #197, #198, #199, #200`) by mention fallback, several flagged
"100% mention fallback without issue commit boundary". So if this closes well
under 3.81, check attribution before reading it as estimate drift — the layout3
trio (#198/#199/#200) shares session time by mention. Recorded now so the ledger
row is not mistaken for calibration evidence (ariadne#117/#127).

### 2026-09-06 — implemented; the backfill, measured

Tasks 1-9 landed. `couch --list` against the real store, with the new binary:

| thread | path | state |
|---|---|---|
| brain | `~/workspace/brain` | live (pid 7821) |
| tools | `~/workspace/tools` | live (pid 43964) |
| parley.nvim | `~/workspace/parley.nvim` | live (pid 8396) |
| ariadne | `~/workspace/ariadne` | detached |
| pair | `~/workspace/pair` | live (pid 7314) |
| arc-agi-3 | `~/workspace/kbench/.../arc-agi-3` | live (pid 89296) |

Six, as the issue estimated. **Every one holds a session** (five live, one
detached), and none carries a `layout` field — they all predate #198, so they
normalize to layout2, which is true: couch pinned layout2 for their whole
lifetime.

**No migration script is needed, and this is why.** A default `couch` requests
layout2, every row normalizes to layout2, so the guard finds zero conflicts and
startup is unaffected. Each thread's witness is then written on its next cold
resume. The absent-field path is pinned by
`TestProjectionNormalizesAbsentLayoutAndDoesNotBlockDefaultStartup`.

**The consequence worth stating plainly:** because all six hold sessions,
`couch --layout3` **refuses today until all six are parked**. That is the
no-mixing rule working as specified rather than a defect — but it makes the
first use of `--layout3` a deliberate act, not a quick experiment. The refusal
names every conflicting thread and the remedy.

**A guard that is not the safety mechanism.** Worth keeping straight for anyone
reading this later: #179 (a warm reattach sends no layout flag) is what prevents
the destructive outcome, and it holds whether or not the guard runs. The guard
only buys predictability. So a race between the guard's snapshot and a session
appearing is cosmetic, which is why a plain startup check was the right
strength and nothing here is transactional with the store.

**Verification so far.** `go test ./cmd/...` green. Mutation checks run on the
four load-bearing invariants — the empty-layout normalization, the blocking
predicate (swapped for `Resumable()`), the warm-reattach argv and witness, and
the guard's IO budget — each confirmed failing before being restored. Task 10
(the manual park/resume cycle) is outstanding and needs the operator.

### 2026-09-06 — manual verification, by the operator

Task 10 run on the real store, against the six-thread inventory recorded above.
What was observed, in order:

1. **`couch --layout3` refused** while all six threads still held sessions,
   naming them. This is the guard's whole point, and it was exercised against
   the live store rather than a fixture.
2. **Every thread parked, then `couch --layout3` started.** The refusal is a
   reachable state, not a trap: parking empties the blocking set, which is
   exactly why parked threads are excluded from it.
3. **`brain` resumed from parked and came back in layout3** — the three-pane
   right side present. This is the cold-resume migration: `brain` was parked
   from a layout2 world and revived into layout3, so its witness followed its
   session rather than the store keeping a stale claim.
4. **Detach tested under layout3** and behaves.

No deletion prompt appeared at any point in the cycle — the #179 failure mode
(pair offering to DELETE a live session when asked for a different layout) did
not surface, which is what the cold/warm split exists to prevent.

Recovery snapshot taken before the switch and kept at
`~/couch-recovery-2026-09-06/` (store backup + per-thread zellij session and
transcript id). Not needed, but the switch was one-way for six live agents and
was worth insuring.

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec                 design=1.10 impl=0.10
item: greenfield-go-module       design=0.20 impl=0.20
item: smaller-go-module          design=0.05 impl=0.14
item: smaller-go-module          design=0.08 impl=0.20
item: smaller-go-module          design=0.06 impl=0.18
item: smaller-go-module          design=0.06 impl=0.16
item: smaller-go-module          design=0.06 impl=0.20
item: cross-cutting-refactor     design=0.08 impl=0.16
item: atlas-docs                 design=0.03 impl=0.08
item: smaller-go-module          design=0.01 impl=0.04
item: milestone-review           design=0.00 impl=0.20
item: milestone-review           design=0.00 impl=0.16
design-buffer: 0.15
total: 3.81
```

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only.* The calibration source reports `stale` (the
ledger is newer than the doc, ariadne#127), so the per-primitive hours are
provisional.

Design is dominated by `issue-spec`, and that is not front-loading credit taken
twice: the Problem section's archaeology was written before this session, and
today added the `## Revisions` (an operator fork plus a correction to the Spec's
mechanism), a ~750-line plan, and **three plan-quality rounds** — round 1 refused
with a Critical (`PQ-1`) whose fix changed the design, adding
`NormalizeLayout`/`LayoutUnknown` and the normalization table; round 2 passed but
raised `PQ-8`; round 3 disposed of it. The remaining design is small because the
plan resolved the open questions: the six-state disposition, the guard's
placement, and the witness's transaction are decided in prose, so each Go item is
transcription against a named anchor rather than a choice.

The `+15%` design buffer (not `+30%`) is v2.1's thorough-plan-doc rule.

**Revised after the estimate-quality check (3.73 → 3.81).** Four of its five
observations said the first block ran low, and all four are accepted:
Task 9 had no line item at all; `milestone-review impl=0.24` was above the
primitive's scaled ceiling (0.08–0.20) *and* was blending a fresh-eyes review
with Task 10's six-step manual park/resume cycle, so those are now two items;
the five `smaller-go-module` rows sat uniformly at the scaled maximum and now
differ by actual scope (Task 3's three files and schema-2 fixture above Task 2's
single pure predicate); and the prose said two gate rounds where the ledger
records three. The fifth observation is about measurement, not the estimate, and
is recorded in `## Log`.

| Slug | Instances |
| --- | --- |
| `issue-spec` | the issue's Problem/Spec, the `## Revisions` entry, the durable plan, and three plan-quality gate rounds |
| `greenfield-go-module` | Task 1 — `layout.go`: a new type with two normalizers and the `LayoutUnknown` sentinel |
| `smaller-go-module` | Task 2 the guard predicate; Task 3 the `ThreadRecord`/`threadrecord` field plus the projection's normalization point and old-record fixture; Task 4 the argv emission and `StartEvent.Layout`; Task 5 the `StartInteractive` guard and its refusal text; Task 6 the CLI flag and its typed plumbing; Task 9 the backfill check |
| `cross-cutting-refactor` | Task 7 — correcting ~19 test premises across six files, one of which (`warmresume_test.go`) must verifiably not change |
| `atlas-docs` | Task 8 — the `couch.go:427` rationale plus four `atlas/couch.md` sites |
| `milestone-review` | ×2 — Task 10's six-step manual verification, and the single close boundary review (this is single-pass work: one `sdlc close`, no `Mx` tags) |

## Revisions

### 2026-09-06 — layout is couch-global, not per-thread; and StartArgs cannot carry it

Two changes to the Spec, one from an operator decision and one from a fact in
the tree that the Spec got wrong.

**1. Operator decision: couch-global, no mixed layouts.** The Spec proposed
per-thread layout ("carried in `StartArgs` alongside agent and argv"). The
operator chose the other fork: *"it should be a couch global setting, either
whole thing is `--layout2` or whole thing is `--layout3`"*, and on being offered
the mixed-state option, *"I like the layout to be predictable, so no don't mix
layout."*

So the flag is `couch --layout3`, applying to every thread that couch starts or
cold-resumes, and a **startup guard refuses to start in a layout that conflicts
with a thread already holding a session in the other one**. The operator
proposed this guard and answered its obvious objection: refusing is not a dead
end, because *"if `couch --layout3` is refused, user can start with
`couch --layout2`, park, and try again"* — the remedy is reachable through the
tool itself. (I had argued the refusal stranded the operator; that was simply
wrong, and conceded.)

**2. Correction: `StartArgs` is not persisted, so it cannot carry the layout.**
The Spec's mechanism does not exist. `StartArgs` is documented as persisted, but
what persists is `ActorRecord.Args` in the **Registry**, which is explicitly *"a
transitional display/handle cache. It decides nothing: ThreadStore is the
durable authority"* (`registry.go:17-23`). A parked thread has no ActorRecord at
all. On resume, `StartArgs` is **rebuilt from scratch** out of `ThreadRecord` +
`LatestLaunchProfile` (`resume.go:445-448`) — agent and argv survive a park via
`LatestLaunchProfile`, *not* via StartArgs.

A `StartArgs.Layout` field would therefore be silently discarded by exactly the
park/resume cycle the Spec's second Done-when requires it to survive. The layout
witness goes on `ThreadRecord` instead.

**Delta to `## Done when`:** the second bullet ("that thread comes back as
layout3 ... a thread started without the flag is layout2") is replaced by: every
thread couch starts or cold-resumes takes couch's process-wide layout, and couch
refuses to start when a session-holding thread disagrees with the requested one.
The warm-reattach bullet, the old-record bullet, the test-premise bullet and the
atlas bullet are unchanged.

**Delta to `## Plan`:** step 1 ("decide the operator surface ... confirm
per-thread persistence") is answered by this revision; the remaining steps are
superseded by `workshop/plans/000198-couch-let-the-operator-choose-layout3-plan.md`.
