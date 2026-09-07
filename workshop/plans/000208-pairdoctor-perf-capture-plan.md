# `:PairDoctor` Performance Capture Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `:PairDoctor` run from the draft pane at the moment the machine feels slow captures a performance snapshot, pairs it with what the operator typed in the buffer, and hands the agent both.

**Architecture:** The existing seam is kept — a pure payload builder (`nvim/doctor.lua`) and a thin IO wrapper in `init.lua`. Two things are added: a snapshot script (`doctor/perf.sh`) that captures the environment, and nvim-side self-timing that only nvim can provide. The operator's buffer becomes the note, consumed the way a normal send consumes it.

**Tech Stack:** POSIX shell (`doctor/`), Lua (`nvim/`), a small Go probe for the one measurement shell cannot make honestly.

---

## Why this exists, in one paragraph

A full session on 2026-09-06 investigated "typing is slow but CPU is fine",
produced three issues, and reached **no theory** — because every measurement was
taken while the machine was healthy, and synthesizing the condition failed (a
deliberate spawn storm reached 1.9× where the fleet reportedly produces 8×).
`:PairDoctor` is already invoked from the draft pane, which is exactly where the
operator stands when typing feels slow. It just captures nothing. See `#208`'s
Problem for the five hypotheses that session settled and the one it could not.

## The two design decisions that shape everything

**1. Capture at invocation, do not instruct.** Today `doctor.lua` builds a
*pointer* — an instruction for the agent to run `doctor.sh` later. That is right
for drift, which is durable: the flight recorder is still there a minute later.
It is **wrong for performance**, which is transient and usually gone by the time
an agent responds. So the perf half runs first and reports numbers. Same seam,
different payload content.

**2. `ps %cpu` is a lifetime average and must not be used.** Learned expensively
the same day: `contactsd` showed ~0% in `ps` (60 min of CPU over 9 days uptime)
while actually burning **42.6%**. A snapshot built on `ps %cpu` would have
missed the one process that mattered. Every per-process number here comes from a
**delta between two samples**, which is what the operator's phrase "resource
usage in previous various windows" requires anyway.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `payload` (drift) | `nvim/doctor.lua` | unchanged |
| `perf_payload` | `nvim/doctor.lua` | new |
| `note_from_lines` | `nvim/doctor.lua` | new |
| `hoprtt` (pipe probe) | `cmd/hoprtt/main.go` | new |

- **`perf_payload(pair_home, note, nvim_timings, env_report)`** — formats the
  message handed to the agent: the operator's note first, then what nvim
  measured about itself, then the environment, then the procedure pointer.
  - **ARCH-PURE:** string in, string out. No vim API, no IO — so it runs under
    `nvim -l` in `make test-lua`, exactly as `payload` does today. That existing
    split is the local precedent and the reason `doctor.lua` is testable at all.
  - **Order is deliberate:** the note leads. An agent reading this needs the
    operator's *symptom* before the numbers, or it will explain whatever is
    largest rather than what was reported.

- **`note_from_lines(lines)`** — the buffer's text as the operator's note:
  trimmed, blank-only → `nil`.
  - **Why pure and separate:** "what counts as a note" is a decision worth
    testing (blank buffer, whitespace-only, very long) without a running editor.

- **`hoprtt`** — two processes ping-ponging a byte over a pipe; reports
  median/p90/p99. **One scheduler wake-up, isolated.**
  - **Why Go and not shell:** the measurement is microseconds (7 µs baseline).
    Shell cannot time that without spawning a clock process per sample — which
    is the exact bug that made the first attempt read **18.7 ms** for
    `/usr/bin/true` against a known 1.9 ms. It measured `python3` startup.
  - It also serves as the **timing harness** for the spawn probes (`-spawn N --
    cmd`), so there is one in-process timer rather than two implementations
    (ARCH-DRY).

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `perf.sh` | `doctor/perf.sh` | new | `top`, `ps`, `vm_stat`, `sysctl`, `iostat`, `zellij` |
| `:PairDoctor` wiring | `nvim/init.lua` | modified | buffer, `vim.system`, agent send |
| nvim self-timing | `nvim/init.lua` | new | `vim.loop.hrtime`, autocmd chain |

- **`doctor/perf.sh`** — the environment snapshot. Shell because it is
  orchestration of system tools, which is what shell is for, and it stays
  runnable standalone (the agent can re-run it without nvim).

- **nvim self-timing — the discriminator, and the reason this issue is worth
  doing at all.** nvim is the only vantage point that can answer *is the editor
  slow, or is the environment slow?*
  - Measures: time to process a synthetic keystroke through the real autocmd
    chain (which is where `#202`'s completion lives, now measurable **in situ**
    rather than in a headless fixture), and redraw cost.
  - **The call it enables:** nvim fast + typing feels slow ⇒ the problem is at
    or above the terminal (transport, render, compositor) and the scheduling
    family (`#201`/`#203`) is excluded for that symptom. nvim slow ⇒ the cause
    is inside the editor and the environment numbers are noise. **Today's
    session could not make that call.**

## What the snapshot captures

Chosen against the operator's ask — *"system load, processes running, resource
usage by those processes in previous various windows"* — and against the five
hypotheses `#208` records, so each row can settle or exclude something.

| group | what | why this row exists |
|---|---|---|
| conditions | load 1/5/15, CPU idle%, memory-pressure level, process count | load was measured **not** to predict degradation (present at 9.5, absent at 15.5), so it is context, not a verdict |
| swap | swapins/swapouts **as a rate** over the window | cumulative counters always look alarming; only a delta is evidence |
| per-process CPU | top ~20 by CPU from a **two-sample delta** | the `contactsd` lesson — a lifetime average hides an active spinner |
| per-process memory | top ~10 by RSS | |
| the fleet | pair/couch/nvim/zellij family: count and aggregate RSS | so "is it my own fleet" is answerable |
| build storm | count of `go`/`compile`/`link`/test processes | `#203`'s variable; correlated with the workload's *phase* rather than load |
| **render path** | WindowServer CPU, alongside CPU idle% | **the untested candidate**, and the only one whose natural units (tens of ms) match visible lag |
| disk | `iostat` tps + MB/s over the window | distinguishes I/O contention from CPU |
| probes | pipe hop, `fork+exec`, `zellij action` — each median/p90 | the three layers, with known baselines to compare against |

**Known baselines to print alongside**, so a reading is interpretable without
hunting: pipe hop ~7 µs, `fork+exec` ~1.5 ms, `zellij action` ~13 ms.

## ARCH-CONSTRAINTS — operating envelope

- **Interaction path: operator-invoked diagnostic, on a machine already
  struggling.** This is the binding constraint. A tool that takes 30 s while the
  operator is suffering is a bad citizen.
- **Budget: ≤ 6 s wall clock, hard.** Composed of one ~3 s delta window (needed
  for per-process rates — it is the point, not overhead) plus probes. Enforced,
  not hoped: `perf.sh` takes a deadline and each probe's sample count is sized
  to fit. Exceeded → the report says which probes were skipped rather than
  running long.
- **`zellij action` sampling is the expensive probe.** At a *degraded* 145 ms it
  is 1.5 s for 10 samples. Cap it at 5 and say so in the output.
- **The capture must not itself perturb what it measures.** Sample counts stay
  small; nothing forks in a loop.
- **Scale:** ~1000 processes on this host. `top -l 2` and two `ps` passes are
  linear and fine.

## ARCH-ORDER — states, events, and the ones the caller cannot block

The capture holds state across one event: **two samples separated by a window.**
Everything else is a single-shot read.

| event | state | -> effect |
|---|---|---|
| invoke with a non-blank buffer | note present | note leads the report; buffer consumed |
| invoke with a blank buffer | no note | report says "no operator note", still valid |
| a process exits between sample 1 and 2 | pid in S1, absent in S2 | dropped from the delta, **counted** in a "vanished: N" line — a spawn storm makes this number large, which is itself signal |
| a process starts between samples | absent in S1 | no delta computable; listed separately as "started during window" |
| the machine is so slow the budget is exceeded | partial capture | report the probes that completed and name the ones skipped — a truncated honest report beats a hung editor |
| `zellij` is absent or the session is gone | probe fails | record `n/a` with the reason; never abort the whole capture |

The event most likely to be mishandled: **a process exiting mid-window**. The
naive delta joins on pid and silently drops it, which under a spawn storm loses
exactly the processes that characterise the storm. Hence the explicit counts.

Nondeterminism enters through sampling; it is bounded by reporting the window
length alongside the numbers so a reader can re-derive the rates.

## ARCH-SECURE

The report **leaves this machine** — it is sent to an agent, and may be pasted
into an issue. So:

- `ps` output includes **full command lines**, which routinely carry paths,
  branch names, and sometimes tokens in argv. The snapshot prints the **process
  name and pid**, not full argv, except for an allowlisted set (`go`, `compile`,
  `link`, `zellij`, `pair*`, `nvim`) where the argv is the diagnostic value.
- The operator's note is their own text, included verbatim by design.
- No environment dump. `env` is where secrets live and it has no diagnostic
  value here.

## ARCH-MOCK

`perf.sh` shells out to system tools that cannot be faked meaningfully, so the
testable seam is the **parsing**, not the collection: `perf.sh` emits a
line-oriented format, and the pure Lua/Go consumers are tested against recorded
fixtures. `hoprtt` needs no double — it takes a command and times it.

## ARCH-PURPOSE

The purpose is to make the **next** slowdown measurable. So the deliverable
includes the baseline capture (a reading taken while healthy is what makes a
degraded one legible) and the `doctor/SKILL.md` procedure — not just a script
that prints numbers.

---

## M1 — the snapshot, with a validated timing harness

**Files:** create `cmd/hoprtt/main.go`, `cmd/hoprtt/main_test.go`, `doctor/perf.sh`

- [ ] **M1.1: `hoprtt` first, because everything else depends on its honesty.**
      Tests: the `-spawn` timer must read a **known quantity** — `/usr/bin/true`
      in 1–4 ms, never 18. That assertion is the positive control, and it is in
      the suite precisely because the first attempt failed it.

```go
// The bug this pins: a timing harness that shells out to read a clock measures
// the clock process. /usr/bin/true is ~1.9ms (#201); a broken harness reads 18ms.
func TestSpawnTimerMeasuresTheCommandNotTheHarness(t *testing.T) {
	med := spawnMedian(20, "/usr/bin/true")
	if med < 0.5 || med > 6.0 {
		t.Fatalf("fork+exec median %.1fms is outside the plausible band; the harness is measuring itself", med)
	}
}

func TestPipeRoundTripIsMicroseconds(t *testing.T) {
	// a hop is ~7us; anything in milliseconds means we timed process startup
}
```

- [ ] **M1.2:** Implement `hoprtt` (pipe ping-pong + `-spawn N -- cmd`), one
      in-process timer shared by both modes.
- [ ] **M1.3:** `doctor/perf.sh` — the snapshot table above, with a deadline, the
      two-sample delta, the vanished/started counts, and baselines printed
      alongside. Line-oriented output.
- [ ] **M1.4:** Verify against today's known values on a quiet machine: pipe ~7 µs,
      fork ~1.5 ms, zellij ~13 ms. A number outside those bands means the probe
      is wrong, not the machine.
- [ ] **M1.5:** Verify the budget: `time sh doctor/perf.sh` ≤ 6 s.
- [ ] **M1.6:** Commit; `sdlc milestone-close --issue 208 --milestone M1`.

## M2 — nvim: the note, the self-timing, the wiring

**Files:** modify `nvim/doctor.lua`, `nvim/init.lua`; extend `doctor/SKILL.md`

- [ ] **M2.1: Pure tests first** (`nvim -l`, no editor):

```lua
-- The buffer is the operator's note. Blank must not become an empty note that
-- reads as "the operator said nothing was wrong".
assert(doctor.note_from_lines({'', '   ', ''}) == nil)
assert(doctor.note_from_lines({'typing slow', 'top took 10s'})
       == 'typing slow\ntop took 10s')

-- The note LEADS: an agent must read the symptom before the numbers, or it
-- explains whatever is largest instead of what was reported.
local out = doctor.perf_payload('/h', 'typing slow', 'nvim: ...', 'env: ...')
assert(out:find('typing slow') < out:find('env:'))

-- Drift payload unchanged -- this issue must not regress #48.
assert(doctor.payload('/h') == <the existing string>)
```

- [ ] **M2.2: Implement** `note_from_lines` and `perf_payload`.
- [ ] **M2.3: nvim self-timing** — the discriminator. Time a synthetic keystroke
      through the real autocmd chain and a redraw, using `vim.loop.hrtime()`.
      Report both, and state the conclusion in words (`editor: fast` /
      `editor: SLOW`), not just numbers.
- [ ] **M2.4: Wire it.** `:PairDoctor` reads the buffer as the note, runs
      `perf.sh` with the budget, combines note + nvim timings + env, sends via
      `send_generated_prompt`, and consumes the buffer the way a normal send
      does. **Settle in review:** whether a *failed* capture should still consume
      the buffer — losing the operator's note to a failed probe would be the
      worst outcome here.
- [ ] **M2.5:** Drift path untouched: `:PairDoctor` with no perf argument still
      produces byte-identical behaviour to today. Whether perf is a mode or
      always-on is decided here and recorded.
- [ ] **M2.6:** `doctor/SKILL.md` gains the perf procedure — how to read the
      report, and explicitly how to use the discriminator to exclude a whole
      family of causes.
- [ ] **M2.7: Capture a real baseline** on the healthy workbench and record it in
      `## Log`. That reading is what makes the next degraded one legible.
- [ ] **M2.8:** `make test` + `make test-lua`; commit; `sdlc close`.

## Rollback

`perf.sh` and `hoprtt` are additive — nothing reads them until M2 wires them in,
so M1 cannot regress anything. M2's only risk to existing behaviour is the
`:PairDoctor` drift path, pinned byte-identical by M2.1's test. If the capture
misbehaves on a struggling machine, the fix is to make the budget smaller, not
to revert — a truncated report is still better than none.
