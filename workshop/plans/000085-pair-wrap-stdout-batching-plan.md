# Pair-wrap Stdout Batching Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Reduce zellij redraw pressure from Codex by batching only filtered `pair-wrap` stdout delivery at a 100ms cadence.

**Architecture:** Keep the raw PTY processing path immediate, and place a small batching seam only between `stdoutChunk(data)` and the actual stdout writer. The core batch state is pure and unit-tested; `masterPump` owns the IO timer and EOF flush. This preserves scrollback offsets and detection responsiveness while smoothing visible writes (`ARCH-PURE`, `ARCH-PURPOSE`).

**Tech Stack:** Go, `cmd/pair-wrap`, standard `testing`, existing wrap-event tracing.

---

## Core Concepts

### Pure Entities

| Name | Lives in | Status |
|------|----------|--------|
| `stdoutBatcher` | `cmd/pair-wrap/main.go` | new |

`stdoutBatcher` accumulates already-filtered stdout bytes and exposes deterministic operations: append bytes, report pending byte/chunk counts, flush into a returned byte slice, and reset. It has no clock or writer dependency.

- **Relationships:** One `masterPump` owns one batcher for the lifetime of a wrapped process.
- **DRY rationale:** Reuses the existing `stdoutChunk` filter; batching starts after filtering so the DEC marker stripping logic is not duplicated.
- **Future extensions:** If #82 proves the experiment useful, the flush interval can become an env/config knob without changing raw capture.

### Integration Points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `stdoutPump` | `cmd/pair-wrap/main.go` | new | `io.Writer` stdout delivery |
| `masterPump stdout flush loop` | `cmd/pair-wrap/main.go` | modified | PTY read channel, `time.Ticker`, `os.Stdout.Write` |

`stdoutPump` receives "queue filtered bytes", "flush tick", and "EOF" events. It owns one pure `stdoutBatcher`, writes flushed bytes through an injected `io.Writer`, and returns trace records for the caller to emit.

- **Injected into:** `proxy.handleChunk` and `masterPump` through the proxy-owned `stdoutPump`.
- **Future extensions:** The flush interval can be made configurable by changing the ticker setup without changing pump semantics.

`masterPump` creates the production `stdoutPump`, appends filtered stdout to it from `handleChunk`, and drives pump flushes on a 100ms ticker plus EOF. `handleChunk` keeps a defensive fallback for direct unit-test use, but production stdout IO stays on the `masterPump`-owned pump.

- **Injected into:** `handleChunk` receives a queueing surface instead of writing stdout directly, keeping the decision testable.
- **Future extensions:** Add a trace-only experiment switch if dogfooding needs A/B comparison.

## Chunk 1: TDD Stdout Batching

### Task 1: Pure Batcher Tests

**Files:**
- Modify: `cmd/pair-wrap/main.go`
- Test: `cmd/pair-wrap/stdout_batch_test.go`

- [x] **Step 1: Write the failing pure batcher test**

```go
func TestStdoutBatcherAccumulatesAndFlushes(t *testing.T) {
	var b stdoutBatcher
	b.append([]byte("ab"))
	b.append([]byte("cd"))

	got, chunks := b.flush()
	if string(got) != "abcd" || chunks != 2 {
		t.Fatalf("flush = %q, %d chunks; want abcd, 2", got, chunks)
	}
	if b.pendingBytes() != 0 {
		t.Fatalf("pendingBytes after flush = %d, want 0", b.pendingBytes())
	}
}
```

- [x] **Step 2: Run the test to verify RED**

Run: `go test ./cmd/pair-wrap -run TestStdoutBatcherAccumulatesAndFlushes -count=1`

Expected: FAIL because `stdoutBatcher` does not exist.

- [x] **Step 3: Implement the minimal pure batcher**

Add a small unexported struct with `append`, `flush`, and `pendingBytes`.

- [x] **Step 4: Run the test to verify GREEN**

Run: `go test ./cmd/pair-wrap -run TestStdoutBatcherAccumulatesAndFlushes -count=1`

Expected: PASS.

### Task 2: Handle Chunk Routes Stdout Into Batcher

**Files:**
- Modify: `cmd/pair-wrap/main.go`
- Test: `cmd/pair-wrap/wrap_events_test.go`

- [x] **Step 1: Write failing tests for filtered stdout queueing and immediate scrollback**

Extend the existing handle-chunk test so `a\x1b[?2026hb` queues `ab` in the stdout pump, while `scrollback.raw` immediately contains the original bytes including the stripped marker.

Expected trace fields for this step:

- `stdout-queue.stdout_len`: filtered bytes queued in this chunk.
- `stdout-queue.stdout_sha256_12`: hash of filtered bytes queued.
- `stdout-queue.queued_chunks`: current pending chunk count after queueing.
- `stdout-queue.queued_bytes`: current pending byte count after queueing.
- No `stdout-batch-flush` event is emitted until a tick or EOF flush.

- [x] **Step 2: Run the focused test to verify RED**

Run: `go test ./cmd/pair-wrap -run TestHandleChunkTracesMasterStdoutAndScrollback -count=1`

Expected: FAIL because `handleChunk` still writes stdout directly and has no batcher to inspect.

- [x] **Step 3: Change `handleChunk` to append filtered stdout to the pump**

Replace the old direct `stdout-write` event on the hot path with `stdout-queue`. Do not move scrollback, span extraction, or detection behind the batch.

- [x] **Step 4: Run the focused test to verify GREEN**

Run: `go test ./cmd/pair-wrap -run 'TestHandleChunkTracesMasterStdoutAndScrollback|TestStdoutBatcher' -count=1`

Expected: PASS.

## Chunk 2: Timer Flush And Verification

### Task 3: Pump Cadence And EOF Tests

**Files:**
- Modify: `cmd/pair-wrap/main.go`
- Test: `cmd/pair-wrap/stdout_batch_test.go`

- [x] **Step 1: Write failing pump tests for cadence and EOF**

Add tests around `stdoutPump` with an injected `bytes.Buffer` writer:

```go
func TestStdoutPumpFlushesOnlyOnTick(t *testing.T) {
	var out bytes.Buffer
	pump := newStdoutPump(&out)

	pump.queue([]byte("a"))
	pump.queue([]byte("b"))
	if out.String() != "" {
		t.Fatalf("stdout written before tick: %q", out.String())
	}

	rec := pump.flush("tick")
	if out.String() != "ab" {
		t.Fatalf("stdout after first tick = %q, want ab", out.String())
	}
	if rec.Chunks != 2 || rec.Bytes != 2 || rec.Reason != "tick" {
		t.Fatalf("flush record = %#v, want 2 chunks/2 bytes/tick", rec)
	}

	pump.queue([]byte("c"))
	if out.String() != "ab" {
		t.Fatalf("stdout wrote before second tick: %q", out.String())
	}
	pump.flush("tick")
	if out.String() != "abc" {
		t.Fatalf("stdout after second tick = %q, want abc", out.String())
	}
}

func TestStdoutPumpFlushesOnEOF(t *testing.T) {
	var out bytes.Buffer
	pump := newStdoutPump(&out)
	pump.queue([]byte("pending"))

	rec := pump.flush("eof")
	if out.String() != "pending" {
		t.Fatalf("stdout after EOF = %q, want pending", out.String())
	}
	if rec.Reason != "eof" {
		t.Fatalf("reason = %q, want eof", rec.Reason)
	}
}
```

- [x] **Step 2: Run the test to verify RED**

Run: `go test ./cmd/pair-wrap -run 'TestStdoutPumpFlushesOnlyOnTick|TestStdoutPumpFlushesOnEOF' -count=1`

Expected: FAIL because no stdout pump exists.

- [x] **Step 3: Implement `stdoutPump`**

The helper writes flushed bytes to the provided writer, reports `stdout-batch-flush` fields, and becomes a no-op when the batch is empty.

Expected trace fields for each actual flush:

- `stdout-batch-flush.reason`: `tick` or `eof`.
- `stdout-batch-flush.write_len`: bytes written to stdout.
- `stdout-batch-flush.stdout_len`: filtered bytes in the batch.
- `stdout-batch-flush.stdout_sha256_12`: hash of filtered bytes in the batch.
- `stdout-batch-flush.chunks`: number of queued chunks in the batch.
- `stdout-batch-flush.error`: write error string, if any.

- [x] **Step 4: Run focused tests to verify GREEN**

Run: `go test ./cmd/pair-wrap -run 'TestStdoutBatcher|TestStdoutPump|TestHandleChunkTracesMasterStdoutAndScrollback' -count=1`

Expected: PASS.

### Task 4: Wire `masterPump`

**Files:**
- Modify: `cmd/pair-wrap/main.go`

- [x] **Step 1: Add the 100ms ticker**

In `masterPump`, create `stdoutFlushTick := time.NewTicker(100 * time.Millisecond)` and defer `Stop`.

- [x] **Step 2: Flush on tick and EOF**

Drive `stdoutPump.flush("tick")` in the ticker case. Before every `masterPump` return caused by closed read channel, EOF, EIO, or read failure, drive `stdoutPump.flush("eof")`.

- [x] **Step 3: Preserve existing timer behavior**

Keep idle and capture timers unchanged; batching must not delay capture finalization or idle detection.

### Task 5: Docs And Verification

**Files:**
- Modify: `atlas/architecture.md`
- Modify: `workshop/issues/000085-pair-wrap-stdout-batching.md`

- [x] **Step 1: Update atlas**

Record that pair-wrap now batches visible stdout delivery while keeping raw scrollback and detection immediate.

- [x] **Step 2: Run verification**

Run:

```sh
gofmt -w cmd/pair-wrap/main.go cmd/pair-wrap/stdout_batch_test.go
go test ./cmd/pair-wrap
go test ./...
make build
git diff --check
```

- [x] **Step 3: Log dogfood instructions**

In #85 log, record that live Pair dogfood requires `make install` before restarting Pair because zellij runs the installed `pair-wrap`.

## Revisions

- 2026-06-29T15:57:00-07:00 — Boundary review found the original Core Concepts table misclassified `stdoutPump` as PURE even though `flush` writes through an injected `io.Writer`. Reclassified `stdoutPump` as an integration point wrapping the pure `stdoutBatcher`; `stdoutBatcher` remains the pure core and `stdoutPump` / `masterPump` form the thin stdout IO shell (`ARCH-PURE`).
- 2026-06-29T16:04:00-07:00 — Second boundary review requested integration coverage for `masterPump` itself. Added pipe-backed tests proving `masterPump` flushes queued stdout on the ticker and on EOF, using a short injected interval instead of sleeping on the production 100ms cadence.
