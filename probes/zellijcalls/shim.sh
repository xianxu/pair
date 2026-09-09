#!/usr/bin/env bash
# A `zellij` that records every invocation, then execs the real one.
#
# WHY A SHIM AND NOT INSTRUMENTED CODE. pair spawns zellij from eight distinct
# call sites (osruntime.go x6, session_quiescence.go x2), so instrumenting "the
# seam" means instrumenting eight of them and silently missing the ninth someone
# adds. The shim sits in PATH and cannot be bypassed by a call site it has never
# heard of -- the same reason #199's paneWriter is not an io.Writer.
#
# Records one TSV row per call: start offset, duration, exit code, argv.
# The real zellij is found by skipping this shim's own directory in PATH, so the
# shim cannot recurse into itself.
#
# Usage:
#   eval "$(probes/zellijcalls/trace.sh arm)"   # puts the shim first in PATH
#   couch                                        # or: pair resume <tag> --layout3
#   probes/zellijcalls/trace.sh report
set -uo pipefail

LOG="${PAIR_ZELLIJ_TRACE:-}"
SELF_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# The real zellij: first match in PATH that is not this shim.
real=""
IFS=: read -ra parts <<< "$PATH"
for d in "${parts[@]}"; do
  [ -n "$d" ] || continue
  [ "$(cd "$d" 2>/dev/null && pwd)" = "$SELF_DIR" ] && continue
  if [ -x "$d/zellij" ]; then real="$d/zellij"; break; fi
done
if [ -z "$real" ]; then
  echo "zellij-shim: no real zellij on PATH (excluding $SELF_DIR)" >&2
  exit 127
fi

if [ -z "$LOG" ]; then
  exec "$real" "$@"
fi

start=$(python3 -c 'import time;print("%.6f"%time.time())')
"$real" "$@"
code=$?
end=$(python3 -c 'import time;print("%.6f"%time.time())')
# Append atomically enough for a trace: one write, one line.
printf '%s\t%s\t%s\t%s\n' "$start" "$end" "$code" "$*" >> "$LOG"
exit $code
