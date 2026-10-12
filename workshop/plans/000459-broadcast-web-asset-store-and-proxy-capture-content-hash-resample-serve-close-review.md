# Boundary Review — pair#459 (whole-issue close)

| field | value |
|-------|-------|
| issue | 459 — broadcast-web: asset store and proxy — capture, content hash, resample, serve |
| repo | pair |
| issue file | workshop/issues/000459-broadcast-web-asset-store-and-proxy-capture-content-hash-resample-serve.md |
| boundary | whole-issue close |
| milestone | — |
| window | fcd5f683d8ec5c4b07a53ea4fdbf5062d16448cd..b11452eb5c573de94c6bc8589e73a19f417631c8 |
| command | sdlc close --issue 459 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T02:52:07-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Reviewing #459's asset store and proxy. I've read the new code; next I'm running the tests and checking module tidiness.

I've finished the review. Verdict below.

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

**Summary.** #459 delivers what its Spec and Plan promise, as amended by the tech lead (TL) on 10-12. There is a real store, `assets.Recorded`: it deduplicates bodies by hash, resamples images to the budget (including WebP), refuses decompression bombs, and makes CSS URLs absolute at `Put`. There is an `assets.Proxy` behind the Server's `<token>/assets/<hash>` route, with a type allow-list, a CSP and nosniff. One CSS scanner handles both directions, and a shared corpus holds the viewer's `css.js` to the same rules. The live viewer tests now go through the real store and proxy, and they check that no request leaves the origin. In the checkout, `go test ./cmd/internal/webcast/...` passed and `node --test tests/webcast/css.test.mjs` passed 7 of 7. `./cmd/internal/artifactpath` failed for an environmental reason, not because of this diff: the generated runtime bundle (`cmd/internal/runtimebundle/assets/runtime/files`) is missing from the review checkout. I did not run the live Chrome tests. Nothing blocks shipping. Four cheap Important items remain: decode memory has no bound, `Proxy.Bind` silently accepts stores it can't serve, `go.mod` is not tidy, and the corpus quietly dropped cases the plan promised.

### 1. Strengths
- **One scanner, two directions.** `RewriteCSS` (`cmd/internal/webcast/assets/css.go:20`) makes URLs absolute at `Put` and points them at `<hash>` at serve time, just by passing a different `to`. A URL is rewritten only when `to` changes it, so an unchanged stylesheet keeps its exact bytes. That keeps `RunStoreContract`'s exact-CSS requirement intact.
- **Bombs are refused before any decode.** `image.DecodeConfig` is checked against a 64 MP ceiling before decoding (`image.go:30-38`). Dropping animated-GIF detection because `gif.DecodeAll` would get past the ceiling was the right call, and the plan's Revisions section records it.
- **The proxy limits what it will serve** (`proxy.go:88-104`): only images, CSS and fonts, with a sandbox CSP and nosniff. A recorded page cannot get HTML or script served from the viewer's origin, and `TestProxyRefusesUnservable` tests that.
- **The live tests now use the real store** (`replay_test.go`, `throughStore`). Doing so exposed a real blind spot: the fixture's 1738×2130 screenshot is over the budget. The no-origin assertion also requires that a stylesheet, a font and an image actually came through the proxy (`viewer_live_test.go:117-136`).
- **`RunStoreContract` runs three times:** against a `DirWriter`, a writer that can't read back, and the fake.

### 2. Critical findings
None.

### 3. Important findings
- **Decode memory has no bound (ARCH-CONSTRAINTS)** — `assets/recorded.go:72-79`, `image.go:46`.
  - Decoding runs outside the lock with no limit on how many run at once.
  - One image just under the 64 MP ceiling can decode to about 256 MB (RGBA) or about 512 MB (16-bit PNG), plus the scaled copy.
  - If #458's recorder calls `Put` from several goroutines, peak memory grows with the number of large images arriving together. Neither the plan nor the code sets a budget.
  - Fix: a small semaphore around the decode/scale step (for example 1–2 slots), and state the per-decode memory budget in the plan.
- **`Proxy.Bind(Store)` accepts stores it can never serve from (ARCH-SECURE)** — `proxy.go:39,50`.
  - Any `Store` is accepted, but `Serve` returns 404 for every hash unless the store also implements `Typed`. For example, binding the #457 `MemStore` fake compiles and then serves nothing, with no signal.
  - Fix: define `type Served interface{ Store; Typed }` and have `Bind` take it, so the mistake becomes a compile error.
- **`go.mod` is not tidy** — `go.mod:28`.
  - `golang.org/x/image` is imported directly (`image.go`) but marked `// indirect`.
  - `go mod tidy -diff` would move it to the direct `require` block and drop the stale `golang.org/x/sys v0.44.0` lines from `go.sum`.
  - Fix: run `go mod tidy`.
- **The shared corpus dropped a case the plan promised** — `tests/webcast/css_urls.json`.
  - Plan Task 1 listed "a URL inside a comment kept" as a corpus case. It is not in the corpus; it appears only in the Go-only `TestRewriteCSSAbsolutizes`.
  - Go and JS actually disagree on these spans. Go skips comments and requires a word boundary (so `myurl(` is not `url(`). The `refs` regex in `css.js` does neither.
  - So the "one written rule in two languages" claim only holds because the cases where they disagree are left out.
  - Fix: add the comment and `myurl(` cases and align `css.js`, or record the divergence in the plan's Revisions section.

### 4. Minor findings
- `Recorded.types[hash]` is last-writer-wins. If the same bytes are captured under two MIME types (say `image/svg+xml`, then `text/html`), the later type wins, and the proxy may then return 404 for the earlier, servable one (`recorded.go:87`).
- Two concurrent `Put`s of the same body both miss the `done` cache and both decode. The "decoded once" guarantee only holds for sequential calls (`recorded.go:65-71`).
- Two concurrent `Put`s of the same URL bind it in lock-acquisition order, not call order. The store's binding (`Lookup`, which the proxy's CSS rewrite uses) can then disagree with the last `asset` record the recorder emitted. Either document that the caller must serialize `Put` per URL, or bind by sequence (ARCH-ORDER).
- The `Capture.DeclaredW/H` and `Policy.MaxPixels` doc comments in `store.go` still say "unless its declared size needs more". `Recorded` ignores the declared size, which the TL decided in (3). The comment should say the declared size is not yet honored.
- `readString` doesn't decode CSS hex escapes (`\41 `), so a URL written with them is resolved literally.
- Absolute URLs inside an `@import`ed stylesheet that are still unbound are loaded by the browser directly and blocked by the viewer's CSP, rather than retried through `resolveAsset` when the asset arrives. No request leaves the origin; the asset just stays missing.
- GIF, SVG and AVIF bodies sit outside the size bound. The atlas documents this; keep it in mind when describing what "a recording's size is bounded" means.
- The `stubLive.Ended` line in `server_test.go` has a gofmt-only whitespace change.

### 5. Test coverage notes
- The pure entities are tested without IO: the corpus, the "exact when unchanged" check, unterminated inputs, the size bound, alpha → PNG, the exact cases, the bomb, and WebP.
- `Recorded` is tested over a real `DirWriter` in `t.TempDir()`. The Proxy tests cover rebinding across later responses.
- Missing: concurrent `Put` (dedup and binding order) and a memory or concurrency test for decodes.
- `throughStore` uses `discardWriter`, so the live tests keep bodies in memory and never exercise reading back from disk. Unit tests cover `DirWriter.Asset`, so this is acceptable.

### 6. Architectural notes for upcoming work
Each principle, pass or flag:
- **ARCH-DRY: pass.** One scanner for both directions; `readAsset` is shared between the reader and the writer.
- **ARCH-PURE: pass.** `RewriteCSS` and `Resample` are pure; `Recorded` and `Proxy` are thin.
- **ARCH-PURPOSE: pass.** The deferred `Bind`/`NewAssets` wiring and the live blog cast are both the TL's 10-12 decisions.
- **ARCH-MOCK: pass.** The `AssetWriter` seam plus a portable temp-directory `DirWriter`.
- **ARCH-CONSTRAINTS: flag** (decode memory, above).
- **ARCH-SECURE: pass with one flag** (`Bind` typing). The bomb ceiling, allow-list and CSP are good.
- **ARCH-ORDER: minor flag** (concurrent rebinding).
- **ARCH-FUNERAL: pass.** `assets/<hash>` files go when the recording directory is deleted, and the in-memory maps die with the cast.

Whichever of #458 and #459 lands second has to wire `Server.Assets().Bind(store)` and point couchcmd's `NewAssets` at `Recorded`. Making `Bind` require the typed interface first would let the compiler check that wiring.

### 7. Plan revision recommendations
- Under Revisions: the corpus leaves out comment and identifier-boundary cases because `css.js` and Go differ there. Either align `css.js` or record the difference as known.
- Under Revisions/Constraints: state the decode memory budget (about 256–512 MB for one decode at the ceiling) and the concurrency limit `Recorded` enforces.

```findings
findings:
  - id: new
    severity: Important
    family: unbounded-resource-per-concurrent-call
    title: |
      Recorded.Put decodes images outside the lock with no concurrency bound; each decode at the 64 MP ceiling allocates 256-512 MB
    detail: |
      recorded.go:72-79 / image.go:46. Concurrent Puts from #458's recorder multiply peak memory. Add a small decode semaphore and declare the budget (ARCH-CONSTRAINTS).
  - id: new
    severity: Important
    family: invalid-state-representable-at-api
    title: |
      Proxy.Bind accepts any Store but Serve 404s everything unless the store also implements Typed
    detail: |
      proxy.go:39,50. Binding MemStore (or any non-Typed store) compiles and silently serves nothing. Make Bind take interface{ Store; Typed } (ARCH-SECURE).
  - id: new
    severity: Important
    family: go-mod-tidy-drift
    title: |
      go.mod marks directly imported golang.org/x/image as indirect; go mod tidy -diff is non-empty
    detail: |
      image.go imports x/image/draw and webp directly. Run go mod tidy (also drops the stale x/sys v0.44.0 go.sum lines).
  - id: new
    severity: Important
    family: shared-corpus-omits-divergent-cases
    title: |
      css_urls.json omits the comment case the plan's Task 1 listed; Go and css.js actually disagree on comments and identifier-boundary url(
    detail: |
      Go skips /* */ and requires identStart; the css.js refs regex does neither. The "one rule, two languages" claim holds only because the divergent cases are left out. Add them and align css.js, or record the divergence in the plan's Revisions.
  - id: new
    severity: Minor
    family: concurrent-binding-order-undefined
    title: |
      Concurrent Put of the same URL rebinds in lock order, not call order; the proxy's Lookup can disagree with the last emitted asset record
    detail: |
      recorded.go:76-80. Document that the caller must serialize Put per URL, or bind by sequence (ARCH-ORDER). Concurrent identical bodies also both decode, so "decoded once" holds only for sequential calls.
  - id: new
    severity: Minor
    family: last-writer-wins-metadata
    title: |
      Recorded.types[hash] is overwritten by the last Put's MIME when identical bytes arrive under different types
  - id: new
    severity: Minor
    family: doc-contract-drift
    title: |
      store.go's Capture.Declared and Policy.MaxPixels docs promise the declared size widens the budget; Recorded ignores it (TL decision 3)
```

---

## Re-review — 2026-10-12T02:58:36-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 459 — broadcast-web: asset store and proxy — capture, content hash, resample, serve |
| repo | pair |
| issue file | workshop/issues/000459-broadcast-web-asset-store-and-proxy-capture-content-hash-resample-serve.md |
| boundary | whole-issue close |
| milestone | — |
| window | fcd5f683d8ec5c4b07a53ea4fdbf5062d16448cd..eeafaf89d62c111bc66ecb6b9814a648fe9d35b9 |
| command | sdlc close --issue 459 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T02:58:36-07:00 |
| verdict | SHIP |

## Review

Disposing round-1 findings: checking that the decode-slot test fails when the fix is reverted.

```verdict
verdict: SHIP
confidence: high
```

All seven round-1 findings are fixed. For each behaviour fix I found a regression test and confirmed that it fails when the fix is reverted. In the disposable checkout:
- With the old `css.js`, the shared corpus test (`css.test.mjs`) fails.
- Without the decode slot, `TestRecordedDecodesOneAtATime` fails with "one body decoded and stored 7 times".
- With last-type-wins restored, `TestRecordedFirstTypeWins` fails.

`go mod tidy -diff` is clean. `go test -race` passes for `assets` and `recording`. The Node CSS tests pass. Nothing blocks SHIP. Two new Minor items are left: a per-hash MIME that can disagree with an asset record, and atlas wording that describes the cast's bind as already wired when it is still waiting on #458.

### 1. Strengths
- `css.js` now uses the same lexical pieces as the Go scanner (`COMMENT`, `STRING`, `NAME`), and comments inside `image-set()` are handled too (`web/css.js:20-28`). The corpus now covers the cases where the two implementations used to disagree, so the "one rule, two languages" claim is actually tested.
- The decode slot checks the store again after waiting (`recorded.go:64-75`). That gives a memory bound and "decoded once" for concurrent calls in one step. It also gives up when the context ends, and fonts and CSS skip it (`TestRecordedWaitHonoursContext`).
- `Proxy.Bind(Served)` makes an unservable binding a compile error, which removes the old runtime type assertion (`proxy.go:28-46`).
- `Resample` checks the image header against the bomb ceiling before decoding, and keeps every GIF exact rather than decoding every frame (`image.go:36-47`).
- The proxy serves only images, stylesheets and fonts, with `nosniff` and a sandboxing CSP (`proxy.go:67-101`).

### 2. Critical findings
None.

### 3. Important findings
None.

### 4. Minor findings
- `recorded.go:124-130` (family `last-writer-wins-metadata`, 2nd finding): BR-6 settled which MIME `TypeOf` returns, but `store()` still returns `MIME: c.MIME`. If the same bytes arrive as `font/woff2` and then as `text/html`, the second asset record says html while the proxy serves the body as font. **The rule:** anything recorded about a hash must be keyed by the same identity as the dedup key, `(hash, MIME)`. Either the binding reports `types[hash]`, or `types` is keyed by `(hash, MIME)`. Fix it for `types` and the emitted `Asset.MIME` together.
- `atlas/webcast.md:86` (family `doc-contract-drift`, 2nd finding): the atlas says "the cast binds the recording's store to it when the recording opens". Nothing in production calls `Bind` or `NewRecorded` yet; per TL decision 1 that waits for whichever of #458/#459 lands second. **The rule:** when the atlas describes wiring owned by an unmerged dependency, it marks that wiring as pending that issue. Sweep the #459 atlas paragraphs for every claim about the bind or the recorder calling the store, not just this line.
- The `Put` doc comment (`recorded.go:53`) says "the recorder calls it from one worker". That is a requirement on a recorder that doesn't exist yet. It should be stated as a requirement on the caller, so #458's author reads it as one.

### 5. Test coverage notes
- `TestRecordedWaitHonoursContext` fills the process-wide `decoding` slot itself. That is fine while tests in the package run one at a time; if any test there uses `t.Parallel`, it will interfere. Worth a comment.
- The `Bind` change has no runtime test, and doesn't need one: the compiler enforces it, and the proxy and server tests bind `Recorded`.

### 6. Architectural notes
- **ARCH-DRY: pass.** `stored()` and `isImage()` helpers were extracted, and `readAsset` is shared by `Recording` and `DirWriter`.
- **ARCH-PURE: pass.** `RewriteCSS` and `Resample` are pure; IO goes through the `AssetWriter` and reader seam.
- **ARCH-PURPOSE: pass.** The production bind is deferred to whichever of #458/#459 lands second, which is a TL decision recorded in the plan. The live check after #458 merges is likewise explicitly not a gate on this issue.
- **ARCH-MOCK: pass.** `memWriter`/`discardWriter` and a `DirWriter` in a temp dir sit behind the real `AssetWriter` seam, and the live viewer tests go through the real `Recorded` and `Proxy`.
- **ARCH-CONSTRAINTS: pass.** There is now a declared budget: one decode at a time process-wide, about 256 MB at the 64 MP ceiling, documented in the code and the atlas.
- **ARCH-SECURE: pass,** apart from the Minor MIME mismatch above. The bomb header is refused before decoding, the hash is validated, there is an allow-list of servable types, and `Served` is typed.
- **ARCH-ORDER: pass.** Concurrent same-URL ordering is documented; identical bodies are decoded once.
- **ARCH-FUNERAL: pass.** The store's maps die with the cast. Asset files follow the recording's lifecycle (atlas `webcast-format.md:295`). The global semaphore is a fixed size.

### 7. Plan revision recommendations
- The plan's Proxy entry (plan line 82) still says `Bind(Store)`. Update it to `Bind(Served)` so it matches the round-1 Revisions note.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Process-wide decode slot honouring ctx; TestRecordedDecodesOneAtATime fails (7 stores) with the slot reverted.
  - id: BR-2
    disposition: addressed
    note: |
      Bind(Served) where Served = Store + TypeOf; enforced by the compiler; proxy/server tests bind Recorded.
  - id: BR-3
    disposition: addressed
    note: |
      x/image is a direct require; go mod tidy -diff is empty at head.
  - id: BR-4
    disposition: addressed
    note: |
      Corpus gains comment/string/name-boundary/image-set-comment/escape cases; css.test.mjs fails against the pre-fix css.js; Go passes the same corpus.
  - id: BR-5
    disposition: addressed
    note: |
      Put documents single-worker caller and finish-order binding; concurrent identical bodies now decode once.
  - id: BR-6
    disposition: addressed
    note: |
      First MIME wins per hash; TestRecordedFirstTypeWins fails when reverted. Residual record/TypeOf mismatch raised as new Minor.
  - id: BR-7
    disposition: addressed
    note: |
      store.go Declared/MaxPixels docs now say the declared size is not yet used (TL decision 3).
findings:
  - id: new
    severity: Minor
    family: last-writer-wins-metadata
    title: |
      Asset record MIME can disagree with the proxy's TypeOf for the same hash
    detail: |
      store() returns MIME c.MIME while types[hash] keeps the first label, so identical bytes put as font then text/html emit a text/html record served as font. Rule: per-hash metadata must be keyed by the dedup identity (hash, MIME); report types[hash] in the binding or key types by (hash, MIME), covering both types and Asset.MIME.
  - id: new
    severity: Minor
    family: doc-contract-drift
    title: |
      atlas/webcast.md says the cast binds the store at go-live; no production caller exists until #458 lands
    detail: |
      Rule: atlas text describing wiring owned by an unmerged dependency must mark it pending that issue. Sweep every #459 atlas claim about Bind or the recorder calling Store.Put, plus the Put doc comment's "the recorder calls it from one worker".
```

---

## Re-review — 2026-10-12T03:01:22-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 459 — broadcast-web: asset store and proxy — capture, content hash, resample, serve |
| repo | pair |
| issue file | workshop/issues/000459-broadcast-web-asset-store-and-proxy-capture-content-hash-resample-serve.md |
| boundary | whole-issue close |
| milestone | — |
| window | e43e3f8dd98817c68625d9947013c8725aec9270..625b9f5eeb186176ff504e18938480f40d328d38 |
| command | sdlc close --issue 459 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T03:01:22-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

I've checked all three changes in the window: the fix for the MIME mismatch, the atlas wording, and the merge-conflict fix in atlas/index.md. The verdict and findings follow.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

BR-8 is fixed for the whole class, not just the one site. `store()` (`cmd/internal/webcast/assets/recorded.go:124-128`) is the only place an asset record gets its MIME, and the record now takes the first MIME stored for that hash. So the recorded MIME and the proxy's `TypeOf` can no longer disagree. I removed the fix and `TestRecordedFirstTypeWins` failed ("served as font/woff2, recorded as text/html"); with the fix back it passes. `go vet` and `go test ./cmd/internal/webcast/...` are clean. The conflict fix in `atlas/index.md` keeps both sides: main's "Agent queries (#462)" and #459's "the asset store and proxy (#459)". No conflict markers are left in `atlas/` or `cmd/`. BR-9 is only partly done. The atlas paragraph now says the bind waits on #458. But the `Put` doc comment, which the finding named, still says the recorder calls it, and `Server.Assets`' comment still says the cast binds the store. Both are Minor and quick to fix, so they don't block. Nothing in the window changes the scope: the plan edit only records the `Bind(Served)` change from round 1, and the lessons.md change just moves an entry.

1. **Strengths**
   - `recorded.go:124-128`: the BR-8 fix is in the one place records are created, so every path is covered. That includes the dedup path: `Put`'s key contains `c.MIME`, so a second label always reaches `store()`.
   - `recorded_test.go:157-160`: the test now checks both the served MIME and the recorded MIME, and it fails when the fix is removed.
   - `atlas/index.md`: the conflict fix keeps both sides correctly.

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - BR-9, still open:
     - `cmd/internal/webcast/assets/recorded.go:53`: "The recorder calls it from one worker" describes a caller that doesn't exist until #458.
     - `cmd/internal/webcast/server.go:70-71`: "the cast binds the recording's store to it when the recording opens" says it happens today.
     - Fix: in both comments, add "(once #458 wires it)" or similar, matching `atlas/webcast.md:86-88`.

5. **Test coverage:** the BR-8 regression test is confirmed (fails without the fix). BR-9 changes wording only, so it needs no test.

6. **Architecture (interdiff only)**
   - **ARCH-DRY: pass.** No duplicated logic.
   - **ARCH-PURE: pass.** The type choice stays inside `store()`'s locked section, and the I/O seam is unchanged.
   - **ARCH-PURPOSE: flag (BR-9).** The finding asked for a sweep of every claim about the binding. Only the atlas instance was fixed; two code comments it named are left.

7. **Plan revisions:** none needed. The plan's Proxy entry now matches the code (`Bind(Served)`).

```findings
dispose:
  - id: BR-8
    disposition: addressed
    note: |
      store() now sets typ to types[hash] when the hash exists, so Asset.MIME equals TypeOf for every path that mints a record; TestRecordedFirstTypeWins checks b.MIME and fails when the fix is removed (verified).
  - id: BR-9
    disposition: not-addressed
    note: |
      atlas/webcast.md:86-88 is fixed, but recorded.go:53 (the Put comment the finding named) and server.go:70-71 (Assets comment) still describe the recorder and bind as present-tense production wiring.
```
