# Couch Live Idle Shading Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fade live thread labels in Couch's tab bar and switcher by idle age
(under 1 day normal; 1–3 days faded; 3 days or more more faded), including the amber
slot glyphs, correctly on dark and light themes.

**Architecture:** A pure idle-age classifier (`IdleLevel`) and a pure color
policy (`FadeStyle`) blend a label's color toward the terminal's own background,
which Couch learns by asking the terminal once (OSC 10/11) at startup. A
background activity pass modeled on the slot-git pass (`console_slotgit.go`)
reads each live thread's last activity: the newest of its bound agent
transcript's mtime, its Pair log's mtime (sends), and its current launch's
pane-birth evidence mtime (launching is an operator action). That definition lives in one shared function, which the title
poller's heat ramp also moves onto. The pass reduces the
result into `MenuState` and repaints. Both renderers derive from the same
level + style, and nothing persists.

**Tech Stack:** Go; `github.com/charmbracelet/ultraviolet` (already decodes
OSC 10/11 replies as `ForegroundColorEvent`/`BackgroundColorEvent`);
`cmd/internal/sessioninventory` for the transcript.

**Operator decisions (2026-09-28, in the issue):** activity is input plus
agent work; thresholds are 1 day / 3 days with 3 levels (revised 2026-09-28 from 1 h/24 h/48 h and 4 levels); fade the normal
foreground and the amber; selection and notifications keep their emphasis;
both themes plus no-color. Colors blend toward the queried background. The
recent-traffic dot moved to #342.

---

## Decisions this plan makes (flag at review if wrong)

1. **Activity = max(bound transcript mtime, Pair log mtime, pane-birth mtime)**,
   in `cmd/internal/threadactivity`. The title poller's `activityMTime` moves
   onto it (ARCH-DRY: one definition).
   - The transcript is written on the operator's prompts and the agent's work.
   - The Pair log (`log-<tag>.md`) is appended only on sends, so it covers an
     agent whose session is not bound yet (see #329).
   - **The draft mtime is dropped, and that changes the title poller's input.**
     Draft autosave runs an unconditional `silent! write` on every FocusLost,
     BufLeave and InsertLeave (`nvim/init.lua:2328`, `:3680`), so its mtime
     means "the operator last left the draft", not input. A thread switch
     would refresh it. The heat ramp inherits the same fix.
   - The pane-birth evidence (`panebirth.Evidence(scopeDir, tag, agent)`) is
     cleared and rewritten on every real launch, create or resume
     (`createflow.go`), and never on attach ("Attach never clears"). Its mtime
     is "this agent was last launched", which is operator activity. It
     survives a Couch restart, because a restart reattaches without relaunching.
   - None of these three change on redraws, cursor blink or polling. Attach
     `Touch`es the draft only with `O_CREATE` (`osfs.Touch`), which leaves an
     existing file's mtime alone; either way the draft is no longer read.
2. **Launch is the lower bound, not a fallback.** A live thread always has a
   pane-birth file, so a thread with no sends and no bound session ages from
   its launch. A new session is fresh, and an old silent one fades.
   - This keeps the title poller's startup behavior. It skips its frame meter
     and heat prefix while activity reads zero (`titlepoller/run.go:151`), and
     used to get a non-zero time from the draft. The pane birth it already
     awaits supplies that now (plan-quality PQ-1).
   - Rejected alternatives:
     - The thread's `CreatedAt`: slot recovery restamps it with the recovery
       time (`slotrecovery.go:326,437`), which fakes freshness.
     - The incarnation's `StartedAt`: detach retires the incarnation, so a
       Couch restart would restamp every thread.
   - A missing pane-birth file and no other signal gives zero, which is
     unknown and shows as level 0 in Couch, as today. That's rare: the thread
     has no evidence of ever launching. The switcher's non-live `AgeUnknown`
     rule is untouched. A future timestamp (clock skew) → level 0.
3. **Precedence:** the selected chip/row, a chip or row with a pending
   notification (`Bell`), and a placeholder are never faded. Idle fading
   applies to every other live label, and to its amber slot glyphs (`±`, `*`).
   A bell's amber stays full-strength, since it is a notification.
4. **Blend weights toward background:** level 0 → 0 %, 1 → 40 %, 2 → 65 %.
   The top weight keeps the label legible (ARCH-PURPOSE: recede, not
   vanish). Tuned at the live smoke; the weights are one table.
5. **Fallbacks:** palette unknown (no reply) → every faded level of the normal
   label is `SGR 90` (the one theme-aware grey, per #217/#225), and faded amber
   stays `attentionSGR`. `NO_COLOR` set → the fade is suppressed and today's
   bytes are kept (amber glyphs included); this issue doesn't take on no-color
   for the rest of the bar. `COLORTERM`
   not `truecolor`/`24bit` → the blended RGB is quantized to the nearest
   xterm-256 color. With an unknown palette, both faded levels render as the
   same `SGR 90`: the two levels are indistinguishable there, and the docs say
   so (there's only one theme-safe grey).
6. **Scope:** live rows only. The switcher's focus view (`MenuViewFocus`) isn't
   faded; the docs say so. A theme change mid-session isn't picked up (the
   query is sent once), which is noted as a limit. Replies to a child
   process's own color queries also land in the palette, which is harmless:
   same terminal, same answer. The switcher's non-live `AgeBand` ramp (fixed dark
   greys) is untouched. Its light-theme inversion belongs to #217/#225.

## ARCH notes

- **ARCH-PURE:** `IdleLevelFor`, `Palette`, `Blend`, `FadeStyle` and
  `Quantize256` are pure with table tests. IO is confined to the host
  query write, the reply capture, and the activity probe.
- **ARCH-CONSTRAINTS:** one activity pass every 60 s (`defaultActivityInterval`)
  on a worker outside `c.mu`, bounded per thread by `activityProbeTimeout =
  2 s`, with the context passed through to `sessioninventory.QuerySessionContext`. Measured: listing `~/.claude/projects` (3,500 files) takes about 40 ms,
  so 20 threads ≈ 0.8 s per pass. Task 6 re-measures one real pass, and the
  budget is < 2 s for 20 threads. Over budget → share one listing per agent per
  pass (noted, not pre-built). Render cost: one map lookup per chip.
- **ARCH-FUNERAL:** creates nothing durable. `MenuState.Activity` is merged per
  pass over the probed set, like `mergeSlotGit`: a failed probe keeps the last
  value, and a thread no longer probed drops out. The palette lives in
  `MenuState` for the Console's lifetime.
- **ARCH-ORDER:** the palette query is written after `MakeRaw` and before the
  presenter's first write, so it cannot interleave with a frame. Replies arrive
  later through the one stdin decoder. An activity result lands only for its
  own schedule generation, as with slot git.
- **ARCH-SECURE:** OSC replies are parsed by ultraviolet's `XParseColor`. A
  malformed reply yields a nil color, which is treated as unknown.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `IdleLevel`, `IdleLevelFor` | `cmd/internal/couchtty/idle_shade.go` | new |
| `Palette` | `cmd/internal/couchtty/idle_shade.go` | new |
| `FadeStyle` (label + amber variants) | `cmd/internal/couchtty/idle_shade.go` | new |
| `blend`, `quantize256` | `cmd/internal/couchtty/idle_shade.go` | new |
| `StatusActor.Idle` | `cmd/internal/couchtty/reserve.go` | modified |
| `StatusModel.Palette` | `cmd/internal/couchtty/reserve.go` | modified |
| `MenuState.Activity`, `MenuState.Palette`, `MenuEventActivity`, `MenuEventPalette` | `cmd/internal/couchtty/menu.go` | modified |
| `threadactivity.Latest` | `cmd/internal/threadactivity/activity.go` | new |

- **IdleLevel / IdleLevelFor(now, last time.Time, known bool) IdleLevel** —
  `IdleFresh` (0), `IdleDay` (1: ≥1 day), `IdleStale` (2: ≥3 days). Unknown or
  future → `IdleFresh`. Boundaries are inclusive at exactly 24 h and 72 h.
  - **DRY rationale:** the one classifier both renderers call.
  - **Future extensions:** thresholds are a table if the operator retunes them.
- **Palette{FG, BG color.RGBA; Known, TrueColor, NoColor bool}** — what Couch
  learned about the host terminal, plus the env-derived modes.
- **FadeStyle(p Palette, level IdleLevel, base styleBase) string** — the SGR
  for a label (`baseDefault`: the terminal fg) or an amber glyph (`baseAmber`:
  xterm 220 = `#ffd700`) at a level. Level 0 → `""` for default and
  `attentionSGR` for amber, byte-identical to today. The fallbacks follow
  Decision 5.
- **threadactivity.Latest(ctx, rt Runtime, t Thread) time.Time**, with
  `Thread{ScopeDir, Scope, Tag, Agent string}` — the newest of the bound
  transcript's `LastActivityAt`, the Pair log's mtime, and the pane-birth
  evidence mtime. Zero means no signal at all.
  - `ScopeDir` is the per-repo data directory
    (`artifactpath.ResolveScopeDir(dataDir, scope)` in Couch, `opts.DataDir`
    in the title poller), not the global data root.
  - `Runtime` is `{ModTime(path) (time.Time, bool); SessionActivity(ctx, scopeDir, scope, tag, agent) (time.Time, bool)}`.
  - Pure given its Runtime; tests use a map-backed fake.
  - **This is the one signature.** The Couch probe returns
    `(threadactivity.Latest(...), nil)`; zero reaches the renderer as "no
    entry", which is level 0.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| palette query write | `cmd/internal/couchtty/console.go` (`Run`) | modified | host tty (`hostty.Host.WriteContext`) |
| palette reply capture | `cmd/internal/couchtty/terminal_input.go` (`routeInputEvent`) | modified | stdin decoder (`uv.*ColorEvent`) |
| `ActivityProbe` + activity pass | `cmd/internal/couchtty/console_activity.go` | new | filesystem + session inventory |
| production probe wiring | `cmd/internal/couchcmd/run.go` | modified | `threadactivity` OS runtime |
| `threadactivity.OSRuntime` | `cmd/internal/threadactivity/os.go` | new | `os.Stat`, `sessioninventory` |
| title poller `activityMTime` | `cmd/internal/titlepoller/run.go` | modified | (now calls `threadactivity.Latest`) |

- **ActivityProbe** `func(ctx, couchcore.ActionableThreadSummary) (time.Time, error)`
  — injected with `Console.SetActivityProbe`, exactly like `SetSlotGitProbe`.
  The test fake is a map keyed by address, with a call log, so a test can
  assert which threads were probed (live only).
- **Clock:** `Console.now func() time.Time` (default `time.Now`) so tests cross
  thresholds with an injected clock. `showMenu` passes `c.now()` to
  `RenderMenuView` instead of `time.Now()` (`console_menu.go:211`), so the
  switcher is reachable from clock-driven tests.

---

## Chunk 1 (M1): pure policy + renderers

### Task 1: `IdleLevelFor`

**Files:** Create `cmd/internal/couchtty/idle_shade.go`, `cmd/internal/couchtty/idle_shade_test.go`

- [x] **Step 1: failing test**

```go
func TestIdleLevelForBoundaries(t *testing.T) {
	now := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name  string
		last  time.Time
		known bool
		want  IdleLevel
	}{
		{"unknown", time.Time{}, false, IdleFresh},
		{"future", now.Add(time.Minute), true, IdleFresh},
		{"just now", now, true, IdleFresh},
		{"23h59m59s", now.Add(-24*time.Hour + time.Second), true, IdleFresh},
		{"24h", now.Add(-24 * time.Hour), true, IdleDay},
		{"71h59m", now.Add(-72*time.Hour + time.Minute), true, IdleDay},
		{"72h", now.Add(-72 * time.Hour), true, IdleStale},
		{"30d", now.Add(-30 * 24 * time.Hour), true, IdleStale},
	} {
		if got := IdleLevelFor(now, tc.last, tc.known); got != tc.want {
			t.Errorf("%s: level = %d, want %d", tc.name, got, tc.want)
		}
	}
}
```

- [x] **Step 2:** `go test ./cmd/internal/couchtty -run TestIdleLevelFor` → FAIL (undefined).
- [x] **Step 3: implement**

```go
type IdleLevel uint8

const (
	IdleFresh IdleLevel = iota // < 1 day, unknown, or future
	IdleDay                    // ≥ 1 day
	IdleStale                  // ≥ 3 days
)

// idleThresholds are the operator's 2026-09-28 bands (#247).
var idleThresholds = [...]time.Duration{24 * time.Hour, 72 * time.Hour}

func IdleLevelFor(now, last time.Time, known bool) IdleLevel {
	if !known || last.IsZero() || last.After(now) {
		return IdleFresh
	}
	age, level := now.Sub(last), IdleFresh
	for i, threshold := range idleThresholds {
		if age >= threshold {
			level = IdleLevel(i + 1)
		}
	}
	return level
}
```

- [x] **Step 4:** test passes. **Step 5:** commit `#247 M1: idle level classifier`.

### Task 2: `Palette`, `blend`, `quantize256`, `FadeStyle`

**Files:** same two files.

- [x] **Step 1: failing tests**, table-driven:
  - `blend(fg, bg, 0) == fg`; `blend(white, black, 0.40)` → `#999999`
    (255·0.60 = 153); `blend(amber #ffd700, white bg, 0.65)`
    moves every channel toward 255.
  - `quantize256` maps `#ff0000` → 196, `#808080` → 244, and is exact for the
    6×6×6 cube corners.
  - `FadeStyle`:
    - level 0 → `""` (default) / `attentionSGR` (amber), in every palette.
    - Known + TrueColor, dark (fg `#ffffff`, bg `#000000`): level 1 default →
      `"\x1b[38;2;153;153;153m"`.
    - Known + TrueColor, light (fg `#000000`, bg `#ffffff`): level 2 default
      is lighter than level 1 (channel sum strictly increases with level).
      This is the "fades toward the background on light themes" invariant,
      stated independently of the weights.
    - Dark: channel sum strictly decreases with level, for default and amber.
    - Known, not TrueColor → `"\x1b[38;5;Nm"` with N = quantize of the blend.
    - Unknown palette → every level ≥1 default is `"\x1b[90m"`; amber is `attentionSGR`.
    - NoColor → today's level-0 bytes for every level: `""` for default and
      `attentionSGR` for amber (the fade is suppressed, nothing else changes).
- [x] **Step 2:** run → FAIL.
- [x] **Step 3: implement** (sketch; exact names as in the table):

```go
type Palette struct {
	FG, BG               color.RGBA
	Known                bool // both OSC 10 and OSC 11 answered
	TrueColor, NoColor   bool
}

type styleBase uint8

const (
	baseDefault styleBase = iota
	baseAmber
)

var amberRGB = color.RGBA{0xff, 0xd7, 0x00, 0xff} // xterm 220 == attentionSGR
var idleBlend = [...]float64{0, 0.40, 0.65}

func FadeStyle(p Palette, level IdleLevel, base styleBase) string {
	if level == IdleFresh || p.NoColor {
		if base == baseAmber {
			return attentionSGR
		}
		return ""
	}
	if !p.Known {
		if base == baseAmber {
			return attentionSGR
		}
		return "\x1b[90m"
	}
	from := p.FG
	if base == baseAmber {
		from = amberRGB
	}
	c := blend(from, p.BG, idleBlend[level])
	if p.TrueColor {
		return fmt.Sprintf("\x1b[38;2;%d;%d;%dm", c.R, c.G, c.B)
	}
	return fmt.Sprintf("\x1b[38;5;%dm", quantize256(c))
}
```

  `blend` mixes each channel `round(a*(1-t) + b*t)`. `quantize256` picks the
  nearest color (squared RGB distance) among the 6×6×6 cube (16–231) and the
  grey ramp (232–255).
- [x] **Step 4:** pass. **Step 5:** commit `#247 M1: fade style toward the terminal background`.

### Task 3: tab bar renders the fade

**Test strategy:** risky function `RenderStatusRow`; guard = golden bytes for a level-1/level-2 chip plus byte-identity of the level-0 fixtures and unchanged `ChipSpan`s; mutation = delete the fade `case`.

**Files:** Modify `cmd/internal/couchtty/reserve.go` (`StatusActor`, `StatusModel`, `RenderStatusRow` style switch). Test in `cmd/internal/couchtty/reserve_test.go`.

- [x] **Step 1: failing tests**
  - An idle (level 1) non-active chip in a known dark truecolor palette starts
    with `FadeStyle(p, IdleDay, baseDefault)`. Its `*` glyph uses
    `FadeStyle(p, IdleDay, baseAmber)`. A level-2 chip uses the `IdleStale` pair.
  - The same chip marked `Active` renders exactly as today (no fade bytes).
    Same for `Bell` (label stays `attentionSGR`) and for `Placeholder`.
  - `IdleFresh` chips are byte-identical to today's output (regression pin:
    render the existing `reserve_test` fixtures with the zero `Idle`).
  - Chip spans (`ChipSpan.Start/End`) are unchanged by the fade (the clipping
    and click-mapping contract).
- [x] **Step 2:** FAIL.
- [x] **Step 3: implement.** Add `Idle IdleLevel` to `StatusActor` and
  `Palette Palette` to `StatusModel`. In the style switch:

```go
switch {
case a.Placeholder:
	style = placeholderSGR
case a.Bell && !a.Active:
	style = attentionSGR
case !a.Active:
	style = FadeStyle(m.Palette, a.Idle, baseDefault)
}
```

  and for glyphs, when `slotGlyphSGR(r) != ""` and the chip is neither
  active, bell nor placeholder, use `FadeStyle(m.Palette, a.Idle, baseAmber)`.
  An active chip's glyph keeps `slotGlyphSGR`.
- [x] **Step 4:** pass, plus the whole `couchtty` package. **Step 5:** commit.

### Task 4: switcher renders the fade for live rows

**Test strategy:** risky function `renderRootMenuFrame`; guard = a faded live row, and a selected/attention/non-live row byte-identical to today; mutation = drop the `thread.Live()` fade branch.

**Files:** Modify `cmd/internal/couchtty/menu_render.go` (`renderRootMenuFrame` live branch, `colorMenuGlyph` gets a glyph-style parameter); `RenderMenuView` gains the palette and activity through `MenuState` (Task 5 adds `Activity`; this task takes a `Palette` field on `MenuState` too). Test in `menu_render_test.go`.

- [x] **Step 1: failing tests**, with `RenderMenuView(state, cols, h, now, true)`:
  - A live, unselected row whose `state.Activity[addr]` is 30 h old (level 1) renders
    wrapped in `FadeStyle(p, IdleDay, baseDefault)`, and its `±` glyph in the
    amber variant.
  - The same row selected → reverse video exactly as today, with no fade bytes.
  - A live row with attention messages → not faded.
  - Non-live rows → the `ageColor(AgeBandFor(...))` bytes, unchanged.
  - No activity entry → byte-identical to today.
- [x] **Step 2:** FAIL. **Step 3:** implement. In the `color256 && frame.View != MenuViewFocus`
  branch, for `thread.Live()` with no attention:
  `outer = FadeStyle(state.Palette, IdleLevelFor(now, at, ok), baseDefault)`,
  where `at, ok := state.Activity[thread.Address]`. (An absent entry, before the
  first pass lands, is level 0.) Pass the matching amber
  style into `colorMenuGlyph`.
- [x] **Step 4:** pass. **Step 5:** commit.
- [x] **M1 boundary:** `sdlc milestone-close --issue 247 --milestone M1`.

## Chunk 2 (M2): IO seams, wiring, docs

### Task 5: shared `threadactivity`, title poller migrated

**Files:** Create `cmd/internal/threadactivity/activity.go`, `activity_test.go`, `os.go`. Modify `cmd/internal/titlepoller/run.go` (`activityMTime`) and `runtime.go` (adapter).

- [x] **Step 1: failing tests** (map-backed fake Runtime):
  - The newest of transcript, log and pane birth wins, each alone included.
  - With no transcript and no log, the answer is the pane-birth mtime (the
    PQ-1 regression: a fresh session is never zero).
  - Nothing at all → zero.
  - A draft mtime newer than everything else is **ignored** (pins Decision 1).
  - A canceled `ctx` returns promptly with what the file mtimes give.
- [x] **Step 2:** FAIL. **Step 3:** implement `Latest` and point
  `titlepoller.activityMTime` at it (the title poller passes `opts.DataDir`,
  its scope key, tag and agent). Add a title-poller test: a session with
  pane birth but no transcript or log still updates its frame titles on the
  first tick. `OSRuntime` in
  `os.go` wraps `os.Stat` plus
  `sessioninventory.NewOSRuntime(home, scopeDir)` +
  `QuerySessionContext` + `ActivityForSession`, following
  `couchcore/switchcontext.go:68-75`. `titlepoller/runtime.go:SessionActivity`
  delegates to it.
- [x] Record in the issue Log that the heat ramp no longer reads the draft.
- [x] **Step 4:** `go test ./cmd/internal/threadactivity ./cmd/internal/titlepoller` pass. **Step 5:** commit.

### Task 6: activity pass in the console

**Test strategy:** risky functions `advanceActivity`/`finishActivity`; guard = a fake probe with a call log and an injected clock, asserting levels on both renderers; mutation = probe non-live rows / ignore the generation.

**Files:** Create `cmd/internal/couchtty/console_activity.go`, `console_activity_test.go`. Modify `console.go` (fields, `Run` select loop: ticker plus the results channel, `now`), `menu.go` (`MenuState.Activity`, `MenuEventActivity` reduce, copy in the state-clone helper next to `SlotGit`), `console_presentation.go` (`statusModelLocked` sets `actor.Idle` and `model.Palette`).

- [x] **Step 1: failing tests** (fake probe + injected clock, following
  `console_slotgit_test.go`):
  - A pass probes live rows only, and each once.
  - The result lands in `MenuState.Activity` and triggers a repaint. The tab
    bar chip for a thread active 2 h ago renders level 0, and one active
    30 h ago renders level 1.
  - Advancing the clock past 72 h plus the next tick moves the 30 h chip to level 2
    with no new activity (repaint driven by the pass).
  - A thread that stops being live drops out of `Activity` on the next pass
    (ARCH-FUNERAL).
  - A result from a stale generation is ignored.
  - A probe error or timeout for one thread keeps its previous value (a stale
    thread must not flash bright) and doesn't fail the pass.
  - The switcher, rendered through `showMenu` with the injected clock, shows
    the same level as the tab bar for the same thread.
  - **Restart:** a second Console built over the same fake probe (whose times
    are 4 days old) renders level 2 after its first pass. Nothing is carried
    in memory; the answer comes from the probe.
  - A pass is requested when the inventory lands and on a thread switch, not
    only on the ticker, so fading appears without a 60 s wait.
- [x] **Step 2:** FAIL. **Step 3:** implement by mirroring `advanceSlotGit` /
  `finishSlotGit` (`RefreshSchedule`, worker outside `c.mu`, per-thread timeout,
  `select` on `c.stop`, `showMenu()` when the panel has focus, else
  `repaint()`). Use `defaultActivityInterval = 60 * time.Second`. Call
  `requestActivity()` beside each existing `requestSlotGit()` (inventory
  landing `console_menu.go:153`, switch `console.go:544`).
- [x] **Step 4:** pass. Measure one production pass against the operator's real
  thread list: log the duration in the issue, budget < 2 s for 20 threads.
- [x] **Step 5:** commit.

### Task 7: palette query + reply capture

**Test strategy:** risky functions `Run` (query order) and `routeInputEvent` (capture); guard = the fake host's write log plus fed OSC replies; mutation = skip the capture before the `Reply` drop.

**Files:** Modify `console.go` (`Run`: after `MakeRaw`, before `applyLayout`, write `"\x1b]10;?\x1b\\\x1b]11;?\x1b\\"` with `c.host.WriteContext`; seed `Palette.TrueColor`/`NoColor` from env at construction), `terminal_input.go` (`routeInputEvent`: before the `Reply` early return, reduce `uv.ForegroundColorEvent`/`uv.BackgroundColorEvent` into `MenuState.Palette` via `MenuEventPalette` under `c.mu`. A nil `Color` (malformed reply) is ignored. `Known` once both have arrived. Then `showMenu()` if the panel has focus, else `repaint()`. `statusModelLocked` reads the palette from `c.menu`; it's stored in one place). Test in `console_test.go`-style harness with the fake host.

- [x] **Step 1: failing tests**
  - `Run` writes the query exactly once, before the first frame (assert the
    order in the fake host's write log).
  - Feeding `"\x1b]11;rgb:ffff/ffff/ffff\x1b\\"` and
    `"\x1b]10;rgb:0000/0000/0000\x1b\\"` to stdin makes the palette known,
    light, and triggers a repaint. A later idle chip uses the light blend.
  - Only one reply → palette stays unknown (the `SGR 90` fallback).
  - A malformed reply → unknown, no panic, not forwarded to the child.
  - Color replies never reach the child (existing `Reply` drop preserved).
- [x] **Step 2:** FAIL. **Step 3:** implement. **Step 4:** pass. **Step 5:** commit.

### Task 8: production wiring

**Files:** Modify `cmd/internal/couchcmd/run.go` next to `SetSlotGitProbe`:

```go
console.SetActivityProbe(func(ctx context.Context, row couchcore.ActionableThreadSummary) (time.Time, error) {
	scopeDir, err := artifactpath.ResolveScopeDir(dataDir, row.Address.RepoScope)
	if err != nil {
		return time.Time{}, err
	}
	return threadactivity.Latest(ctx, threadactivity.NewOSRuntime(home), threadactivity.Thread{
		ScopeDir: scopeDir, Scope: row.Address.RepoScope, Tag: row.Address.Tag, Agent: row.Agent,
	}), nil
})
```

where `dataDir := launcher.ResolveDataDir(...)` is the one `couchcmd/run.go:103`
already computes. (Check `ResolveScopeDir`'s exact signature at
implementation; switchcontext.go reaches the same dir through
`artifactpath.Resolve(...).ScopeDir()`, and either is fine.)

- [x] No existing test pins `SetSlotGitProbe`'s wiring. This wiring is
  covered by the live smoke. Commit.

### Task 9: docs + verification

- [x] README Couch section and the Alt+h help: one line saying live threads
  fade after 1 day and again after 3 days without activity (your input or the agent's
  work), and that the focused/selected thread and notifications never fade.
- [x] `atlas/`: add the activity pass and palette query to the Couch
  presentation map. Link from `atlas/index.md` if a new file is added.
- [x] Full suite (`go test ./... -count=1`, then `make test`, with the
  session env scrubbed; see the #329 log for the command).
- [x] Mutation checks (cp/cmp revert): drop the fade case in
  `RenderStatusRow` → Task 3 tests fail; skip the reply capture → Task 7 fails;
  probe non-live rows → Task 6 fails.
- [x] Live smoke by the operator on a dark AND a light theme: an idle thread
  recedes in both bars; the selected thread and a notified thread don't fade;
  `NO_COLOR=1 couch` shows no fade bytes. On the light theme, check that the
  65 % amber (≈ `#fff1a6`) is still visible; if not, cap the amber weight
  (one table entry).
- [x] **M2 boundary / close:** `sdlc close --issue 247 --verified '…'`.

## Revisions

### 2026-09-28 — fresh-eyes plan review (approve-with-fixes), folded in

- Critical: the probe used Couch's global data root where the per-repo scope
  dir is needed, and named a nonexistent `environment.PairLifecycleDataDir`.
  Now `artifactpath.ResolveScopeDir` + `launcher.ResolveDataDir`, following
  `couchcore/switchcontext.go`.
- `ctx` reaches `QuerySessionContext` (the timeout now bounds the probe).
  `showMenu` uses the injected clock. `NO_COLOR` keeps today's bytes instead of
  also stripping amber. A failed probe keeps the last value. Passes are
  requested on inventory landing and on switch. Added a restart test. Repaint
  follows `finishSlotGit`. The palette is stored once, in `MenuState`. Dropped
  the claim about a nonexistent wiring test.
- Unknown activity: replaced "unknown → level 0" with a `CreatedAt` floor.
  Checking the reviewer's point turned up that draft mtime is focus noise
  (unconditional autosave `write`), so the draft is dropped from the activity
  definition in favor of the Pair log. The title poller follows (Decision 1).

### 2026-09-28 — operator shrinks the ramp to 3 levels at 1 day / 3 days

- Levels: under 1 day normal (`IdleFresh`), 1 day to under 3 days
  (`IdleDay`), 3 days or more (`IdleStale`). `IdleHour` is gone.
- Blend weights: 0 / 40 / 65 % (was 0/35/55/70). Test boundaries, worked
  examples (`#999999`, amber ≈ `#fff1a6` on white) and docs updated to match.

### 2026-09-28 — plan-quality gate (change-code) findings

- PQ-1 (Important): moving the title poller off the draft with a zero floor
  would have disabled its frame meter and heat prefix on new sessions. Fixed
  at the class: activity now includes the current launch's pane-birth mtime,
  which both consumers have. A title-poller regression test is added.
- PQ-2 (Minor): the signatures were inconsistent. Now one
  `Latest(ctx, rt, Thread) time.Time`, and the probe returns `(time.Time, error)`.
- PQ-3 (Minor): the `CreatedAt` floor could be restamped by slot recovery.
  Dropped: pane birth replaces it, and `ActionableThreadSummary` is unchanged.
- PQ-4 (Minor): each risky task now has one test-strategy line (function,
  guard, mutation).
- PQ-5 (Minor): the unknown-palette fallback collapses both faded levels to
  SGR 90, which is now documented.

### 2026-09-28 — M1 delivered, reconciled against the plan

- Tasks 1–4 as planned. Delta from the table: `MenuState.Activity` and
  `MenuState.Palette` landed in M1 (Task 4 renders from them). Their
  reducer events (`MenuEventActivity`, `MenuEventPalette`) and the clone of
  `Activity` in the state-copy helper stay in M2 (Tasks 6–7), where the first
  writer arrives. Until then nothing writes either field in production, so
  M1 changes nothing on screen by construction.
- `colorMenuGlyph` takes the row's amber colour as a parameter (one call site).

### 2026-09-28 — M2 delivered, reconciled against the plan

- Task 5: `threadactivity.Latest(ctx, rt, Thread) time.Time` as planned, plus
  `InScope(dataDir, scope, tag, agent)`. `couchcmd` is a non-artifact source
  that may not import `artifactpath`, so scope resolution moved next to the
  reads it serves. The title poller adapts its own runtime (`pollerActivity`).
  Its pane-birth ordering test now compares zellij calls only (the activity read
  stats the pane too) and no longer seeds a draft. The launch alone drives the
  first render, which is the PQ-1 regression.
- Task 6: `console_activity.go` as planned. `mergeSlotGit` became the generic
  `mergeObservations`, shared by both passes. The stale-generation check is the
  shared `RefreshSchedule`'s, and it is exercised through the fixture rather
  than by an injected late result. A new `activity` trace event is documented
  in atlas (`trace_test` requires it).
- Task 7: `console_palette.go` (`paletteQuery`, `capturePalette`,
  `ensureMenuLocked`, `SetColorModes`, `repaintAfterPalette`). Delta: the two
  lazy `c.menu` builds now go through `ensureMenuLocked`, which keeps a palette
  that arrived first. The soak test's vt host now drains its emulator's replies
  (it hung on the new query).
- Task 8: `wireIdleFading` in `consoleRunner`, the composition root that
  already reads the environment.
- Task 9: README only. The Alt+h help lists keys, and idle fading has none.
  Measured cost: 114 ms mean per thread (56 real threads in 6.4 s, slowest
  577 ms), so ~2.3 s for 20 threads, about 15 % over the < 2 s budget. It runs
  on the background worker once a minute. The planned follow-up (one shared
  store listing per agent per pass) is not built; raised with the operator.
- M1 review Minors folded in: BR-2 (`slotGlyphBase`), BR-3 (atlas fields),
  BR-4 (cross-referenced precedence comments).

### 2026-09-28 — operator smoke (M2 review BR-5)

- Built `7363b5e1` in `pair:0` (`bin/pair`, `bin/couch` at that revision, not
  modified). In live Couch the operator saw distinct shades by idle age and
  passed the smoke; colour tuning is deferred. Not exercised live: the
  light-theme amber legibility and `NO_COLOR`. Both moved to #343 with the
  probe-cost decision (BR finding on the 2.3 s/20-thread pass), which the
  operator left open.
- A spot check during the smoke: `brain:0` (`couch-5f43ed128f6135e9`, codex)
  read 28.5 h idle from its Pair log. It has no established binding, so its
  transcript is not a signal (recorded in #343).
