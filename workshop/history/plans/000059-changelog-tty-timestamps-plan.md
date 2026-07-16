# Change-log TTY Timestamps Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Capture wall-clock timestamps in the scrollback sidecar so the change log dates entries by **real change-time** (`## YYYY-MM-DD` day headers), reversing #58's date removal — but now fed honest dates instead of distill-time.

**Architecture:** Three additive seams over existing machinery. (1) `pair-wrap` drops minute-debounced `time` events into the existing `events.jsonl` sidecar (offset-anchored, raw untouched). (2) `pair-scrollback-render`, behind `--with-timestamps`, snapshots the emulator's scrollback length at each `time`-event offset and interleaves `⟦pair:ts DATE⟧` marker lines where the date changes — the offset→rendered-line mapping the render alone can compute. (3) `pair-changelog` parses those markers into per-line dates, splits the slice into per-day segments, and assembles each day's new entries under its real `## DATE` header. No markers present → byte-for-byte the #58 header-free behavior, so the feature is purely additive.

**Tech Stack:** Go (`cmd/pair-wrap`, `cmd/pair-scrollback-render`, `cmd/pair-changelog`), the `charmbracelet/x/vt` emulator, sh orchestrator (`bin/pair-changelog-open`). Tests: Go `testing` + the existing process-level `fakeClaude` + headless smoke.

---

## Core concepts

The feature threads one fact — "what wall-clock time produced this scrollback region" — from capture (pair-wrap) through projection (render) to consumption (distiller). The marker line is the wire format between render and distiller; the events sidecar is the wire format between pair-wrap and render. Both are offset-anchored so the byte stream stays the single source of position.

### Pure entities (the conceptual core)

| Name | Lives in | Status |
|------|----------|--------|
| `dueForTimeEvent` | `cmd/pair-wrap/main.go` | new |
| `dateOf` | `cmd/pair-scrollback-render/main.go` | new |
| `interleaveDateMarkers` | `cmd/pair-scrollback-render/main.go` | new |
| `scrollbackEvent` (was `resizeEvent`) | `cmd/pair-scrollback-render/main.go` | modified |
| `tsMarkerRe` | `cmd/pair-changelog/distill.go` | new |
| `parseDatedLines` | `cmd/pair-changelog/distill.go` | new |
| `splitByDate` | `cmd/pair-changelog/distill.go` | new |
| `assemble` | `cmd/pair-changelog/distill.go` | modified |
| `lastHeaderDate` + `headerDateRe` | `cmd/pair-changelog/distill.go` | new (re-add) |
| `stripDateHeaders` + `dateHeaderRe`/`multiBlankRe` | `cmd/pair-changelog/distill.go` | deleted |

- **`dueForTimeEvent(last, now time.Time) bool`** — the minute-debounce decision: `last.IsZero() || now.Sub(last) >= time.Minute`. First output of a session emits immediately (zero `last`); thereafter ≤1/minute.
  - **DRY rationale:** the one place the cadence rule lives; the IO method `maybeLogTime` just calls it.
  - **Future extensions:** a configurable interval becomes a param instead of the `time.Minute` literal.

- **`dateOf(ts string) string`** — RFC3339 → `YYYY-MM-DD` (the `ts[:10]` day, parsed defensively; `""` on a malformed ts so a corrupt event degrades to undated, never panics).

- **`interleaveDateMarkers(lines []string, marks []dateMark) []string`** — given the rendered output lines and `marks` (`{scrollbackLineIndex int, date string}` snapshots), insert a `⟦pair:ts DATE⟧` line immediately before the first line of each new date-run. Lines before the first mark stay undated (no marker). Pure; the heart of the render feature and directly unit-testable without an emulator.
  - **Relationships:** consumes the marks built by the IO snapshot pass; 1 marker emitted per date *change*, not per mark.
  - **DRY rationale:** isolates the "where does a date boundary fall in the line list" logic from the emulator-feeding IO.

- **`scrollbackEvent`** — `resizeEvent` renamed + widened with `Ts string` (`json:"ts,omitempty"`); `parseEvents` stops hard-filtering to `resize` and returns **all** known event types; `initialSize`/`feedSegments` filter `Type=="resize"` at their use sites.
  - **DRY rationale:** one event parser, one struct — not a parallel `timeEvent` type + second parse pass.
  - **Future extensions:** any later sidecar event (cursor marks, annotations) reuses the same struct + parser.

- **`tsMarkerRe`** — `^⟦pair:ts (\d{4}-\d{2}-\d{2})⟧$`. Recognizes a render-emitted day marker. Distinctive sentinel (guillemets + `pair:ts` prefix) so it can't collide with agent output.

- **`parseDatedLines(lines []string) (content []string, dates []string)`** — strips marker lines, returns the content lines plus a parallel `dates` slice (`dates[i]` = the day of the most recent preceding marker, `""` before the first). The single point where markers leave the pipeline — everything downstream (anchor, turn-count, locate, slice, model input) sees clean content.
  - **Relationships:** `len(content) == len(dates)`. Run **before** `trimLiveTail`; the footer trim then slices `dates` to `len(trimmedContent)` (trim only removes from the tail).
  - **DRY rationale:** markers are consumed once, here — no marker-awareness leaks into locate/assemble/anchor.

- **`splitByDate(content, dates []string) []dateSegment`** — groups consecutive same-date runs into `{date string, lines []string}` segments (oldest→newest). A multi-day slice → multiple segments → multiple `## DATE` sections. Pure.

- **`assemble(frozenPrefix, ekPrime, newEntries, date, lastDate string)`** — **re-add the `date`/`lastDate` params #58 removed**: insert `## <date>` before `newEntries` iff `newEntries != "" && date != "" && date != lastDate`. `date==""` → no header (the #58 header-free behavior, preserved for undated content).
  - **DRY rationale:** restores the single date-header authority; the date now comes from a marker, not `--today`.

- **`lastHeaderDate` + `headerDateRe`** — re-added (`(?m)^## (\d{4}-\d{2}-\d{2})\s*$`): the last `## DATE` in the running log, for `assemble`'s rollover check.

- **`stripDateHeaders` (+ `dateHeaderRe`, `multiBlankRe`)** — **deleted.** #58 added it to migrate distill-time headers off; now real headers are wanted, and stripping the prior log's headers on read would zero `lastHeaderDate` → duplicate headers every press. Legacy header-free logs have nothing to strip, so removal is safe.

### Integration points (where pure meets the world)

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `maybeLogTime` | `cmd/pair-wrap/main.go` | new | clock + `events.jsonl` FD |
| time-snapshot pass | `cmd/pair-scrollback-render/main.go` | modified | `vt.Emulator` scrollback |
| `--with-timestamps` + segment loop | `cmd/pair-changelog/main.go` | modified | flag + distill orchestration |
| render-step flag | `bin/pair-changelog-open` | modified | sh orchestrator |

- **`maybeLogTime()`** — proxy method: if `dueForTimeEvent(p.lastTimeEvent, p.now())`, call `logScrollbackEvent("time", {"ts": p.now().Format(time.RFC3339)})` and set `p.lastTimeEvent = p.now()`. Called from the scrollback tee at `main.go:1835` (only when output was actually written → "new output" requirement).
  - **Injected into:** uses `p.now` (new field, defaults to `time.Now`) so tests drive a fake clock; reuses the existing generic `logScrollbackEvent`.
  - **Future extensions:** richer event payloads (e.g. an idle marker) reuse the same method shape.

- **time-snapshot pass (render)** — extend the segment feeder so it also stops at `time`-event offsets and records `dateMark{scrollbackLineIndex: em.Scrollback().Len(), date: dateOf(e.Ts)}`. Only meaningful uncapped (the changelog path; the viewer is capped + never asks for markers). Feeds `interleaveDateMarkers`.
  - **Injected into:** produces the `marks` consumed by the pure `interleaveDateMarkers`.

- **`--with-timestamps` + segment loop (distiller)** — `main.go`: `parseDatedLines` → `trimLiveTail` (slice dates too) → existing anchor/turn-count/locate on content → `splitByDate(slice, sliceDates)` → for each segment, `chunkLines` + `distillStep(newLog, chunk, agent, model, segment.date)` (new `date` param threaded to `assemble`). `--today` and its plumbing stay deleted.

- **render-step flag** — `bin/pair-changelog-open`: add `--with-timestamps` to the `$PCL_DISTILL`'s upstream render invocation (`$PCL_RENDER ... "$PCL_CLEANED"`). The Alt+/ scrollback path (`bin/pair-scrollback-open`) does NOT pass it → scrollback stays marker-free.

---

## Chunk 1: capture + render

### Task 1: `dueForTimeEvent` (pure debounce)

**Files:**
- Modify: `cmd/pair-wrap/main.go`
- Test: `cmd/pair-wrap/main_test.go` (or a new `time_test.go` if main_test is crowded)

- [ ] **Step 1: Write the failing test**

```go
func TestDueForTimeEvent(t *testing.T) {
	base := time.Date(2026, 6, 14, 10, 0, 0, 0, time.UTC)
	if !dueForTimeEvent(time.Time{}, base) {
		t.Fatal("first event (zero last) should be due")
	}
	if dueForTimeEvent(base, base.Add(59*time.Second)) {
		t.Fatal("59s < 1min should not be due")
	}
	if !dueForTimeEvent(base, base.Add(60*time.Second)) {
		t.Fatal("60s should be due")
	}
}
```

- [ ] **Step 2: Run → fail** — `go test ./cmd/pair-wrap/ -run TestDueForTimeEvent` → undefined.
- [ ] **Step 3: Implement**

```go
// dueForTimeEvent reports whether a new scrollback time event should be logged:
// the first one always (zero last), then at most one per minute of activity.
func dueForTimeEvent(last, now time.Time) bool {
	return last.IsZero() || now.Sub(last) >= time.Minute
}
```

- [ ] **Step 4: Run → pass.**
- [ ] **Step 5: Commit** — `#59: pair-wrap minute-debounce decision for time events`

### Task 2: `maybeLogTime` wiring (clock seam + emit)

**Files:**
- Modify: `cmd/pair-wrap/main.go` (proxy struct: add `now func() time.Time`, `lastTimeEvent time.Time`; new method; call at the tee)
- Test: `cmd/pair-wrap/main_test.go`

- [ ] **Step 1: Write the failing test** — construct a proxy with a temp `eventsFD` and a fake clock; call `maybeLogTime` across a clock advance; assert event count + that each line is `{"type":"time","ts":...}`.

```go
func TestMaybeLogTimeDebounced(t *testing.T) {
	dir := t.TempDir()
	f, _ := os.Create(filepath.Join(dir, "e.jsonl"))
	defer f.Close()
	clock := time.Date(2026, 6, 14, 10, 0, 0, 0, time.UTC)
	p := &proxy{eventsFD: f, now: func() time.Time { return clock }}
	p.maybeLogTime()                 // first → emit
	p.maybeLogTime()                 // same instant → skip
	clock = clock.Add(61 * time.Second)
	p.maybeLogTime()                 // >1min → emit
	f.Close()
	data, _ := os.ReadFile(filepath.Join(dir, "e.jsonl"))
	n := strings.Count(string(data), `"type":"time"`)
	if n != 2 {
		t.Fatalf("got %d time events, want 2:\n%s", n, data)
	}
}
```

- [ ] **Step 2: Run → fail.**
- [ ] **Step 3: Implement** — add struct fields (`now` defaulted to `time.Now` wherever the proxy is constructed in `run`/`main`), the method:

```go
// maybeLogTime drops a minute-debounced wall-clock event into the scrollback
// sidecar so the change-log render can date scrollback regions (#59). Offset is
// the current scrollbackBytes (set by logScrollbackEvent); the raw is untouched.
func (p *proxy) maybeLogTime() {
	now := p.now()
	if !dueForTimeEvent(p.lastTimeEvent, now) {
		return
	}
	p.logScrollbackEvent("time", map[string]any{"ts": now.Format(time.RFC3339)})
	p.lastTimeEvent = now
}
```

- [ ] **Step 4: Wire the call** — at `main.go:1835`, inside the `if p.scrollbackFD != nil` block, after `p.scrollbackBytes += int64(wn)` (only on a successful write), call `p.maybeLogTime()`. Ensure every `&proxy{...}` construction sets `now: time.Now` (grep `&proxy{`); default-nil-guard inside `maybeLogTime` (`if p.now == nil { p.now = time.Now }`) as a belt-and-suspenders.
- [ ] **Step 5: Run → pass; full `go test ./cmd/pair-wrap/`.**
- [ ] **Step 6: Commit** — `#59: pair-wrap emits minute-debounced time events to the sidecar`

### Task 3: render parses all events + `dateOf`

**Files:**
- Modify: `cmd/pair-scrollback-render/main.go` (rename `resizeEvent`→`scrollbackEvent`, `+Ts`, `parseEvents` returns all; `initialSize`/`feedSegments` filter `resize`)
- Test: `cmd/pair-scrollback-render/*_test.go`

- [ ] **Step 1: Failing tests** — `parseEvents` over a sidecar mixing `resize` + `time` returns both; `initialSize` still picks the first resize; `dateOf("2026-06-14T10:30:00Z") == "2026-06-14"`, `dateOf("garbage") == ""`.
- [ ] **Step 2: Run → fail.**
- [ ] **Step 3: Implement** — widen the struct, drop the `Type=="resize"` filter in `parseEvents` (keep skipping unknown types if desired), add `resize`-filtering where resize is consumed, add `dateOf`.
- [ ] **Step 4: Run → pass (existing render tests still green — resize behavior unchanged).**
- [ ] **Step 5: Commit** — `#59: scrollback-render parses time events; dateOf helper`

### Task 4: `interleaveDateMarkers` (pure) + snapshot pass + `--with-timestamps`

**Files:**
- Modify: `cmd/pair-scrollback-render/main.go`
- Test: `cmd/pair-scrollback-render/*_test.go`

- [ ] **Step 1: Failing test for `interleaveDateMarkers`**

```go
func TestInterleaveDateMarkers(t *testing.T) {
	lines := []string{"a", "b", "c", "d"}        // 4 scrollback lines
	marks := []dateMark{{0, "2026-06-13"}, {2, "2026-06-14"}}
	got := interleaveDateMarkers(lines, marks)
	want := []string{
		"⟦pair:ts 2026-06-13⟧", "a", "b",
		"⟦pair:ts 2026-06-14⟧", "c", "d",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q", got)
	}
	// undated leading region (first mark at line 1) → those lines carry no marker
	got2 := interleaveDateMarkers(lines, []dateMark{{1, "2026-06-14"}})
	if got2[0] != "a" || got2[1] != "⟦pair:ts 2026-06-14⟧" {
		t.Fatalf("leading undated region mishandled: %q", got2)
	}
}
```

- [ ] **Step 2: Run → fail.**
- [ ] **Step 3: Implement** `dateMark` struct + `interleaveDateMarkers` (walk lines; before the line at each mark's index, if the date differs from the running date, emit the marker).
- [ ] **Step 4: Snapshot pass + flag (IO).** Add `--with-timestamps` (default false). When set + `maxLines<=0`: during the segment feed, also stop at `time`-event offsets and append `dateMark{em.Scrollback().Len(), dateOf(e.Ts)}`; after building the scrollback-line slice (before the visible buffer), run `interleaveDateMarkers`. Visible-buffer lines (newest) inherit the last mark's date — append a trailing mark at `len(scrollbackLines)` with the last date so they're covered. Markers only emitted in `--with-timestamps` mode.
- [ ] **Step 5: Integration test** — feed a raw fixture + a sidecar with two `time` events at offsets that land in scrollback; render with `--with-timestamps`; assert `⟦pair:ts ⟧` lines appear at the right boundaries; render WITHOUT the flag → no markers.
- [ ] **Step 6: Run → pass; full `go test ./cmd/pair-scrollback-render/`.**
- [ ] **Step 7: Commit** — `#59: scrollback-render --with-timestamps interleaves day markers`

## Chunk 2: distiller consumes markers

### Task 5: `parseDatedLines` + `tsMarkerRe` (pure)

**Files:**
- Modify: `cmd/pair-changelog/distill.go`
- Test: `cmd/pair-changelog/distill_test.go`

- [ ] **Step 1: Failing test** — markers stripped, parallel `dates` carry-forward; lines before the first marker → `""`.

```go
func TestParseDatedLines(t *testing.T) {
	in := []string{"old", "⟦pair:ts 2026-06-13⟧", "x", "y", "⟦pair:ts 2026-06-14⟧", "z"}
	content, dates := parseDatedLines(in)
	if !reflect.DeepEqual(content, []string{"old", "x", "y", "z"}) {
		t.Fatalf("content %q", content)
	}
	if !reflect.DeepEqual(dates, []string{"", "2026-06-13", "2026-06-13", "2026-06-14"}) {
		t.Fatalf("dates %q", dates)
	}
}
```

- [ ] **Step 2-4: red → implement → green.**
- [ ] **Step 5: Commit** — `#59: distiller parses ts markers into per-line dates`

### Task 6: `splitByDate` (pure)

**Files:** `cmd/pair-changelog/distill.go` + test.

- [ ] **Step 1: Failing test** — consecutive same-date runs grouped; leading `""` is its own undated segment.

```go
func TestSplitByDate(t *testing.T) {
	content := []string{"a", "b", "c", "d"}
	dates := []string{"", "", "2026-06-14", "2026-06-14"}
	segs := splitByDate(content, dates)
	// segs == [{"", [a b]}, {"2026-06-14", [c d]}]
}
```

- [ ] **Step 2-4: red → implement → green.**
- [ ] **Step 5: Commit** — `#59: distiller splits a slice into per-day segments`

### Task 7: re-add `assemble` date param; delete `stripDateHeaders`

**Files:** `cmd/pair-changelog/distill.go`, `distill_test.go`

- [ ] **Step 1: Update tests** — `assemble(frozen, ek, new, date, lastDate)`: `date!=lastDate` inserts `## date`; `date==""` → no header; restore a rollover test. Re-add `lastHeaderDate` test. **Delete** `TestStripDateHeaders` + `TestStripsLegacyDateHeaders`.
- [ ] **Step 2: Run → fail (signature mismatch).**
- [ ] **Step 3: Implement** — re-add the date branch to `assemble` + `lastHeaderDate`/`headerDateRe`; delete `stripDateHeaders`/`dateHeaderRe`/`multiBlankRe`.
- [ ] **Step 4: Run → pass.**
- [ ] **Step 5: Commit** — `#59: restore assemble date headers (fed real dates); drop stripDateHeaders`

### Task 8: thread dates through `main.go` (the segment loop)

**Files:** `cmd/pair-changelog/main.go`, `main_test.go`

- [ ] **Step 1: Update `distillStep`** to take a `date string` param, passed to `assemble(frozen, ekPrime, newEntries, date, lastHeaderDate(priorLog))`. Remove the call site's reliance on the old signature.
- [ ] **Step 2: Rework `main`'s flow** — after reading cleaned: `content, dates := parseDatedLines(splitLines(...))`; `content = trimLiveTail(content, agent)`; `dates = dates[:len(content)]`; run the existing `stripDateHeaders`-free prior-log read, anchor/turn-count/locate/no-op on `content`; `slice := content[sliceStart:]`, `sliceDates := dates[sliceStart:]`; `for seg := range splitByDate(slice, sliceDates) { for chunk := range chunkLines(seg.lines, maxSliceLines) { newLog = distillStep(newLog, join(chunk), agent, model, seg.date) ... } }`. Keep the anchor-advance + writeReady changes from #58.
- [ ] **Step 3: Integration tests (the done-when)** — using the `fakeClaude` harness:
  - **two-day:** a cleaned with injected `⟦pair:ts D1⟧ … ⟦pair:ts D2⟧ …` → assert the log has `## D1` and `## D2` sections over the right entries.
  - **no-marker (regression):** a cleaned with NO markers → header-free log (byte-identical to current behavior). This guards the "purely additive" property.
  - **incremental same-day:** second press, new content under today's marker → one new `## today` section, frozen prefix preserved.
- [ ] **Step 4: Run → pass; full `go test ./cmd/pair-changelog/`.**
- [ ] **Step 5: Commit** — `#59: distiller dates entries per-day from ts markers`

### Task 9: orchestrator flag

**Files:** `bin/pair-changelog-open`

- [ ] **Step 1:** add `--with-timestamps` to the `$PCL_RENDER` invocation inside `INNER`.
- [ ] **Step 2: End-to-end marker-survival test (plan-quality finding 2).** A Go (or shell) test that drives the **real render→cleaned→distill seam**: build a raw + events fixture with two `time` events, run `pair-scrollback-render --with-timestamps` to produce a cleaned file, assert the `⟦pair:ts …⟧` markers landed on their own lines (so `tsMarkerRe`'s `^…$` anchor will match), then run `pair-changelog` over that cleaned and assert the log carries the two `## DATE` headers. This guards the seam Task 8 skips (Task 8 injects markers into a cleaned fixture directly).
- [ ] **Step 3: Run** `make test-changelog` smoke (assert no `⟦pair:ts` leaks into a viewer-rendered scrollback via the existing scrollback test; the changelog log may now carry `## DATE`).
- [ ] **Step 4: Commit** — `#59: changelog-open renders with --with-timestamps + e2e marker-survival test`

## Chunk 3: docs + verify

### Task 10: atlas + live verify

**Files:** `atlas/architecture.md`

- [ ] **Step 1:** update the Change-log section — date headers are back, sourced from `time` events (not `--today`); the render's `--with-timestamps` marker mechanism; the scrollback/events-sidecar gains a `time` event type. Cite ARCH-DRY (reused `logScrollbackEvent` + single event parser) and ARCH-PURE (pure `interleaveDateMarkers`/`parseDatedLines`/`splitByDate` vs the thin emulator/clock seams).
- [ ] **Step 2: Live verify** — rebuild binaries; against the live `changelog-pair-claude.cleaned` (after a real session with the new pair-wrap running), confirm `Alt+l` produces real `## DATE` headers and the Alt+/ scrollback shows no markers.
- [ ] **Step 3: Commit** — `#59: atlas — change-log dates sourced from captured time events`

## Done when

(mirrors the issue) — minute-debounced `time` events in `events.jsonl`; scrollback viewer visually unchanged; change log dates entries by change-time (two-day integration test); no-marker stream → header-free (regression test); full go + lua + smoke green; live `Alt+l` shows real dates.

## Notes / risks

- **Offset→line mapping correctness** rests on deterministic replay: the render's emulator at byte offset N has the same `Scrollback().Len()` pair-wrap saw when it stamped the `time` event at N. Holds because the render replays the identical raw and the changelog path is uncapped (`--max-lines 0` → no eviction to perturb indices). The viewer path is capped but never requests markers. `feedSegments` becomes the **single offset-ordered walk over all events** — act on `resize`, snapshot on `time` — not a parallel feeder (events.jsonl is naturally offset-sorted; both types share the monotonic `scrollbackBytes`).
- **Visible-buffer lag (accepted):** the snapshot reads `Scrollback().Len()` *during* the feed, so the ~`Height()` rows still on screen at that instant haven't scrolled into scrollback yet. A day-boundary marker at index `L` can therefore precede up to one screenful of the *previous* day's tail (written before the stamp, not yet evicted), tagging it with the new date. Negligible at day granularity — accepted, not fixed.
- **Marker collision:** `⟦pair:ts YYYY-MM-DD⟧` is a deliberately unlikely sentinel; `tsMarkerRe` anchors the whole line. If an agent ever printed it verbatim it'd be mis-stripped — acceptable, vanishingly unlikely.
- **ARCH-DRY:** reuse `logScrollbackEvent` (don't add a parallel writer); one `scrollbackEvent` struct + parser (don't add a `timeEvent` type). **ARCH-PURE:** `dueForTimeEvent`, `dateOf`, `interleaveDateMarkers`, `parseDatedLines`, `splitByDate`, `assemble` are pure + unit-tested; the clock (`p.now`), the emulator snapshot, and the flag plumbing are the thin seams.
- **Single review boundary:** capture+render produce no user-visible change alone (markers only matter to the distiller), so this ships as one `sdlc close` — plain checkboxes, no `Mx` split (AGENTS.md §3).
