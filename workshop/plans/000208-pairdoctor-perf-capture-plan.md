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

## The four design decisions that shape everything

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

**3. `:PairDoctor` always does both — there is no perf mode.** Spec open
question 2, settled here. The operator's use case is *"next time there's a
slowdown, I'll run it"*: one command, at a bad moment, with no flag to remember.
A mode would also split the buffer-note convention in two. The drift half is a
pointer and costs nothing; the perf half is bounded at 6 s. The drift
*instruction text* stays byte-identical (pinned by a test) — what changes is
that the message now carries perf data alongside it, which is additive.

**4. Captures accumulate.** Spec open question 4, settled here: **build the
rolling file.** One snapshot answers "what is happening now"; a series answers
"what changed", and the absence of any prior reading is precisely why the
2026-09-06 investigation had nothing to compare against. A hand-pasted number in
an issue Log does not survive — issues archive to `workshop/history/`. Each
capture appends one line to a rolling JSONL under the session's data dir, so the
*next* investigation starts comparative instead of absolute. It is an append,
not a subsystem.

## Non-goals

- **Continuous or background monitoring.** This fires only when the operator
  invokes it. A sampler running always is a different tool with a different
  budget, and it would perturb the thing it watches.
- **Fixing anything.** This issue captures; `#201`/`#203` remediate. A capture
  that also tuned parallelism would make its own readings uninterpretable.
- **Replacing `doctor.sh`'s drift analysis.** That path is untouched, and
  proving so is a task.
- **Cross-machine comparison or any upload.** The rolling file is local. The
  report goes only where the operator sends it.
- **Diagnosing the render path.** The snapshot *captures* WindowServer CPU
  because that is the untested candidate; interpreting it is downstream work.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `payload` (drift) | `nvim/doctor.lua` | unchanged |
| `perf_payload` | `nvim/doctor.lua` | new |
| `note_from_lines` | `nvim/doctor.lua` | new |
| `pair-hoprtt` (pipe probe) | `cmd/pair-hoprtt/main.go` | new |
| `CaptureRecord` (one rolling row) | `nvim/doctor.lua` | new |
| `delta` (two-sample join) | `nvim/doctor.lua` | new |

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

- **`pair-hoprtt`** — two processes ping-ponging a byte over a pipe; reports
  median/p90/p99. **One scheduler wake-up, isolated.**
  - **Where it lives and how it gets built — verified, not assumed.**
    `make build` is driven by a **hand-maintained** `GO_BINS` list
    (`Makefile.local:32`, `:80`) which overrides the base layer's `cmd/*/main.go`
    scan, so dropping a `cmd/pair-hoprtt/main.go` in would produce **no binary and a
    `perf.sh` with nothing to call**. It goes in `GO_BINS` with its per-binary
    recipe stanza, which is the mechanism the Makefile's own comment
    (`:11-14`) documents.
  - **`perf.sh` locates it as `$PAIR_HOME/bin/pair-hoprtt` and degrades**: probe rows
    print `n/a (hoprtt not built — run make build)` rather than failing the
    capture. A diagnostic that dies because a helper is missing is worse than
    one that reports a gap.
  - The existing probe homes `probes/` (`termsmoke`, `zellijpark`) and
    `cmd/probes/` (`couchstartrecovery`) were considered and rejected: neither is
    on the `make build` path, and `perf.sh` needs a binary that exists after a
    normal build rather than one requiring `go run` per sample.
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
- **The capture must not block the nvim UI. This is the binding constraint, not
  the 6 s budget.** Everything in this repo shells out with `vim.fn.system`,
  which is SYNCHRONOUS — a 6 s capture through it would freeze the editor at
  exactly the moment the operator is already suffering, which is worse than the
  problem being diagnosed. The capture runs via `vim.system(cmd, opts, on_exit)`
  (async; nvim 0.11.7 here), the operator keeps typing throughout, and the send
  happens in the callback. This is the first async shell-out in `nvim/`, so it
  is called out rather than assumed.
- **The capture must not itself perturb what it measures.** Sample counts stay
  small; nothing forks in a loop.
- **Scale:** ~1000 processes on this host. `top -l 2` and two `ps` passes are
  linear and fine.

## ARCH-ORDER — states, events, and the ones the caller cannot block

The capture holds state across **two** spans, and the second only exists because
the capture is async (see ARCH-CONSTRAINTS): the **two-sample window** inside
`perf.sh`, and the **in-flight capture** between `:PairDoctor` returning and its
`on_exit` callback firing. The editor stays live for that whole second span, so
the operator can do anything during it — which is where the interesting events
are.

| event | state | -> effect |
|---|---|---|
| invoke with a non-blank buffer | note present | note leads the report; buffer consumed |
| invoke with a blank buffer | no note | report says "no operator note", still valid |
| a process exits between sample 1 and 2 | pid in S1, absent in S2 | dropped from the delta, **counted** in a "vanished: N" line — a spawn storm makes this number large, which is itself signal |
| a process starts between samples | absent in S1 | no delta computable; listed separately as "started during window" |
| the machine is so slow the budget is exceeded | partial capture | report the probes that completed and name the ones skipped — a truncated honest report beats a hung editor |
| `zellij` is absent or the session is gone | probe fails | record `n/a` with the reason; never abort the whole capture |
| **`:PairDoctor` invoked again while one is in flight** | capture pending | **ignore, and say so.** A second capture would both perturb the first's numbers and race it to consume the buffer. Queueing is wrong for the same reason — the operator wants *this* moment measured, not a later one |
| **the operator keeps typing during the capture** | note read at t0, buffer now differs | **do not consume.** Send the note as read, and leave the buffer alone — see below |
| **the draft buffer is gone at callback time** (closed, or nvim exiting) | no buffer to consume | send anyway if possible, skip the consume, never error into the operator's face |
| **timing fails or throws** | scratch buffer may survive | tear it down in a `pcall`-protected finally; a leaked scratch buffer is a visible bug in the operator's buffer list |

Two events are most likely to be mishandled, one per span.

**In the sample window: a process exiting mid-window.** The naive delta joins on
pid and silently drops it, which under a spawn storm loses exactly the processes
that characterise the storm. Hence the explicit vanished/started counts.

**In the in-flight span: consuming a buffer the operator has since edited.**
This is the only **data-loss** path in the design. The note is read at t0; the
callback fires seconds later; if it clears the buffer unconditionally it deletes
whatever the operator typed in between. The rule: **consume only if the buffer
still matches what was read**, otherwise leave it untouched and note in the sent
message that the buffer was preserved. Combined with the failure rule in M2.4
(consume only on a successful send), the operator's text survives every path —
which matters more than tidiness, because their description of the symptom is
the one thing here that cannot be re-measured.

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

- **`delta(sample_a, sample_b, window_seconds)`** — the two-sample join: per-process
  CPU rate, swap rate, and the vanished/started counts. **Pure Lua in
  `doctor.lua`**, which is where this repo's pure-and-headless-tested code
  already lives (`make test-lua` runs it under `nvim -l`).
  - **Why not in the shell:** the plan's own ARCH-ORDER table calls the pid join
    the thing most likely to be mishandled. Untestable shell is the wrong home
    for the one piece of logic already identified as error-prone, so `perf.sh`
    emits **raw samples** and does no arithmetic.
  - **The three cases it must state and test**, driven from recorded sample
    pairs: a pid in A but not B (**vanished** — counted, never silently dropped,
    because under a spawn storm those are exactly the processes that
    characterise it); a pid in B but not A (**started** — no rate computable,
    listed separately); and a pid present in both whose start-time differs
    (**reused pid** — treated as started, not as a process with an absurd rate).
  - Truncated `top` output is a fourth: the join reports how many rows each
    sample carried, so a comparison across unequal captures is visible.

## ARCH-MOCK

`perf.sh` shells out to system tools that cannot be faked meaningfully, so the
testable seam is the **join**, not the collection: `perf.sh` emits raw
line-oriented samples and `doctor.delta` is tested against recorded fixture
pairs checked into the repo. `pair-hoprtt` needs no double — it takes a command and
times it.

## ARCH-PURPOSE

The purpose is to make the **next** slowdown measurable. So the deliverable
includes the baseline capture (a reading taken while healthy is what makes a
degraded one legible) and the `doctor/SKILL.md` procedure — not just a script
that prints numbers.

---

## M1 — the snapshot, with a validated timing harness

**Files:** create `cmd/pair-hoprtt/main.go`, `cmd/pair-hoprtt/main_test.go`, `doctor/perf.sh`

- [x] **M1.1: `pair-hoprtt` first, because everything else depends on its honesty.**
      Tests: the `-spawn` timer must read a **known quantity** — `/usr/bin/true`
      in 1–4 ms, never 18. That assertion is the positive control, and it is in
      the suite precisely because the first attempt failed it.

```go
// The bug this pins: a timing harness that shells out to read a clock measures
// the clock process. /usr/bin/true is ~1.9ms (#201); a broken harness reads 18ms.
func TestSpawnTimerMeasuresTheCommandNotTheHarness(t *testing.T) {
	// Band is deliberately WIDE. This runs inside `make test`'s parallel
	// `go test ./... -count=1`, where the baseline 1.5ms can reach ~4.5ms under
	// the suite's own spawn load. A 15ms ceiling still catches the 18.7ms bug
	// this control exists for, without going red on a busy machine.
	med := spawnMedian(20, "/usr/bin/true")
	if med < 0.5 || med > 15.0 {
		t.Fatalf("fork+exec median %.1fms is outside the plausible band; the harness is measuring itself", med)
	}
}

func TestPipeRoundTripIsMicroseconds(t *testing.T) {
	// a hop is ~7us; anything in milliseconds means we timed process startup
}
```

- [x] **M1.2:** Implement `pair-hoprtt` (pipe ping-pong + `-spawn N -- cmd`), one
      in-process timer shared by both modes.
- [x] **M1.2b: Add it to `GO_BINS` + its recipe stanza** (`Makefile.local:32,80`)
      and verify `make build` actually produces `bin/pair-hoprtt`. Without this the
      binary does not exist — the hand-maintained list overrides the base layer's
      `cmd/*/main.go` scan.
- [x] **M1.3:** `doctor/perf.sh` — the snapshot table above, with a deadline and
      baselines printed alongside. It emits **raw two-sample output and does no
      arithmetic**: the join is `doctor.delta` (M2.2b), because the pid join is
      the piece ARCH-ORDER flags as most likely to be mishandled and shell is the
      wrong home for it. It degrades per-probe (`n/a` + reason) rather than
      aborting.
- [x] **M1.4:** Verify against today's known values on a quiet machine: pipe ~7 µs,
      fork ~1.5 ms, zellij ~13 ms. A number outside those bands means the probe
      is wrong, not the machine.
- [x] **M1.5:** Verify the budget: `time sh doctor/perf.sh` ≤ 6 s.
- [x] **M1.6:** Commit; `sdlc milestone-close --issue 208 --milestone M1`.

## M2 — nvim: the note, the self-timing, the wiring

**Files:** modify `nvim/doctor.lua`, `nvim/init.lua`; extend `doctor/SKILL.md`

- [x] **M2.1: Pure tests first** (`nvim -l`, no editor):

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

- [x] **M2.2: Implement** `note_from_lines` and `perf_payload`.
- [x] **M2.2b: Implement `doctor.delta`** against checked-in fixture sample
      pairs, covering vanished / started / reused-pid / truncated-sample.
- [x] **M2.3: nvim self-timing** — the discriminator, and the Spec calls it the
      single most valuable bit, so it is specified rather than sketched.

      **`doctor.verdict(insert_ms, redraw_ms)` → `'fast' | 'slow'`**, pure and
      tested. Threshold **16 ms**, with a basis: one frame at 60 Hz is the floor
      of human perceptibility, so an editor that handles a keystroke inside a
      frame cannot be what the operator is feeling. Stated as a named constant,
      not a magic number.

      **It runs on a scratch buffer, never the draft.** The draft buffer is the
      operator's note (read by M2.4) and is also where `#202`'s `TextChangedI`
      chain lives — timing a synthetic keystroke in place would **insert a
      character into the note**. The measurement uses a throwaway buffer with the
      draft's filetype so the same autocmd chain fires, and the note is read
      **before** any timing runs. That ordering is a test, not a comment.

      Timings come from `vim.loop.hrtime()`; the report states the verdict in
      words alongside both numbers.
- [x] **M2.4: Wire it, asynchronously.** `:PairDoctor` reads the note FIRST,
      then runs `perf.sh` via `vim.system(..., on_exit)` — **not**
      `vim.fn.system`, which every other shell-out in `nvim/` uses and which
      would freeze the editor for the whole capture. The operator keeps typing;
      the combine-and-send happens in the callback.

      **The buffer is consumed only on a successful send, and only if it still
      matches what was read** (ARCH-ORDER: the operator keeps typing during the
      async window; clearing unconditionally would delete text written after
      invocation). A failed capture leaves the note exactly where the operator
      typed it — losing their
      description of the symptom to a probe error would be the worst outcome
      this feature could produce, and it is the one behaviour a test pins.
- [x] **M2.5: Drift text pinned byte-identical.** There is no perf argument —
      `:PairDoctor` always does both (design decision 3). The test asserts the
      existing `payload()` string is unchanged and that it still appears verbatim
      inside the combined message, so `#48`'s procedure cannot silently drift.
- [x] **M2.6: Append the capture to the rolling file** (design decision 4): one
      JSONL row per invocation under the session's data dir, via
      `artifactpath` rather than a hand-built path. Include the window length and
      the probe baselines in the row, so a row is interpretable years later
      without this plan. Test the pure `CaptureRecord` shaping; cap the file so
      it cannot grow without bound.
- [x] **M2.7:** `doctor/SKILL.md` gains the perf procedure — how to read the
      report, and explicitly how to use the discriminator to exclude a whole
      family of causes.
- [x] **M2.8: Capture a real baseline** on the healthy workbench and record it in
      `## Log`. That reading is what makes the next degraded one legible.
- [x] **M2.9:** `make test` + `make test-lua`; commit; `sdlc close`.

## Rollback

`perf.sh` and `pair-hoprtt` are additive — nothing reads them until M2 wires them in,
so M1 cannot regress anything. M2's only risk to existing behaviour is the
`:PairDoctor` drift path, pinned byte-identical by M2.1's test. If the capture
misbehaves on a struggling machine, the fix is to make the budget smaller, not
to revert — a truncated report is still better than none.

---

## Revisions

### 2026-09-07 — M2 boundary review (REWORK → addressed)

The M2 close came back REWORK with one Critical and four Importants. The plan
statements those findings contradict are corrected here rather than left to
read as delivered.

1. **M2.6 landed via `pair_data_dir()`, not `artifactpath`.** Two of its
   sub-requirements were **not** delivered and are now explicit rather than
   implied by an unticked box: the row carries **no window length**, and
   `perf-captures.jsonl` is **uncapped**. Both are deferred to `#210` alongside
   the time-bounding gap; neither affects a single capture's correctness, and
   the row's baselines travel with it so a row is still legible alone.
2. **M2.3's discriminator did not exercise the chain it named.** As first built,
   `time_editor` fired a synthetic `TextChangedI`, but `:PairDoctor` runs from a
   `:` command where mode is Normal, so `run_completers` returned at its
   insert-mode gate — the number measured the gate, not `#202`'s chain, while
   `doctor/SKILL.md` instructed the reader to EXCLUDE `#201`/`#203` on it. Now
   fixed rather than rescoped: the gate is split off as `complete_now`, the
   timing calls the chain directly against the scratch buffer, and a leg that
   cannot run renders `n/a (<why>)` and forces `verdict = unknown`. `verdict` is
   variadic over its legs so this holds at any arity.
3. **Core concepts additions.** Entities that shipped without a row:
   `strip_samples`, `headline`, `format_delta`, `parse_samples`,
   `parse_duration`, `verdict`/`FRAME_MS`, and — added at the boundary —
   `probes_from`, `PROBE_KEYS`/`HEADLINE_KEYS`/`BASELINES`, `safe_comm`. The
   `CaptureRecord` row is `capture_record` in the code.
4. **The key set is now single-sourced.** It had been restated in four
   hand-maintained places (perf.sh's `kv` calls, `headline`'s allowlist,
   `capture_record`'s baselines, and a regex in `init.lua`), three of which had
   already drifted. `doctor.PROBE_KEYS`/`HEADLINE_KEYS`/`BASELINES` is the one
   declaration; `probes_from` replaced the regex, which was silently wrong —
   Lua's `%w` excludes `_`, so every row recorded `hop_ms` for `pipe_hop_ms`.
5. **A failed collector now renders under its SUCCESS key** (`swapins_per_s=n/a
   (…)`, not `swap=n/a`). Otherwise every consumer needs a second list of
   failure key names, and one that has only the first drops the row silently —
   an absent line reads as "no such section", which is worse than a wrong value.
6. **`ps`-derived text is filtered at the point of emission**, in `sample()`,
   not at one consumer. The earlier `safe_comm` covered `format_delta` only,
   while the sidecar — the file `doctor/SKILL.md` tells the agent to open —
   carried raw `comm` verbatim, so the `^N` hazard reached the terminal by a
   different path.
7. **The sidecar is named per capture**, not `-latest`. The documented response
   to a truncated send is to re-run `:PairDoctor`, which under a fixed name
   overwrites the file the earlier prompt points at.
8. **The wiring has a test** (`tests/pair-doctor-test.sh`, `_G.PairDoctorTest`
   with an injectable capture runner). Both defects that shipped lived in the
   glue between `doctor.lua` and `init.lua`, which no suite executed — while the
   pure test asserting the intended probe key stayed green throughout. It also
   pins plan M2.4's stated rule, **consume only on a successful send**, which
   the code was not honouring: the note was cleared even when the capture failed.


### 2026-09-07 — the probe is a `pair` subcommand, not a binary

**What the plan says above is wrong from M1 round 3 onward.** Every reference to
`cmd/pair-hoprtt`, `bin/pair-hoprtt`, a `GO_BINS` entry, or a per-binary recipe
describes a design the boundary review reversed. As built:
`cmd/internal/hoprttcmd`, invoked as **`pair hoprtt`**, dispatched from
`cmd/pair-go/main.go` beside `term`/`wrap`/`clip`.

**Why the plan's version was wrong, and why the plan-quality gate missed it.**
The gate correctly forced `GO_BINS` (PQ-1: `make build` is driven by a
hand-maintained list that overrides the base-layer scan, so a bare
`cmd/hoprtt/main.go` would never have been built). That fix was right about the
checkout and wrong about everything else: `make install` is only one of the ways
pair ships. The Homebrew formula builds solely `./cmd/pair-go`, and `PAIR_HOME`
at runtime is the extracted bundle root, which has carried no helper binaries
since `#104` M3. So a standalone probe binary would have been permanently
`n/a` for every installed pair — the operator this whole capture exists for.

`pair` is the only binary guaranteed to exist in every distribution, which makes
a subcommand the sole correct home. The plan should have asked "which
distributions ship this?" rather than "how does `make build` find it".

**The review's method is worth recording too**: it reverted the PATH-first fix
in a scratch copy, found the suite still green, and concluded nothing pinned it.
The replacement test builds the *real* `pair` binary and invokes the subcommand
through it, so the wiring cannot silently regress.

### Also reversed in the same round

- **Budget shedding order.** The plan's ARCH-CONSTRAINTS implied guarding the
  probes. That sheds exactly the wrong thing: the probes are the cheapest and
  most valuable rows, so a degraded machine — the only condition this tool
  targets — would have produced a capture with no probe data. Shedding is now
  reverse-value-order (iostat → top → sample window) with a reserved probe
  slice.
- **`verdict` is asymmetric.** The plan specified a single 16 ms threshold. As
  built, partial evidence can prove `slow` but never `fast`: one timing over
  budget is positive evidence, one timing under it says nothing about the
  measurement that is missing.
- **The delta fixture lives in `doctor/fixtures/`,** not `nvim/`. The runtime
  bundle walks `nvim/` wholesale, so a fixture there would ship into every
  user's extracted session.

**The rule this is the second instance of** (the first was `BR-15` on this
issue's own plan gate): the plan is verified against the tree at the round's
FINAL commit. Both instances were the same mistake — writing the entry from the
mid-round state and leaving it describing code a later commit replaced.
