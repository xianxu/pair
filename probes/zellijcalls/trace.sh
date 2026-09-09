#!/usr/bin/env bash
# Arm the zellij call tracer, then report what startup actually spent.
#
# #215 raised couch's registration deadline from 5s to 15s against a MEASURED
# 8.85s startup -- but nothing measured where those 8.85s go. nvim is 117ms
# (measured with --startuptime) and one `zellij list-sessions` is 43ms at 26
# sessions, so neither is the answer on its own. This counts the calls.
#
#   eval "$(probes/zellijcalls/trace.sh arm)"
#   couch                     # drive the thing you want to measure, then quit
#   probes/zellijcalls/trace.sh report
#   probes/zellijcalls/trace.sh disarm
set -uo pipefail

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LOG="${PAIR_ZELLIJ_TRACE:-${TMPDIR:-/tmp}/pair-zellij-trace.tsv}"

case "${1:-}" in
arm)
  : > "$LOG"
  # Printed for `eval` so the exports land in the CALLER's shell -- a child
  # process could not change the shell that launches couch.
  echo "export PAIR_ZELLIJ_TRACE='$LOG'"
  echo "export PATH='$DIR':\$PATH"
  echo "echo 'zellij tracing armed -> $LOG'" ;;

disarm)
  echo "unset PAIR_ZELLIJ_TRACE"
  echo "export PATH=\$(echo \"\$PATH\" | tr ':' '\\n' | grep -v '^$DIR\$' | paste -sd: -)"
  echo "echo 'zellij tracing disarmed'" ;;

report)
  if [ ! -s "$LOG" ]; then
    echo "no calls recorded in $LOG"
    echo "  (armed? run: eval \"\$($0 arm)\" in the shell that launches couch)"
    exit 1
  fi
  python3 - "$LOG" <<'PY'
import sys, collections
rows = []
for line in open(sys.argv[1]):
    parts = line.rstrip("\n").split("\t", 3)
    if len(parts) != 4:
        continue
    start, end, code, argv = parts
    rows.append((float(start), float(end), code, argv))
if not rows:
    sys.exit("no parseable rows")
rows.sort()
t0 = rows[0][0]
span = rows[-1][1] - t0
total = sum(e - s for s, e, _, _ in rows)

def kind(argv):
    # The SUBCOMMAND, not the whole argv: `action rename-pane` and
    # `action write-chars` are different costs and must not collapse together,
    # while the session name they carry must not split one kind into hundreds.
    w = argv.split()
    out = []
    for tok in w:
        if tok.startswith("-"):
            continue
        out.append(tok)
        if len(out) == 2:
            break
    return " ".join(out) or "(bare)"

by = collections.defaultdict(lambda: [0, 0.0])
for s, e, _, argv in rows:
    k = by[kind(argv)]
    k[0] += 1
    k[1] += e - s

print(f"calls          : {len(rows)}")
print(f"wall span      : {span:.2f}s   (first call start -> last call end)")
print(f"in-zellij time : {total:.2f}s   ({100*total/span:.0f}% of the span)")
print()
print(f"{'subcommand':<28}{'n':>5}{'total s':>10}{'mean ms':>10}")
for k, (n, t) in sorted(by.items(), key=lambda kv: -kv[1][1]):
    print(f"{k:<28}{n:>5}{t:>10.2f}{1000*t/n:>10.1f}")
print()
print("slowest 10 individual calls:")
for s, e, code, argv in sorted(rows, key=lambda r: r[1] - r[0], reverse=True)[:10]:
    print(f"  {e-s:6.2f}s  +{s-t0:6.2f}s  rc={code}  {argv[:88]}")
PY
  ;;
*)
  echo "usage: $0 {arm|report|disarm}" >&2
  exit 2 ;;
esac
