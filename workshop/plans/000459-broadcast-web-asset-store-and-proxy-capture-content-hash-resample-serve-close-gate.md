---
gate: boundary-review
issue: 459
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-12T02:52:07-07:00"
      dispatched: "2026-10-12T02:49:35-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: Recorded.Put decodes images outside the lock with no concurrency bound; each decode at the 64 MP ceiling allocates 256-512 MB
          detail: 'recorded.go:72-79 / image.go:46. Concurrent Puts from #458''s recorder multiply peak memory. Add a small decode semaphore and declare the budget (ARCH-CONSTRAINTS).'
          family: unbounded-resource-per-concurrent-call
          round: 1
        - id: BR-2
          severity: Important
          title: Proxy.Bind accepts any Store but Serve 404s everything unless the store also implements Typed
          detail: proxy.go:39,50. Binding MemStore (or any non-Typed store) compiles and silently serves nothing. Make Bind take interface{ Store; Typed } (ARCH-SECURE).
          family: invalid-state-representable-at-api
          round: 1
        - id: BR-3
          severity: Important
          title: go.mod marks directly imported golang.org/x/image as indirect; go mod tidy -diff is non-empty
          detail: image.go imports x/image/draw and webp directly. Run go mod tidy (also drops the stale x/sys v0.44.0 go.sum lines).
          family: go-mod-tidy-drift
          round: 1
        - id: BR-4
          severity: Important
          title: css_urls.json omits the comment case the plan's Task 1 listed; Go and css.js actually disagree on comments and identifier-boundary url(
          detail: Go skips /* */ and requires identStart; the css.js refs regex does neither. The "one rule, two languages" claim holds only because the divergent cases are left out. Add them and align css.js, or record the divergence in the plan's Revisions.
          family: shared-corpus-omits-divergent-cases
          round: 1
        - id: BR-5
          severity: Minor
          title: Concurrent Put of the same URL rebinds in lock order, not call order; the proxy's Lookup can disagree with the last emitted asset record
          detail: recorded.go:76-80. Document that the caller must serialize Put per URL, or bind by sequence (ARCH-ORDER). Concurrent identical bodies also both decode, so "decoded once" holds only for sequential calls.
          family: concurrent-binding-order-undefined
          round: 1
        - id: BR-6
          severity: Minor
          title: Recorded.types[hash] is overwritten by the last Put's MIME when identical bytes arrive under different types
          family: last-writer-wins-metadata
          round: 1
        - id: BR-7
          severity: Minor
          title: store.go's Capture.Declared and Policy.MaxPixels docs promise the declared size widens the budget; Recorded ignores it (TL decision 3)
          family: doc-contract-drift
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - "n": 2
      timestamp: "2026-10-12T02:58:37-07:00"
      dispatched: "2026-10-12T02:56:43-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: Process-wide decode slot honouring ctx; TestRecordedDecodesOneAtATime fails (7 stores) with the slot reverted.
          round: 2
        - id: BR-2
          disposition: addressed
          note: Bind(Served) where Served = Store + TypeOf; enforced by the compiler; proxy/server tests bind Recorded.
          round: 2
        - id: BR-3
          disposition: addressed
          note: x/image is a direct require; go mod tidy -diff is empty at head.
          round: 2
        - id: BR-4
          disposition: addressed
          note: Corpus gains comment/string/name-boundary/image-set-comment/escape cases; css.test.mjs fails against the pre-fix css.js; Go passes the same corpus.
          round: 2
        - id: BR-5
          disposition: addressed
          note: Put documents single-worker caller and finish-order binding; concurrent identical bodies now decode once.
          round: 2
        - id: BR-6
          disposition: addressed
          note: First MIME wins per hash; TestRecordedFirstTypeWins fails when reverted. Residual record/TypeOf mismatch raised as new Minor.
          round: 2
        - id: BR-7
          disposition: addressed
          note: store.go Declared/MaxPixels docs now say the declared size is not yet used (TL decision 3).
          round: 2
      findings:
        - id: BR-8
          severity: Minor
          title: Asset record MIME can disagree with the proxy's TypeOf for the same hash
          detail: 'store() returns MIME c.MIME while types[hash] keeps the first label, so identical bytes put as font then text/html emit a text/html record served as font. Rule: per-hash metadata must be keyed by the dedup identity (hash, MIME); report types[hash] in the binding or key types by (hash, MIME), covering both types and Asset.MIME.'
          family: last-writer-wins-metadata
          round: 2
        - id: BR-9
          severity: Minor
          title: 'atlas/webcast.md says the cast binds the store at go-live; no production caller exists until #458 lands'
          detail: 'Rule: atlas text describing wiring owned by an unmerged dependency must mark it pending that issue. Sweep every #459 atlas claim about Bind or the recorder calling Store.Put, plus the Put doc comment''s "the recorder calls it from one worker".'
          family: doc-contract-drift
          round: 2
      recipe: milestone-review
      isolation: fenced
      reviewed: eeafaf89d62c111bc66ecb6b9814a648fe9d35b9
      blocked: false
---

# Gate ledger — pair#459 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-12T02:52:07-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `unbounded-resource-per-concurrent-call` Recorded.Put decodes images outside the lock with no concurrency bound; each decode at the 64 MP ceiling allocates 256-512 MB
  recorded.go:72-79 / image.go:46. Concurrent Puts from #458's recorder multiply peak memory. Add a small decode semaphore and declare the budget (ARCH-CONSTRAINTS).
- **BR-2** [Important] `invalid-state-representable-at-api` Proxy.Bind accepts any Store but Serve 404s everything unless the store also implements Typed
  proxy.go:39,50. Binding MemStore (or any non-Typed store) compiles and silently serves nothing. Make Bind take interface{ Store; Typed } (ARCH-SECURE).
- **BR-3** [Important] `go-mod-tidy-drift` go.mod marks directly imported golang.org/x/image as indirect; go mod tidy -diff is non-empty
  image.go imports x/image/draw and webp directly. Run go mod tidy (also drops the stale x/sys v0.44.0 go.sum lines).
- **BR-4** [Important] `shared-corpus-omits-divergent-cases` css_urls.json omits the comment case the plan's Task 1 listed; Go and css.js actually disagree on comments and identifier-boundary url(
  Go skips /* */ and requires identStart; the css.js refs regex does neither. The "one rule, two languages" claim holds only because the divergent cases are left out. Add them and align css.js, or record the divergence in the plan's Revisions.
- **BR-5** [Minor] `concurrent-binding-order-undefined` Concurrent Put of the same URL rebinds in lock order, not call order; the proxy's Lookup can disagree with the last emitted asset record
  recorded.go:76-80. Document that the caller must serialize Put per URL, or bind by sequence (ARCH-ORDER). Concurrent identical bodies also both decode, so "decoded once" holds only for sequential calls.
- **BR-6** [Minor] `last-writer-wins-metadata` Recorded.types[hash] is overwritten by the last Put's MIME when identical bytes arrive under different types
- **BR-7** [Minor] `doc-contract-drift` store.go's Capture.Declared and Policy.MaxPixels docs promise the declared size widens the budget; Recorded ignores it (TL decision 3)

## Round 2 — 2026-10-12T02:58:37-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — Process-wide decode slot honouring ctx; TestRecordedDecodesOneAtATime fails (7 stores) with the slot reverted.
- BR-2 — addressed — Bind(Served) where Served = Store + TypeOf; enforced by the compiler; proxy/server tests bind Recorded.
- BR-3 — addressed — x/image is a direct require; go mod tidy -diff is empty at head.
- BR-4 — addressed — Corpus gains comment/string/name-boundary/image-set-comment/escape cases; css.test.mjs fails against the pre-fix css.js; Go passes the same corpus.
- BR-5 — addressed — Put documents single-worker caller and finish-order binding; concurrent identical bodies now decode once.
- BR-6 — addressed — First MIME wins per hash; TestRecordedFirstTypeWins fails when reverted. Residual record/TypeOf mismatch raised as new Minor.
- BR-7 — addressed — store.go Declared/MaxPixels docs now say the declared size is not yet used (TL decision 3).

### Raised

- **BR-8** [Minor] `last-writer-wins-metadata` Asset record MIME can disagree with the proxy's TypeOf for the same hash
  store() returns MIME c.MIME while types[hash] keeps the first label, so identical bytes put as font then text/html emit a text/html record served as font. Rule: per-hash metadata must be keyed by the dedup identity (hash, MIME); report types[hash] in the binding or key types by (hash, MIME), covering both types and Asset.MIME.
- **BR-9** [Minor] `doc-contract-drift` atlas/webcast.md says the cast binds the store at go-live; no production caller exists until #458 lands
  Rule: atlas text describing wiring owned by an unmerged dependency must mark it pending that issue. Sweep every #459 atlas claim about Bind or the recorder calling Store.Put, plus the Put doc comment's "the recorder calls it from one worker".

## Open findings

- **BR-8** [Minor] `last-writer-wins-metadata` Asset record MIME can disagree with the proxy's TypeOf for the same hash
- **BR-9** [Minor] `doc-contract-drift` atlas/webcast.md says the cast binds the store at go-live; no production caller exists until #458 lands
