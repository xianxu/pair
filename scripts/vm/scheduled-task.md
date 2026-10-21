# Nightly conformance handoff

Pair#743 supplies one deterministic command. Pair#737 owns scheduling;
operations owns registration. Until that scheduler lands, the hub or operator
invokes this command manually. Registration is pending.

After main contains #743 and ariadne#509's supported non-required `conformance-vm`
declaration, invoke:

```sh
bash /Users/xianxu/workspace/pair/scripts/vm/nightly.sh --checkout /Users/xianxu/workspace/pair
```

The caller needs SDLC at or after ariadne commit `9ffe23d0`, Git origin access and the qualified image. Version-2 declaration uses `scope: landing`, `resource: runner`, `required: false`; named execution is manual-only on pinned main, and `--all` excludes it. SDLC takes no suite slot; the Pair runner alone owns VM capacity.
The command fetches origin/main once, resolves its exact commit and runs
`sdlc local-check run --name conformance-vm --rev SHA` once. The checkout keeps its
branch and local edits. A concurrent push can cause SDLC's exact-tip guard to
refuse; preserve that failure. No retry occurs.

The operations native Go task adapter under
`scripts/scheduled/nightly-conformance/` invokes that argv. Revalidate #737's
landed declaration parser before registration; these are handoff values:

| Setting | Value |
| --- | --- |
| Cron | `30 3 * * *`, local time |
| Scope | machine |
| Missed | skip |
| Timeout | 30 minutes, including setup and the 20-minute VM execution plus cleanup |
| Cancellation grace | SDLC provider's approved 30 seconds; runner cleanup up to 10 seconds |
| Resource | runner-owned train lane through #509; no additional suite slot or VM lease |
| Execution | one invocation; no retries or promotion counter; exit 10 wakes the hub |

Map `version`, `cron`, `timeout`, `scope`, `missed`, `script_run` and `heavy`
according to the landed #737 schema and #509 resource semantics. Do not guess
that `heavy` means another suite allocation. Operations owns task code, project
updates and registration. This handoff installs no service or private timer.

Evidence defaults to the native account home's `~/.local/share/pair/conformance-nightly/YYYY-MM-DD/`.
Inherited HOME does not change managed-storage authority or that default.
`--artifacts-root ABSOLUTE_PATH` selects a separate caller-owned root;
`--date YYYY-MM-DD` selects an explicit date for manual invocation. A date
reservation persists after fetch/setup/execution/cleanup failure. A partial
directory after abrupt termination is an unconfirmed night: inspect it, never
count it as passed or overwrite it.

`nightly.json` records exact source and exit code (null while incomplete).
Its fresh `vm/` child passes through host-only `PAIR_VM_ARTIFACTS_DIR` to
SDLC/Make, then the generic runner's `--artifacts`. The VM result retains
image/tool provenance; raw stdout/stderr retain strict case evidence. SDLC owns
the exact-commit status and local-check log. A successful wrapper exit alone
does not prove those records. No candidate declaration injection or hand-written
status publisher is an acceptable bootstrap.

Admission retains at most sixteen nightly directories and 4 GiB of logical
evidence bytes. Every retained date charges max(actual measured bytes, the full
worst-case runner bundle), regardless of manifest or exit code. SDLC leader exit
cannot prove that descendants stopped writing. No reservation is released until
explicit archive. Enumeration and metadata errors refuse; unreadable evidence
never counts as zero.

At twelve retained dates or 75% of reserved bytes (and at either hard cap), the
command launches no check and exits **10: wake the hub**. Stderr reports the
population and prints one shell-quoted `mv` command to move this entire evidence
root to a unique sibling archive. The shared task must surface exit 10 to its TL;
it is attention required, never a passed night. After inspecting failed/partial
results and reconciling any surviving writers, run that exact emitted command.
The next invocation creates a fresh evidence root. The archive keeps all dates,
logs and metadata unchanged; the hub owns its subsequent #745 decision and
explicit disposal. There is no automatic eviction. Ordinary runner retention
cannot prune these caller-owned bundles. The artifact-admission lock is held
through the check; do not archive while any invocation or unconfirmed writer is
active. This is an explicit operator archive step, not another cleanup process.

After #743 lands, bootstrap the named check on exact main and verify retained
status/log, image identity, required cases and clean VM release. Give the hub
the observed result directory. GitHub PR conformance stays required; activation
stays off. #745 owns seven distinct clean nights and promotion. #744 owns genuine
fleet/agent coverage; current M2 native coverage uses legacy stand-ins.
