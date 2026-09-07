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
kv "pair_family_procs" "$(ps -Ao comm= 2>/dev/null | grep -cE 'pair|couch|nvim|zellij' || echo 0)"
kv "build_procs" "$(ps -Ao comm= 2>/dev/null | grep -cE 'compile$|link$|^go$' || echo 0)"
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
if [ -x "$PROBE" ]; then
	"$PROBE" | awk '{printf "pipe_hop_ms=%s\npipe_hop_p90=%s\n", $1, $2}'
	"$PROBE" -spawn 30 -- /usr/bin/true | awk '{printf "fork_exec_ms=%s\nfork_exec_p90=%s\n", $1, $2}'
	if command -v zellij >/dev/null 2>&1; then
		"$PROBE" -spawn "$ZJ_SAMPLES" -- zellij action query-tab-names \
			| awk '{printf "zellij_action_ms=%s\nzellij_action_p90=%s\n", $1, $2}'
	else
		kv "zellij_action_ms" "n/a (zellij not on PATH)"
	fi
else
	kv "probes" "n/a (pair-hoprtt not built — run make build)"
fi

say ""
say "# end"
