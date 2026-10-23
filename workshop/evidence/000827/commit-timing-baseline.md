# #827 baseline: BeginPark commit latency on real ThreadStore (SSD)

Raw run: `commit-timing-baseline.txt` (fixture `TestParkCommitTimingRealStore`, head 228fdf67a,
go1.27.3 darwin/arm64, APFS `/System/Volumes/Data` via `~/Library/Caches`, empty-base env,
fan-out LifecycleParallelism=6, peak 6, current threshold 1s).

| slots | wall | p50 | p95 | max | errors | staged bytes |
| --- | --- | --- | --- | --- | --- | --- |
| 15 | 398 ms | 106 ms | 342 ms | 342 ms | 0 | 55,908 |
| 30 | 920 ms | 122 ms | 367 ms | 709 ms | 0 | 111,870 |

Machine state: load averages 64.80 / 30.04 / 23.37 at start (fleet co-tenancy: other
slots' suites; `ps` is sandbox-denied, so co-tenants are not itemized). This is a
*loaded* run, not a healthy-idle one; it is still not the operator's 30-slot incident
load, so the incident's causal chain remains unproven.

Reading (observed, not an inferred tail bound): p50 is flat with N, max roughly doubles
with N (writers serialize on the shared publication file). At 30 slots under load ~65
the worst commit already used 71% of the 1s threshold, so a 1s overrun under heavier
load is plausible. 10s gives ~14x headroom over the observed max; 5s ~7x.
Decision per approved plan: confirm the provisional 10s threshold. Owned cache dir removed
(verified empty `ls ~/Library/Caches | grep pair-827`).
