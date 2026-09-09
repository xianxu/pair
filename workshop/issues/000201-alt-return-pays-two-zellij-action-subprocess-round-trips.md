---
id: 000201
status: punt
deps: []
github_issue:
created: 2026-09-06
updated: 2026-09-09
estimate_hours:
started: 2026-09-06T22:06:07-07:00
---

# alt+Return pays two zellij action subprocess round-trips

## Problem

Operator report: `alt+Return` from the draft pane visibly pastes into the agent
pane, **hangs for about a second**, then the submitting Return arrives. The
pause between the two halves is the symptom, and it is the mechanism showing
through.

`alt+Return` is not one operation. It is two `zellij action` invocations —
`write-chars` then the focus/Return — and each is a **fresh subprocess** that
loads the zellij binary, connects to the server socket, and waits for a round
trip.

### Measured, on this machine (12 cores, M-series)

Decomposed by differencing three timings:

| step | median | what it adds |
|---|---|---|
| `/usr/bin/true` | 1.9 ms | OS process-spawn floor |
| `zellij --version` | 9.5 ms | + loading the zellij binary (**7.6 ms**) |
| `zellij --session S action query-tab-names` | 36.2 ms | + connect & server round-trip (**26.7 ms**) |

So on a **calm** machine `alt+Return` costs **≥72 ms** before any useful work.

**Corrected 2026-09-06:** re-measured on a genuinely quiet host (2 agents, load
2.3) the round-trip is **17.6 ms**, so the true floor is **~35 ms**, not 72 —
the earlier 36 ms reading was itself taken under residual load. The floor is
lower than first filed and still worth removing; the numbers below stand as the
loaded case.
Under the load this fleet actually runs at (concurrent `go test` across sessions,
load 25–100 on 12 cores) the same call measured **145 ms median, 467 ms max** —
putting `alt+Return` at **290 ms typical and ~930 ms at the tail**, which is the
"about a second" the operator sees.

Note the shape: the subprocess path is **4× more load-sensitive** than a plain
process wake-up. A controlled experiment (12 CPU burners at normal priority)
moved simple wake-up latency by only 0.3 ms — but spawn→dynamic-link→connect→
round-trip degrades hard, because it is four scheduling points, not one.

### The blast radius is wider than alt+Return

`zellij action` as subprocess-per-operation is used across pair, so every one of
these pays the same 36 ms floor:

- `termcmd/run.go:191` — `RunZellijAction("focus-pane-id", ...)`
- `termcmd/run.go:959-963` — `rename-pane`, on **every tab title change**
- `layoutcmd/layoutcmd.go:57` — `focus-pane-id`
- `clipcmd/runtime.go:142,145` — `focus-pane-id`, twice (bare then `terminal_`
  form), on the copy-on-select handoff path (`#125`)

`clipcmd` is the notable one: it tries the bare form, and on failure tries the
prefixed form — so a miss costs two full round-trips before the paste lands.

## Spec

**Stop paying a process spawn per zellij operation on interactive paths.**

Three directions, cheapest first; the plan picks after measuring which of the
7.6 ms (binary load) and 26.7 ms (connect + round-trip) actually dominates in
the real call, since `query-tab-names` may not be representative of
`write-chars`:

1. **Collapse the two calls into one.** If `write-chars` can carry the
   submitting Return in the same payload, `alt+Return` halves immediately with
   no architectural change. Check whether the two-step exists for a reason —
   the agent may need the paste to settle before the Return, in which case the
   gap is deliberate and the fix is a different one. **Settle this first; it may
   be the whole issue.**
2. **Hold a persistent connection.** pair already runs long-lived processes next
   to every session (`pair term`, the title poller). One that keeps the zellij
   socket open turns each action into a socket write rather than a spawn.
3. **Bypass zellij for writes we own.** couch already owns the child pty and
   `pair term` owns its pane's; writing bytes directly is cheaper than asking
   zellij to do it. Bounded by which panes we actually own.

Out of scope: the load that multiplies this 4× — that is `#203`. This issue is
about the 72 ms floor that exists on an idle machine.

## Done when

- `alt+Return` is a single round-trip, or none.
- Benchmarks are taken on a quiet host with the agent population recorded
  alongside — an ambient-load number is not a floor (this issue's own first
  measurement was 2× off for exactly that reason).
- Measured before/after on a calm machine, reported as medians with the same
  method as above — the floor must actually move, not just feel better.
- The `rename-pane`-per-title-change path (`termcmd/run.go:959`) is addressed or
  explicitly excluded with a reason.
- `clipcmd`'s two-attempt focus (`runtime.go:142,145`) no longer costs two
  spawns on the miss path.
- No regression in `#125`'s copy-on-select handoff or the source-pane gate.

## Plan

- [ ] Settle Spec item 1 — can the two calls be one? Check whether the paste
      must settle before the Return.
- [ ] Measure `write-chars` specifically (not `query-tab-names`) and split its
      cost between binary load and round-trip.
- [ ] Implement the cheapest sufficient direction.
- [ ] Re-measure `alt+Return` end to end, calm and under load.
- [ ] Sweep the other `RunZellijAction` interactive callers listed above.

## Log

### 2026-09-06

Found while debugging an operator report of general workbench sluggishness. Three
distinct causes came out of that session and are filed separately: this one
(`#201`), the draft's per-keystroke work (`#202`), and unbounded build
parallelism across couch sessions (`#203`).

Method note worth keeping: the decomposition above (`true` → `--version` →
`action`) is what turned "zellij feels slow" into "7.6 ms of binary load and
26.7 ms of round-trip", which is what makes the three fix directions
distinguishable. A single end-to-end timing would not have.

Correction recorded because it shaped the filing: the session's first hypothesis
was that machine load explained the slowness. A controlled experiment falsified
it as a *general* cause — 12 CPU burners at normal priority moved wake-up
latency 2.51 ms → 2.21 ms, i.e. not at all, because macOS boosts recently-blocked
threads. Load turned out to matter **specifically** for this multi-stage
spawn+connect path (36 ms → 145 ms), which is why the two effects are filed as
separate issues rather than one.

### 2026-09-09 — punted: the cost is real, the frequency makes it not worth paying for

Operator decision: *"the single alt+return send latency doesn't justify the
complexity."* Measured first, so the punt rests on numbers rather than a shrug.

**Corrections to this issue as filed.** Recorded so a future reader does not
re-derive them:

1. **It is FOUR zellij calls per send, not two.** `draft_send.lua:4-21` —
   `move-focus up`, `write-chars <body>`, `send-keys Alt Enter`, `move-focus
   down`. The per-call figure (~17ms) was right; the multiplier was wrong, so
   the corrected "~35ms floor" is really ~70ms.

2. **Measured from the recorded action traces, not a synthetic benchmark.**
   `PairZellijTrace` already writes `duration_ms` per action to
   `zellij-actions-*.jsonl`. Across 21,243 recorded actions / **5,015 real
   sends**:

   | | p50 | p90 | p95 | p99 | max |
   |---|---|---|---|---|---|
   | per send (4 calls) | **70ms** | 88ms | 100ms | 264ms | 18.2s |

   `>=200ms: 1.40%`, `>=500ms: 0.50%`, `>=1000ms: 0.38%`.

3. **This is per alt+Return, not per keystroke** (per-keystroke draft work is
   `#202`). At a few dozen deliberate sends a day, a 70ms floor is not worth
   architecture. That frequency — not the measurement — is what closes this.

4. **Spec item 1 is foreclosed.** "Collapse the two calls into one" cannot work:
   the submit must be **Alt+Enter**, because pair-wrap's stdin translator
   rewrites `\r` into the agent's insert-newline sequence
   (`init.lua:725-729`), and `write-chars` writes literal characters and cannot
   express a modified chord. Item 1's own precondition — *"check whether the
   two-step exists for a reason... Settle this first; it may be the whole
   issue"* — is answered YES in the code, and was never checked.

5. **The "about a second" is two different things, neither of them spawn cost.**
   `draft_send.lua:48` → `init.lua:745` sleeps a deliberate **100ms** after the
   write when the body is multi-line or >200 chars — sitting exactly between the
   paste and the Return, which is where the pause was reported. Its reason is
   documented: `write-chars` returns when bytes are queued, not delivered, so an
   immediate submit can land inside the bracketed-paste boundary. A realistic
   multi-line send is therefore ~170ms. The genuine >=1s cases are the 0.38%
   tail, which is the zellij **server stalling**, not spawn cost.

6. **Direction 2 (persistent connection) rejected on merit, not just cost.** It
   removes the ~9.5ms process half of each call (70ms → ~30ms) and touches
   neither the 100ms settle nor the tail — so it cannot fix the only symptom
   with an operator-visible effect, while adding a resident daemon per session
   (the send is issued from nvim, so a held socket needs our own IPC hop), a
   private-protocol coupling against a zellij version `#213` already records we
   want to upgrade, and a fallback spawn path we would keep anyway.

**What survives this punt, so it is not lost with it:**

- **The tail is the only part with a symptom.** 0.38% of sends >=1s, max 18.2s,
  cause unknown and server-side. Nothing tracks it. If it recurs and annoys,
  that is its own issue and the trace evidence above is its starting point.
- **The two `move-focus` calls are removable.** Both `write-chars` and
  `send-keys` accept `--pane-id` in zellij 0.44.3, and
  `draftroute/route.go:103` already uses that form; the focus dance exists only
  because the send targets the focused pane. Worth ~34ms of the 70ms — but as
  tidiness for whoever is next in that file, not as a performance need. Unchecked
  precondition: whether the agent pane needs focus for anything else.
- **The blast-radius list is untouched.** `rename-pane` per title change and
  `clipcmd`'s two-attempt focus (`runtime.go:142,145`) are separate paths;
  clipcmd's is per-selection, i.e. more frequent than per-send.

**Method note.** This issue measured `query-tab-names` in isolation while the
real send path was being traced to disk the whole time. Reach for the recorded
trace before synthesizing a benchmark — the instrument already existed, and it
disagreed with the model by 2x on call count.
