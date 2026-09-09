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
BIN="$DIR/bin"
# A per-invocation trace file, not a fixed name: a predictable path in a shared
# /tmp is someone else's to clobber, and `arm` truncates what it points at.
LOG="${PAIR_ZELLIJ_TRACE:-${TMPDIR:-/tmp}/pair-zellij-trace.$$.tsv}"

case "${1:-}" in
arm)
  mkdir -p "$BIN" || exit 1
  # The shim must BE `zellij` on PATH. Built here so the trace can never run a
  # stale binary against changed source.
  (cd "$DIR/../.." && go build -o "$BIN/zellij" ./probes/zellijcalls) || {
    echo "echo 'zellijcalls: build failed; not arming'"; exit 1; }
  : > "$LOG"
  # Printed for `eval` so the exports land in the CALLER's shell -- a child
  # process could not change the shell that launches couch.
  echo "export PAIR_ZELLIJ_TRACE='$LOG'"
  echo "export PATH='$BIN':\$PATH"
  echo "echo 'zellij tracing armed -> $LOG  (PATH gains $BIN until you disarm)'" ;;

disarm)
  echo "unset PAIR_ZELLIJ_TRACE"
  # Exact string match on the path element, not a regex: a repo path containing
  # regex metacharacters would over-match.
  echo "export PATH=\$(printf '%s' \"\$PATH\" | tr ':' '\\n' | while IFS= read -r p; do [ \"\$p\" = '$BIN' ] || printf '%s:' \"\$p\"; done | sed 's/:\$//')"
  echo "echo 'zellij tracing disarmed'" ;;

overhead)
  (cd "$DIR/../.." && go build -o "$BIN/zellij" ./probes/zellijcalls) || exit 1
  PAIR_ZELLIJ_TRACE="" "$BIN/zellij" --overhead ;;

report)
  if [ ! -s "$LOG" ]; then
    echo "no calls recorded in $LOG"
    echo "  (armed? run: eval \"\$($0 arm)\" in the shell that launches couch)"
    exit 1
  fi
  python3 - "$LOG" <<'PY_REPORT'
import sys, collections, shlex
rows = []
for line in open(sys.argv[1]):
    parts = line.rstrip("\n").split("\t", 3)
    if len(parts) != 4:
        continue
    start_ns, dur_ns, code, argv = parts
    try:
        rows.append((int(start_ns) / 1e9, int(dur_ns) / 1e9, code, argv))
    except ValueError:
        continue
if not rows:
    sys.exit("no parseable rows")
rows.sort()
t0 = rows[0][0]
span = max(s + d for s, d, _, _ in rows) - t0
total = sum(d for _, d, _, _ in rows)

VALUE_FLAGS = {"--session", "-s", "--config-dir", "--config", "--layout",
               "--layout-dir", "--pane-id", "--new-session-with-layout"}

def kind(argv):
    # The SUBCOMMAND, not the whole argv: `action rename-pane` and
    # `action write-chars` are different costs and must not collapse together,
    # while the session name they carry must not split one kind into hundreds.
    try:
        words = shlex.split(argv)
    except ValueError:
        words = argv.split()
    out = []
    skip_next = False
    for tok in words:
        if skip_next:
            # A flag's VALUE. Without this, `--session <name> action ...` groups
            # by the session NAME and one kind becomes one row per session --
            # the exact split this function exists to prevent.
            skip_next = False
            continue
        if tok.startswith("-"):
            skip_next = "=" not in tok and tok in VALUE_FLAGS
            continue
        out.append(tok)
        if len(out) == 2:
            break
    return " ".join(out) or "(bare)"

by = collections.defaultdict(lambda: [0, 0.0])
for _, d, _, argv in rows:
    k = by[kind(argv)]
    k[0] += 1
    k[1] += d

print(f"calls          : {len(rows)}")
print(f"wall span      : {span:.2f}s   (first call start -> last call end)")
print(f"in-zellij time : {total:.2f}s   ({100*total/span:.0f}% of the span)")
print()
print(f"{'subcommand':<28}{'n':>5}{'total s':>10}{'mean ms':>10}")
for k, (n, t) in sorted(by.items(), key=lambda kv: -kv[1][1]):
    print(f"{k:<28}{n:>5}{t:>10.2f}{1000*t/n:>10.1f}")
print()
print("slowest 10 individual calls:")
for s, d, code, argv in sorted(rows, key=lambda r: r[1], reverse=True)[:10]:
    print(f"  {d:6.2f}s  +{s-t0:6.2f}s  rc={code}  {argv[:88]}")
PY_REPORT
  ;;
*)
  echo "usage: $0 {arm|report|disarm|overhead}" >&2
  exit 2 ;;
esac
