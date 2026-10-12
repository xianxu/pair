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

---

## Re-review — 2026-10-12T02:38:46-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 458 — broadcast-web: recorder — CDP DOM stream per frame plus isolated-world observer |
| repo | pair |
| issue file | workshop/issues/000458-broadcast-web-recorder-cdp-dom-stream-per-frame-plus-isolated-world-observer.md |
| boundary | milestone M3 |
| milestone | M3 |
| window | 52fe755e5376db2fe8a4cf98c209786c597261f6..be308479357f0746c4c9aa42049a1e44f6a944c7 |
| command | sdlc milestone-close --issue 458 --milestone M3 |
| reviewer | claude |
| isolation | checkout |
| timestamp | 2026-10-12T02:38:46-07:00 |
| verdict | REWORK |

## Review

Checking the stress page assets, Store.Put dedupe, and the new tests.

Found a potential divide-by-zero in the new `LowRes` for DPR < 0.5 (Chrome browser zoom at 25–33% reports such DPRs). Verifying with a quick test.

Confirmed: `LowRes` panics at DPR 0.33/0.25. Now running the suite and checking the remaining minors' evidence.

```verdict
verdict: REWORK
confidence: high
```

The M3 review fixes mostly hold up. Frames are now sized from the DPR and labelled with a measured scale. A Network response now takes precedence over a DOM reference whichever arrives first, and both orders are tested. The stress page is committed under testdata and the Revisions entry is in place. `go test ./cmd/internal/webcast/...` passes at the head. The BR-23 fix added a crash, though. The new `LowRes(full, dpr)` panics with an integer divide-by-zero for any DPR below 0.5, and `shotWorker` has no `recover`, so the panic takes down the process. I reproduced it in the disposable checkout: the DPRs 1, 1.25, 1.5, 2 and 0.5 work, while 0.33 and 0.25 panic. Chrome reports DPRs that low on an ordinary DPR-1 display when the operator zooms the cast window to 33% or 25%. That Critical crash is the only thing forcing REWORK, and the fix is two lines. BR-24 also stays open, because a `src`-plus-`srcset` image still gets its `src` loaded.

### 1. Strengths
- **`side.go:39-60`, the ownership rule.** It is written as one comment block and enforced in one place (`own`, `queueAsset`). This is the class-level rule the family asked for, not a fix to one site.
- **`TestRecorderAssetOwnerIndependentOfOrder` (`recorder_paths_test.go`).** It runs both interleavings through the production recorder, with a fake that fails on `loadNetworkResource`. Reverting to the old immediate `queueURL` would turn the "ref first" case red.
- **`TestRecorderScreenshotAtDPR2`.** It pins 500×350 at scale 0.5 for a 2000×1400 device-pixel shot. The old `LowRes` would have produced 1000×700, so this is real regression evidence for BR-23.
- **`serveStress` (`live_record_test.go`).** It serves the committed page with a second origin for the out-of-process frame, so the Log's stress numbers can now be reproduced from the repo. The README explains why `late.ttf` is left out.
- **`LowRes` now loops over typed `Pix` buffers**, so the boxed `At`/`Convert` calls are gone (BR-27).

### 2. Critical findings
- **`cmd/internal/webcast/recorder/side.go:470-496`: `LowRes` divides by zero when `dpr < 0.5`.**
  - **Cause:** with `f = 2*dpr < 1`, the bounds `x0 = int(x*f)` and `x1 = int((x+1)*f)` can be equal, so `n == 0` and `sum[c]/n` panics.
  - **Where `dpr` comes from:** `vp.DPR` is the page's own `devicePixelRatio` (`recorder.go:735`). Chrome lowers it with browser zoom (33% zoom on a DPR-1 display gives 0.33).
  - **Effect:** the panic happens inside the `shotWorker` goroutine, which has no `recover`, so it crashes the process.
  - **Fix sketch:** clamp `f = max(2*dpr, 1)` (never upsample) and keep `scale = w/vp.W` as measured. Or guarantee `x1 >= x0+1` and `y1 >= y0+1`.
  - **Test to add:** a table test of `LowRes` over DPRs {0.25, 0.33, 0.5, 1, 1.25, 1.5, 2, 3} and odd sizes, with no panic and `w, h ≥ 1`.
  - **Family:** this is the 2nd finding in `css-px-contract`. The rule: the device-pixel → frame-pixel mapping must hold for every DPR Chrome can report (zoom 25%–500% × display DPR), and every output pixel must cover at least one source pixel.

### 3. Important findings
- **BR-24 stays not-addressed (the remainder only):**
  - **`src` of a `srcset` image:** `queueRef` still loads the `src` of an `<img>` that also has `srcset`, after the settle and with credentials, even though the browser fetched a srcset candidate instead. `TestRecorderLoadsAssetsChromeDoesntHold` pins this: `avatar.png` comes from an img with `srcset`. The same happens to an `<img src>` inside `<picture>`. The prior finding asked for "only currentSrc".
  - **Cheap fix:** skip `src` when `srcset` is present or the parent is `<picture>`.
  - **Remaining race:** a response that arrives after `refSettle` (an image slower than 2 s) still causes a second fetch and `Put`. Document that bound in the rule comment, or have the ref load hand ownership to a late response without a second `Put`.

### 4. Minor findings
- **Atlas:** `webcast-format.md:188` doesn't say what `frame.scale` means (image px per CSS px). The URL-ownership and settle rule, and the "never `srcset` or lazy" policy, are in the plan Revision but not in `atlas/webcast.md:67-68`.
- **Regression tests are missing for BR-26** (shots skipped without a live replay; the selection logic could be pulled out of `Run` as a pure function) **and BR-31** (retry after a queue-full drop). Both are disposed not-addressed per the evidence rule.
- **BR-30 was not touched** in the fix commit.
- **`TestRecorderScreenshotsAndLayout` now expects scale 0.25.** That means its fake screenshot is half the viewport's CSS width at DPR 1. The fixture is inconsistent with a real capture, so make it a full-viewport image.

### 5. Test coverage notes
- Both BR-23 and BR-24's ordering fix have tests that would fail without the fix (checked by reading what the old code would do).
- No test covers `LowRes` directly, for fractional DPR, DPR below 1 or odd sizes. That gap is how the Critical got through.
- The ordering test sleeps `3*refSettle`, so it only exercises a response that arrives *within* the settle window, not the late-response case.

### 6. Architecture
- **ARCH-DRY: pass.** The duplicate Chrome launch is justified in a comment at `run.go:131`.
- **ARCH-PURE: pass.** `LowRes`, `resolveURL` and `refs` are pure. `flushRefs` decides from a clock that is passed in.
- **ARCH-PURPOSE: flag.** See BR-24's remainder.
- **ARCH-MOCK: pass.** The fake models `loadNetworkResource` and DPR.
- **ARCH-CONSTRAINTS: flag.** See the Critical: the input domain (DPR range) isn't enforced.
- **ARCH-SECURE: flag (part of BR-24).** The page's DOM, which is untrusted, still drives a credentialed load of a `src` the page didn't fetch.
- **ARCH-ORDER: pass with a note.** The `owned`/`pending` maps are now a clear per-URL state (pending → owned). The late-response case is the one ordering left untested.
- **ARCH-FUNERAL: pass.** `owned` and `pending` die with the recorder, as documented, and the fidelity profile is now removed on early errors.

### 7. Plan revision recommendations
- Add to the 2026-10-12 Revision: frames are "half CSS size, never upsampled (DPR clamped)", and the `src`/`srcset` rule as finally built.

```findings
dispose:
  - id: BR-23
    disposition: addressed
    note: |
      DPR-aware LowRes + measured scale; TestRecorderScreenshotAtDPR2 fails on the old code. The fix introduced a new crash for dpr<0.5, raised separately.
  - id: BR-24
    disposition: not-addressed
    note: |
      Ordering fixed and tested in both orders, but the src of an img with srcset (or inside picture) is still loaded with credentials (the test pins avatar.png), and a response arriving after refSettle still double-fetches.
  - id: BR-25
    disposition: addressed
    note: |
      testdata/stress committed and served by serveStress; the plan Revision covers LowRes and recorder-side loading.
  - id: BR-26
    disposition: not-addressed
    note: |
      Skipped counter present in run.go:110, but no test; extract shot selection as a pure fn and test it.
  - id: BR-27
    disposition: addressed
    note: |
      Typed Pix loops in LowRes.
  - id: BR-28
    disposition: addressed
    note: |
      fail() cleanup on early errors; duplication justified in a comment at run.go:131.
  - id: BR-29
    disposition: addressed
    note: |
      TestLiveRecordURL now errors on Losses() or Desync.
  - id: BR-30
    disposition: not-addressed
    note: |
      No change: replay scroll still targets only the top document and querySelector in it.
  - id: BR-31
    disposition: not-addressed
    note: |
      delete(owned) on drop is present, but no test fills the queue and checks a later sighting retries.
findings:
  - id: new
    severity: Critical
    family: css-px-contract
    title: |
      LowRes panics (integer divide by zero) for DPR < 0.5, crashing the process from shotWorker
    detail: |
      This is the 2nd finding in family css-px-contract. Rule: the device-px to frame-px mapping must hold for every DPR Chrome reports (browser zoom 25-500% times display DPR); every output pixel covers at least one source pixel. side.go LowRes with f=2*dpr<1 yields x0==x1, so n==0 and sum/n panics; reproduced at dpr 0.33 and 0.25. vp.DPR is the page's devicePixelRatio, which browser zoom lowers. Clamp f>=1 (or x1>=x0+1), keep scale measured, and add a LowRes table test over fractional and sub-1 DPRs.
  - id: new
    severity: Minor
    family: readme-user-surface
    title: |
      Atlas doesn't state frame.scale's meaning or the asset URL-ownership/settle/no-srcset rule
    detail: |
      This is the 2nd finding in family readme-user-surface. Rule: every contract stated in a plan Revision that a consumer reads (record fields, recorder policy) is also in the atlas page for that surface. webcast-format.md:188 lists scale without units; webcast.md:67-68 omits the ownership rule.
```

---

## Re-review — 2026-10-12T02:45:24-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 458 — broadcast-web: recorder — CDP DOM stream per frame plus isolated-world observer |
| repo | pair |
| issue file | workshop/issues/000458-broadcast-web-recorder-cdp-dom-stream-per-frame-plus-isolated-world-observer.md |
| boundary | milestone M3 |
| milestone | M3 |
| window | 52fe755e5376db2fe8a4cf98c209786c597261f6..8c5a8aba0e2ce469802f0c99132cf27114ce9e80 |
| command | sdlc milestone-close --issue 458 --milestone M3 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T02:45:24-07:00 |
| verdict | SHIP |

## Review

Checking prior findings against the head; now looking at the fidelity replay and DPR sources.

Tests pass. Now reverting the fixes in the disposable checkout to confirm the regression tests go red.

```verdict
verdict: SHIP
confidence: high
```

Round 3 confirms the two blocking findings from round 2 are fixed. **BR-32:** `LowRes` now keeps every output pixel at one or more source pixels, so it can no longer divide by zero. **BR-24:** each asset URL is now fetched once per recording, by whichever source claims it first. Every `webcast/...` package passes `go test` at head. I put the round-2 versions of `side.go` and `mirror.go` back, and three tests failed without the fixes: `TestLowResAnyDPR` panicked with integer divide by zero, `TestRecorderLateResponseDoesntRefetch` saw the late response fetched a second time, and `TestRecorderLoadsAssetsChromeDoesntHold` timed out. I then put the head versions back. The base commit had unresolved merge-conflict markers in `cast.go`, `record.go` and `reduce.go`, and this range removes them all. Nothing left open is above Minor: one new Minor finding about DPR below 0.5, and two Minor findings from earlier rounds that are still open.

1. **Strengths**
   - `side.go:39-62`: one rule decides who fetches an asset. `own()` is the only place that claims a URL, it removes any waiting reference when it does, and `TestRecorderAssetOwnerIndependentOfOrder` (`recorder_paths_test.go:431`) tests both orders, reference first and response first.
   - `mirror.go:463-464`: the new `Chosen` flag means an image whose source the browser picks (it has a `srcset` or sits in a `<picture>`) is never loaded from its `src`. The test covers both cases.
   - `fidelity/run.go`: `Pick` and `Summarize` are pure functions now. A shot with no live replay is counted as skipped instead of being scored as 101, and `TestPickAndSummarize` tests this without starting Chrome (ARCH-PURE).
   - `side.go:342`: `frame.scale` is computed from the actual image and viewport sizes, not set as a constant.

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - **DPR below 0.5 (3rd finding in family `css-px-contract`).** When `devicePixelRatio` is under 0.5, `LowRes` uses one source pixel per output pixel, so the recorded frame's `scale` equals the DPR rather than 0.5. Two places still assume 0.5:
     - `atlas/webcast-format.md:188` says the scale is 0.5 "whatever the device pixel ratio".
     - The fidelity replay (`run.go:351`, `LowRes(full,1)`) always produces a half-size image. The sizes then differ, `DiffImages` returns an error, and `Run` stops.

     The rule to fix rather than this one instance: `Frame.Scale` is the single source for converting CSS pixels to frame pixels, and every consumer reads it. The fidelity replay should size its output from `f.Scale`, and the atlas should say "0.5, or the DPR when it is under 0.5".
   - **BR-30** is still open. In `run.go:330-335`, scroll is restored with `document.querySelector` in the top document only, so scrollers inside srcdoc iframes and shadow roots stay at 0.
   - **BR-31** is fixed in the code but has no test. `side.go:76` gives up the URL when the queue is full, but no test fills `assetQ`. Note also that a drop that started from a reference has already lost its waiting entry, so it is retried only if the page references that URL again.

5. **Test coverage notes:** the reverts above showed that the three new or changed tests fail without their fixes. The fidelity tests passed (in 0.15 s), and nothing tests the case where the recorded frame and the replay come out at different sizes.

6. **Architecture**

   | Principle | Result | Why |
   |---|---|---|
   | ARCH-DRY | pass | |
   | ARCH-PURE | pass | `Pick` and `Summarize` are pure |
   | ARCH-PURPOSE | pass | |
   | ARCH-MOCK | pass | the fake Chrome keeps state across calls, and both asset orders run against it |
   | ARCH-CONSTRAINTS | pass | `owned` and `pending` grow with the number of distinct URLs and live only as long as the recording |
   | ARCH-SECURE | pass | `includeCredentials` loads now happen only for images the page loaded eagerly, in its own context |
   | ARCH-ORDER | pass | ownership goes through `own()`, and both interleavings are tested |
   | ARCH-FUNERAL | pass | the maps are dropped with the `Recorder`, and `FIDELITY_OUT` is only set by hand during development |

7. **Plan revisions:** add a Revisions entry saying `frame.scale` is 0.5 when DPR is 0.5 or more, and equals the DPR below that.

```findings
dispose:
  - id: BR-24
    disposition: addressed
    note: |
      own() is the single URL claimer, refs settle 2s, Chosen/srcset/lazy skipped; both orders tested and late-response test goes red on revert.
  - id: BR-26
    disposition: addressed
    note: |
      Pick returns Diff -1 with no tries, Summarize counts it skipped; TestPickAndSummarize pins it.
  - id: BR-30
    disposition: not-addressed
    note: |
      run.go:330-335 still restores scroll only via top-document querySelector.
  - id: BR-31
    disposition: not-addressed
    note: |
      side.go:76 deletes owned on drop, but no test fills assetQ to pin the retry.
  - id: BR-32
    disposition: addressed
    note: |
      f=max(2*dpr,1) plus clamped bounds; TestLowResAnyDPR panics on revert.
  - id: BR-33
    disposition: addressed
    note: |
      webcast-format.md:188 states scale units; webcast.md:69-74 states the ownership, settle and no-srcset rule.
findings:
  - id: new
    severity: Minor
    family: css-px-contract
    title: |
      Below DPR 0.5, frame.scale is the DPR, but the atlas and the fidelity replay still assume 0.5
    detail: |
      3rd in family. The rule: Frame.Scale is the single source for the CSS-px to frame-px mapping, and every consumer reads it. Today webcast-format.md:188 says 0.5 whatever the DPR. fidelity run.go:351 always replays at LowRes(full,1), so a cast recorded at DPR under 0.5 has a different image size and DiffImages makes Run fail. Size the replay from f.Scale and correct the atlas.
```
