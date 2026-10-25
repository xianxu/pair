# pair#865 initial read-only measurements — 2026-10-24

The #860 merge runner retains the checkout; no source or checkout file was changed. #865 is reserved by pair:9 and its card is retitled Live Couch sustains ~1.3 cores: attribute and bound it. Full details/plan publication waits until #860 settles.

Live Couch PID4409 started Oct23 20:41:49 PDT. A25.0015s proc_pid_rusage window measured135.6886% CPU,180376.98 physical write bytes/s, five new frames. Mach ticks converted with SDK timebase125/3. Raw window is /tmp/pair865-window.json. fseventsd rusage refuses EPERM and fs_usage requires root; no causality established.

The actual current cast is segment13, a local Cast Notes layout mock, viewport1728x997 at DPR2. Stored frames864x498, scale0.5, about5.25s apart; normal ShotEvery is5s. This differs from the echo busy board present when the issue was filed. No viewer count has been independently established yet.

10s sample at1ms interval:5957 counted symbolized Couch leaf samples (flat output includes only functions with >=5 samples); image path201 (~3.37%), terminal/ingest702, GC/memory1756, JSON623, operator224, other2451. These are sample counts, not a CPU-profile attribution. Go unwinding and background threads limit inference. Raw /tmp/pair865-couch-sample-dense.txt; short initial sample /tmp/pair865-couch-sample.txt; top intervals /tmp/pair865-couch-top.txt.

The mapped binary UUID in sample is71435B3F-E53E-3B8F-9BAA-F82E6EBF9EA4; the current file at the same path has UUID DCDA620B-E040-3662-B779-91DADAD6D406. It was replaced after process launch. Do not attribute the running process to the on-disk vcs revision or assume the source checkout matches it. A fresh staged profiling baseline needs a pinned matching executable.

No pprof symbol/CPUProfile hook was found in command source and no TCP listener is exposed by PID4409. Thus add opt-in local profiling only after the profiling-first plan is approved and #860 merge settles. Do not optimize the screenshot path based on this sample. The fseventsd question remains observational pending a controlled window or permitted privileged trace.

In the25s window, five new PNGs total245785bytes (9830.8B/s), plus event stream growth12654bytes. This is logical recording growth, not physical per-file write attribution.

## Existing in-process Console control (03:06–03:07 PDT)

Existing TestCouchProductionSoak passed in30.17s with PAIR_TERMINAL_SOAK_DURATION=30s; source HEAD839377c1dd90236a167979a6063cfe741cb7a1fd remained clean and unchanged. This exercises the production Console/presenter with two real PTY helpers, switching, resizing, held ordering and attachment replacement; it excludes full native-agent/Zellij/message-service/Chrome workload. It is not a reproduction of the live incident.

583 iterations,145 attachment replacements,13,637,824 parent-output bytes,7289 writes, max input-visible45.301833ms. The matching retained test executable is /tmp/pair865-console.test (SHA256ae8b3b2034f5cc1da6d1c8b2de51e16eb910aa9ff91644e61d22cbf3ef66c1e6).

CPU profile /tmp/pair865-console-cpu.pprof parses with go tool pprof:30.28s duration,13.78s sampled CPU (45.51%). Presenter.run cumulative4.19s; terminal paintEndpoint3.88s. These overlap and must not be summed. The emulated physical host is test overhead included in these totals.

Allocation profile /tmp/pair865-console-allocs.pprof parses:10396.63MB cumulative alloc_space (sampling estimate, not RSS), cloneCells3400.08MB flat, slices.Clone of terminal cells2799.41MB flat, Compose1192.87MB flat, Endpoint.capture894.03MB flat. The workload deliberately drives repeated resize/replacement/history and includes fixture validation; do not project this allocation rate onto live Couch. CPU and alloc top reports saved alongside. No production optimization selected.

Existing BenchmarkEndpointFeedSnapshot also passed on the same untouched checkout (one1s sample per geometry, darwin/arm64 Apple M2 Max, Go reported GOMAXPROCS3):80x24=423601ns/op,443233B/op,27allocs/op;240x80=3069153ns/op,4311650B/op,27allocs/op. Raw /tmp/pair865-terminal-bench.txt. The benchmark measures Feed plus Snapshot for a fixed short output sequence; it does not cover idle polling, history growth, Chrome or fleet ingestion.

Existing BenchmarkHistoryPublication passed9.384s with a2000-row prefill and one1s sample per case. Raw /tmp/pair865-history-bench.txt.80x24 unchanged publication=3.587ms/op,4,715,783B/op,473allocs; fresh=3.614ms/op,4,937,357B/op,477allocs.240x80 unchanged=4.009ms/op,6,399,363B/op,159allocs; fresh=4.138ms/op,8,555,148B/op,163allocs. This existing benchmark explicitly documents bounded full-window copy cost; it is a component measurement, not proof of repeated live calls. Keep it for comparison only after the native profile identifies the actual caller/rate.

Native harness prerequisite: pair#863 owns the explicit-hub startup regression on main. Sent owner pair:7 receipt2f725f30-01ba-4a56-be29-12c1ec49245c (submitted) requesting landed revision and limits; no private workaround or branch stacking.

Queue correction at03:52: TL briefly dispatched #868 ahead of profiling, then withdrew it after ancestry/source verification. The failing #830 head a520f3a5af36b66ca0a612aee8deaa95f1a0ec34 lacks #825 fix917c97e6a4bda90ecfac2f11e879e685539fdc40 and still uses len(entries)!=0; no failing head containing the fix is known. Peer pair:6 confirmed. TL7b582422-42e6-429b-a1b5-0cc721e21657 says do not claim #868; resume #865 after #860. Current unchanged #860-head cancellation reuse and real-record controls passed count3 in8.978s (/tmp/pair868-current-controls.log). No issue or checkout edits made.
