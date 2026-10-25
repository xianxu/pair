# Endpoint capture allocation correction

2026-10-24, Darwin arm64, Apple M2 Max, Go1.27.3. This qualifies the allocator
named by the operator's startup profile, not later live CPU or an aged workload.
The staged pair:0 executable and live fleet were unchanged.

## Method and ownership

Baseline commit9205907e2 pins the new tests and synchronized DEC2026 benchmark
with unchanged production capture. Candidate changes only Endpoint.capture:
reuse the private published slices at equal geometry/length, otherwise allocate
exact current dimensions; overwrite all rows/cells, explicitly zero missing cells.
The existing mutex serializes access. Public Frame.Clone ownership, history,
parser behavior, repair and synchronized-output state transitions are unchanged.
One current-geometry buffer belongs to each endpoint, with no pool/spare cache.

Sequential commands, each count1, with no second full-suite check:

```
go test ./cmd/internal/terminal -run '^TestEndpointCapture' -count=1 -v
go test ./cmd/internal/terminal -run '^$' -bench '^BenchmarkEndpointCapture$' -benchtime=100x -count=1 -benchmem
go test ./cmd/internal/ptychild -run '^$' -bench '^BenchmarkChildIngestRedraw$' -benchtime=100x -count=1 -benchmem
go test -race ./cmd/internal/terminal -run 'TestEndpoint.*(Capture|Sync|Resize|Publication|EndInput)|Test.*History' -count=1
```

Allocation counts use AllocsPerRun(3) after warmup; each observation executes
64 or128 private captures, excluding consumer clones/parser/history. Public
Feed/Snapshot independently verifies stable backing through128 real synchronized
cycles. Capture benchmark measures only capture; Child benchmark includes real
production ingest, synchronization, public consumer publication and validation.
ASCII/mixed and64/4096-byte transport chunks use identical before/after payloads.

## Counted acceptance

| Geometry | Captures | Baseline allocated objects | Candidate |
|---|---:|---:|---:|
|80×24|64|128|0|
|80×24|128|256|0|
|160×24|64|128|0|
|160×24|128|256|0|

Baseline allocation tests failed at all four cases and public synchronized
capture replaced backing at cycle0. Candidate passes all cases and128 cycles.

| Capture replay | Baseline B/op | Candidate B/op | Baseline allocs/op | Candidate | Baseline ns/op | Candidate |
|---|---:|---:|---:|---:|---:|---:|
|80×24|221568|0|2|0|49362|33135|
|160×24|434562|0|2|0|112715|64931|

Capture-owned allocated bytes fall100%, exceeding the90% acceptance budget.

| Full ingest replay | Baseline B/op | Candidate B/op | Baseline allocs/op | Candidate | Baseline ns/op | Candidate |
|---|---:|---:|---:|---:|---:|---:|
|ASCII/chunk64|680652|237453|1769|1765|421787|304147|
|ASCII/chunk4096|680544|237404|1739|1735|454789|272438|
|Mixed/chunk64|688459|245324|2360|2356|808982|472542|
|Mixed/chunk4096|688344|245207|2315|2311|552585|423465|

The full replay removes about443KB/op (64–65%) and four allocations, corresponding
to two private captures. Remaining consumer/parser allocations are intentional
and separately owned. Short100-iteration wall timings are corroborating only;
machine load varies and these are not live Couch CPU savings or throughput bounds.

## Semantic and adversarial controls

Both complete affected Go packages also pass: terminal44.432s, ptychild1.045s.
Focused capture/sync/resize/publication/EOF controls PASS0.228s; focused race plus
history controls PASS2.541s. Added controls use the actual emulator and existing
stateful ttyio fake. They cover erased glyph/style/link and row metadata, retained
publication and mutated consumer independence, and held no-op/refused/successful
resize with grow/shrink/exact storage, epoch and cancellation assertions. Existing
same-feed hold, repeated-begin deadline, release/timeout and EOF controls are reused.

The absent-cell control deliberately shrinks only the real backend, creating an
adversarial partial screen rather than misrepresenting public Resize. An isolated
Go overlay removing only the new nil-cell assignment fails at cell1,0, which
retains B; normal candidate passes. The tracked source was never mutated for that
experiment. Existing repair/strict validation remains intact.

These measurements precede the full recorded check and final boundary reviews;
the issue Log and SDLC records carry their later results. After #879 landed
PR601, the exact branch and M1 provenance moved into canonical pair:9, landed
main was imported cleanly, and the deferred lesson was added as approved.
