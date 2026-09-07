#!/bin/sh
# doctor/perf.sh — capture the machine's state at a moment of felt slowness.
#
# Invoked by :PairDoctor (#208), and standalone by an agent that wants to
# re-take a reading. Prints a line-oriented report on stdout.
#
# TWO RULES SHAPE THIS FILE.
#
# 1. It emits RAW two-sample output and does no arithmetic. The pid join
#    (per-process CPU rate, vanished/started counts) is the piece most likely to
#    be mishandled, so it lives in `doctor.delta` where it is unit-tested against
#    fixtures. Shell is the wrong home for error-prone logic.
#
# 2. `ps %cpu` is a LIFETIME AVERAGE and is never used for a verdict. Measured
#    2026-09-06: contactsd read ~0% in ps (60 min of CPU over 9 days uptime)
#    while actually burning 42.6%. Everything per-process here comes from two
#    samples separated by a window.
#
# Every probe degrades to `n/a` with a reason rather than aborting the capture.
# A partial report beats none when the operator is already suffering.

WINDOW="${PAIR_PERF_WINDOW:-2}"     # seconds between the two samples
PROBE="${PAIR_HOME:-}/bin/pair-hoprtt"
ZJ_SAMPLES=5                        # capped: at a degraded 145ms this is 0.7s

say() { printf '%s\n' "$*"; }
kv()  { printf '%s=%s\n' "$1" "$2"; }

# A collector that fails must render as n/a, not as a value. The file's own rule
# said so; without this helper an empty or failed command printed `key=` and a
# reader could not tell "zero" from "we could not measure".
collect() {
	_key=$1; shift
	_val=$("$@" 2>/dev/null) || _val=""
	case "$_val" in
		"") kv "$_key" "n/a (collector failed)" ;;
		*)  kv "$_key" "$_val" ;;
	esac
}

# comm on macOS is a FULL PATH (/usr/local/bin/go), so `^go$` never matched and
# the loose `pair` pattern matched anything with pair in its path. Count on the
# basename.
count_comm() { ps -Ao comm= 2>/dev/null | awk -F/ '{print $NF}' | grep -cE "$1"; }

# The 6s budget, ENFORCED rather than declared. Probes are the tail of the run,
# so anything past the deadline is skipped and SAID -- a truncated honest report
# beats a capture that outlives the operator's patience.
BUDGET="${PAIR_PERF_BUDGET:-6}"
STARTED=$(date +%s)
over_budget() { [ $(( $(date +%s) - STARTED )) -ge "$BUDGET" ]; }

say "# pair perf capture"
kv  "captured_at" "$(date '+%Y-%m-%dT%H:%M:%S%z')"
kv  "window_seconds" "$WINDOW"
kv  "host_cores" "$(sysctl -n hw.ncpu 2>/dev/null || echo '?')"

say ""
say "## conditions"
kv "load" "$(sysctl -n vm.loadavg 2>/dev/null | tr -d '{}' | awk '{print $1"/"$2"/"$3}')"
kv "memory_pressure_level" "$(sysctl -n kern.memorystatus_vm_pressure_level 2>/dev/null || echo 'n/a')"
kv "process_count" "$(ps -Ao pid= 2>/dev/null | wc -l | tr -d ' ')"

# ONE top invocation feeds both cpu-idle and WindowServer: top -l 2 costs ~2s
# and calling it twice was the whole budget overrun. Its SECOND sample is a
# delta; the first is a lifetime average (rule 2), which is why -l 2 is required.
TOP_OUT=$(top -l 2 -n 60 -stats command,cpu 2>/dev/null)
# "CPU usage: x% user, y% sys, z% idle" -- the number is $(NF-1); $NF is the
# word "idle".
kv "cpu_idle_pct" "$(printf '%s' "$TOP_OUT" | awk '/^CPU usage/{v=$(NF-1)} END{print v}' | tr -d '%')"

say ""
say "## fleet"
kv "pair_family_procs" "$(count_comm '^(pair|pair-.*|couch|nvim|zellij)$')"
kv "build_procs" "$(count_comm '^(compile|link|go|vet|asm|cgo)$')"
kv "windowserver_cpu_pct" "$(printf '%s' "$TOP_OUT" | awk '/WindowServer/{v=$NF} END{print (v==""?"n/a":v)}')"

# --- the two samples. Raw; doctor.delta joins them. -------------------------
# Fields: pid, cumulative-cpu-seconds, rss-kb, comm. Cumulative CPU is what
# makes a rate computable; %cpu deliberately absent (see rule 2).
# `etime` (elapsed since start) is carried so the join can detect a REUSED pid:
# a pid whose etime went DOWN between samples is a different process that
# inherited the number, and reporting a rate for it would be nonsense.
sample() {
	ps -Ao pid=,etime=,rss=,comm= 2>/dev/null | awk '{printf "%s\t%s\t%s\t%s\n", $1, $2, $3, $4}'
}
cputimes() {
	ps -Ao pid=,time= 2>/dev/null | awk '{printf "%s\t%s\n", $1, $2}'
}

# Taken before the window so the swap rate below costs no extra sleep.
SWAP_A=""
command -v vm_stat >/dev/null 2>&1 && SWAP_A=$(vm_stat | awk '/Swapins/{si=$2} /Swapouts/{so=$2} /Pageins/{pi=$2} END{print si, so, pi}' | tr -d '.')

say ""
say "## sample_a"
kv "at_ns" "$(date +%s)"
say "### cputime"
cputimes
say "### procs"
sample

sleep "$WINDOW"

say ""
say "## sample_b"
kv "at_ns" "$(date +%s)"
say "### cputime"
cputimes
say "### procs"
sample

# --- swap as a RATE. The cumulative counters always look alarming. ----------
say ""
say "## swap_rate"
# Rides the MAIN window rather than paying its own sleep -- SWAP_A was taken
# before it. Rates are per-second over WINDOW.
if [ -n "$SWAP_A" ]; then
	b=$(vm_stat | awk '/Swapins/{si=$2} /Swapouts/{so=$2} /Pageins/{pi=$2} END{print si, so, pi}' | tr -d '.')
	echo "$SWAP_A $b" | awk -v w="$WINDOW" '{printf "swapins_per_s=%.1f\nswapouts_per_s=%.1f\npageins_per_s=%.1f\n", ($4-$1)/w, ($5-$2)/w, ($6-$3)/w}'
else
	kv "swap" "n/a (no vm_stat)"
fi

say ""
say "## disk"
if command -v iostat >/dev/null 2>&1; then
	iostat -d -w 1 -c 2 2>/dev/null | tail -1 | awk '{printf "kb_per_transfer=%s\ntps=%s\nmb_per_s=%s\n", $1, $2, $3}'
else
	kv "disk" "n/a (no iostat)"
fi

# --- probes, with their known-good baselines printed alongside so a reading
#     is interpretable without hunting for this file. -------------------------
say ""
say "## probes"
say "# baselines on a healthy host: pipe_hop ~0.007ms, fork_exec ~1.5ms, zellij ~13ms"
# A probe line carrying a 5th field means invocations FAILED: the command
# errored fast, which would otherwise read as excellent latency.
probe_line() {
	_name=$1; shift
	if over_budget; then kv "${_name}_ms" "n/a (budget exceeded, probe skipped)"; return; fi
	_out=$("$@" 2>/dev/null) || { kv "${_name}_ms" "n/a (probe failed)"; return; }
	printf '%s' "$_out" | awk -v n="$_name" '{
		if (NF >= 5 && $5 > 0)
			printf "%s_ms=n/a (%s of %s invocations failed)\n", n, $5, $4;
		else
			printf "%s_ms=%s\n%s_p90=%s\n", n, $1, n, $2;
	}'
}

if [ -x "$PROBE" ]; then
	probe_line pipe_hop "$PROBE"
	probe_line fork_exec "$PROBE" -spawn 30 -- /usr/bin/true
	if command -v zellij >/dev/null 2>&1; then
		probe_line zellij_action "$PROBE" -spawn "$ZJ_SAMPLES" -- zellij action query-tab-names
	else
		kv "zellij_action_ms" "n/a (zellij not on PATH)"
	fi
else
	kv "probes" "n/a (pair-hoprtt not built — run make build)"
fi
kv "elapsed_seconds" "$(( $(date +%s) - STARTED ))"
kv "budget_seconds" "$BUDGET"

say ""
say "# end"
