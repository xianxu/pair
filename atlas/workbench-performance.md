# Workbench counted performance invariants

The [workbench latency target](../workshop/targets/workbench-latency.md) bounds
work by what changed. #204 makes these promises release tests in the normal
`make test` suite. They measure bytes and native reader calls, never elapsed time.

| Interaction | Counted boundary | Bound |
|---|---|---|
| Idle UI activity | Console request → activity worker → completed generation and menu reducer; 20 owners with valid 30,000,000-byte ledgers | Each unchanged pass reads 0 historical body bytes and supplies 0 decoder-input bytes |
| Peer message | Real private socket request → broker → submitted receipt and confirmed endpoint delivery; the same 20-owner scale | Each unchanged request reads/decodes 0 historical/index body bytes |
| Selected Couch attach | Public listed attach parent plus Pair helper, through blocking attach handoff | Parent 0 full tables, helper at most 1 |
| Standalone attach | Launch entry through handoff | At most 2 full tables |
| Continuation falling through to attach | Continuation launch entry through handoff | At most 3 full tables |
| Owner pass | Shared initial capture plus Probe/Revalidate for N callers | At most 1+2N full tables |

The last three bounds are **ratchets**, not the desired final design. #567 owns
lowering them to one without dropping safety checks. `SnapshotNamed` enumerates
the whole table too; it filters subsequent argv enrichment. Targeted PID identity
and liveness reads are separate operations and remain fresh. An initial owner
capture count alone is not a count of the entire owner workflow.

The zero-byte bounds cover unchanged ledgers, session indexes, catalogs and
native transcript bodies. Metadata fingerprints and small current ready/PID
security files are deliberately still observed. Tests also require successful
activity/delivery/attachment and stale-binding refusal: skipping authority is
not a valid way to obtain zero work.

## Run the release net

```sh
go test ./cmd/internal/couchtty ./cmd/internal/couchcmd ./cmd/internal/launcher ./cmd/internal/couchcore -run '^TestWorkbenchPerformance' -count=1
```

These tests need no real agents, Zellij server, operator window or network
service. They use temporary homes, artifacts, private sockets, terminal doubles
and a controlled native process reader. Context deadlines only bound a hung test;
they are not latency assertions. `make test` also runs them without a special flag.

- `TestWorkbenchPerformanceIdleUITick` uses the same
  `couchtty.ThreadActivityProbe` adapter as production `wireIdleFading`, drives
  the exact activity schedule events used by the UI ticker and applies results
  to the real menu reducer. It actually parses 600 MB cold, then measures every
  warm pass independently. One owner mutation performs owner-local work, followed
  by another zero-work pass.
- `TestWorkbenchPerformanceMessageUnchangedEvidence` uses the production
  recorded-authority adapter and the existing socket-service fixture. Every send
  must reach its endpoint and a submitted receipt before measurement. Index
  mutation invalidates only its owner; current pane, PID and ready checks remain
  active. The valid large ledgers must remain unread on this path.
- `TestWorkbenchPerformanceAttachTableBudgets`,
  `TestWorkbenchPerformanceListedAttachParentHasNoTableScan` and
  `TestWorkbenchPerformanceOwnerPassCountsEveryNativeFence` count the low-level
  reader shared by production ancestry, snapshots, initial capture and live
  fences. Native defaults retain fresh observations; instance-local injection
  provides isolation, not caching.

#559's smaller component tests retain invalidation, replacement, failed evidence
and allocation-scale controls. The consumer tests own the release-scale envelope.
Earlier submit/span/build counts and nonblocking timing work are preserved in
unassigned #566.

## Add a counted invariant

Choose an interaction and name its full observation boundary. Define an upper
bound, with its workload and exclusions, before choosing a seam. Measure the
operation at the lowest existing source boundary; counting high-level calls can
miss work hidden inside them. `procutil.Source` reuses the actual native snapshot
parser and reader accounting, while #559's `QueryWorkObserver` and
`SessionNameIndexCacheIO` distinguish filesystem bytes from decoder inputs.

Reuse `workbenchperftest` for valid evidence and thread-safe independent read/
decode totals. Its snapshots own their maps; deltas retain negative values so a
reset cannot masquerade as zero. Shared accounting is checked against an
independent generated-event oracle and joined concurrent updates under race.

Require a positive cold control, a controlled mutation, and correct successful
behavior. Assert each unchanged operation separately. Reintroduce the redundant
work through a temporary overlay and demonstrate the invariant fails; disconnect
its observer and demonstrate the positive control fails. Preserve privacy and
fresh authority checks. Keep the new test under the normal suite and document
its count boundary here. Do not add a timing threshold or a second test runner.

## Couch acceptance: idle CPU remains an operator smoke

On every Couch acceptance session, the TL/operator uses the approved 20-slot
fixture with representative 30 MB ledgers, restarts Couch onto the candidate,
and waits for attaches and other startup work to settle. This worker item does
not stage or restart the operator's Couch. Record the actual fixture size,
concurrently working agents, machine/build and settled state; a different scale
is a stated coverage gap rather than an assumed 20-slot result.

Measure the **supervisor PID**, not its shell or a worker. With `COUCH_PID` set
from the live supervisor's identity, this reads cumulative process CPU twice:

```sh
python3 - "$COUCH_PID" <<'PY'
import subprocess, sys, time
pid = int(sys.argv[1])
def cpu_seconds():
    value = subprocess.check_output(
        ['ps', '-p', str(pid), '-o', 'cputime='], text=True).strip()
    days, sep, clock = value.partition('-')
    if not sep:
        clock, days = days, '0'
    parts = [float(p) for p in clock.split(':')]
    seconds = 0
    for part in parts:
        seconds = seconds * 60 + part
    return int(days) * 86400 + seconds
def identity():
    return subprocess.check_output(
        ['ps', '-p', str(pid), '-o', 'lstart='], text=True).strip()
started = identity()
start = time.monotonic()
first = cpu_seconds()
time.sleep(60)
last = cpu_seconds()
elapsed = time.monotonic() - start
if identity() != started or last < first:
    raise SystemExit('supervisor changed during the sample; no result')
percent = 100 * (last - first) / elapsed
print(f'pid={pid} mean_cpu={percent:.2f}% over {elapsed:.1f}s')
PY
```

Acceptance is **mean supervisor CPU below 5% over 60 seconds after settling**,
plus an ordinary successful peer-message round trip. If it fails, retain the
build/PID, population/working-agent count and a profile; follow the hot path
rather than widening a threshold. The fixture and process must remain the same
throughout the sample. This manual smoke is separate from deterministic CI;
no timed capture can turn a failed count into a pass.
