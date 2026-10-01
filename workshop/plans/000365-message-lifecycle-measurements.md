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

_(Pending: `sdlc move :0`, `make build`, operator relaunches Couch and every slot, then the same commands with the same slot count.)_
