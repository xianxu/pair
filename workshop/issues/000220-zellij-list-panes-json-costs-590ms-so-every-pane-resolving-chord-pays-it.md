---
id: 000220
status: working
deps: []
github_issue:
created: 2026-09-09
updated: 2026-09-09
estimate_hours: 1.07
started: 2026-09-09T13:39:07-07:00
---

# zellij list-panes --json costs 590ms, so every pane-resolving chord pays it

## Problem

Operator report: `Alt+Shift+←/→` (switch the right pane's tabs from the draft,
`#216`) has *"too much of a delay"* next to focusing the right pane and pressing
`Alt+←/→`.

That comparison is not the defect. The in-pane chord is in-process — bytes reach
`pair term`'s stdin and `handleTerminalChord` calls `mux.previousTab()` — so it
is ~0ms and nothing crossing a process boundary will match it. The defect is
what the crossing costs, and it is **not** the process boundary either.

### Measured on this machine, 2026-09-09

| call | median of 5 |
|---|---|
| `/usr/bin/true` | 6.0 ms |
| `bin/pair --version` (binary load) | 9.0 ms |
| `zellij action list-panes` (plain) | 22.6 ms |
| **`zellij action list-panes --json`** | **590.7 ms** |
| `zellij action list-panes --json --command --state --geometry` | 598.6 ms |
| `pair layout switch-terminal-tab` end-to-end | **631 ms** |
| `pair layout focus-terminal` end-to-end | **671 ms** |

**It is `--json` alone.** Adding `--command`, `--state` and `--geometry` costs
nothing measurable on top; dropping `--json` takes 590ms to 22ms. zellij
0.44.3's JSON pane serialization is 26x its plain listing.

`layoutcmd.OSRuntime.ListPanesJSON` runs
`zellij action list-panes --json --command --state --geometry`, so **every
pane-resolving action pays ~600ms**:

- `pair layout focus-terminal` — `Alt+k` from the draft or agent pane
- `pair layout toggle-focused` — `Alt+Shift+Enter`
- `pair layout switch-terminal-tab` — `Alt+Shift+←/→` (`#216`)

**This is pre-existing.** `Alt+k` has cost ~670ms since `layoutcmd` was written;
`#216` inherited it and put it on a chord pressed often enough to notice.

### Why the call is avoidable

`resolveRightTerminal` needs one thing: the right terminal's pane id. Both
callers use only `terminal.ID`. Two sidecars already answer that without zellij:

- `$PAIR_TERMINAL_PANES_PATH` — the TerminalPaneRegistry, `paneID pid` per line,
  self-registered by each `pair term`, deduped by pane id and filtered by pid
  liveness.
- `$PAIR_LAST_TERMINAL_PANE_PATH` — the recorded last-used split half.

The pane list is only needed for the tie-break `pickRightTerminal` applies when
the registry is ambiguous AND no half is recorded — zellij focus, then pane
order.

Second, smaller cost on the same path: `procutil.Alive` shells out to
`kill -0 <pid>` — one subprocess per registry line (~6ms each) for what is a
single syscall.

## Spec

**Resolve the right terminal from the sidecars when they can answer, and call
zellij only when they cannot.**

- Fast path, no `list-panes`: read the live registry ids. If the recorded
  last-terminal is among them, that is the answer. If there is exactly one live
  id, that is the answer — there is nothing to tie-break.
- Slow path, unchanged: two or more live ids and no recorded preference. Fall
  back to today's `list-panes --json` + `pickRightTerminal`, preserving the
  zellij-focus-then-pane-order tie-break exactly.
- The fast path must produce the SAME id the slow path would in the cases it
  handles, or `Alt+k` and `Alt+Shift+←/→` start disagreeing about which split
  half they mean — the property `#216` BR-10 exists to protect.
- `procutil.Alive` becomes a `syscall.Kill(pid, 0)` check. `EPERM` means the
  process exists but is not ours: alive. Every caller benefits.

**Not in scope:** matching the in-pane chord's latency. A cross-process delivery
cannot reach ~0ms without a resident control channel in `pair term`'s input
loop, which is a much larger change to a surface `#199`/`#216` showed is
delicate. The goal is to remove the 590ms that has no reason to be there.

## Done when

- `pair layout switch-terminal-tab` and `pair layout focus-terminal` no longer
  call `list-panes --json` when the registry can answer, measured before/after
  on a quiet host and recorded in the `## Log`.
- The fast path and the slow path are proven to agree on the same inputs.
- The slow path still runs, and is still correct, when the registry is ambiguous
  and no half is recorded.
- `procutil.Alive` performs no subprocess.
- No regression in `#216`'s tests or `#123`/`#199`'s focus behaviour.

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: pensive              design=0.10 impl=0.05
item: smaller-go-module    design=0.05 impl=0.10
item: smaller-go-module    design=0.10 impl=0.16
item: smaller-go-module    design=0.10 impl=0.16
item: milestone-review     design=0.00 impl=0.20
design-buffer: 0.15
total: 1.07
```

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only. Rows: the measurement already taken and its
Log write-up; `procutil.Alive`; the shared sidecar-first resolver plus its
agreement and ambiguity tests; the two `termcmd` callers; one close review. Frequency note per `#201`: this is an
interactive navigation gesture pressed repeatedly, so latency is worth spending
on here in a way it was not for a per-send cost. (`sdlc estimate-source` reports
the calibration doc `[stale]`, #127.)

## Plan

- [x] **The rule (PQ-3), because two of five callers is the easy subset.**
      Every `ListPanesJSON()` caller is either resolving a pane **ID** — which
      sidecars already answer — or needing **geometry / the full pane set**,
      which only zellij can answer. Enumerated, all five:

      | caller | needs | disposition |
      |---|---|---|
      | `layoutcmd:61` `resolveRightTerminal` (`Alt+k` from draft/agent, `Alt+Shift+←/→`) | right terminal ID | sidecar-first |
      | `termcmd:568` `currentRightTerminalPane` (`Alt+Shift+d` split) | right terminal ID | sidecar-first, same resolver |
      | `termcmd:144` `focusedWorkbenchPanes` (`Alt+j`/`Alt+k` **from the terminal** — PQ-3's named instance) | **not IDs only** — `run.go:129` passes the whole `panes.focused` to `RoleForPaneWith`, which reads `IsPlugin`/`TerminalCommand`/`Title` | role is known **by construction** on the live path: `run.go:150` already records that bytes on `pair term`'s stdin can only come from its own pane, so role = `PaneRoleRightTerminal`, focused ID = `CurrentPaneID()`, draft ID = cached. Falls back to the pane list when `CurrentPaneID()` is empty (the `--test-shortcut` path, which has no live pane) |
      | `draftroute:81` `RouteLua` | draft ID | **already** sidecar-first — the precedent, see below |
      | `layoutcmd:189` `RunToggleFocused` (`Alt+Shift+Enter`) | pane **geometry** for the resize planner | keeps `list-panes --json`; no sidecar carries geometry |

- [x] **Reuse `draftroute`'s shape, do not invent one (ARCH-DRY).** `RouteLua`
      already does cache-then-`list-panes`-on-miss (`route.go:60-70`, `:79-87`),
      and the part to copy is the PURE half: `ValidateCachedDraftPane(data, session,
      alive)` takes bytes and an aliveness predicate and returns an id — no
      Runtime, no IO (`ARCH-PURE`). `RouteLua` itself is the IO shell around it.
      The terminal resolver mirrors the validator, not the shell.
- [x] **What the registry is, and is not (PQ-1).** `$PAIR_TERMINAL_PANES_PATH`
      is written by each `pair term` at startup, so it is a **subset** of the
      right terminals — a pane that has not registered yet is absent. The fast
      path is therefore conditional, never assumed:
      - `0` live ids → fall back. Absence proves nothing.
      - recorded last-terminal is live **and in the registry** → answer.
      - exactly `1` live id **and no record** → answer.
      - `≥2` live ids and no record → fall back; that is precisely the
        zellij-focus tie-break only the pane list can make.
      The failure mode if registration races: the chord targets a registered
      half while an unregistered one exists — the same wrong-half risk the slow
      path already carries from a stale record. Acceptable, and stated rather
      than discovered. Note the slow path **already trusts this registry**:
      `isRightTerminal` / `RoleForPaneWith` take `terminalPaneIDs` to recognise
      split halves at all, so this is not new trust, only earlier trust.
- [x] **`procutil.Alive` without a subprocess, guarded (PQ-4).**
      `syscall.Kill(pid, 0)`, with `EPERM` meaning "exists but not ours" →
      alive. **Route through the existing `positivePID`**, which
      `identity_darwin.go:14` and `identity_other.go:14` already use and `Alive`
      does not: `kill(0, …)` signals the whole process group and `kill(-1, …)`
      every process the user owns, so an unguarded pid is a correctness bug of a
      different order than a slow one.
- [x] **The pure decision, named (PQ-2).**
      `resolveFromSidecars(liveIDs []string, lastTerminal string) (string, bool)`
      — no `Runtime`, no IO, mirroring `ValidateCachedDraftPane`'s shape
      (`ARCH-PURE`). It returns `false` for "sidecars cannot answer", which is
      the only signal the IO shell needs to decide whether to spend the 590ms.
- [x] **Agreement by GENERATION, not by hand (PQ-2).** The adversarial class is
      *the registry disagreeing with the pane report*, and hand-picked cases are
      blind to it by construction — I would only write the disagreements I
      already thought of. So: generate the cross-product of
      **pane set** {command-classified, title-classified, registry-only-visible}
      x **registry subset** {empty, proper subset, disjoint from the pane set}
      x **record** {none, live-and-registered, stale, unregistered-but-present},
      and assert the one property that matters:
      **`resolveFromSidecars` answering implies its answer equals
      `pickRightTerminal`'s** on the same inputs. Where it declines, assert only
      that the caller falls back. That property is what stops `Alt+k` and
      `Alt+Shift+←/→` from disagreeing about a split half (`#216` BR-10), and it
      holds or fails over the whole generated space rather than over my
      imagination.
- [x] **Fall-back-still-runs test.** With two live ids and no record, assert the
      pane list IS consulted — otherwise the fast path silently eats the
      zellij-focus tie-break.
- [x] **Re-measure end to end** on a quiet host, before/after, and record both
      in the `## Log` with the agent population, per `#201`'s Done-when.

## Log

### 2026-09-09

Filed from operator report on `#216`. Measured before designing, per `#201`'s
lesson — the first hypothesis was the process boundary and the subprocess count,
and the measurement refuted both: the boundary costs ~31ms of the ~631ms, and a
single zellij flag costs the rest.

### 2026-09-09 — implementation and measurement

**Before/after**, same host, 26 zellij sessions, load 3.3 — median of 6, the
same method as the `## Problem` table:

| path | before | after |
|---|---|---|
| `pair layout switch-terminal-tab` (`Alt+Shift+←/→`) | 631 ms | **55 ms** |
| `pair layout focus-terminal` (`Alt+k` from draft/agent) | 671 ms | **27 ms** |

The residue is honest and irreducible without a new mechanism: ~9ms `pair`
binary load + ~22ms for the `zellij action write` that actually delivers the
chord. Matching the in-pane chord's ~0ms needs a resident control channel in
`pair term`'s input loop; out of scope, and stated as such in the Spec.

**The generated agreement test earned its keep immediately.** It found two
divergence classes before the comment describing them existed:

- `registry=subset, record=none` — one right terminal registered, another not:
  fast answers the registered half, `pickRightTerminal` answers the pane-order
  first. The startup race.
- `registry=disjoint` — a registry id naming no pane at all: fast returns an id
  that does not exist.

Hand-picked cases would have been blind to both, exactly as PQ-2 predicted. So
the property is **scoped rather than weakened**: agreement is asserted over
registry-CONSISTENT pane sets (the state the registry invariant maintains —
`LiveIDs` filters on the registering `pair term` being alive, and that process
does not outlive its pane), 216 generated cases, 54 of which the fast path
answers. For the inconsistent classes the weaker invariant that actually holds
is asserted instead: **the fast path never returns an id the registry did not
list**. With an incomplete registry and no recorded half both paths are
guessing, and asserting they guess alike would assert a coincidence, not a
contract.

**Deployment note, discovered by the operator rather than predicted.** The
speed-up reached a running session with **no restart**: the draft's chord does
`jobstart({… '/bin/pair', 'layout', 'switch-terminal-tab', dir})`, a fresh
process per keypress, so rebuilding `bin/pair` is enough. What does NOT reach a
running session is anything compiled into a long-lived process — the two
`termcmd` fast paths (`focusedWorkbenchPanes`, `currentRightTerminalPane`) live
inside `pair term` and need that pane to restart. Worth stating because it is
the inverse of `#216`, which needed a restart precisely because its change was
new Lua held in nvim's memory.

**`procutil.Alive`** is now `syscall.Kill(pid, 0)` routed through the existing
`positivePID`, with `EPERM` read as alive. The guard is not decoration:
`kill(0, …)` signals the caller's whole process group and `kill(-1, …)` every
process the user owns, and `Alive` was the one caller not gating on it — safe
only because a subprocess `kill` merely failed on those inputs.

### 2026-09-09 — boundary review round 1: FIX-THEN-SHIP, addressed

**BR-7 — the generated space never reached the branch it claimed to cover.**
The first generator used two-pane worlds only, so a registry of size 1 was
always *incomplete* and `resolveFromSidecars`' single-live-id branch either went
unexercised or diverged. The axis that matters is **completeness, not size**:
rebuilt over one- and two-pane worlds with registries that are empty or
complete. 360 cases, 108 answered, **54 through the single-live-id branch**, all
agreeing. A generated space proves nothing about a case it cannot produce, and
the first one could not produce this.

**BR-3 — both `termcmd` fast paths executed in zero tests.** Written and
shipped-to-review with a coverage count of 0 on every line. Now: the saving is
asserted (no `ListPanesJSON` call when registered) and so is the gate (falls
back when unregistered, when there is no current pane id, and when the draft
cache misses).

**BR-4 — reached past the injectable seam.** `focusedWorkbenchPanes` called
`draftroute.CachedDraftPaneIDFromEnv` while `Runtime.CachedDraftPaneID`
(`run.go:29`) existed for exactly this. That is what made the branch untestable
in the first place; BR-3 and BR-4 are the same mistake seen from two sides.

**BR-5 — the real defect of the five.** The fast path synthesised a
right-terminal role from `ZELLIJ_PANE_ID` alone. That env var says *which* pane
we are, not *what kind* — so any pane running that code would have had every
chord routed as a right terminal. Both fast paths now gate on a shared
`registered(rt, paneID)` predicate, which is the signal that actually means
"right terminal" and the same one `RoleForPaneWith` uses for split halves that
zellij reports without a `terminal_command`. Verified by mutation: dropping the
gate reddens.

**BR-6 — and a claim of mine that was wrong.** `Alive`'s two declared behaviours
were unpinned; both now have tests, and dropping the `positivePID` guard reddens
with `Alive("0") = true`. While writing the test I had to correct the comment I
had written with the change: **signal 0 is the null signal**, so the guard does
not prevent a delivered broadcast — it prevents a *false positive*, since
`Kill(0, 0)` and `Kill(-1, 0)` both succeed and a registry line reading `1 0`
would report a live pane that does not exist. The pid-selector danger is why the
guard exists in the codebase, not what it stops here.

### 2026-09-09 — boundary review round 2: the reachability guard was itself unfalsifiable

**BR-12.** The guard I added for BR-7 — "the generated space must reach the
single-live-id branch" — counted `len(registry) == 1`, an **input** property. A
one-entry registry also answers through the *record* branch, so the guard
survived deleting the single-live-id branch outright: it was asserting something
about the generator's inputs while claiming something about the code's paths.

Now it counts the branch's own signature — answered **with no recorded half**,
which only that branch can produce. The count fell from 54 to 18, so 36 of the
cases I had been reporting as branch coverage were record-hits. Verified by
mutation: deleting the branch now fails with *"the single-live-id branch is
unreached, so this space cannot prove what it claims"*.

Worth naming as a class, because it is the third variant of one mistake in this
session: a guard that reports on its own inputs rather than on the behaviour it
exists to pin. `#216`'s route scan asserted its premise instead of the routing;
its first cut mis-scanned and called eight families unroutable; and this one
counted a fixture shape instead of an execution.

### 2026-09-09 — boundary review round 3

**BR-13 — the rule: a fast path is pinned by its ANSWER, not by its saving.**
Asserting `listCalls == 0` and that the gate declines pins the speed-up and the
guard while leaving the actual result unchecked. Only one of three fast paths
had a differential oracle (`resolveFromSidecars`, 360 cases). Both `termcmd`
fast paths now run against the SAME fixture as the slow path — gate on, gate off
— and assert the two answers are identical.

The finding also caught a hole I had built in: my fixture set `cachedDraft: "2"`
while its draft pane was also id 2, so "reads the cache" and "agrees with the
report" were indistinguishable, and a fast path reading the wrong sidecar would
have passed. The draft is now id 7, distinct from every other id in the fixture.
Verified by mutation: returning `draftID + "9"` fails with *fast = draft "79";
slow = draft "7" — the two paths disagree*.

**BR-14 (Minor)** — registry membership was open-coded a third time.
`workbenchshortcut.Registered(ids, paneID)` now owns it, and `RoleForPaneWith`,
`resolveFromSidecars` and `termcmd.registered` all call it. The registry owns
its own predicate, so a normalisation rule or a second field has one home.
