---
name: messageidle
description: Count the zellij/ps processes an idle Couch spawns and its CPU-seconds over a fixed window; #365's before/after measurement.
---

# Measuring idle Couch messaging cost

```bash
eval "$(probes/messageidle/run.sh arm)"      # the shell that will launch couch
couch                                        # attach the usual slots, then leave idle
SLOTS=4 probes/messageidle/run.sh window 120 before   # from another shell
eval "$(probes/messageidle/run.sh disarm)"
```

`arm` builds one shim twice, as `zellij` and as `ps`, into `probes/messageidle/bin`
and prepends it to `PATH`. Every Couch call site resolves both through `PATH`. The
shim records a row **only when its parent process is `couch`**: agents in the slots
inherit the same `PATH` and run `ps` themselves, and that is not Couch's cost.

`window` measures the running couch for N seconds (default 120):
`couch_cpu_seconds` (from `ps -o cputime`), `couch_spawns` per subcommand, and a
10-second `/usr/bin/sample` profile saved next to the summary under `$TMPDIR`. Run
it unsandboxed: `sample` needs task access to couch. For the running-thread view used
in the original investigation, also run
`sudo spindump couch 10 1 -onlyRunnable -file /tmp/couch-cpu.spindump.txt`.

Not every spawn is messaging's. The Console's own slot-git pass (`git`, not shimmed)
and on-demand `zellij action list-clients` also appear. #365 attributes by subcommand:
`ps -axo` and `zellij action list-panes` / `list-sessions` come from the
messaging ownership probe (`launcher/session_owner_os.go`).

Cost of the instrument: one extra fork+exec per shimmed call plus a parent
`sysctl`. It is the same for before and after, so it doesn't bias the delta.
