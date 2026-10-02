# #365 measurements — idle Couch messaging cost

## Before (2026-10-01, live, read-only)

- Couch: `~/workspace/pair/bin/couch`, vcs.revision `6a64e5d2` (includes the #360 10 s-window stopgap); pid 97616, 9.5 min old, 11 `pair wrap` processes; operator idle.
- **Couch CPU: 101.6 CPU-seconds in a 120 s window (≈85% of one core).**
- Couch children seen by a 10 Hz `ps -axo pid=,ppid=,comm=` poll over the same 120 s (distinct PIDs; undercounts short-lived ones):

  | child | distinct PIDs / 120 s |
  |---|---|
  | ps | 639 |
  | zellij | 179 |
  | sdlc | 141 |
  | git | 94 |

  `sdlc workspace --json` is `ResolveWorkspace` (`couchcore/provision.go:48`), which messaging runs only for a binding it doesn't know. 141 of them in 2 minutes means registrations are repeatedly failing and being re-admitted. Each failed heartbeat costs a full check plus a workspace resolve, every second, so failing wrappers are the most expensive case for the polling design.
- `sample 97616 10`: CPU frames are again VT emulation (`vt.handleGrapheme`, `ultraviolet.Line.Set`, `ansi.Parser`), GC (`scanObject`, `memclr`) and `fork`/`wait4`, matching the project log's 08:55 capture.
- In-tree count (`TestMessageIdleRunsNoProbes`, skipped until M2): for one healthy wrapper over an idle minute, 12 full checks (24 ownership probes ≈ 48 `ps` + 24 `zellij list-panes`) + 12 process checks + 6 `git status`.

Commands (re-runnable without relaunching Couch):

```bash
pid=$(pgrep -x couch | head -1)
cpu() { ps -o cputime= -p $pid | awk -F'[:.]' '{n=NF; s=$(n-1)+60*$(n-2); if(n>3)s+=3600*$(n-3); print s"."$n}'; }
c0=$(cpu); end=$((SECONDS+120)); : > children.txt
while [ $SECONDS -lt $end ]; do ps -axo pid=,ppid=,comm= | awk -v p=$pid '$2==p{print $1, $3}' >> children.txt; sleep 0.1; done
c1=$(cpu); echo "cpu seconds: $(python3 -c "print(round($c1-$c0,2))")"
sort -u children.txt | awk '{print $2}' | tr -d '()' | sort | uniq -c | sort -rn
sample $pid 10 -file couch.sample.txt
```

`probes/messageidle` gives exact spawn counts when Couch is launched from an armed shell (optional).

### Before, re-taken right before the switch (same workload as After)

- Couch pid 76438, build `60bb0490` (main, includes #360), 11 `pair wrap` processes, operator idle.
- **Couch CPU: 60.93 CPU-seconds in 120 s (≈51% of one core).**
- Children over 120 s (10 Hz poll, distinct PIDs): ps 536, sdlc 155, zellij 107, git 68.

## After

Couch build `e0a8c053` (this branch, `sdlc move :0` + `make build`), restarted by the operator. The 11 slots' wrappers kept running the old binary, so they still send the legacy 1 s `register`, now answered `unsupported` with no checks. Same commands, operator idle.

| window | Couch CPU-s / 120 s | ps | zellij | sdlc | git |
|---|---|---|---|---|---|
| before, main `6a64e5d2` (pid 97616) | 101.56 | 639 | 179 | 141 | 94 |
| before, main `60bb0490` (pid 76438, same workload as after) | 60.93 | 536 | 107 | 155 | 68 |
| after, Couch age 1 min (pid 22471) | 25.05 | 0 | 0 | 0 | 40 |
| after, Couch age 3.5 min | 22.52 | 0 | 0 | 0 | 38 |

- Messaging-attributable process spawns went from ≈800 per 2 min (`ps`, `zellij list-panes`/`list-sessions`, `sdlc workspace`) to **0**.
- Couch CPU fell **63%** against the same-workload baseline (60.9 → 22.5 CPU-s per 120 s, ≈51% → ≈19% of a core). Against the first capture it fell 78%.
- The remaining `git` (≈19/min) is the Console's own slot-git glyph pass (10 s per checkout), not messaging.
- The `sample` after the switch has no VT-emulation or fork/wait4 hot frames. The heaviest Couch frames are `ttyio` waits and JSON decoding, consistent with the legacy wrappers' 11 `register` requests per second.
- **Hypothesis tested and rejected:** I guessed that the old wrappers' legacy `register` requests were most of the remainder. The operator relaunched 6 of 11 slots on the new wrapper, and the next window (Couch age 10.5 min) measured **27.33 CPU-s / 120 s** (ps 0, zellij 0, sdlc 0, git 39). That is no lower, so the legacy requests are not the remainder.
- **What the remainder looks like:** over 30 s, Couch used 1.66 s user and 2.18 s system CPU, roughly 13% of a core. The window-to-window spread is wide (13–23%). It is mostly *system* time. The `sample` shows the hot leaves as `__fcntl`, `open`/`openat`, `lstat`, `getdirentries` and `mkdir`, plus `fork` for the Console's git passes, and some `terminal` cell work (`plainText`, `cloneCell`). Those are file-system polling and terminal rendering, not messaging. Candidates outside #365: the Console's 500 ms continuation-store reads, the 10 s slot-git pass, and diagnostic/storage writes (compare #370).
