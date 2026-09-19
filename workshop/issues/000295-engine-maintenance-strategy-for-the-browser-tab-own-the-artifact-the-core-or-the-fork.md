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
and the iteration loop are not (a Chromium build is ~100 GB and over an hour,
per platform, and four platform binaries are published today).

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
- Mirror the v0.0.3 release for the platforms we use, with checksums.
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
applying, and stand up builds for four platforms.
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
- This is the option to spend engineering on if the browser tab earns its place.

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

- [ ] Tier 1: mirror + checksum the v0.0.3 artifacts, document the install,
  wire the version check.
- [ ] Spike (timeboxed, ~1 day): a CDP text-extraction renderer against stock
  Chrome — inject the text/box reporter, screencast for graphics, paint cells.
  Measure against three real pages (a local dev server, a docs site, a
  dashboard): text fidelity, CPU, latency. Compare with Carbonyl side by side.
- [ ] Decide tiers 2 and 3 from the spike, and record the decision + triggers.
- [ ] Atlas: the tier, the seam, and the exit path.

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
