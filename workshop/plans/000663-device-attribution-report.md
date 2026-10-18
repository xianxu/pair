# pair#663: approved observer comparison

## Result

Both arms completed all five invocations of TestSlotPreferencesIndependentAcrossRestartResumeAndFresh and exited 0. The full observer arm recorded 3.40 times the device writes of endpoints only. **This does not identify observer cost:** device traffic was already higher during quiet-before and declined strongly across the ordered experiment. Attribution remains open. No repeat, host census or concurrency experiment was run.

| Interval | Device bytes | Seconds | Decimal MB/s |
|---|---:|---:|---:|
| Quiet-before, no owned test running | 1,678,868,480 | 30.001173 | 55.960095 |
| Full observer, five completed invocations | 1,016,655,872 | 22.891279 | 44.412367 |
| Endpoints only, five completed invocations | 298,835,968 | 24.377118 | 12.258872 |
| Quiet-after, no owned test running | 253,427,712 | 30.025743 | 8.440348 |

The 717,819,904-byte arm difference (70.606% lower for endpoints only) is an observation, not a causal saving. Quiet-before alone wrote more than either arm. The quiet windows still sampled IORegistry at about 1 Hz; they are not a zero-observer control. Other host writers and delayed writes from preceding work remain possible. Do not subtract quiet slopes to manufacture a process attribution.

## Method and validity

- One ordered pair: full observation first, endpoints only second, five fixed invocations each. Each arm had a 60-second limit with an independent deadline watchdog and bounded shutdown. No deadline fired.
- The test binary was compiled once outside the window. Assertions, Git operations, durability calls and normal shared caches were unchanged. Each arm used a fresh private TMPDIR. The full observer reused the retained #658 device/ps/native/FD sampling, with a 0.5-second sleep between observations and five-second subprocess limits. Endpoints only had no in-loop observation.
- Both normal machine suite tokens were acquired at 2026-10-18 03:39:14 PDT after 1,268.782 seconds in the corrected queue. They remained held through the 30-second quiet-before, both arms and 30-second quiet-after. The supervisor then exited 0, releasing admission. No other full suite could run during this window; other host activity and filesystem writeback were not excluded.
- Full arm ended at 03:40:07 PDT; endpoints arm ended at 03:40:31 PDT. Both direct test processes were reaped, and both process groups were absent on the post-run check. Group termination was requested before ending counters; this is not proof of exact descendant writeback completion at the boundary.
- Stable physical driver: disk0, APPLE SSD AP1024Z Media, registry 4294970125. Full endpoints: 35,934,125,985,792 to 35,935,142,641,664 bytes. Endpoints-only: 35,935,142,641,664 to 35,935,441,477,632 bytes. Exact subtraction agrees with each reported total. No counter identity/reset error occurred.
- The full observer sampled only one owned process generation, totaling 3,399,680 lifetime physical-write bytes. Brief Git children can be missed. Endpoints-only process bytes were not observed; they are not zero. These counters do not partition host device bytes.
- A single fixed order leaves cache, background and delayed-write confounds. This selected test is not the whole package or suite. No APFS or NAND-wear attribution follows from this comparison. The concurrency budget question remains deferred by the hub until the writer is known.

## Provenance

Clean source at admission: 57e1e2b7b158162b28fccc7384ba7040e8db4023. Only documentation changed after compilation; no product/test code changed. Binary SHA256: d657cbc055c87885b9da421b34e47a69de5e542a095f7662862b0fe430d8cb56. Diagnostic driver SHA256: 0bc29a8e04cfd39b21f5415746fd396eea56a8119ecaea368d9d2d1205920103. Original #658 observer SHA256: bf5d37c01c26e289ca7821039d173d6f81c232537fd986e696796547091eadf9.

Local supporting root: /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair663-observer.cJE4Uv. The driver is observer_control.py; comparison/ holds result.json, manifest.json, queue.json, device.jsonl and both arm logs. There are 99 device samples. result.json SHA256: 523bf0598421b5bdc8b0c7c8016eebe4971828b83656931433527742bbba569b; device.jsonl SHA256: 1e5fec0a97614018d90a282614ac0ef360945357a299e0cddfc4e17a576ae1c0. Each arm log records five passes. Temporary raw files are supporting evidence; the measurements and limits above are the durable record.

## Next evidence

Hub receipt 99719b95-da83-4047-9c8d-1f380df71cbb approves requesting two all-host fs_usage arms in one paste: 60 seconds during final Go, then 60 seconds with no suite running. The ready-to-paste command is in 000663-operator-trace.md. It starts no suite, bounds captured output, and writes under TMPDIR. The operator arranges and confirms the two conditions. No privileged trace has been executed by this slot.
