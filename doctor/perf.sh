#!/bin/sh
# doctor/perf.sh — capture the machine's state at a moment of felt slowness.
#
# Invoked by :PairDoctor (#208), and standalone by an agent that wants to
# re-take a reading. Prints a line-oriented report on stdout.
#
# THREE RULES SHAPE THIS FILE.
#
# 1. A collector that fails renders as `n/a (<why>)`, NEVER as a value. An empty
#    `load=` or a fabricated `process_count=0` is indistinguishable from a real
#    reading, and this file is routinely run by sandboxed agents with exactly
#    these tools denied. Every collector goes through collect().
#
# 2. It emits RAW two-sample output and does no arithmetic. The pid join
#    (per-process CPU rate, vanished/started/reused counts) is the piece most
#    likely to be mishandled, so it lives in `doctor.delta` where it is
#    unit-tested. Shell is the wrong home for error-prone logic.
#
# 3. `ps %cpu` is a LIFETIME AVERAGE and is never used. Measured 2026-09-06:
#    contactsd read ~0% in ps (60 min of CPU over 9 days uptime) while actually
#    burning 42.6%. Everything per-process comes from two samples.
#
# The budget sheds work in REVERSE VALUE ORDER, and the ordering is the design.
# `top -l 2` and `iostat` are most of the wall clock but are the LEAST valuable
# rows on a slow machine; the three probes are the most valuable and the
# cheapest, so they must survive a squeeze. An earlier version guarded only the
# probes -- which shed exactly the wrong thing, yielding a capture with no probe
# rows in the one condition this tool exists for.
#
# Shed order: iostat, then top, then the sample window. The probes are reserved
# a slice of the budget and are never skipped while any budget remains.

WINDOW="${PAIR_PERF_WINDOW:-2}"
BUDGET="${PAIR_PERF_BUDGET:-6}"
ZJ_SAMPLES=5
STARTED=$(date +%s)

say() { printf '%s\n' "$*"; }
kv()  { printf '%s=%s\n' "$1" "$2"; }

elapsed()     { echo $(( $(date +%s) - STARTED )); }
over_budget() { [ "$(elapsed)" -ge "$BUDGET" ]; }
# Collectors stop early enough to leave PROBE_RESERVE seconds for the probes,
# which are the rows that actually distinguish a slow machine from a busy one.
PROBE_RESERVE="${PAIR_PERF_PROBE_RESERVE:-2}"
collectors_done() { [ "$(elapsed)" -ge $(( BUDGET - PROBE_RESERVE )) ]; }

# collect KEY TOOL FUNC — run FUNC; render `n/a (<why>)` on a missing tool, a
# failure, or an empty result. TOOL is named in the reason so a reader knows
# what was missing rather than seeing a bare blank.
collect() {
	_key=$1; _tool=$2; _fn=$3
	if ! command -v "$_tool" >/dev/null 2>&1; then
		kv "$_key" "n/a ($_tool unavailable)"; return
	fi
	_val=$("$_fn" 2>/dev/null) || { kv "$_key" "n/a ($_tool failed)"; return; }
	case "$_val" in
		"") kv "$_key" "n/a ($_tool returned nothing)" ;;
		*)  kv "$_key" "$_val" ;;
	esac
}

_now()       { date '+%Y-%m-%dT%H:%M:%S%z'; }
_loadavg()   { sysctl -n vm.loadavg | tr -d '{}' | awk '{print $1"/"$2"/"$3}'; }
_cores()     { sysctl -n hw.ncpu; }
_mempress()  { sysctl -n kern.memorystatus_vm_pressure_level; }
# `awk END` with a guard: a ps that produced no rows must FAIL rather than
# report 0, which would be a fabricated claim.
_proccount() { ps -Ao pid= | awk 'END{if (NR==0) exit 1; print NR}'; }
# comm on macOS is a FULL PATH (/usr/local/bin/go), so `^go$` never matches and a
# loose `pair` matches any path containing it. Count on the basename.
#
# Counting is done in awk, not `grep -c`, for two reasons that bit in review:
# `grep -c` prints 0 AND exits 1 on no match, so `|| echo 0` emitted a stray
# second line, and swallowing that exit with `|| true` turned a FAILED ps into a
# fabricated `0` -- indistinguishable from "no such processes". awk prints the
# count and exits 0, and ps's failure is checked separately so it can propagate.
_countmatching() {
	_procs=$(ps -Ao comm=) || return 1
	[ -n "$_procs" ] || return 1
	printf '%s\n' "$_procs" | awk -F/ -v re="$1" '{ if ($NF ~ re) k++ } END{ print k+0 }'
}
_family()    { _countmatching '^(pair|pair-.*|couch|nvim|zellij)$'; }
_builds()    { _countmatching '^(compile|link|go|vet|asm|cgo)$'; }

say "# pair perf capture"
collect "captured_at" date _now
kv  "window_seconds" "$WINDOW"
kv  "budget_seconds" "$BUDGET"
collect "host_cores" sysctl _cores

say ""
say "## conditions"
collect "load" sysctl _loadavg
collect "memory_pressure_level" sysctl _mempress
collect "process_count" ps _proccount

# ONE top invocation feeds both cpu-idle and WindowServer: `top -l 2` costs ~2s
# and calling it twice was an early budget overrun. The SECOND sample is a delta;
# the first is a lifetime average (rule 3), which is why -l 2 is required.
if collectors_done; then
	kv "cpu_idle_pct" "n/a (budget reserved for probes; top skipped)"
	kv "windowserver_cpu_pct" "n/a (budget reserved for probes; top skipped)"
elif ! command -v top >/dev/null 2>&1; then
	kv "cpu_idle_pct" "n/a (top unavailable)"
	kv "windowserver_cpu_pct" "n/a (top unavailable)"
else
	TOP_OUT=$(top -l 2 -n 60 -o cpu -stats command,cpu 2>/dev/null || true)
	if [ -z "$TOP_OUT" ]; then
		kv "cpu_idle_pct" "n/a (top failed)"
		kv "windowserver_cpu_pct" "n/a (top failed)"
	else
		# "CPU usage: x% user, y% sys, z% idle" — the number is $(NF-1); $NF is
		# the word "idle".
		_idle=$(printf '%s' "$TOP_OUT" | awk '/^CPU usage/{v=$(NF-1)} END{print v}' | tr -d '%')
		kv "cpu_idle_pct" "${_idle:-n/a (top had no CPU line)}"
		_ws=$(printf '%s' "$TOP_OUT" | awk '/WindowServer/{v=$NF} END{print v}')
		kv "windowserver_cpu_pct" "${_ws:-n/a (WindowServer absent from top output)}"
	fi
fi

say ""
say "## fleet"
collect "pair_family_procs" ps _family
collect "build_procs" ps _builds

# --- the two samples. Raw; doctor.delta joins them. -------------------------
# `etime` is carried so the join can detect a REUSED pid: a pid whose etime went
# DOWN between samples is a different process that inherited the number.
# ps is captured BEFORE the pipe, not piped straight into awk. In a pipeline the
# exit status is awk's, so a denied or failed ps exits 0 with no output -- and
# `cputimes || say n/a` never fires, emitting an EMPTY section that reads as "no
# processes" rather than "not measured". That is rule 1's defect class, and it
# was live here until a sandboxed `make test` (where ps is denied outright)
# rendered both sections blank.
# comm is filtered HERE, at the single point it is emitted, rather than at each
# consumer. A process name is attacker- and accident-controlled: any app can be
# named with control bytes in it, and this text reaches a terminal (the report),
# a file the agent is told to open (the sidecar), and a committed fixture. A ^N
# is SO and garbles every following line; a newline would split a sample row and
# could forge a `key=value` line inside the reader's own evidence. Filtering at
# one egress and not the others just moves the hazard one file over.
sample() {
	_p=$(ps -Ao pid=,etime=,rss=,comm=) || return 1
	[ -n "$_p" ] || return 1
	# Everything from field 4 onward, not $4 alone: a comm may contain SPACES
	# ("/Applications/Google Chrome.app/.../Google Chrome"), and taking one field
	# basenames it to "Google" -- a wrong name attributed to a real pid, which is
	# worse than an ugly one.
	printf '%s\n' "$_p" | awk '{
		c = $0
		sub(/^[[:space:]]*[0-9]+[[:space:]]+[^[:space:]]+[[:space:]]+[0-9]+[[:space:]]+/, "", c)
		n = split(c, parts, "/"); c = parts[n]
		gsub(/[^[:print:]]/, "?", c)
		if (c == "") c = "?"
		printf "%s\t%s\t%s\t%s\n", $1, $2, $3, c
	}'
}
cputimes() {
	_p=$(ps -Ao pid=,time=) || return 1
	[ -n "$_p" ] || return 1
	printf '%s\n' "$_p" | awk '{printf "%s\t%s\n", $1, $2}'
}

_swapnow() {
	_vm=$(vm_stat 2>/dev/null) || return 1
	[ -n "$_vm" ] || return 1
	printf '%s\n' "$_vm" | awk '/Swapins/{si=$2} /Swapouts/{so=$2} /Pageins/{pi=$2}
		END{ if (si=="" || so=="" || pi=="") exit 1; print si, so, pi }' | tr -d '.'
}
SWAP_A=$(_swapnow 2>/dev/null || true)

emit_sample() {
	say ""
	say "## $1"
	collect "at_s" date "date_epoch"
	say "### cputime"
	cputimes 2>/dev/null || say "n/a (ps failed or is unavailable)"
	say "### procs"
	sample 2>/dev/null || say "n/a (ps failed or is unavailable)"
}
date_epoch() { date +%s; }

emit_sample sample_a
WINDOW_ELAPSED=0
if collectors_done; then
	say ""
	say "## sample_b"
	kv "skipped" "budget reserved for probes; no rates computable"
else
	sleep "$WINDOW"
	WINDOW_ELAPSED=1
	emit_sample sample_b
fi

say ""
say "## swap_rate"
# A failed collector renders under the SAME keys as a successful one. Emitting
# `swap=n/a` instead means every consumer must know both the success key AND the
# failure key, and any that knows only the first drops the row silently -- which
# is worse than a wrong value, because an absent line is indistinguishable from a
# tool that has no such section. Same reason the `## disk` and `## probes`
# sections below do it too.
# na_for REASON KEY... — one loop, three call sites. The three used to be three
# copies differing only in their key list.
na_for() { _r=$1; shift; for k in "$@"; do kv "$k" "n/a ($_r)"; done; }
swap_na()   { na_for "$1" swapins_per_s swapouts_per_s pageins_per_s; }
disk_na()   { na_for "$1" kb_per_transfer tps mb_per_s; }
probes_na() { na_for "$1" pipe_hop_ms fork_exec_ms zellij_action_ms; }
# Rides the MAIN window rather than paying its own sleep.
if [ "$WINDOW_ELAPSED" != 1 ]; then
	# The sleep never ran, so there is no interval to divide by. Reporting
	# (b-a)/WINDOW here would be a rate over time that did not pass -- the same
	# fabrication the n/a rule exists to prevent, in shell rather than Lua.
	swap_na "sample window was shed; no interval to rate over"
elif [ -z "$SWAP_A" ]; then
	swap_na "vm_stat unavailable"
else
	_b=$(_swapnow 2>/dev/null || true)
	if [ -z "$_b" ]; then
		swap_na "vm_stat failed on the second read"
	else
		echo "$SWAP_A $_b" | awk -v w="$WINDOW" '{printf "swapins_per_s=%.1f\nswapouts_per_s=%.1f\npageins_per_s=%.1f\n", ($4-$1)/w, ($5-$2)/w, ($6-$3)/w}'
	fi
fi

say ""
say "## disk"
if collectors_done; then
	disk_na "budget reserved for probes; iostat skipped"
elif ! command -v iostat >/dev/null 2>&1; then
	disk_na "iostat unavailable"
else
	_io=$(iostat -d -w 1 -c 2 2>/dev/null | tail -1)
	if [ -z "$_io" ]; then
		disk_na "iostat failed"
	else
		printf '%s' "$_io" | awk '{printf "kb_per_transfer=%s\ntps=%s\nmb_per_s=%s\n", $1, $2, $3}'
	fi
fi

# --- probes -----------------------------------------------------------------
# The probe is a SUBCOMMAND of pair, not a separate binary, and that is what
# makes it work in a shipped pair at all. A standalone pair-hoprtt existed only
# after `make install`: the Homebrew formula builds just ./cmd/pair-go, and
# PAIR_HOME at runtime is the extracted bundle root, which carries no helper
# binaries (since #104 M3). So the probes would have been permanently n/a for
# every installed pair -- the operator this capture exists for.
PROBE=""
if command -v pair >/dev/null 2>&1; then
	PROBE="pair"
elif [ -n "${PAIR_HOME:-}" ] && [ -x "${PAIR_HOME}/bin/pair" ]; then
	PROBE="${PAIR_HOME}/bin/pair"
fi

say ""
say "## probes"
say "# baselines on a healthy host: pipe_hop ~0.007ms, fork_exec ~1.5ms, zellij ~13ms"

# A probe line carrying a 5th field means invocations FAILED: the command
# errored fast, which would otherwise read as excellent latency.
probe_line() {
	_name=$1; shift
	if over_budget; then kv "${_name}_ms" "n/a (budget exceeded, probe skipped)"; return; fi
	_out=$("$@" 2>/dev/null) || { kv "${_name}_ms" "n/a (probe failed)"; return; }
	[ -n "$_out" ] || { kv "${_name}_ms" "n/a (probe returned nothing)"; return; }
	printf '%s' "$_out" | awk -v n="$_name" '{
		if (NF >= 5 && $5 > 0)
			printf "%s_ms=n/a (%s of %s invocations failed)\n", n, $5, $4;
		else
			printf "%s_ms=%s\n%s_p90=%s\n", n, $1, n, $2;
	}'
}

if [ -n "$PROBE" ]; then
	probe_line pipe_hop "$PROBE" hoprtt
	probe_line fork_exec "$PROBE" hoprtt -spawn 30 -- /usr/bin/true
	if command -v zellij >/dev/null 2>&1; then
		probe_line zellij_action "$PROBE" hoprtt -spawn "$ZJ_SAMPLES" -- zellij action query-tab-names
	else
		kv "zellij_action_ms" "n/a (zellij not on PATH)"
	fi
else
	probes_na "pair not on PATH"
fi

kv "elapsed_seconds" "$(elapsed)"
say ""
say "# end"
