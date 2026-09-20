---
id: 000295
status: open
deps: [pair#292]
github_issue:
created: 2026-09-19
updated: 2026-09-19
estimate_hours:
---

# Engine maintenance strategy for the browser tab: own the artifact, the core, or the fork

## Problem

pair#292 puts a Carbonyl browser tab in the right pane. Carbonyl is
**unmaintained**: last upstream commit 2023-02-26, last release v0.0.3
(2023-02-18), bundling Chromium 111. The operator's position, 2026-09-19: *if we
rely on it, we own it.* That is right — the question this issue settles is
**which layer we own**, with the cost of each measured rather than assumed.

Nothing here blocks #292. It ships behind a swappable seam ("launch a binary,
speak CDP") precisely so this decision can be made later, on evidence from use.

### Measured 2026-09-19

**The thing we'd be taking over is small.** At tag v0.0.3:

| Piece | Size |
|---|---|
| Chromium patch series | 14 patches, 54 files, **+1,444 / −267** |
| Skia patches | 2 patches, 6 files, +11 / −7 |
| WebRTC patch | 1 patch, 1 file, +1 / −1 |
| Carbonyl's own source | **3,968 lines** (3,221 Rust, 314 `.cc`, 285 `.h`, 126 `.gn`, 22 `.mojom`) |
| Total surface | ≈ 5,400 lines |

And the patch series is mostly **additive glue**, not rewriting of Chromium
logic:

| Patch | Lines | Nature |
|---|---|---|
| 0009 Bridge browser into Carbonyl library | +462 / −28 | additive |
| 0002 Add Carbonyl service (Mojo text service) | +409 / −3 | additive |
| 0013 Refactor rendering bridge | +351 / −84 | reworks its own earlier additions |
| 0006 DPI, 0007 text effects, 0005 assertions, 0010 text rendering, Skia | ≈ ±120 | the genuinely invasive edits |

**Only ~270 lines of pre-existing Chromium code are modified at all.**

**What the patches buy** (why this can't be a library):
1. **Text is intercepted before rasterization.** Blink's
   `platform/fonts/font.cc` `DrawBlobs` returns early, Skia's
   `SkBitmapDevice::onDrawGlyphRunList` is gated on `Bridge::BitmapMode()`, and
   a new `carbonyl::mojom::CarbonylRenderService` ships `TextData` (string,
   position, color, size) from renderer to browser. Graphics are drawn as
   pixels; text is re-emitted as terminal text. This is why Carbonyl is
   readable where screenshot-downscaling never can be.
2. **A terminal-sized software frame.** viz's `SoftwareOutputDeviceProxy` and
   the `LayeredWindowUpdater` Mojo interface (normally the Windows
   layered-window path) are repurposed to land each frame in shared memory.
   Headless Chrome's ordinary output is the screenshot API, far too slow.
3. **DPI redefined** so layout happens at the terminal grid
   (`ui/display/display.cc` → `Bridge::GetCurrent()->GetDPI()`).

Chromium exposes no embedder API for any of the three, which is why it's a fork.

**What a rebase would cost.** Checked against current Chromium:
- `headless/app/headless_shell.cc` still exists, so the entry hook survives.
- `DrawBlobs` is **gone** from `font.cc` (now 369 lines), and
  `ng_text_painter_base.cc` is gone — the `ng_` paint generation was renamed to
  `text_painter.cc` when LayoutNG became the default.

So the two load-bearing interception points must be **re-derived** against a
refactored paint pipeline, not re-applied. The volume is small; the expertise
and the iteration loop are not (a Chromium build is ~100 GB and over an hour;
upstream publishes four platform binaries, though pair needs only macos-arm64).

**Not a factor:** sandboxing. Measured 2026-09-19 — Carbonyl 0.0.3's renderer,
GPU and network processes all run **with Chromium's sandbox** (no
`--no-sandbox`; the network utility names its sandbox type), so a hostile page
still has to escape the sandbox.

## Spec

Decide, and record, which ownership tier pair commits to. They are cumulative,
cheapest first.

**Tier 1 — own the artifact (do this regardless).** The binary must not be able
to disappear, and #292 must not depend on npm's `latest` tag (which still points
at 0.0.2, the build that spins a CPU core when idle).
- Mirror the v0.0.3 release for macos-arm64, with checksums.
- A documented install path pair supports, wired through `PAIR_CARBONYL`.
- `pair doctor` (or the browser tab's own notice) names the supported version.

**Tier 2 — own the Rust core (cheap, on demand).** `libcarbonyl` builds in
seconds with cargo and drops into a release tree; 3,221 lines. It is where the
idle-CPU bug lived and where the navigation bar lives.
- Concrete target already known: the bar has no select-all or clear, which is
  why #292 builds its own URL field. ~20 lines of Rust would fix it upstream.
- **Cost that must not be waved away:** shipping a patched `libcarbonyl.dylib`
  means pair distributes a Rust build artifact, and pair is deliberately a
  single Go binary (see the Homebrew release runbook). Keep any fork a separate
  tap the operator installs, wired via `PAIR_CARBONYL` — do not pull Rust into
  pair's build.

**Tier 3 — own the Chromium fork (a project, not a task).** Re-derive ~270
invasive lines against current Blink/viz, keep ~1,400 lines of additive glue
applying, and stand up a build — macos-arm64 only, so one artifact on the
operator's machine, not Carbonyl's four.
- **Only worth it against a stated trigger** (below). Absent one, it buys a
  stale engine *and* a standing obligation.

**Alternative to tier 3 — own the renderer instead of the engine.** Browsh's
architecture on Chrome: stock headless Chrome over CDP, a script injected with
`Page.addScriptToEvaluateOnNewDocument` reporting text runs and boxes,
`Page.startScreencast` for graphics, and our own Go painter producing cells.
- No patches, no Chromium builds; Homebrew keeps the engine patched; CDP
  sharing (#292's point) is intact by construction; the code is Go we can test.
- Costs fidelity against engine-level interception: canvas/WebGL text,
  transforms, shadow DOM and generated content are weaker — the same class of
  gap Browsh has.
- **Must reproduce the cell-to-CSS-pixel contract.** Carbonyl maps one column
  to ~5.29 CSS px at zoom 100 (measured 2026-09-19), which is what lets
  pair#292 derive `--zoom` from the pane width and hand pages a ~1024 px
  viewport at 94 columns. Any replacement engine or renderer owes the same two
  properties: a *controllable* viewport width independent of the cell grid, and
  text drawn one glyph per cell so zooming out costs no legibility. A renderer
  that can only downscale pixels has neither, which is the real reason the
  screenshot approach was rejected.
- This is the option to spend engineering on if the browser tab earns its place.

### Tier 3 spike — the bounded rebase experiment

Hardware is not the constraint (measured 2026-09-19 on the operator's M2, 96 GB
RAM: 302 GB free disk; Chromium needs ~100–150 GB). What is unknown is exactly
one thing — whether the text interception can be re-derived against a paint
pipeline that has been refactored since Chromium 111. The spike exists to answer
that, and nothing else, before anyone commits to tier 3.

**Build the CURRENT stable Chromium, never 111.** A Feb-2023 tree on macOS 26
with a 2026 SDK is *harder* to build than current Chromium: it is pinned to an
old SDK and to fetch dependencies that may no longer resolve. So the rebase IS
the experiment; reproducing v0.0.3 byte-for-byte is not a goal.

**We need one platform, not four.** Carbonyl publishes macos-{arm64,amd64} and
linux-{arm64,amd64}. Pair needs **macos-arm64** only, so builds happen on the
operator's machine and the release matrix collapses to a single artifact. If
pair ever targets Linux, this assumption is void and the cost rises.

**Prerequisites** (~half a day, mostly waiting):
- **Full Xcode.app.** Only Command Line Tools are installed
  (`/Library/Developer/CommandLineTools`); Chromium's macOS build expects the
  full SDK. ~10 GB.
- `depot_tools` + `gclient sync` at a current stable tag: tens of GB, bandwidth
  bound.
- `ccache`, and `out/` on the internal SSD.

**The four re-derivation targets** — the ~270 lines that modify pre-existing
Chromium code, in dependency order. Each one gets a yes/no answer:
1. **Text interception in Blink** (was `platform/fonts/font.cc`, `DrawBlobs`
   early return). `DrawBlobs` **no longer exists**; find the current call path
   that bloberizes and paints text runs, and suppress it there while reporting
   the run to the bridge. **This is the make-or-break target.**
2. **Skia glyph gating** (`SkBitmapDevice::onDrawGlyphRunList` wrapped in
   `Bridge::BitmapMode()`). Expected to survive nearly unchanged; the function
   is stable.
3. **viz software output** (`SoftwareOutputDeviceProxy` +
   `LayeredWindowUpdater` Mojo repurposed for a shared-memory terminal frame).
   Churn is likely but the concept is intact.
4. **DPI** (`ui/display/display.cc` → the bridge's DPI). Small and localized.

Plus: confirm the additive glue still applies — the Mojo text service
(`render_frame_host_impl` / `render_frame_impl` / `browser_interface_binders`)
and the headless shell hook (`headless/app/headless_shell*.cc`, verified still
present upstream 2026-09-19).

**Stop rule (go/no-go).** If target 1 is not rendering text through the bridge
after **one day of focused work**, stop and take the renderer option (stock
Chrome over CDP with our own Go painter). Targets 2–4 are not worth attempting
if 1 fails, since 1 is the only reason the fork exists. Record the outcome
either way — a failed spike that names *where* it stalled is the evidence that
keeps this from being re-litigated.

**What a successful spike does NOT settle.** Per-milestone upkeep (hours every
~4 weeks, forever) and the CVE cadence remain tier 3's standing cost. A green
spike means the fork is *possible*, not that it is *worth it*.

**Triggers that would force a tier change** (write the answer before it's
urgent):
- The URL field or #293's Alt+click needs to open untrusted pages routinely →
  tier 3 or the renderer option (Chromium 111 is 3.5 years unpatched).
- A Chrome-111 rendering or protocol gap blocks a real dev-preview workflow.
- The v0.0.3 artifact becomes unavailable → tier 1 already covers it.
- Carbonyl gains an active maintainer → drop to tier 1.

## Done when

- Tier 1 is implemented: a mirrored, checksummed v0.0.3 artifact, a documented
  install, and a version check that names the supported build.
- Tiers 2 and 3, and the renderer alternative, each have a written decision:
  taken, or declined with the trigger that would reopen it.
- The spike below has measured the renderer option's fidelity and cost against
  the pages pair is actually used with, so tier 3 is never chosen by default.
- `atlas/` records which tier pair is on and what the exit path is, so a future
  session doesn't re-derive this analysis.

## Plan

- [ ] **M1 — Tier 1, own the artifact.** Mirror + checksum the v0.0.3
  macos-arm64 build, document the supported install, wire the version check
  and its notice. Independent of every decision below; it is what stops a
  silent fall back to npm's 0.0.2.
- [ ] **M2 — Renderer spike (~1 day).** A CDP text-extraction renderer against
  stock Chrome: inject the text/box reporter with
  `Page.addScriptToEvaluateOnNewDocument`, `Page.startScreencast` for graphics,
  paint cells in Go. Measure on three real pages (a local dev server, a docs
  site, a dashboard): text fidelity, CPU, latency, side by side with Carbonyl.
- [ ] **M3 — Chromium rebase spike (~1 day, go/no-go).** Per "Tier 3 spike"
  above:
  - [ ] Prerequisites: Xcode.app, depot_tools, `gclient sync` at current stable.
  - [ ] Apply the additive glue; confirm the Mojo service and headless-shell
        hooks still land.
  - [ ] Target 1, text interception in Blink — the make-or-break; **stop rule:
        one day**.
  - [ ] Targets 2–4 (Skia gating, viz output, DPI) only if target 1 works.
  - [ ] Build and run; compare rendering against 0.0.3 on the same pages.
  - [ ] Record where it stalled if it stalls; that is the deliverable either
        way.
- [ ] **M4 — Decide and record.** Pick the tier from M2/M3 evidence, write the
  triggers that would reopen it, and update `atlas/` with the tier, the seam
  and the exit path.

Order note: M2 before M3 deliberately. The renderer spike is the cheaper
experiment and it bounds M3's value — if the renderer is good enough, the fork
is unnecessary regardless of whether the rebase would have worked.

## Log

### 2026-09-19

Filed from the pair#292 planning session, after the operator asked whether we
should fork Carbonyl and maintain it. All measurements above were taken that
day: patch counts from the v0.0.3 tag, upstream churn from current Chromium
(`chromium.googlesource.com`), the sandbox check by running 0.0.3 under a pty
and reading its child process flags, and maintenance dates from the GitHub API
(carbonyl last commit 2023-02-26; browsh last commit 2025-07-05).

Browsh was considered as a maintained substitute and rejected for #292's
purpose: it tracks current Firefox (so it stays patched) and also renders real
text, but it reads text from the DOM via a WebExtension rather than from the
paint pass, and Firefox cannot be driven over CDP — which would cost the
operator/agent shared-browser story that is the point of #292.

### 2026-09-19 — hardware and prerequisites measured

The operator asked whether their M2 with 96 GB RAM can build Chromium. Measured
on the machine: **302 GB free** (Chromium needs ~100–150 GB), macOS 26.6.2, and
**no Xcode.app — Command Line Tools only**, which is a prerequisite to install.
RAM is not the limiting factor for a Chromium build (core- and I/O-bound, a few
GB per link); 96 GB simply removes any worry about parallelism. Carbonyl's own
readme puts fetch+build at ~1 h on an M1 Max, and incremental rebuilds after
editing a patched file are minutes — so the iteration loop while re-deriving
patches is tolerable.

Conclusion recorded in the Spec: hardware is not the constraint, the single
unknown is re-deriving the Blink text interception, and the spike is bounded by
a one-day stop rule. Also recorded: build current stable Chromium rather than
111, and pair needs only the macos-arm64 artifact where Carbonyl publishes four.

### 2026-09-19 — the viewport contract a replacement must honor

Measured while answering "how well does Carbonyl work at 94 columns" (full
table in pair#292's Log): one column is ~5.29 CSS px at zoom 100, so the right
pane sees a 497 px viewport by default — a tablet breakpoint, and a
`min-width:1024px` app loses half its columns. pair#292 therefore derives
`--zoom` from the pane width, targeting ~1024 px.

That turns two engine properties into requirements for any tier-3 fork or
renderer replacement: a viewport width that can be set independently of the
cell grid, and text drawn one glyph per cell (so zoom buys layout width without
shrinking text). Both fall out of the same patches this issue is about — the
text interception and the DPI override. A renderer that only downscales
screenshots satisfies neither.

### 2026-09-19 — the cell budget a replacement has to paint into

The 94 columns above was an estimate; measured on the operator's M2, the right
pane is **93 columns** collapsed and **123** expanded (`Alt+Shift+Enter`,
`pair layout toggle-focused`). Full table in pair#292's Log. The 5.29 CSS px
per column contract is unchanged, and so is the requirement it imposes on any
fork or replacement renderer.

What the real numbers sharpen is the *size of the job*. Even at 123 columns a
page sees only 651 px at zoom 100 — still a tablet breakpoint — so the
viewport-independent-of-the-grid property is not a nicety for narrow panes, it
is load-bearing at every pane width pair actually uses. And the worst case a
replacement must render legibly is the **collapsed** pane: ~1024 CSS px of
layout painted into 93 cells. A renderer that only downscales pixels fails
that outright; a Browsh-style text-run painter has to crowd or truncate runs
there, which is where its fidelity gap will first show.
