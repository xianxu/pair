# Boundary Review — pair#458 (milestone M3)

| field | value |
|-------|-------|
| issue | 458 — broadcast-web: recorder — CDP DOM stream per frame plus isolated-world observer |
| repo | pair |
| issue file | workshop/issues/000458-broadcast-web-recorder-cdp-dom-stream-per-frame-plus-isolated-world-observer.md |
| boundary | milestone M3 |
| milestone | M3 |
| window | 52fe755e5376db2fe8a4cf98c209786c597261f6..24e938aafe21f57cc254e9e9ee883cb6bad0f183 |
| command | sdlc milestone-close --issue 458 --milestone M3 |
| reviewer | claude |
| isolation | checkout |
| timestamp | 2026-10-12T02:29:45-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Inspecting the M3 diff now. The base is a synthetic "rebased" commit that still has conflict markers, so I'm checking that head resolves them cleanly.

Checking go-live ordering of snapshot refs vs `goAssets` (affects whether the new load path duplicates fetches).

Reviewing the live-check deliverables now: the plan promised a stress page "served from testdata", so I'm checking whether it was committed.

Running the package tests to confirm head is green before finalizing findings.

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

**VERDICT: FIX-THEN-SHIP**

M3 delivers what it set out to do. There is a fidelity tool (static render of the reduced state, offline asset serving through `Fetch`, pixelmatch-equivalent diff). Screenshots are now taken of the whole viewport, which fixes the earlier bug where a clipped capture of a scrolled page came out offset. The catch-up now keeps only the latest `hover` and `ptr`, with an equivalence check (`CheckCatchUp`). `hover` and `ptr` now carry `by` attribution. The atlas is updated, and the merge conflict markers left in the synthetic base are resolved cleanly at head.

`go test ./cmd/internal/webcast/...` passes, and `-race -count=3` passes on recorder and recording. `couchcmd` fails at setup because generated `runtimebundle` assets are missing from this checkout. Nothing in this diff touches them, so that failure is environmental.

Three Important findings keep this from SHIP:
- **Wrong frame scale on high-DPI screens.** Frames record a constant `scale: 0.5`, but on any screen with a device pixel ratio other than 1 the image is a different size. A real Retina cast records wrong metadata, and the new fidelity tool fails on it with "sizes differ".
- **Assets fetched twice or more.** The new DOM-reference asset loading downloads most images twice. It also fetches every `srcset` candidate and lazy images the page never loaded, with the page's credentials.
- **Live check not reproducible.** The plan's "stress page served from testdata" was never committed, so the stress-page live check can't be rerun from the repo.

### 1. Strengths
- `recording/catchup.go:42-100`: hover and ptr are treated as state (latest of each, kept in place), and `CheckCatchUp` now compares `Hover`/`Ptr` (`contract.go:306`). The test checks every cut, including one where the hovered node is removed.
- `recorder/side.go:283-292`: the screenshot is stamped at the midpoint of the capture call, and the whole viewport is captured instead of a CSS clip. Both are real findings from the measurement, written up in the issue Log.
- `fidelity/run.go:205-223`: the replay never contacts the page's origin; every request is fulfilled from the recording or failed. So the fidelity number measures the recording, not the live site.
- `fidelity/render.go`: rendering is pure and tested without IO (`TestRenderIsStaticAndKeepsState`). It drops scripts and `on*` handlers, renders shadow roots as declarative templates, inlines frames as `srcdoc`, and lets form state override attributes.
- The baseline is measured with the same tool on #457's fixtures. That makes the comparison between this recorder and #453's prototype fair.

### 2. Critical findings
None.

### 3. Important findings

**I-1 — `frame.scale` is wrong whenever the device pixel ratio isn't 1** (`recorder/side.go:287`, `side.go:419-445`; ARCH-PURPOSE)
- **What happens:** `Page.captureScreenshot` without a clip returns device pixels. `LowRes` halves that, but `shotDone.scale` is always `shotScale = 0.5`. The fixtures define scale as image pixels per CSS pixel (640×400 for a 1280×800 viewport at 0.5).
- **When it bites:** the cast Chrome runs windowed by default (`chrome.go:157`). On the operator's Retina MacBook (DPR 2), a 1280×800 viewport records 1280×800 frames labelled `scale 0.5`.
- **Effect on the fidelity tool:** it replays at forced DPR 1, so `DiffImages` returns "sizes differ" and `Run` aborts.
- **Why tests missed it:** the live runs forced `deviceScaleFactor: 1`, and the fake returns a DPR-1 image.
- **Fix:** downscale by `2·DPR`, or set `scale = w / vp.W`. Add a fake-page test with DPR 2.
- This is new family `css-px-contract`. The old clip code probably had the same mismatch, but this window rewrote that code and built a consumer on top of it.

**I-2 — asset acquisition depends on arrival order, so images are fetched twice; `srcset` and lazy images are over-fetched** (`side.go:40-88`, `recorder.go:916-924`, `mirror.go:436-461`; ARCH-ORDER, ARCH-SECURE, ARCH-FUNERAL)
- **The race:** `asked` assumes the network response arrives before the DOM reference ("its response is coming; no load needed"). In practice an inserted `<img>` reaches `afterInput` → `queueURL` first, which queues a `Network.loadNetworkResource` load. Then `loadingFinished` queues a `requestID` job anyway.
- **Result:** every image inserted after go-live, and every image on a new document, is loaded a second time, `Put` twice, and gets two `asset` records.
- **Over-fetch:** `queueRef` also loads every `srcset` candidate (1x, 2x, 3x, or every `w` width) and images marked `loading=lazy` that the page never fetched. These are credentialed loads (`includeCredentials: true`) on whatever site the cast follows.
- **Test gap:** `TestRecorderLoadsAssetsChromeDoesntHold` sends no Network events, so it only exercises one ordering. It also pins fetching both `x1` and `x2` as intended behaviour.
- **This is the 2nd finding in family `asset-record-dedupe-source`.** The rule that covers both: *one URL-keyed owner decides whether a body is fetched, and the decision must not depend on arrival order across the three acquisition paths (resource tree, Network response, DOM/font reference).*
- **Fix for the rule:** treat a DOM reference as a deferred candidate, and fetch it only if no Network response for that URL arrives by the document's load/settle point. A pending or completed response always supersedes it. Load only the candidate Chrome actually chose (`currentSrc`), not every `srcset` entry. Then add a test that interleaves a reference with a response in both orders.

**I-3 — the plan's M3 claims don't match what was built** (plan Milestones M3, Revisions; ARCH-PURPOSE)
- **This is the 2nd finding in family `plan-drift`.** The rule: *every concrete artifact a Plan checklist item names must exist at the boundary head, or a `## Revisions` entry in the same range says what replaced it.*
- **The missing artifact:** M3 promises a "stress page served from testdata". No stress page source exists anywhere in the tree; `TestLiveRecordURL` points at an external `http://127.0.0.1:8801/stress.html`. The stress fidelity numbers in the Log therefore can't be reproduced from the repo.
- **Undocumented design changes:** the recorder now fetches assets itself through `loadNetworkResource`. The plan and Spec assign capture to #459's store ("capture and storage are pair#459's"). The screenshot form changed as well. Neither change has a Revisions entry.
- **Fix:** commit the stress page under `testdata`, served by the live test, and add Revisions entries for the asset loading and `LowRes`.

### 4. Minor findings
- **`fidelity/run.go:84,100-106`:** a shot with no live state at any offset keeps the sentinel `Diff: 101` and is counted in the median and max as if it were a measurement. Skip it and report it as unmeasured instead (new family `sentinel-as-evidence`).
- **`side.go:428-433`:** `LowRes` calls `src.At` and `NRGBAModel.Convert` per pixel (4M boxed calls at DPR 2). It runs every 5 s on the side worker and 17× per shot in the fidelity tool. **This is the 2nd finding in family `hot-path-allocation`.** The rule: per-pixel and per-node loops work on typed buffers (`*image.NRGBA.Pix`, or a type switch), never interface calls.
- **`fidelity/run.go:123-170`** (ARCH-DRY): `launch` re-implements webcast's Chrome launch (pipes, args, `ExtraFiles`). It also leaks the profile directory and pipes when `os.Pipe` or `cmd.Start` fails.
- **`fidelity/run.go:272-280`:** scroll offsets are only restored in the top document. Iframes rendered as `srcdoc` and shadow-root scrollers keep scroll position 0.
- **`live_record_test.go:342-356`:** `TestLiveRecordURL` only logs `Stats`. The "0 dropped, 0 desync, 0 lost" in the Log isn't asserted.
- **`side.go:56-60`:** when the asset queue is full, the job is dropped but `asked[url]` stays true, so that URL is never retried in this recording.
- **`fidelity/render.go:186`:** `option` has an unused `_ int` parameter. Attribute names and tag names are written without validation; acceptable for a development tool.
- **Blog result:** the blog's max (2.97) is slightly above the baseline's (2.94). The Log's explanation (the page settling at go-live) is plausible, but the Done-when wording "at least matches" is met on the median only.

### 5. Test coverage notes
- Mutation reasoning: reverting the hover/ptr filter in `CatchUp` makes `TestCatchUpKeepsLatestHoverAndPtr` fail (its count assertion). Reverting `samePtr` in the contract has no test that fails on its own, but the cut-sweep exercises it.
- No test sees a device pixel ratio other than 1 (I-1), interleaves a Network response with a DOM reference (I-2), or covers `resolveURL` or `LowRes` directly. Both are pure and cheap to table-test: odd dimensions, relative and protocol-relative references, `data:`.
- Fidelity and live-URL runs are env-gated manual runs, which suits a development tool, but their inputs (stress page, recordings) aren't committed (I-3).

### 6. Architectural notes
- **ARCH-DRY:** flag (Minor, the Chrome launch duplication). `fup` vs `up` is fine.
- **ARCH-PURE:** pass. `Render`, `Diff`, `LowRes`, `resolveURL` and `mirror.refs` are pure; IO stays in `run.go` and `side.go`.
- **ARCH-PURPOSE:** flag (I-1, I-3).
- **ARCH-MOCK:** pass. The new CDP surface (`loadNetworkResource`, `IO.read`/`close`) is modelled in the fake Chrome. The fidelity tool is real-Chrome-only by design.
- **ARCH-CONSTRAINTS:** flag (Minor `LowRes`). The 32 MiB per-body cap on loads is good, though it's checked after `append`, so it can overshoot by one 1 MiB chunk.
- **ARCH-SECURE:** flag (part of I-2). The page's DOM, which is untrusted, now drives credentialed loads beyond what the page itself fetched.
- **ARCH-ORDER:** flag (I-2). The meaning of `asked` depends on arrival order, and the test covers only one interleaving.
- **ARCH-FUNERAL:** pass with a note. `asked` grows per distinct URL and dies with the recorder (documented). The fidelity profile is removed on `close` but not on early launch errors. Double `Put` and over-fetched `srcset` assets grow each recording's asset store per image (I-2).
- **Upcoming wave-2 fidelity work** will consume `frame.scale`, so settle I-1 before then.

### 7. Plan revision recommendations
- `## Revisions` — 2026-10-12, M3 as built: screenshots are the whole viewport downscaled in Go (`recorder.LowRes`), stamped at the capture midpoint. `frame.scale` is image pixels per CSS pixel, derived from DPR (after the I-1 fix).
- `## Revisions` — the recorder loads assets Chrome doesn't hold in the page's context (`Network.loadNetworkResource`). This is a scope shift from "capture is #459's", so state the dedupe rule and the `srcset`/lazy policy (I-2).
- `## Revisions` (or commit the page) — the stress page lives at `cmd/internal/webcast/testdata/...`, served by the live check; or state why it lives elsewhere and how to reproduce the runs.

```findings
findings:
  - id: new
    severity: Important
    family: css-px-contract
    title: |
      frame.scale is the constant 0.5 but LowRes halves device pixels, so DPR 2 casts record wrong scale and break the fidelity diff
    detail: |
      side.go:287 sets scale: shotScale while LowRes halves captureScreenshot's device-pixel image. On the default windowed cast Chrome on a Retina Mac (DPR 2), a 1280x800 viewport yields 1280x800 frames labelled 0.5 (fixtures: 640x400 at 0.5 = image px per CSS px). fidelity.Run replays at DPR 1 and aborts with "sizes differ". Fix: downscale by 2*DPR or set scale = w/vp.W; add a DPR-2 fake test.
  - id: new
    severity: Important
    family: asset-record-dedupe-source
    title: |
      Asset fetch dedupe depends on arrival order: DOM-ref loads race Network responses (double fetch and Put), and every srcset candidate and lazy image is loaded with credentials
    detail: |
      This is the 2nd finding in family asset-record-dedupe-source. Rule: one URL-keyed owner decides whether a body is fetched, independent of arrival order across the resource tree, Network responses and DOM/font references. Today queueURL (afterInput) runs before loadingFinished for inserted or new-document images, so both jobs run. queueRef loads all srcset candidates and never-loaded lazy images via loadNetworkResource with includeCredentials. The test covers only the no-Network ordering and pins x1+x2 loading. Defer refs until load/settle, let responses supersede them, load only currentSrc, and test both interleavings.
  - id: new
    severity: Important
    family: plan-drift
    title: |
      M3 promised a stress page served from testdata; none is committed, and recorder-side asset loading and LowRes have no Revisions entry
    detail: |
      This is the 2nd finding in family plan-drift. Rule: every concrete artifact a Plan checklist item names must exist at the boundary head, or a Revisions entry in the same range says what replaced it. TestLiveRecordURL targets an external 127.0.0.1:8801/stress.html, so the Log's stress numbers can't be reproduced from the repo. The Spec assigns capture to #459, but the recorder now fetches bodies itself.
  - id: new
    severity: Minor
    family: sentinel-as-evidence
    title: |
      fidelity.Run reports a shot with no live replay as Diff 101 and includes it in median and max
  - id: new
    severity: Minor
    family: hot-path-allocation
    title: |
      LowRes does per-pixel src.At plus color-model Convert (millions of boxed calls per shot); use typed Pix buffers
    detail: |
      This is the 2nd finding in family hot-path-allocation. Rule: per-pixel and per-node loops work on typed buffers, never interface calls. Not on the recorder goroutine (side worker every 5 s), but it runs 17x per shot in the fidelity tool.
  - id: new
    severity: Minor
    family: shared-chrome-launch
    title: |
      fidelity launch duplicates webcast's Chrome launch and leaks the profile dir and pipes on early errors
  - id: new
    severity: Minor
    family: test-asserts-claimed-numbers
    title: |
      TestLiveRecordURL logs Stats but doesn't assert the 0 dropped, desync and lost figures the Log reports
  - id: new
    severity: Minor
    family: fidelity-scroll-scope
    title: |
      fidelity replay restores scroll only in the top document; srcdoc iframes and shadow-root scrollers stay at 0
  - id: new
    severity: Minor
    family: asset-retry-after-drop
    title: |
      A queue-full drop in queueAsset leaves asked[url] set, so the asset is never retried in that recording
```
