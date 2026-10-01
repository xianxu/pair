#!/usr/bin/env bash
# Measure what an idle Couch spends on messaging (#365): processes Couch spawns
# (zellij, ps) and Couch CPU-seconds over a fixed idle window, plus a 10s
# `sample` profile. Same commands before and after, same slot count.
#
#   eval "$(probes/messageidle/run.sh arm)"   # in the shell that launches couch
#   couch                                     # attach the usual slots; leave idle
#   probes/messageidle/run.sh window [seconds] [label]   # from another shell
#   eval "$(probes/messageidle/run.sh disarm)"
set -uo pipefail

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN="$DIR/bin"
LOG="${PAIR_MESSAGEIDLE_TRACE:-${TMPDIR:-/tmp}/pair-messageidle.tsv}"

case "${1:-}" in
arm)
  mkdir -p "$BIN" || exit 1
  (cd "$DIR/../.." && go build -o "$BIN/zellij" ./probes/messageidle && cp "$BIN/zellij" "$BIN/ps") || {
    echo "echo 'messageidle: build failed; not arming'"; exit 1; }
  : > "$LOG"
  echo "export PAIR_MESSAGEIDLE_TRACE='$LOG'"
  echo "export PATH='$BIN':\$PATH"
  echo "echo 'messageidle armed -> $LOG'" ;;

disarm)
  echo "unset PAIR_MESSAGEIDLE_TRACE"
  echo "export PATH=\$(printf '%s' \"\$PATH\" | tr ':' '\\n' | while IFS= read -r p; do [ \"\$p\" = '$BIN' ] || printf '%s:' \"\$p\"; done | sed 's/:\$//')"
  echo "echo 'messageidle disarmed'" ;;

window)
  secs="${2:-120}"; label="${3:-run}"
  pid="$(pgrep -x couch | head -1)"
  [ -n "$pid" ] || { echo "no couch process" >&2; exit 1; }
  [ -s "$LOG" ] || echo "warning: $LOG is empty — was couch launched from an armed shell?" >&2
  out="${TMPDIR:-/tmp}/pair-messageidle-$label"
  cpu() { ps -o cputime= -p "$pid" | awk -F'[:.]' '{ n=NF; s=$(n-1)+60*$(n-2); if (n>3) s+=3600*$(n-3); print s"."$n }'; }
  t0="$(python3 -c 'import time; print(time.time_ns())')"; c0="$(cpu)"
  sleep "$secs"
  t1="$(python3 -c 'import time; print(time.time_ns())')"; c1="$(cpu)"
  sample "$pid" 10 -file "$out.sample.txt" >/dev/null 2>&1 || echo "sample failed (try without sandbox)" >&2
  {
    echo "label=$label couch_pid=$pid window_s=$secs slots_attached=${SLOTS:-unset}"
    echo "couch_cpu_seconds=$(python3 -c "print(round($c1-$c0,2))")"
    python3 - "$LOG" "$t0" "$t1" "$secs" <<'PY'
import sys, collections
log, t0, t1, secs = sys.argv[1], int(sys.argv[2]), int(sys.argv[3]), float(sys.argv[4])
by = collections.Counter()
for line in open(log):
    p = line.rstrip("\n").split("\t", 2)
    if len(p) == 3 and t0 <= int(p[0]) <= t1:
        words = [w for w in p[2].split() if not w.startswith("-")]
        if p[1] == "zellij" and "--session" in p[2].split():
            a = p[2].split(); i = a.index("--session"); words = [w for w in a[i+2:] if not w.startswith("-")]
        by[p[1] + " " + " ".join(words[:2])] += 1
total = sum(by.values())
print(f"couch_spawns={total} per_minute={60*total/secs:.1f}")
for k, n in by.most_common():
    print(f"  {n:6d}  {k}")
PY
    echo "sample_file=$out.sample.txt"
  } | tee "$out.txt" ;;

*)
  echo "usage: $0 {arm|disarm|window [seconds] [label]}" >&2
  exit 2 ;;
esac
