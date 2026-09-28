---
id: 000295
status: open
created: 2026-09-19
updated: 2026-09-19
estimate_hours:
github_issue:
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
